package gutsuite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/verify/godotsuite"
)

// TestAgainstARealEngine runs a GUT suite with one passing and one failing
// test through loomux, on each line whose engine and addon are given:
// GODOT3_BIN with the GUT 7 addon directory in LOOMUX_GUT7_DIR, GODOT_BIN
// with the GUT 9 one in LOOMUX_GUT9_DIR. The project gets what it needs the
// way a user gets it, through godot rescan. Without them a line skips, as CI
// has no engine.
func TestAgainstARealEngine(t *testing.T) {
	for _, l := range []struct{ name, bin, addon, fixture string }{
		{"Godot 3 + GUT 7", os.Getenv("GODOT3_BIN"), os.Getenv("LOOMUX_GUT7_DIR"), "project3"},
		{"Godot 4 + GUT 9", os.Getenv("GODOT_BIN"), os.Getenv("LOOMUX_GUT9_DIR"), "project4"},
	} {
		t.Run(l.name, func(t *testing.T) {
			if l.bin == "" || l.addon == "" {
				t.Skip("needs the engine and the addon directory")
			}
			root := t.TempDir()
			if err := os.CopyFS(root, os.DirFS(filepath.Join("testdata", l.fixture))); err != nil {
				t.Fatal(err)
			}
			if err := os.CopyFS(filepath.Join(root, "addons", "gut"), os.DirFS(l.addon)); err != nil {
				t.Fatal(err)
			}
			notes, err := godotsuite.Rescan(godotsuite.RescanOptions{Root: root, Godot: l.bin,
				Status: func(string) (string, error) { return "", nil }})
			if err != nil {
				t.Fatalf("rescan: %v %v", err, notes)
			}
			r, err := Run(Options{Root: root, Godot: l.bin, RunID: "real"})
			if err != nil {
				t.Fatal(err)
			}
			if len(r.Findings) != 1 || !strings.HasPrefix(r.Findings[0], "test/test_a.gd test_fails: [1] expected to equal [2]") {
				t.Fatalf("findings %q", r.Findings)
			}
		})
	}
}
