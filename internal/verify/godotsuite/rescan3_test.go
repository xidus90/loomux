package godotsuite

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
)

const godot3Original = "; Engine configuration file.\nconfig_version=4\n\n[application]\n\nconfig/name=\"p\"\n"

// godot3Rescan is a Godot 3 project with one named script, whose editor --
// like Godot 3.6's -- writes the class list into project.godot, rewrites its
// header and never quits on its own, so the run ends by its time limit.
func godot3Rescan(t *testing.T, writes bool) (root string, spec *child.Spec, start func(child.Spec) child.Result) {
	t.Helper()
	root = t.TempDir()
	for path, body := range map[string]string{
		"project.godot": godot3Original,
		"src/player.gd": "extends Node\nclass_name Player\n",
	} {
		p := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	spec = &child.Spec{}
	start = func(s child.Spec) child.Result {
		*spec = s
		if writes {
			edited := "; Engine configuration file.\n; It's best edited using the editor UI.\nconfig_version=4\n\n" +
				"_global_script_classes=[ {\n\"base\": \"Node\",\n\"class\": \"Player\",\n\"language\": \"GDScript\",\n\"path\": \"res://src/player.gd\"\n} ]\n" +
				"_global_script_class_icons={\n\"Player\": \"\"\n}\n\n[application]\n\nconfig/name=\"p\"\nrun/main_scene=\"res://x.tscn\"\n"
			if err := os.WriteFile(filepath.Join(root, "project.godot"), []byte(edited), 0o644); err != nil {
				panic(err)
			}
		}
		return child.Result{Code: -1, TimedOut: true}
	}
	return root, spec, start
}

// Godot 3 keeps its class list in project.godot, so a rescan keeps that list
// and only that: the editor's other rewrites go back.
func TestRescanOfAGodot3ProjectKeepsOnlyTheClassList(t *testing.T) {
	root, spec, start := godot3Rescan(t, true)
	notes, err := Rescan(RescanOptions{Root: root, Godot: "godot3", Start: start, Window: time.Second,
		Status: func(string) (string, error) { return "", nil }})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(spec.Argv, []string{"godot3", "--no-window", "--editor", "--path", root}) || spec.Timeout != time.Second {
		t.Fatalf("spec %+v", *spec)
	}
	b, _ := os.ReadFile(filepath.Join(root, "project.godot"))
	got := string(b)
	for _, want := range []string{"\"class\": \"Player\"", "_global_script_class_icons={\n\"Player\": \"\"\n}", "config/name=\"p\""} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	for _, gone := range []string{"It's best edited", "run/main_scene"} {
		if strings.Contains(got, gone) {
			t.Errorf("kept the editor's %q in\n%s", gone, got)
		}
	}
	if !slices.Contains(notes, "kept the class list the editor wrote into project.godot; restored the rest") {
		t.Errorf("notes %v", notes)
	}
}

// An editor that writes no class list in its window leaves the project as it
// was and says which classes it is still missing.
func TestRescanOfAGodot3ProjectGivesUp(t *testing.T) {
	root, _, start := godot3Rescan(t, false)
	_, err := Rescan(RescanOptions{Root: root, Godot: "godot3", Start: start, Window: time.Millisecond,
		Status: func(string) (string, error) { return "", nil }})
	if err == nil || !strings.Contains(err.Error(), "the editor listed no class for: Player") {
		t.Fatalf("%v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "project.godot")); string(b) != godot3Original {
		t.Fatalf("project.godot changed:\n%s", b)
	}
}

// The ways a Godot 3 rescan ends early, each saying why.
func TestRescanOfAGodot3ProjectReportsEachFailure(t *testing.T) {
	okStatus := func(string) (string, error) { return "", nil }
	t.Run("status before", func(t *testing.T) {
		root, _, start := godot3Rescan(t, true)
		_, err := Rescan(RescanOptions{Root: root, Godot: "g", Start: start, Status: func(string) (string, error) { return "", errors.New("no git") }})
		if err == nil || err.Error() != "no git" {
			t.Fatalf("%v", err)
		}
	})
	t.Run("engine does not start", func(t *testing.T) {
		root, _, _ := godot3Rescan(t, true)
		start := func(child.Spec) child.Result { return child.Result{Err: errors.New("not found"), Code: -1} }
		_, err := Rescan(RescanOptions{Root: root, Godot: "g", Start: start, Status: okStatus})
		if err == nil || !strings.HasPrefix(err.Error(), "the engine failed: ") {
			t.Fatalf("%v", err)
		}
	})
	// An editor that ends before its window, with an error, did not run: its
	// output says why, not a list it never had the time to write.
	t.Run("engine exits at once", func(t *testing.T) {
		root, _, _ := godot3Rescan(t, false)
		start := func(child.Spec) child.Result { return child.Result{Code: 1, Stdout: "crash at start"} }
		_, err := Rescan(RescanOptions{Root: root, Godot: "g", Start: start, Status: okStatus})
		if err == nil || err.Error() != "the engine failed: crash at start" {
			t.Fatalf("%v", err)
		}
	})
	t.Run("default window", func(t *testing.T) {
		root, spec, start := godot3Rescan(t, true)
		if _, err := Rescan(RescanOptions{Root: root, Godot: "g", Start: start, Status: okStatus}); err != nil || spec.Timeout != rescanWindow {
			t.Fatalf("%v %v", err, spec.Timeout)
		}
	})
	t.Run("a class the list lacks", func(t *testing.T) {
		root, _, start := godot3Rescan(t, true)
		if err := os.WriteFile(filepath.Join(root, "src", "enemy.gd"), []byte("class_name Enemy\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Rescan(RescanOptions{Root: root, Godot: "g", Start: start, Status: okStatus})
		if err == nil || err.Error() != "the editor listed no class for: Enemy" {
			t.Fatalf("%v", err)
		}
		if b, _ := os.ReadFile(filepath.Join(root, "project.godot")); !strings.Contains(string(b), "\"class\": \"Player\"") {
			t.Fatalf("the partial list is gone:\n%s", b)
		}
	})
	t.Run("status after", func(t *testing.T) {
		root, _, start := godot3Rescan(t, true)
		calls := 0
		status := func(string) (string, error) {
			calls++
			if calls == 2 {
				return "", errors.New("git gone")
			}
			return "", nil
		}
		if _, err := Rescan(RescanOptions{Root: root, Godot: "g", Start: start, Status: status}); err == nil || err.Error() != "git gone" {
			t.Fatalf("%v", err)
		}
	})
	for _, writes := range []bool{true, false} {
		t.Run("read-only project.godot", func(t *testing.T) {
			root, _, start := godot3Rescan(t, writes)
			project := filepath.Join(root, "project.godot")
			readOnly := func(s child.Spec) child.Result {
				r := start(s)
				if err := os.Chmod(project, 0o444); err != nil {
					t.Fatal(err)
				}
				return r
			}
			_, err := Rescan(RescanOptions{Root: root, Godot: "g", Start: readOnly, Status: okStatus})
			_ = os.Chmod(project, 0o644)
			if err == nil {
				t.Skip("this user can write a read-only file")
			}
			if strings.HasPrefix(err.Error(), "the editor listed") {
				t.Fatalf("the write error was lost: %v", err)
			}
		})
	}
}

func TestClassListBlocks(t *testing.T) {
	src := "a=1\n_global_script_classes=[ {\n\"x\": [ 1 ]\n} ]\nb=2\n_global_script_class_icons={\n}\n"
	blocks, rest := cutClassList(src)
	if blocks != "_global_script_classes=[ {\n\"x\": [ 1 ]\n} ]\n_global_script_class_icons={\n}\n" || rest != "a=1\nb=2\n" {
		t.Fatalf("%q %q", blocks, rest)
	}
	// Keys apart in the original are both replaced, not one.
	if got := withClassList("config_version=4\n"+src, "N\n"); got != "config_version=4\nN\na=1\nb=2\n" {
		t.Fatalf("%q", got)
	}
	if got := withClassList("x=1\n", "_global_script_classes=[  ]\n"); got != "_global_script_classes=[  ]\nx=1\n" {
		t.Fatalf("no config_version: %q", got)
	}
}
