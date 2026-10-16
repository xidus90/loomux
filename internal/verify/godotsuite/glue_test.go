package godotsuite

import (
	"os"
	"path/filepath"
	"testing"
)

var glueNames = []string{"run_GdUnitCmdTool.gd", "hook.gd", "disk.gd"}

func TestWriteGlueWritesTheEmbeddedScripts(t *testing.T) {
	root := t.TempDir()
	if err := WriteGlue(root); err != nil {
		t.Fatal(err)
	}
	for _, name := range glueNames {
		want, err := glue.ReadFile("gd/" + name)
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(root, ".godot", "loomux", name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("%s differs from the embedded copy", name)
		}
	}
}

func TestWriteGlueOverwritesAChangedFile(t *testing.T) {
	root := t.TempDir()
	if err := WriteGlue(root); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, ".godot", "loomux", "hook.gd")
	if err := os.WriteFile(p, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteGlue(root); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	if string(got) == "stale" {
		t.Error("changed file was not overwritten")
	}
}

func TestWriteGlueFailsWhenGodotIsAFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".godot"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteGlue(root); err == nil {
		t.Error("expected an error")
	}
}

func TestWriteGlueFailsWhenAScriptPathIsADirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".godot", "loomux", "hook.gd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteGlue(root); err == nil {
		t.Error("expected an error")
	}
}
