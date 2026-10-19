package verify

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// godot3Project is a Godot 3 project with two named scripts, one of them in
// an addon, and the class list project.godot holds for them.
func godot3Project(t *testing.T, listed ...string) string {
	t.Helper()
	dir := t.TempDir()
	classes := ""
	for i, c := range listed {
		if i > 0 {
			classes += ", "
		}
		classes += "{\n\"base\": \"Node\",\n\"class\": \"" + c + "\",\n\"language\": \"GDScript\",\n\"path\": \"res://x.gd\"\n}"
	}
	project := "; Engine configuration file.\nconfig_version=4\n\n"
	if classes != "" {
		project += "_global_script_classes=[ " + classes + " ]\n"
	}
	project += "\n[application]\n\nconfig/name=\"p\"\n"
	for path, body := range map[string]string{
		"project.godot":            project,
		"src/player.gd":            "extends Node\nclass_name Player\n",
		"addons/gut/test.gd":       "class_name GutTest, \"res://icon.png\"\nextends Node\n",
		"src/plain.gd":             "extends Node\n",
		".import/stale.gd":         "class_name Stale\n",
		"other/.godot/leftover.gd": "class_name Leftover\n",
	} {
		p := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// Godot 3 has no class cache of its own: it resolves a class_name through the
// list in project.godot, which the editor writes. A project is ready once
// that list names every class_name it declares, its addons' included.
func TestImportReadyForAGodot3Project(t *testing.T) {
	if !ImportReady(godot3Project(t, "Player", "GutTest")) {
		t.Error("a complete class list is ready")
	}
	if ImportReady(godot3Project(t, "Player")) {
		t.Error("GutTest is missing from the list")
	}
	if ImportReady(godot3Project(t)) {
		t.Error("no list at all is not ready")
	}
	godot4 := t.TempDir()
	if err := os.WriteFile(filepath.Join(godot4, "project.godot"), []byte("config_version=5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ImportReady(godot4) {
		t.Error("a Godot 4 project without its class cache is not ready")
	}
}

func TestClassListMissingNamesWhatTheListLacks(t *testing.T) {
	missing, godot3 := ClassListMissing(godot3Project(t, "Player"))
	if !godot3 || !slices.Equal(missing, []string{"GutTest"}) {
		t.Fatalf("%v %v", missing, godot3)
	}
	if _, godot3 := ClassListMissing(t.TempDir()); godot3 {
		t.Error("no project.godot is no Godot 3 project")
	}
}
