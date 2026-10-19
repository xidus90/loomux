package verify

import (
	"os"
	"path/filepath"
	"testing"
)

// The class list holds what Godot 3's editor scans, so a class_name where it
// never looks is no missing class: a hidden directory (a worktree copy under
// .claude, say), one with a .gdignore, and a nested project of its own.
func TestClassListMissingSkipsWhatGodotSkips(t *testing.T) {
	dir := godot3Project(t, "Player", "GutTest")
	for path, body := range map[string]string{
		".claude/worktrees/x/src/hidden.gd": "class_name Hidden\n",
		"tools/.gdignore":                   "",
		"tools/tool.gd":                     "class_name Tool\n",
		"sample/project.godot":              "config_version=4\n",
		"sample/s.gd":                       "class_name Sample\n",
	} {
		p := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if missing, _ := ClassListMissing(dir); len(missing) != 0 {
		t.Fatalf("missing %v", missing)
	}
}
