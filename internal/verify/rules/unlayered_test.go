package rules

import (
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

func TestUnlayered(t *testing.T) {
	rels := []string{
		"godot/sim/a.gd", "godot/sim/b.tscn", "godot/sim/c.tres", "project.godot", "README.md",
		"tools/x.py", "cmd/main.go", "godot/free/d.gd", "godot/sim/E.GD",
	}
	m, errs := Assign(config.Rules{Layers: []config.Layer{
		{Name: "free", Paths: []string{"godot/free/**"}, Checked: false},
	}}, rels)
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	const tail = "is in no layer; add it to a layer in [rules.layers], or to one with checked = false"
	want := []Finding{
		{Path: "cmd/main.go", Text: tail},
		{Path: "godot/sim/a.gd", Text: tail},
		{Path: "tools/x.py", Text: tail},
	}
	if got := Unlayered(rels, m); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	const printed = "tools/x.py is in no layer; add it to a layer in [rules.layers], or to one with checked = false"
	if got := Unlayered(rels, m)[2].String(); got != printed {
		t.Fatalf("printed %q, want %q", got, printed)
	}
}
