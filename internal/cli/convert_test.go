package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/convert"
	"github.com/xidus90/loomux/internal/child"
)

// convertWorld is a state directory with one area `knowledge` whose inbox is
// `00 Eingang`, the process pointed at it and at an empty working directory.
func convertWorld(t *testing.T) (state, inbox string) {
	t.Helper()
	root := t.TempDir()
	state = filepath.Join(root, "state")
	area := filepath.Join(root, "vault")
	inbox = filepath.Join(area, "00 Eingang")
	for _, dir := range []string{state, inbox, filepath.Join(area, ".loomux")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(area, ".loomux", "config.toml"), "[area]\nscope = \"knowledge\"\n\n[layout]\ninbox = \"00 Eingang\"\n")
	writeFile(t, filepath.Join(state, "registry.toml"), "[[area]]\nscope = \"knowledge\"\npath = \""+filepath.ToSlash(area)+"\"\n")
	t.Setenv("LOOMUX_STATE_DIR", state)
	t.Chdir(t.TempDir())
	return state, inbox
}

// noPDFTools stands for a machine whose pdftotext is never asked.
func noPDFTools(t *testing.T) {
	t.Helper()
	saved := convertTools
	t.Cleanup(func() { convertTools = saved })
	convertTools = convert.Tools{
		Look: func(string) (string, error) { t.Fatal("pdftotext was looked up"); return "", nil },
		Run:  func(child.Spec) child.Result { t.Fatal("a program was started"); return child.Result{} },
	}
}

func TestConvertNamesWhatItWroteAndExitsZero(t *testing.T) {
	_, inbox := convertWorld(t)
	noPDFTools(t)
	writeFile(t, filepath.Join(inbox, "video.txt"), "[00:00] Hallo.\n")
	code, out, errOut := run("convert")
	if code != 0 || out != filepath.Join(inbox, "video.txt.md")+"\n" || errOut != "" {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
	if code, out, _ := run("convert"); code != 0 || out != "" {
		t.Fatalf("the second run: %d %q", code, out)
	}
}

func TestConvertListsWhatIsLeftOnStderrAndExitsOne(t *testing.T) {
	_, inbox := convertWorld(t)
	noPDFTools(t)
	writeFile(t, filepath.Join(inbox, "notiz.txt"), "Nur Prosa.\n")
	writeFile(t, filepath.Join(inbox, "video.txt"), "[00:00] Hallo.\n")
	code, out, errOut := run("convert")
	if code != 1 || !strings.HasSuffix(out, "video.txt.md\n") || errOut != "skipped: notiz.txt: no converter knows this format\n" {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}

func TestConvertOfOneFileNeedsNoRegistry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", filepath.Join(dir, "nowhere"))
	t.Chdir(dir)
	noPDFTools(t)
	writeFile(t, filepath.Join(dir, "video.txt"), "[00:00] Hallo.\n")
	if code, out, errOut := run("convert", "video.txt"); code != 0 || out != "video.txt.md\n" || errOut != "" {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}

func TestConvertWithoutARegistryIsOneErrorLine(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", dir)
	t.Chdir(dir)
	if code, _, errOut := run("convert"); code != 1 || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("%d %q", code, errOut)
	}
}

func TestConvertStopsAtABrokenModelBlock(t *testing.T) {
	state, inbox := convertWorld(t)
	writeFile(t, filepath.Join(state, "config.toml"), "[model]\nenabled = 5\n")
	writeFile(t, filepath.Join(inbox, "video.txt"), "[00:00] Hallo.\n")
	if code, out, errOut := run("convert"); code != 1 || out != "" || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}

func TestConvertStopsAtABrokenDeclaration(t *testing.T) {
	_, inbox := convertWorld(t)
	writeFile(t, filepath.Join(filepath.Dir(inbox), ".loomux", "config.toml"), "[area]\nscope = \"knowledge\"\n\n[privacy]\nmode = \"cloud\"\n")
	writeFile(t, filepath.Join(inbox, "video.txt"), "[00:00] Hallo.\n")
	if code, out, errOut := run("convert"); code != 1 || out != "" || !strings.Contains(errOut, "mode") {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}

// An inbox that passes as a directory but cannot be listed stops the run.
// What the inboxes before it wrote and left is still printed, and the error
// line comes after it: nothing already written goes unlisted.
func TestConvertListsWhatItDidBeforeAnInboxThatCannotBeListed(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the listing is denied through icacls")
	}
	state, first := convertWorld(t)
	noPDFTools(t)
	writeFile(t, filepath.Join(first, "notiz.txt"), "Nur Prosa.\n")
	writeFile(t, filepath.Join(first, "video.txt"), "[00:00] Hallo.\n")
	second := filepath.Join(filepath.Dir(filepath.Dir(first)), "project")
	closed := filepath.Join(second, "00 Eingang")
	if err := os.MkdirAll(closed, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(second, ".loomux", "config.toml"), "[area]\nscope = \"project/x\"\n\n[layout]\ninbox = \"00 Eingang\"\n")
	registry := filepath.Join(state, "registry.toml")
	before, err := os.ReadFile(registry)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, registry, string(before)+"\n[[area]]\nscope = \"project/x\"\npath = \""+filepath.ToSlash(second)+"\"\n")
	// Only the listing is denied: the run must still see a directory, or the
	// inbox would be left out instead of failing.
	user := os.Getenv("USERNAME")
	if said, err := exec.Command("icacls", closed, "/deny", user+":(RD)").CombinedOutput(); err != nil {
		t.Fatalf("%s %v", said, err)
	}
	t.Cleanup(func() { exec.Command("icacls", closed, "/remove:d", user).Run() })
	if _, err := os.ReadDir(closed); err == nil {
		t.Skip("this account lists the directory despite the deny, e.g. an administrator on CI")
	}
	code, out, errOut := run("convert")
	skipped := "skipped: notiz.txt: no converter knows this format\n"
	if code != 1 || out != filepath.Join(first, "video.txt.md")+"\n" || !strings.HasPrefix(errOut, skipped+"error: ") || strings.Count(errOut, "\n") != 2 {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}

func TestConvertTakesOnePathAtMost(t *testing.T) {
	convertWorld(t)
	if code, _, errOut := run("convert", "a", "b"); code != 2 || !strings.Contains(errOut, "unrecognized arguments: b") {
		t.Fatalf("%d %q", code, errOut)
	}
	if code, _, _ := run("convert", "--nope"); code != 2 {
		t.Fatal(code)
	}
}

// The reference's convert and fetch take --channel and --state-dir from
// _add_common (cli.py:450-452); loomux takes neither, a row of the parity
// list: neither command hands anything out, and the state comes from
// LOOMUX_STATE_DIR. Both end before anything is looked up or started.
func TestConvertAndFetchTakeNoChannelAndNoStateDir(t *testing.T) {
	convertWorld(t)
	noYtdlp(t)
	url := "https://www.youtube.com/watch?v=mHSOsy_usAg"
	for _, args := range [][]string{
		{"convert", "--channel", "local"},
		{"convert", "--state-dir", "x"},
		{"fetch", "--channel", "local", url},
		{"fetch", "--state-dir", "x", url},
	} {
		want := "flag provided but not defined: " + strings.TrimPrefix(args[1], "-")
		if code, out, errOut := run(args...); code != 2 || out != "" || !strings.HasPrefix(errOut, want) {
			t.Errorf("%q: %d %q %q", args, code, out, errOut)
		}
	}
}

func TestConvertRefusesWhereTheProjectSwitchedTheBrainOff(t *testing.T) {
	convertWorld(t)
	project := t.TempDir()
	os.MkdirAll(filepath.Join(project, ".loomux"), 0o755)
	writeFile(t, filepath.Join(project, ".loomux", "config.toml"), "[modules]\nbrain = false\n")
	t.Chdir(project)
	code, _, errOut := run("convert")
	if code != 1 || !strings.Contains(errOut, "[modules] brain = false") || !strings.Contains(errOut, filepath.Join(project, ".loomux", "config.toml")) {
		t.Fatalf("%d %q", code, errOut)
	}
}

func TestConvertReportsAnUnreadableModulesTable(t *testing.T) {
	convertWorld(t)
	project := t.TempDir()
	os.MkdirAll(filepath.Join(project, ".loomux"), 0o755)
	writeFile(t, filepath.Join(project, ".loomux", "config.toml"), "[modules\n")
	t.Chdir(project)
	if code, _, errOut := run("convert"); code != 1 || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("%d %q", code, errOut)
	}
}

func TestConvertRunsWhereTheProjectKeepsTheBrainOn(t *testing.T) {
	_, inbox := convertWorld(t)
	noPDFTools(t)
	project := t.TempDir()
	writeFile(t, filepath.Join(project, ".loomux", "config.toml"), "[modules]\nbrain = true\n")
	t.Chdir(project)
	writeFile(t, filepath.Join(inbox, "video.txt"), "[00:00] Hallo.\n")
	if code, out, errOut := run("convert"); code != 0 || out != filepath.Join(inbox, "video.txt.md")+"\n" || errOut != "" {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}

func TestConvertOfOneFileListsWhatIsLeft(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", filepath.Join(dir, "nowhere"))
	t.Chdir(dir)
	noPDFTools(t)
	writeFile(t, filepath.Join(dir, "notiz.txt"), "Nur Prosa.\n")
	if code, out, errOut := run("convert", "notiz.txt"); code != 1 || out != "" || errOut != "skipped: notiz.txt: no converter knows this format\n" {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}

// A suggestion stands on stdout after the written paths, and it is no work
// left for a person: the exit code stays 0.
func TestConvertNamesWhereAWrittenFileBelongs(t *testing.T) {
	state, inbox := convertWorld(t)
	noPDFTools(t)
	writeFile(t, filepath.Join(state, "config.toml"), "[model]\nenabled = true\nendpoint = \"http://127.0.0.1:11435\"\nroles = { place = true }\n")
	writeFile(t, filepath.Join(state, "ollama-fixture.json"), `{"response": "{\"scope\": \"knowledge\", \"grund\": \"Es passt.\"}"}`+"\n")
	serveFakeOllama(t, state)
	writeFile(t, filepath.Join(inbox, "video.txt"), "[00:00] Hallo.\n")
	code, out, errOut := run("convert")
	want := filepath.Join(inbox, "video.txt.md") + "\nsuggested: video.txt.md: belongs in knowledge, left in the inbox\n"
	if code != 0 || out != want || errOut != "" {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}

// fakeYtdlp stands for a yt-dlp that leaves the given files in its working
// directory and exits with 0.
func fakeYtdlp(t *testing.T, files map[string]string) {
	t.Helper()
	saved := convertTools
	t.Cleanup(func() { convertTools = saved })
	convertTools = convert.Tools{
		Look: func(name string) (string, error) { return name, nil },
		Run: func(spec child.Spec) child.Result {
			for name, text := range files {
				writeFile(t, filepath.Join(spec.Dir, name), text)
			}
			return child.Result{}
		},
	}
}

func TestFetchWritesIntoTheScopedInbox(t *testing.T) {
	_, inbox := convertWorld(t)
	fakeYtdlp(t, map[string]string{
		"v.info.json": `{"title": "Der echte Weg", "subtitles": {"de": [{"ext": "json3"}]}}`,
		"v.de.json3":  `{"events": [{"tStartMs": 0, "segs": [{"utf8": "Hallo."}]}]}`,
	})
	code, out, errOut := run("fetch", "https://www.youtube.com/watch?v=mHSOsy_usAg", "--scope", "knowledge")
	want := filepath.Join(inbox, "Der echte Weg (mHSOsy_usAg).txt")
	if code != 0 || out != want+"\n" || errOut != "" {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}

func TestFetchRefusesAnAreaItCannotWriteInto(t *testing.T) {
	state, _ := convertWorld(t)
	bare := t.TempDir()
	corpus := t.TempDir()
	os.MkdirAll(filepath.Join(state, "areas", "corpus", ".loomux"), 0o755)
	writeFile(t, filepath.Join(state, "areas", "corpus", ".loomux", "config.toml"), "[area]\nscope = \"corpus\"\n\n[layout]\ninbox = \"00 Eingang\"\n")
	registry := filepath.Join(state, "registry.toml")
	data, _ := os.ReadFile(registry)
	// archive is read-only and declares no inbox: read-only is the answer
	// that holds whatever its manifest says, so it comes first.
	archive := t.TempDir()
	writeFile(t, registry, string(data)+"\n[[area]]\nscope = \"project/x\"\npath = \""+filepath.ToSlash(bare)+"\"\n\n[[area]]\nscope = \"corpus\"\npath = \""+filepath.ToSlash(corpus)+"\"\nreadonly = true\n\n[[area]]\nscope = \"archive\"\npath = \""+filepath.ToSlash(archive)+"\"\nreadonly = true\n")
	fakeYtdlp(t, nil)
	for scope, want := range map[string]string{
		"knowledge2": "no area named 'knowledge2' in the registry",
		"project/x":  "area 'project/x' declares no inbox",
		"corpus":     "area 'corpus' is read-only",
		"archive":    "area 'archive' is read-only",
	} {
		if code, _, errOut := run("fetch", "https://x", "--scope", scope); code != 1 || !strings.Contains(errOut, want) {
			t.Errorf("%s: %d %q", scope, code, errOut)
		}
	}
}

// noYtdlp stands for a machine whose yt-dlp is never asked.
func noYtdlp(t *testing.T) {
	t.Helper()
	saved := convertTools
	t.Cleanup(func() { convertTools = saved })
	convertTools = convert.Tools{
		Look: func(string) (string, error) { t.Fatal("yt-dlp was looked up"); return "", nil },
		Run:  func(child.Spec) child.Result { t.Fatal("yt-dlp was started"); return child.Result{} },
	}
}

// The usage errors speak argparse's words, as the reference's do, and end
// before anything is looked up or started.
func TestFetchNeedsExactlyOneURL(t *testing.T) {
	convertWorld(t)
	noYtdlp(t)
	for args, want := range map[string]string{
		"fetch":             "loomux fetch: the following arguments are required: url\n",
		"fetch a b c":       "loomux fetch: unrecognized arguments: b c\n",
		"fetch --nope a":    "flag provided but not defined: -nope",
		"fetch -- --plugin": "loomux fetch: a URL does not begin with '-': --plugin\n",
		"fetch -":           "loomux fetch: a URL does not begin with '-': -\n",
	} {
		if code, _, errOut := run(strings.Fields(args)...); code != 2 || !strings.HasPrefix(errOut, want) {
			t.Errorf("%q: %d %q", args, code, errOut)
		}
	}
}

// A URL that begins with a dash would reach yt-dlp as an option, after the
// terminator too: `--` ends loomux's flags, not yt-dlp's.
func TestFetchPassesNoOptionToYtdlp(t *testing.T) {
	_, inbox := convertWorld(t)
	noYtdlp(t)
	if code, out, errOut := run("fetch", "--", "--plugin-dirs=x"); code != 2 || out != "" || errOut != "loomux fetch: a URL does not begin with '-': --plugin-dirs=x\n" {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
	if entries, err := os.ReadDir(inbox); err != nil || len(entries) != 0 {
		t.Fatalf("%v %v", entries, err)
	}
}

// `--no-playlist` narrows a watch URL that carries `list=` to its video; a
// URL that names only a playlist it does not narrow, and yt-dlp would walk
// every entry into the same names. Such a URL ends before yt-dlp is looked
// up, on every host YouTube serves the playlist page from. A watch URL with
// `list=` and no video is one too: yt-dlp sends it to the playlist page.
func TestFetchRefusesAURLThatNamesOnlyAPlaylist(t *testing.T) {
	_, inbox := convertWorld(t)
	noYtdlp(t)
	for _, url := range []string{
		"https://www.youtube.com/playlist?list=PLx",
		"https://youtube.com/playlist?list=PLx",
		"https://m.youtube.com/playlist?list=PLx",
		"https://music.youtube.com/playlist?list=PLx",
		"http://WWW.YouTube.com:443/playlist/?list=PLx",
		"www.youtube.com/playlist?list=PLx",
		"https://www.youtube.com/watch?list=PLx",
		"https://m.youtube.com/watch/?feature=share&list=PLx",
		"https://www.youtube.com/watch?v=&list=PLx",
		"https://www.youtube.com/watch?list=&list=PLx",
		"youtube.com/watch?list=PLx&next=https://x",
	} {
		want := "loomux fetch: a URL that names only a playlist is not fetched, give the URL of one video: " + url + "\n"
		if code, out, errOut := run("fetch", url); code != 2 || out != "" || errOut != want {
			t.Errorf("%s: %d %q %q", url, code, out, errOut)
		}
	}
	if entries, err := os.ReadDir(inbox); err != nil || len(entries) != 0 {
		t.Fatalf("%v %v", entries, err)
	}
}

// What `--no-playlist` narrows to one video goes on to yt-dlp, and so does a
// playlist page on a host that is not YouTube's. A watch URL without a
// playlist is yt-dlp's to refuse.
func TestFetchTakesAVideoURLThatAlsoNamesAPlaylist(t *testing.T) {
	_, inbox := convertWorld(t)
	fakeYtdlp(t, map[string]string{
		"v.info.json": `{"title": "Der echte Weg", "subtitles": {"de": [{"ext": "json3"}]}}`,
		"v.de.json3":  `{"events": [{"tStartMs": 0, "segs": [{"utf8": "Hallo."}]}]}`,
	})
	for _, url := range []string{
		"https://www.youtube.com/watch?v=mHSOsy_usAg&list=PLx",
		"https://www.youtube.com/watch?list=PLx&v=mHSOsy_usAg",
		"https://www.youtube.com/watch?v=mHSOsy_usAg&list=",
		"https://www.youtube.com/watch?v=&v=mHSOsy_usAg&list=PLx",
		"https://example.com/watch?list=PLx",
		"https://www.youtube.com/watch?feature=share",
		"https://youtu.be/mHSOsy_usAg?list=PLx",
		"https://example.com/playlist?list=PLx",
		"https://www.youtube.com/playlists?list=PLx",
		"https://[youtube.com/playlist?list=PLx",
	} {
		if code, out, errOut := run("fetch", url); code != 0 || !strings.HasPrefix(out, inbox) || errOut != "" {
			t.Errorf("%s: %d %q %q", url, code, out, errOut)
		}
	}
}

func TestFetchReportsWhatYtdlpCouldNotDo(t *testing.T) {
	_, inbox := convertWorld(t)
	fakeYtdlp(t, map[string]string{})
	if code, _, errOut := run("fetch", "https://x"); code != 1 || !strings.Contains(errOut, "yt-dlp found no video") {
		t.Fatalf("%d %q", code, errOut)
	}
	if entries, err := os.ReadDir(inbox); err != nil || len(entries) != 0 {
		t.Fatalf("the failed fetch left %v in the inbox (%v)", entries, err)
	}
}

func TestFetchRefusesWhereTheProjectSwitchedTheBrainOff(t *testing.T) {
	convertWorld(t)
	project := t.TempDir()
	writeFile(t, filepath.Join(project, ".loomux", "config.toml"), "[modules]\nbrain = false\n")
	t.Chdir(project)
	fakeYtdlp(t, nil)
	if code, _, errOut := run("fetch", "https://x"); code != 1 || !strings.Contains(errOut, "loomux fetch: the brain module is off") {
		t.Fatalf("%d %q", code, errOut)
	}
}

func TestFetchWithoutARegistryIsOneErrorLine(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", dir)
	t.Chdir(dir)
	fakeYtdlp(t, nil)
	if code, _, errOut := run("fetch", "https://x"); code != 1 || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("%d %q", code, errOut)
	}
}

func TestFetchStopsAtABrokenDeclaration(t *testing.T) {
	_, inbox := convertWorld(t)
	writeFile(t, filepath.Join(filepath.Dir(inbox), ".loomux", "config.toml"), "[area]\nscope = \"knowledge\"\n\n[privacy]\nmode = \"cloud\"\n")
	fakeYtdlp(t, nil)
	if code, _, errOut := run("fetch", "https://x"); code != 1 || !strings.Contains(errOut, "mode") {
		t.Fatalf("%d %q", code, errOut)
	}
}

// A file where the inbox should be: ToInbox cannot make the directory.
func TestFetchReportsAnInboxItCannotWriteInto(t *testing.T) {
	_, inbox := convertWorld(t)
	if err := os.RemoveAll(inbox); err != nil {
		t.Fatal(err)
	}
	writeFile(t, inbox, "not a directory\n")
	fakeYtdlp(t, map[string]string{
		"v.info.json": `{"title": "Der echte Weg", "subtitles": {"de": [{"ext": "json3"}]}}`,
		"v.de.json3":  `{"events": [{"tStartMs": 0, "segs": [{"utf8": "Hallo."}]}]}`,
	})
	if code, out, errOut := run("fetch", "https://www.youtube.com/watch?v=mHSOsy_usAg"); code != 1 || out != "" || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("%d %q %q", code, out, errOut)
	}
}
