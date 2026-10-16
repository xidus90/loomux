package godotsuite

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/child"
)

const original = "; comment\nconfig_version=5\n"

// rescanWorld is a project whose engine rewrites project.godot like the real
// one does and, unless told not to, writes the class cache.
type rescanWorld struct {
	root   string
	argv   []string
	cache  bool
	result child.Result
	rewrit string
}

func newRescan(t *testing.T) *rescanWorld {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "project.godot"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	return &rescanWorld{root: root, cache: true, rewrit: "config_version=5\n"}
}

func (w *rescanWorld) start(s child.Spec) child.Result {
	w.argv = s.Argv
	if err := os.WriteFile(filepath.Join(w.root, "project.godot"), []byte(w.rewrit), 0o644); err != nil {
		panic(err)
	}
	if w.cache {
		dir := filepath.Join(w.root, ".godot")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "global_script_class_cache.cfg"), nil, 0o644); err != nil {
			panic(err)
		}
	}
	return w.result
}

func (w *rescanWorld) rescan(status func(string) (string, error)) ([]string, error) {
	if status == nil {
		status = func(string) (string, error) { return "", nil }
	}
	return Rescan(RescanOptions{Root: w.root, Godot: "godot", Start: w.start, Status: status})
}

func (w *rescanWorld) project(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(w.root, "project.godot"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRescanRestoresProjectGodotAndSaysSo(t *testing.T) {
	w := newRescan(t)
	notes, err := w.rescan(nil)
	if err != nil || !slices.Equal(notes, []string{"restored project.godot, which the engine rewrote"}) {
		t.Fatalf("notes %q err %v", notes, err)
	}
	if got := w.project(t); got != original {
		t.Fatalf("project.godot is %q", got)
	}
	if want := []string{"godot", "--path", w.root, "--headless", "--import"}; !slices.Equal(w.argv, want) {
		t.Fatalf("argv %q", w.argv)
	}
}

func TestRescanSaysNothingWhenProjectGodotStays(t *testing.T) {
	w := newRescan(t)
	w.rewrit = original
	if notes, err := w.rescan(nil); err != nil || len(notes) != 0 {
		t.Fatalf("notes %q err %v", notes, err)
	}
}

func TestRescanNamesWhatElseTheEngineChanged(t *testing.T) {
	w := newRescan(t)
	w.rewrit = original
	calls := 0
	notes, err := w.rescan(func(string) (string, error) {
		calls++
		if calls == 1 {
			return " M kept.tres\n", nil
		}
		return " M kept.tres\n M ui/theme/x.tres\n?? core/a.gd.uid\n", nil
	})
	want := []string{
		"the engine also changed: core/a.gd.uid, ui/theme/x.tres",
		"*.uid and *.import files are products of the engine; they are reported, not reset",
	}
	if err != nil || !slices.Equal(notes, want) {
		t.Fatalf("notes %q err %v", notes, err)
	}
}

func TestRescanNamesOtherChangesWithoutTheProductNote(t *testing.T) {
	w := newRescan(t)
	w.rewrit = original
	n := 0
	notes, _ := w.rescan(func(string) (string, error) {
		n++
		if n == 1 {
			return "", nil
		}
		return " M ui/theme/x.tres\n", nil
	})
	if !slices.Equal(notes, []string{"the engine also changed: ui/theme/x.tres"}) {
		t.Fatalf("notes %q", notes)
	}
}

func TestRescanRestoresProjectGodotWhenTheEngineFails(t *testing.T) {
	w := newRescan(t)
	w.result = child.Result{Code: 1, Stdout: "boom out", Stderr: " boom err"}
	_, err := w.rescan(nil)
	if err == nil || !strings.Contains(err.Error(), "boom out boom err") {
		t.Fatalf("err %v", err)
	}
	if got := w.project(t); got != original {
		t.Fatalf("project.godot is %q", got)
	}
}

func TestRescanReportsAnEngineThatDoesNotStart(t *testing.T) {
	w := newRescan(t)
	w.result = child.Result{Err: errors.New("no such file")}
	if _, err := w.rescan(nil); err == nil || !strings.Contains(err.Error(), "no such file") {
		t.Fatalf("err %v", err)
	}
}

func TestRescanWantsTheClassCacheAfterwards(t *testing.T) {
	w := newRescan(t)
	w.cache = false
	_, err := w.rescan(nil)
	if err == nil || err.Error() != "the engine ran but wrote no .godot/global_script_class_cache.cfg" {
		t.Fatalf("err %v", err)
	}
}

func TestRescanNeedsAProjectAndAWorkingStatus(t *testing.T) {
	empty := RescanOptions{Root: t.TempDir(), Godot: "g", Status: func(string) (string, error) { return "", nil }}
	if _, err := Rescan(empty); err == nil {
		t.Fatal("no project.godot, no error")
	}
	w := newRescan(t)
	calls := 0
	_, err := w.rescan(func(string) (string, error) {
		calls++
		if calls == 2 {
			return "", errors.New("git gone")
		}
		return "", nil
	})
	if err == nil || err.Error() != "git gone" || w.project(t) != original {
		t.Fatalf("err %v project %q", err, w.project(t))
	}
	w = newRescan(t)
	_, err = w.rescan(func(string) (string, error) { return "", errors.New("no git") })
	if err == nil || err.Error() != "no git" || w.argv != nil {
		t.Fatalf("err %v argv %q", err, w.argv)
	}
}

func TestRescanUsesTheRealEngineRunnerByDefault(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "project.godot"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Rescan(RescanOptions{Root: root, Godot: filepath.Join(root, "absent"), Status: func(string) (string, error) { return "", nil }})
	if err == nil {
		t.Fatal("a binary that is not there ran")
	}
}

func TestRescanReportsAProjectGodotItCannotRestore(t *testing.T) {
	w := newRescan(t)
	project := filepath.Join(w.root, "project.godot")
	// A read-only file the engine "rewrote": the write back must fail loudly.
	start := func(s child.Spec) child.Result {
		w.start(s)
		if err := os.Chmod(project, 0o444); err != nil {
			t.Fatal(err)
		}
		return child.Result{}
	}
	_, err := Rescan(RescanOptions{Root: w.root, Godot: "g", Start: start, Status: func(string) (string, error) { return "", nil }})
	_ = os.Chmod(project, 0o644)
	if err == nil {
		t.Skip("this user can write a read-only file")
	}
}

func TestRescanKeepsTheEngineErrorWhenTheWriteBackFailsToo(t *testing.T) {
	w := newRescan(t)
	project := filepath.Join(w.root, "project.godot")
	w.result = child.Result{Code: 1, Stdout: "boom out"}
	start := func(s child.Spec) child.Result {
		w.start(s)
		if err := os.Chmod(project, 0o444); err != nil {
			t.Fatal(err)
		}
		return w.result
	}
	_, err := Rescan(RescanOptions{Root: w.root, Godot: "g", Start: start, Status: func(string) (string, error) { return "", nil }})
	_ = os.Chmod(project, 0o644)
	if err == nil || !strings.Contains(err.Error(), "boom out") {
		t.Fatalf("the engine error is lost: %v", err)
	}
	if w.project(t) == original {
		t.Skip("this user can write a read-only file")
	}
	if !strings.Contains(err.Error(), "project.godot") {
		t.Fatalf("the write error is lost: %v", err)
	}
}
