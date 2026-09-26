package cli

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/tui"
)

// localOnlyWorld is initWorld with a project that declares local_only, so
// the part model is on by default, and an Ollama that has no model.
func localOnlyWorld(t *testing.T, pull string) (string, *initSeams) {
	t.Helper()
	root, s := initWorld(t)
	writeAt(t, filepath.Join(root, ".loomux", "config.toml"), "[privacy]\nmode = \"local_only\"\n")
	useOllama(t, s, `{"models":[]}`, pull)
	return root, s
}

func TestInitDryRunShowsThePullAndPullsNothing(t *testing.T) {
	root, s := localOnlyWorld(t, `{"status":"success"}`)
	code, out, errOut := run("init", "--root", root, "--yes", "--dry-run")
	if code != 0 || !strings.Contains(out, "  model-pull: ollama pull "+config.DefaultModelName+"\n") || len(s.pulls) != 0 {
		t.Fatalf("code %d, pulls %v: %s\n%s", code, s.pulls, errOut, out)
	}
}

func TestInitPullsAMissingModel(t *testing.T) {
	root, s := localOnlyWorld(t, `{"status":"success"}`)
	code, out, errOut := run("init", "--root", root, "--yes")
	if code != 0 || !slices.Equal(s.pulls, []string{config.DefaultModelName}) ||
		!strings.Contains(out, "written: model-pull\n") || !strings.Contains(errOut, "model-pull: success\n") {
		t.Fatalf("code %d, pulls %v: %s\n%s", code, s.pulls, errOut, out)
	}
}

// A pull that fails is a note; init writes the rest and exits 0.
func TestInitGoesOnAfterAFailedPull(t *testing.T) {
	root, s := localOnlyWorld(t, `{"error":"pull model manifest: file does not exist"}`)
	code, out, errOut := run("init", "--root", root, "--yes")
	want := "note: model-pull: pull model manifest: file does not exist; run it by hand: ollama pull " + config.DefaultModelName + "\n"
	if code != 0 || len(s.pulls) != 1 || !strings.Contains(out, want) || strings.Contains(out, "failed:") ||
		!there(root, ".claude/settings.json") {
		t.Fatalf("code %d, pulls %v: %s\n%s", code, s.pulls, errOut, out)
	}
}

// Off by default, the part keeps a fresh project's brain module from being
// offered as all: the human meets the download in the part list.
func TestInitOffersTheBrainPartsOneByOne(t *testing.T) {
	root, _ := initWorld(t)
	term := tui.Script(100, 40, tui.Keys("enter", "enter", "enter", "enter", "enter", "enter", "enter")...)
	withTerminal(t, term, nil)
	code, out, errOut := run("init", "--root", root, "--dry-run")
	if code != 0 || !strings.Contains(term.Output(), "module brain (wiki) (area, merge-hook, brain-skills, model): each") ||
		strings.Contains(out, "model-pull") {
		t.Fatalf("code %d: %s\n%s\nscreen:\n%s", code, errOut, out, term.Output())
	}
}

// With the model in Ollama already, nothing is planned or pulled.
func TestInitPullsNothingWithTheModelThere(t *testing.T) {
	root, s := initWorld(t)
	writeAt(t, filepath.Join(root, ".loomux", "config.toml"), "[privacy]\nmode = \"local_only\"\n")
	code, out, errOut := run("init", "--root", root, "--yes")
	if code != 0 || len(s.pulls) != 0 || strings.Contains(out, "model-pull") {
		t.Fatalf("code %d, pulls %v: %s\n%s", code, s.pulls, errOut, out)
	}
}
