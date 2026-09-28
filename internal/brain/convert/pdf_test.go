package convert

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
)

const popplerV = "pdftotext version 25.07.0\nCopyright 2005-2025 The Poppler Developers - http://poppler.freedesktop.org\n"

const longText = "Hallo aus dem Pruefbestand. Diese Zeile testet die Extraktion aus einer " +
	"von Hand geschriebenen PDF-Datei, ohne Texterkennung und ohne fremde Erzeugerbibliothek."

// fakeTools answers pdftotext from a table keyed by the argument after the
// program, and counts the calls.
type fakeTools struct {
	missing  bool
	version  child.Result
	pages    map[string]child.Result
	calls    []child.Spec
	versions int
}

func (f *fakeTools) tools() Tools {
	return Tools{
		Look: func(name string) (string, error) {
			if f.missing {
				return "", exec.ErrNotFound
			}
			return `C:\bin\` + name + ".exe", nil
		},
		Run: func(spec child.Spec) child.Result {
			f.calls = append(f.calls, spec)
			if spec.Argv[1] == "-v" {
				f.versions++
				return f.version
			}
			return f.pages[spec.Argv[6]]
		},
	}
}

func poppler(pages map[string]child.Result) *fakeTools {
	return &fakeTools{version: child.Result{Stderr: popplerV}, pages: pages}
}

func ok(out string) child.Result { return child.Result{Stdout: out} }

func TestATextPDFYieldsItsWashedText(t *testing.T) {
	f := poppler(map[string]child.Result{"text.pdf": ok("   " + longText + "      Ende.\f")})
	e, err := newPDFToText(f.tools()).extract(filepath.Join(`C:\inbox`, "text.pdf"))
	if err != nil || e.pages != 1 || e.skipped != 0 || !strings.Contains(e.text, "Pruefbestand") || strings.Contains(e.text, "  ") {
		t.Fatalf("%+v %v", e, err)
	}
}

// Review Focus 5: the name goes relative, the inbox is the directory.
func TestPdftotextGetsTheNameAndTheInbox(t *testing.T) {
	f := poppler(map[string]child.Result{"Bericht März.pdf": ok(longText + "\f")})
	path := filepath.Join(`C:\Vault\00 Eingang`, "Bericht März.pdf")
	if _, err := newPDFToText(f.tools()).extract(path); err != nil {
		t.Fatal(err)
	}
	call := f.calls[len(f.calls)-1]
	want := []string{`C:\bin\pdftotext.exe`, "-layout", "-enc", "UTF-8", "-eol", "unix", "Bericht März.pdf", "-"}
	if strings.Join(call.Argv, "|") != strings.Join(want, "|") || call.Dir != `C:\Vault\00 Eingang` || call.Timeout != 2*time.Minute {
		t.Fatalf("%+v", call)
	}
}

func TestScanPagesAreCountedPerPage(t *testing.T) {
	umlauts := func(n int) string { return strings.Repeat("\U000000e4", n) }
	for _, c := range []struct {
		name, out            string
		text                 bool
		pages, skipped       int
		containsPruefbestand int
	}{
		{"blank", "\f", false, 1, 1, 0},
		// xpdf's answer to a PDF without pages; Poppler ends it with 99
		// (TestAPagelessPDFUnderPopplerIsUnreadable).
		{"nooutput", "", false, 0, 0, 0},
		{"mixed", strings.Repeat(longText+"\f", 3) + strings.Repeat("\f", 7), true, 10, 7, 3},
		{"allscan", "\f\f", false, 2, 2, 0},
		{"short", "zu kurz\f", false, 1, 1, 0},
		// The threshold counts code points, as len() of a str does: 60
		// umlauts are 120 bytes and still a scan.
		{"umlauts100", umlauts(100) + "\f", true, 1, 0, 0},
		{"umlauts99", umlauts(99) + "\f", false, 1, 1, 0},
		{"umlauts60", umlauts(60) + "\f", false, 1, 1, 0},
		// Python's space around the page counts for nothing, \x1f included.
		{"padded", "   \x1f" + umlauts(99) + "\U00003000\n\f", false, 1, 1, 0},
	} {
		f := poppler(map[string]child.Result{c.name + ".pdf": ok(c.out)})
		e, err := newPDFToText(f.tools()).extract(c.name + ".pdf")
		if err != nil || (e.text != "") != c.text || e.pages != c.pages || e.skipped != c.skipped ||
			strings.Count(e.text, "Pruefbestand") != c.containsPruefbestand {
			t.Errorf("%s: %+v %v", c.name, e, err)
		}
	}
}

func TestBlankLinesSeparateParagraphs(t *testing.T) {
	if got := wash("erste  Zeile\nzweite\t\tZeile\n\n\ndritte\n"); got != "erste Zeile zweite Zeile\n\ndritte" {
		t.Fatalf("%q", got)
	}
	if got := wash("erster Absatz\n\n"); got != "erster Absatz" {
		t.Fatalf("%q", got)
	}
}

func TestSplitPagesDropsTheLastFormFeed(t *testing.T) {
	if got := splitPages("a\fb\f"); len(got) != 2 || got[1] != "b" {
		t.Fatalf("%q", got)
	}
	if got := splitPages("a\fb"); len(got) != 2 {
		t.Fatalf("%q", got)
	}
	if got := splitPages(""); len(got) != 0 {
		t.Fatalf("%q", got)
	}
}

func TestAPDFPdftotextRefusesIsUnreadable(t *testing.T) {
	for name, res := range map[string]child.Result{
		"corrupt":  {Code: 1, Stderr: "Syntax Error: Couldn't find trailer dictionary\n"},
		"notstart": {Code: -1, Err: errors.New("boom")},
		"slow":     {Code: -1, TimedOut: true},
		"latin1":   {Stdout: "\xe4\f"},
	} {
		f := poppler(map[string]child.Result{name + ".pdf": res})
		_, err := newPDFToText(f.tools()).extract(name + ".pdf")
		var tool *toolError
		if err == nil || errors.As(err, &tool) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Poppler ends a PDF without pages with 99 and says why on stderr, with CRLF
// (measured): the PDF is unreadable, not a scan, and the CR stays out of the
// message.
func TestAPagelessPDFUnderPopplerIsUnreadable(t *testing.T) {
	f := poppler(map[string]child.Result{"pageless.pdf": {
		Code:   99,
		Stderr: "Syntax Error: Invalid page count 0\r\nCommand Line Error: Wrong page range given: the first page (1) can not be after the last page (0).\r\n",
	}})
	_, err := newPDFToText(f.tools()).extract("pageless.pdf")
	var tool *toolError
	if err == nil || errors.As(err, &tool) || err.Error() != "pdftotext exited 99: Syntax Error: Invalid page count 0" {
		t.Fatalf("%q", err)
	}
}

// Review Focus 3: Git Bash finds xpdf. Its -v ends with 99 and writes to
// stdout with CRLF (measured): the exit code decides nothing, the text does,
// and no CR lands in the message.
func TestOnlyPopplerIsTaken(t *testing.T) {
	f := &fakeTools{version: child.Result{Code: 99, Stdout: "pdftotext version 4.06 [www.xpdfreader.com]\r\nCopyright 1996-2025 Glyph & Cog, LLC\r\n"}}
	_, err := newPDFToText(f.tools()).extract("text.pdf")
	var tool *toolError
	if !errors.As(err, &tool) || !strings.Contains(err.Error(), "(pdftotext version 4.06 [www.xpdfreader.com]);") ||
		!strings.Contains(err.Error(), "winget install --id oschwartz10612.Poppler -e") {
		t.Fatal(err)
	}
}

// A -v that gave no answer is not the wrong program: the message says what
// happened instead of telling the user to install what they have.
func TestAVersionWithoutAnAnswerSaysWhy(t *testing.T) {
	for name, c := range map[string]struct {
		res  child.Result
		want string
	}{
		"notstart": {
			child.Result{Code: -1, Err: errors.Join(errors.New(`C:\bin\pdftotext.exe`), errors.New("Access is denied."))},
			`C:\bin\pdftotext.exe -v did not start: C:\bin\pdftotext.exe: Access is denied.; install Poppler with: winget install --id oschwartz10612.Poppler -e`,
		},
		"slow": {
			child.Result{Code: -1, TimedOut: true},
			`C:\bin\pdftotext.exe -v took longer than 2m0s; install Poppler with: winget install --id oschwartz10612.Poppler -e`,
		},
		// 0xC0000135: Windows ends a program whose DLL is missing with it,
		// before the program says anything.
		"silent": {
			child.Result{Code: 3221225781, Stderr: "\r\n"},
			`C:\bin\pdftotext.exe -v exited 3221225781 and said nothing; install Poppler with: winget install --id oschwartz10612.Poppler -e`,
		},
	} {
		f := &fakeTools{version: c.res}
		_, err := newPDFToText(f.tools()).extract("text.pdf")
		var tool *toolError
		if !errors.As(err, &tool) || err.Error() != c.want {
			t.Errorf("%s: %q", name, err)
		}
	}
}

func TestAPdftotextThatSaysNothingIsNamedSo(t *testing.T) {
	f := poppler(map[string]child.Result{"a.pdf": {Code: 1}, "b.pdf": {Code: -1, Err: errors.Join(errors.New(`C:\bin\pdftotext.exe`), errors.New("Access is denied."))}})
	p := newPDFToText(f.tools())
	if _, err := p.extract("a.pdf"); err == nil || err.Error() != "pdftotext exited 1 and said nothing" {
		t.Errorf("%q", err)
	}
	if _, err := p.extract("b.pdf"); err == nil || err.Error() != `pdftotext did not start: C:\bin\pdftotext.exe: Access is denied.` {
		t.Errorf("%q", err)
	}
}

func TestAMissingPdftotextNamesItsInstaller(t *testing.T) {
	f := &fakeTools{missing: true}
	_, err := newPDFToText(f.tools()).extract("text.pdf")
	var tool *toolError
	if !errors.As(err, &tool) || !strings.Contains(err.Error(), "pdftotext is not on PATH; install it with: winget install --id oschwartz10612.Poppler -e") {
		t.Fatal(err)
	}
}

func TestTheVersionIsAskedOncePerRun(t *testing.T) {
	f := poppler(map[string]child.Result{"a.pdf": ok(longText + "\f"), "b.pdf": ok(longText + "\f")})
	p := newPDFToText(f.tools())
	for _, name := range []string{"a.pdf", "b.pdf"} {
		if _, err := p.extract(name); err != nil {
			t.Fatal(err)
		}
	}
	if f.versions != 1 {
		t.Fatalf("asked %d times", f.versions)
	}
}

// The wrong program is found once as well: the second PDF gets the same
// answer without another -v.
func TestAWrongPdftotextIsAskedOncePerRun(t *testing.T) {
	f := &fakeTools{version: child.Result{Code: 99, Stdout: "pdftotext version 4.06 [www.xpdfreader.com]\r\n"}}
	p := newPDFToText(f.tools())
	_, first := p.extract("a.pdf")
	_, second := p.extract("b.pdf")
	var tool *toolError
	if !errors.As(first, &tool) || second != first || f.versions != 1 || len(f.calls) != 1 {
		t.Fatalf("%v %v, asked %d times, %d calls", first, second, f.versions, len(f.calls))
	}
}

func TestSystemToolsAreChildAndLookPath(t *testing.T) {
	tools := SystemTools()
	if tools.Run == nil || tools.Look == nil {
		t.Fatal(tools)
	}
}
