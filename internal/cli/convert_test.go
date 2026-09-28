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
	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", filepath.Join(root, "legacy"))
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
