package convert

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/config"
)

// world is one machine: a state directory with a registry, and the areas
// below root.
type world struct {
	t        *testing.T
	root     string
	state    string
	registry strings.Builder
	tools    Tools
}

func newWorld(t *testing.T) *world {
	t.Helper()
	root := t.TempDir()
	state := filepath.Join(root, "state")
	if err := os.MkdirAll(state, 0o755); err != nil {
		t.Fatal(err)
	}
	w := &world{t: t, root: root, state: state, tools: poppler(nil).tools()}
	w.flush()
	return w
}

func (w *world) flush() {
	if err := os.WriteFile(filepath.Join(w.state, "registry.toml"), []byte(w.registry.String()), 0o644); err != nil {
		w.t.Fatal(err)
	}
}

// area registers scope; a declaration of "" writes none. It answers the
// area's directory.
func (w *world) area(scope, declaration string, readonly bool) string {
	dir := filepath.Join(w.root, strings.ReplaceAll(scope, "/", "-"))
	if err := os.MkdirAll(filepath.Join(dir, ".loomux"), 0o755); err != nil {
		w.t.Fatal(err)
	}
	if declaration != "" {
		if err := os.WriteFile(filepath.Join(dir, ".loomux", "config.toml"), []byte(declaration), 0o644); err != nil {
			w.t.Fatal(err)
		}
	}
	fmt.Fprintf(&w.registry, "[[area]]\nscope = %q\npath = %q\n", scope, filepath.ToSlash(dir))
	if readonly {
		w.registry.WriteString("readonly = true\n")
	}
	w.registry.WriteString("\n")
	w.flush()
	return dir
}

// inbox registers an area whose declaration names `00 Eingang`, with extra
// tables after it, and answers the inbox.
func (w *world) inbox(scope, extra string) string {
	dir := w.area(scope, fmt.Sprintf("[area]\nscope = %q\n\n[layout]\ninbox = \"00 Eingang\"\n\n%s", scope, extra), false)
	inbox := filepath.Join(dir, "00 Eingang")
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		w.t.Fatal(err)
	}
	return inbox
}

func (w *world) run() (Outcome, error) {
	w.t.Helper()
	areas, err := config.ReadRegistry(w.state)
	if err != nil {
		return Outcome{}, err
	}
	entries, err := Areas(areas, w.state, filepath.Join(w.root, "legacy"))
	if err != nil {
		return Outcome{}, err
	}
	return ConvertAll(context.Background(), w.tools, entries, w.state)
}

func put(t *testing.T, dir, name, data string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestATranscriptBecomesAMarkdownFile(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	put(t, inbox, "video (mHSOsy_usAg).txt", "[00:00] Hallo zusammen.\n")
	out, err := w.run()
	target := filepath.Join(inbox, "video (mHSOsy_usAg).txt.md")
	if err != nil || !slices.Equal(out.Written, []string{target}) || len(out.Skipped) != 0 {
		t.Fatalf("%+v %v", out, err)
	}
	text := read(t, target)
	for _, want := range []string{"converter: brain-transcript/1\n", "source_url: https://www.youtube.com/watch?v=mHSOsy_usAg\n", "asr: true\n", "---\n\n[00:00] Hallo zusammen.\n"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in %q", want, text)
		}
	}
}

func TestTheSecondRunWritesNothing(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	put(t, inbox, "video.txt", "[00:00] Hallo zusammen.\n")
	w.run()
	first := read(t, filepath.Join(inbox, "video.txt.md"))
	out, err := w.run()
	if err != nil || len(out.Written) != 0 || len(out.Skipped) != 0 || read(t, filepath.Join(inbox, "video.txt.md")) != first {
		t.Fatalf("%+v %v", out, err)
	}
}

// retrieved is the source's modification day in UTC, never "now" and never
// the local day. The file system keeps no zone, so the machine's own zone is
// what would leak in: under UTC+14 12:30Z on the 2nd is already the 3rd.
func TestRetrievedIsTheSourcesDayInUTC(t *testing.T) {
	saved := time.Local
	t.Cleanup(func() { time.Local = saved })
	time.Local = time.FixedZone("far", 14*3600)
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	source := put(t, inbox, "video.txt", "[00:00] Hallo.\n")
	noon := time.Date(2026, 1, 2, 12, 30, 0, 0, time.UTC)
	if err := os.Chtimes(source, noon, noon); err != nil {
		t.Fatal(err)
	}
	if _, err := w.run(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(read(t, source+".md"), "retrieved: 2026-01-02\n") {
		t.Fatal(read(t, source+".md"))
	}
}

func TestTwoSourcesWithTheSameStemBothSurvive(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	w.tools = poppler(map[string]child.Result{"doku.pdf": ok(longText + "\f")}).tools()
	put(t, inbox, "doku.pdf", "%PDF-1.4\n")
	put(t, inbox, "doku.txt", "[00:00] Aus dem Transkript.\n")
	if out, err := w.run(); err != nil || len(out.Written) != 2 {
		t.Fatalf("%+v %v", out, err)
	}
	pdf := read(t, filepath.Join(inbox, "doku.pdf.md"))
	if !strings.Contains(pdf, "Pruefbestand") || !strings.Contains(pdf, "converter: brain-pdf/2\n") || !strings.Contains(pdf, "asr: false\n") {
		t.Fatal(pdf)
	}
	if !strings.Contains(read(t, filepath.Join(inbox, "doku.txt.md")), "Aus dem Transkript.") {
		t.Fatal("the transcript's target")
	}
}

func TestWhatIsLeftBehindIsReportedAndTheRunGoesOn(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	w.tools = poppler(map[string]child.Result{
		"scan.pdf": ok("\f"),
		"leer.pdf": ok(""),
		"teil.pdf": ok(strings.Repeat(longText+"\f", 3) + strings.Repeat("\f", 7)),
		// pdftotext writes its errors with CRLF under Windows (measured).
		"kaputt.pdf": {Code: 1, Stderr: "Syntax Error\r\nSyntax Error: Couldn't read xref table\r\n"},
	}).tools()
	put(t, inbox, "notiz.txt", "Nur Prosa.\n")
	put(t, inbox, "kaputt.txt", "[00:00] Anfang ist sauber.\n"+strings.Repeat("x", 8300)+"\n"+strings.Repeat("\xff\xfe", 50))
	put(t, inbox, "hand.txt", "[00:00] Hallo.\n")
	put(t, inbox, "hand.txt.md", "---\ntitle: Meine Notiz\n---\n\nHandarbeit.\n")
	put(t, inbox, "bytes.txt", "[00:00] Hallo.\n")
	put(t, inbox, "bytes.txt.md", strings.Repeat("\xff\xfe", 8192))
	for _, name := range []string{"scan.pdf", "leer.pdf", "teil.pdf", "kaputt.pdf"} {
		put(t, inbox, name, "%PDF-1.4\n")
	}
	put(t, inbox, "video.txt", "[00:00] Hallo.\n")
	out, err := w.run()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"notiz.txt: no converter knows this format",
		"kaputt.txt: cannot be read as UTF-8 text",
		"hand.txt.md: not written by us, left untouched",
		"bytes.txt.md: cannot be read as UTF-8 text",
		"scan.pdf: no extractable text, looks like a scan",
		"leer.pdf: no pages to extract",
		"teil.pdf: 7 page(s) skipped as scanned",
		"kaputt.pdf: cannot be read as a PDF (pdftotext exited 1: Syntax Error)",
	} {
		if !slices.Contains(out.Skipped, want) {
			t.Errorf("no %q in %q", want, out.Skipped)
		}
	}
	if !slices.Contains(out.Written, filepath.Join(inbox, "video.txt.md")) || !slices.Contains(out.Written, filepath.Join(inbox, "teil.pdf.md")) {
		t.Fatalf("%q", out.Written)
	}
	if read(t, filepath.Join(inbox, "hand.txt.md")) != "---\ntitle: Meine Notiz\n---\n\nHandarbeit.\n" {
		t.Fatal("a hand-written file was touched")
	}
}

// Review Focus 3: xpdf leaves every PDF for a person; the transcript still
// goes. One pdftotext serves the whole run, so -v is asked once for both.
func TestXpdfLeavesThePDFsAndTheTranscriptGoes(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	f := &fakeTools{version: child.Result{Code: 99, Stdout: "pdftotext version 4.06 [www.xpdfreader.com]\r\n"}}
	w.tools = f.tools()
	put(t, inbox, "buch.pdf", "%PDF-1.4\n")
	put(t, inbox, "heft.pdf", "%PDF-1.4\n")
	put(t, inbox, "video.txt", "[00:00] Hallo.\n")
	out, _ := w.run()
	if len(out.Skipped) != 2 || f.versions != 1 {
		t.Fatalf("%q, -v asked %d times", out.Skipped, f.versions)
	}
	// The whole line: a wrong program is the tool's message, never "cannot
	// be read as a PDF".
	for i, name := range []string{"buch.pdf", "heft.pdf"} {
		want := name + `: C:\bin\pdftotext.exe is not Poppler's pdftotext (pdftotext version 4.06 [www.xpdfreader.com]); install Poppler with: winget install --id oschwartz10612.Poppler -e`
		if out.Skipped[i] != want {
			t.Errorf("%q", out.Skipped[i])
		}
		if _, err := os.Stat(filepath.Join(inbox, name+".md")); err == nil {
			t.Errorf("%s: a PDF target was written", name)
		}
	}
	if !slices.Equal(out.Written, []string{filepath.Join(inbox, "video.txt.md")}) {
		t.Fatalf("%q", out.Written)
	}
}

// Windows refuses to replace a read-only file (measured with Go 1.27: access
// denied); POSIX asks the directory, and the rename would go through.
func TestAnUnwritableTargetIsReported(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("a read-only file blocks its replacement only under Windows")
	}
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	put(t, inbox, "a.txt", "[00:00] Hallo.\n")
	target := put(t, inbox, "a.txt.md", "---\nsource_url:\nretrieved: 2020-01-01\nconverter: brain-transcript/1\nasr: true\n---\n\nAlt.\n")
	if err := os.Chmod(target, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(target, 0o644) })
	put(t, inbox, "b.txt", "[00:00] Welt.\n")
	out, _ := w.run()
	if len(out.Skipped) != 1 || !strings.HasPrefix(out.Skipped[0], "a.txt.md: cannot be written (") {
		t.Fatalf("%q", out.Skipped)
	}
	if !slices.Equal(out.Written, []string{filepath.Join(inbox, "b.txt.md")}) {
		t.Fatalf("%q", out.Written)
	}
}

func TestAnUnreadableTimestampIsReported(t *testing.T) {
	saved := statSource
	t.Cleanup(func() { statSource = saved })
	statSource = func(string) (os.FileInfo, error) { return nil, os.ErrPermission }
	source := put(t, t.TempDir(), "video.txt", "[00:00] Hallo.\n")
	target, message := ConvertFile(context.Background(), poppler(nil).tools(), source)
	if target != "" || !strings.HasPrefix(message, "video.txt: cannot be read (") {
		t.Fatalf("%q %q", target, message)
	}
}

func TestAreasWithoutAWritableInboxAreLeftAlone(t *testing.T) {
	w := newWorld(t)
	w.area("project/x", "", false)
	w.area("project/y", "[area]\nscope = \"project/y\"\n", false)
	// A readonly area's declaration lies where ResolvedAreaDir reads it,
	// under the state directory; in the area it would leave corpus without
	// an inbox, and the test would pass for that reason.
	corpus := w.area("corpus", "", true)
	declared := filepath.Join(w.state, "areas", "corpus", ".loomux")
	if err := os.MkdirAll(declared, 0o755); err != nil {
		t.Fatal(err)
	}
	put(t, declared, "config.toml", "[area]\nscope = \"corpus\"\n\n[layout]\ninbox = \"00 Eingang\"\n")
	if err := os.MkdirAll(filepath.Join(corpus, "00 Eingang"), 0o755); err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(corpus, "00 Eingang"), "video.txt", "[00:00] Hallo.\n")
	gone := w.area("gone", "[area]\nscope = \"gone\"\n\n[layout]\ninbox = \"00 Eingang\"\n", false)
	registered, err := config.ReadRegistry(w.state)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := Areas(registered, w.state, filepath.Join(w.root, "legacy"))
	if i := slices.IndexFunc(entries, func(a Area) bool { return a.Scope == "corpus" }); err != nil || i < 0 || entries[i].Inbox == "" {
		t.Fatalf("corpus has no inbox, so ReadOnly is not what leaves it alone: %+v %v", entries, err)
	}
	out, err := w.run()
	if err != nil || len(out.Written)+len(out.Skipped) != 0 {
		t.Fatalf("%+v %v", out, err)
	}
	if _, err := os.Stat(filepath.Join(corpus, "00 Eingang", "video.txt.md")); err == nil {
		t.Fatal("a readonly area was written into")
	}
	if _, err := os.Stat(filepath.Join(gone, "00 Eingang")); err == nil {
		t.Fatal("an inbox was made")
	}
}

func TestMarkdownAndDirectoriesInTheInboxAreNoSources(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	put(t, inbox, "fertig.MD", "[00:00] Hallo.\n")
	if err := os.Mkdir(filepath.Join(inbox, "ordner.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := w.run(); err != nil || len(out.Written)+len(out.Skipped) != 0 {
		t.Fatalf("%+v %v", out, err)
	}
}

// Review Focus 1: the order is the reference's under Windows, by bytes
// elsewhere.
func TestTheInboxIsWalkedInThePlatformsOrder(t *testing.T) {
	names := []string{"B.txt", "a.txt", "_z.txt", "b.pdf"}
	if got := sortedOn("windows", slices.Clone(names)); !slices.Equal(got, []string{"_z.txt", "a.txt", "b.pdf", "B.txt"}) {
		t.Fatalf("windows: %q", got)
	}
	if got := sortedOn("linux", slices.Clone(names)); !slices.Equal(got, []string{"B.txt", "_z.txt", "a.txt", "b.pdf"}) {
		t.Fatalf("linux: %q", got)
	}
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	put(t, inbox, "B.txt", "[00:00] Zwei.\n")
	put(t, inbox, "a.txt", "[00:00] Eins.\n")
	want := []string{filepath.Join(inbox, "a.txt.md"), filepath.Join(inbox, "B.txt.md")}
	if runtime.GOOS != "windows" {
		slices.Reverse(want)
	}
	if out, err := w.run(); err != nil || !slices.Equal(out.Written, want) {
		t.Fatalf("%q %v", out.Written, err)
	}
}

// Review Focus 2.
func TestACRLFTranscriptConvertsAsLF(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	put(t, inbox, "video.txt", "00:00:00 - 00:00:05\r\nHallo zusammen.\r\n\r\n00:00:06 - 00:00:09\r\nUnd weiter.\r\n")
	w.run()
	text := read(t, filepath.Join(inbox, "video.txt.md"))
	if strings.Contains(text, "\r") || !strings.Contains(text, "[00:00] Hallo zusammen. Und weiter.\n") {
		t.Fatalf("%q", text)
	}
}

func TestABrokenDeclarationStopsTheRunBeforeAnythingIsWritten(t *testing.T) {
	w := newWorld(t)
	inbox := w.inbox("knowledge", "")
	put(t, inbox, "video.txt", "[00:00] Hallo.\n")
	w.inbox("project/x", "[privacy]\nmode = \"cloud\"\n")
	_, err := w.run()
	if err == nil || !strings.Contains(err.Error(), "mode") {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(inbox, "video.txt.md")); err == nil {
		t.Fatal("a file was written before the refusal")
	}
}

// An absolute inbox would leave the area tree; the declaration is refused
// like any broken one, before a file is touched.
func TestAnAbsoluteInboxInTheDeclarationStopsTheRun(t *testing.T) {
	w := newWorld(t)
	elsewhere := filepath.ToSlash(filepath.Join(w.root, "elsewhere"))
	w.area("knowledge", fmt.Sprintf("[area]\nscope = \"knowledge\"\n\n[layout]\ninbox = %q\n", elsewhere), false)
	if _, err := w.run(); err == nil || !strings.Contains(err.Error(), "[layout] inbox must be relative to the area") {
		t.Fatal(err)
	}
}

// An inbox that passes as a directory but cannot be listed stops the run;
// what an earlier inbox wrote is still listed.
func TestAnInboxThatCannotBeListedStopsTheRun(t *testing.T) {
	w := newWorld(t)
	first := w.inbox("knowledge", "")
	put(t, first, "video.txt", "[00:00] Hallo.\n")
	closed := w.inbox("project/x", "")
	put(t, closed, "video.txt", "[00:00] Hallo.\n")
	// Only the listing fails: isDir still sees a directory, so the inbox is
	// not left out but stops the run.
	saved := readDir
	t.Cleanup(func() { readDir = saved })
	readDir = func(name string) ([]os.DirEntry, error) {
		if name == closed {
			return nil, fs.ErrPermission
		}
		return saved(name)
	}
	out, err := w.run()
	if !errors.Is(err, fs.ErrPermission) || !slices.Equal(out.Written, []string{filepath.Join(first, "video.txt.md")}) {
		t.Fatalf("%+v %v", out, err)
	}
}

func TestOneNamedFileConvertsOnItsOwn(t *testing.T) {
	source := put(t, t.TempDir(), "video.txt", "[00:00] Hallo.\n")
	target, message := ConvertFile(context.Background(), poppler(nil).tools(), source)
	if target != source+".md" || message != "" || strings.Contains(read(t, target), "description") {
		t.Fatalf("%q %q", target, message)
	}
}

func TestWriteIfChangedLeavesAnEqualFile(t *testing.T) {
	path := put(t, t.TempDir(), "x.md", "same\n")
	if written, err := writeIfChanged(path, "same\n"); written || err != nil {
		t.Fatal(written, err)
	}
	if written, err := writeIfChanged(path, "other\n"); !written || err != nil || read(t, path) != "other\n" {
		t.Fatal(written, err)
	}
}

// The comparison is byte for byte, as `newline=""` reads: a target saved with
// CRLF is rewritten with LF, not taken as equal.
func TestWriteIfChangedComparesTheRawBytes(t *testing.T) {
	path := put(t, t.TempDir(), "x.md", "same\r\n")
	if written, err := writeIfChanged(path, "same\n"); !written || err != nil || read(t, path) != "same\n" {
		t.Fatal(written, err)
	}
}
