package rules

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

const projectGodot = "config_version=5\n[autoload]\nGame=\"*res://autoload/game.gd\"\n"

// edgeFindings runs Edges over an in-memory tree.
func edgeFindings(t *testing.T, tree map[string]string, layers ...config.Layer) []string {
	t.Helper()
	var files []File
	var rels []string
	for rel, content := range tree {
		files = append(files, File{Rel: rel, Content: []byte(content)})
		rels = append(rels, rel)
	}
	m, _ := Assign(config.Rules{Present: true, Layers: layers}, rels)
	got, err := Edges(files, m)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range got {
		out = append(out, f.String())
	}
	return out
}

func checked(name string, paths ...string) config.Layer {
	return config.Layer{Name: name, Paths: paths, Checked: true}
}

func withMayUse(l config.Layer, use ...string) config.Layer {
	l.HasMayUse, l.MayUse = true, use
	return l
}

func withBases(l config.Layer, bases ...string) config.Layer {
	l.HasBases, l.Bases = true, bases
	return l
}

func godot(extra map[string]string) map[string]string {
	tree := map[string]string{
		"project.godot":             projectGodot,
		"autoload/game.gd":          "extends Node\nfunc x():\n\tpass\nfunc y():\n\tpass\n",
		"ui/panels/market_lines.gd": "class_name MarketLines\nextends Node\nstatic func has_market():\n\tpass\n",
		"ui/x.gd":                   "extends Node\n",
	}
	for k, v := range extra {
		tree[k] = v
	}
	return tree
}

func layersFor(core config.Layer) []config.Layer {
	return []config.Layer{
		core, withMayUse(checked("ui", "ui/**")), withMayUse(checked("autoload", "autoload/**")),
		withMayUse(config.Layer{Name: "harness", Paths: []string{"harness/**"}}),
		withMayUse(checked("root", "project.godot")),
	}
}

func TestEdgesReportsTheIncidentOfAugust(t *testing.T) {
	tree := godot(map[string]string{
		"model/game_state.gd": "extends Node\n\nfunc f():\n\tpass\n\tpass\n\tpass\n\tpass\n\tMarketLines.has_market()\n\tpass\n\tMarketLines.has_market()\n",
	})
	// The first use counts, and the symbol is the class, not the method.
	want := []string{"model/game_state.gd:8 core uses ui (MarketLines) — not in may_use of core; allow it with: loomux config set rules.layers.core.may_use … --propose"}
	got := edgeFindings(t, tree, layersFor(withMayUse(checked("core", "model/**"), "autoload"))...)
	if !slices.Equal(got, want) {
		t.Fatalf("got %q", got)
	}
}

func TestEdgesAutoloadNeedsItsLayerInMayUse(t *testing.T) {
	tree := godot(map[string]string{"model/a.gd": "class_name A\nextends RefCounted\nfunc f():\n\tGame.x()\n"})
	denied := edgeFindings(t, tree, layersFor(withBases(withMayUse(checked("core", "model/**")), "RefCounted"))...)
	if len(denied) != 1 || !strings.HasPrefix(denied[0], "model/a.gd:4 core uses autoload (autoload/game.gd) — not in may_use of core") {
		t.Fatalf("%q", denied)
	}
	allowed := edgeFindings(t, tree, layersFor(withBases(withMayUse(checked("core", "model/**"), "autoload"), "RefCounted"))...)
	if len(allowed) != 0 {
		t.Fatalf("%q", allowed)
	}
}

func TestEdgesPreloadOfAScene(t *testing.T) {
	tree := godot(map[string]string{
		"ui/x.tscn":  "[gd_scene]\n",
		"model/a.gd": "class_name A\nextends RefCounted\nvar s = preload(\"res://ui/x.tscn\")\n",
	})
	got := edgeFindings(t, tree, layersFor(withBases(withMayUse(checked("core", "model/**")), "RefCounted"))...)
	want := "model/a.gd:3 core uses ui (ui/x.tscn) — not in may_use of core; allow it with: loomux config set rules.layers.core.may_use … --propose"
	if !slices.Equal(got, []string{want}) {
		t.Fatalf("%q", got)
	}
}

func TestEdgesIgnoresComments(t *testing.T) {
	tree := godot(map[string]string{"model/a.gd": "class_name A\nextends RefCounted\n# MarketLines\n"})
	if got := edgeFindings(t, tree, layersFor(withBases(withMayUse(checked("core", "model/**")), "RefCounted"))...); len(got) != 0 {
		t.Fatalf("%q", got)
	}
}

func TestEdgesAllowedShapes(t *testing.T) {
	use := "class_name A\nextends RefCounted\nfunc f():\n\tMarketLines.has_market()\n"
	cases := map[string]struct {
		tree   map[string]string
		layers []config.Layer
	}{
		"own layer": {godot(map[string]string{"ui/b.gd": use}), layersFor(checked("core", "model/**"))},
		"no may_use": {
			godot(map[string]string{"model/b.gd": use}),
			layersFor(withBases(checked("core", "model/**"), "RefCounted")),
		},
		"unchecked source": {
			godot(map[string]string{"harness/b.gd": use}),
			layersFor(checked("core", "model/**")),
		},
		"target without layer": {
			godot(map[string]string{"model/b.gd": use}),
			[]config.Layer{withBases(withMayUse(checked("core", "model/**")), "RefCounted")},
		},
	}
	for name, c := range cases {
		if got := edgeFindings(t, c.tree, c.layers...); len(got) != 0 {
			t.Errorf("%s: %q", name, got)
		}
	}
}

func TestEdgesIntoAnUncheckedLayerNeedMayUse(t *testing.T) {
	tree := godot(map[string]string{
		"harness/h.gd": "class_name Harness\nextends RefCounted\n",
		"model/a.gd":   "class_name A\nextends RefCounted\nfunc f():\n\tHarness.new()\n",
	})
	got := edgeFindings(t, tree, layersFor(withBases(withMayUse(checked("core", "model/**")), "RefCounted"))...)
	if len(got) != 1 || !strings.Contains(got[0], "core uses harness (Harness)") {
		t.Fatalf("%q", got)
	}
	got = edgeFindings(t, tree, layersFor(withBases(withMayUse(checked("core", "model/**"), "harness"), "RefCounted"))...)
	if len(got) != 0 {
		t.Fatalf("%q", got)
	}
}

func TestEdgesBases(t *testing.T) {
	inner := "class_name A\nextends RefCounted\n\n\n\n\n\n\n\n\n\nclass Inner extends Node:\n\tpass\n"
	tree := godot(map[string]string{"core/a.gd": inner})
	core := checked("core", "core/**")
	got := edgeFindings(t, tree, layersFor(withBases(core, "RefCounted"))...)
	if !slices.Equal(got, []string{"core/a.gd:12 core extends Node — not in bases of core"}) {
		t.Fatalf("%q", got)
	}
	if got := edgeFindings(t, tree, layersFor(core)...); len(got) != 0 {
		t.Fatalf("without bases: %q", got)
	}
}

func TestEdgesAmbiguousBase(t *testing.T) {
	tree := godot(map[string]string{
		"core/d1.gd": "class_name Dup\nextends RefCounted\n",
		"core/d2.gd": "class_name Dup\nextends RefCounted\n",
		"core/a.gd":  "extends Dup\n",
	})
	core := checked("core", "core/**")
	for _, l := range []config.Layer{core, withBases(core, "Dup", "RefCounted")} {
		got := edgeFindings(t, tree, layersFor(l)...)
		if !slices.Contains(got, "core/a.gd:1 ambiguous base Dup: more than one class declares it") {
			t.Fatalf("%q", got)
		}
	}
}

func TestEdgesBasesOfAnUncheckedLayerAreIgnored(t *testing.T) {
	tree := godot(map[string]string{"harness/a.gd": "extends Dup\n", "harness/d.gd": "class_name Dup\n", "harness/e.gd": "class_name Dup\n"})
	if got := edgeFindings(t, tree, layersFor(checked("core", "core/**"))...); len(got) != 0 {
		t.Fatalf("%q", got)
	}
}

func TestEdgesAcrossLayersInGoAndPython(t *testing.T) {
	tree := map[string]string{
		"go.mod":        "module example.com/m\n\ngo 1.22\n",
		"core/a.go":     "package core\n\nimport \"example.com/m/ui\"\n\nvar _ = ui.F\n",
		"ui/u.go":       "package ui\n\nfunc F() {}\n",
		"core/a.py":     "import pyui.uimod\n",
		"pyui/uimod.py": "X = 1\n",
	}
	layers := []config.Layer{
		withMayUse(checked("core", "core/**")), checked("ui", "ui/**"), checked("pyui", "pyui/**"),
	}
	got := edgeFindings(t, tree, layers...)
	if len(got) != 2 || !strings.HasPrefix(got[0], "core/a.go:3 core uses ui (ui/u.go) ") ||
		!strings.HasPrefix(got[1], "core/a.py:1 core uses pyui (pyui/uimod.py) ") {
		t.Fatalf("%q", got)
	}
}

func TestEdgesReportsExtractionErrors(t *testing.T) {
	m, _ := Assign(config.Rules{}, nil)
	_, err := Edges([]File{{Rel: "a.go", Content: []byte("package")}}, m)
	if err == nil || !strings.HasPrefix(err.Error(), "a.go: ") {
		t.Fatalf("%v", err)
	}
}

func TestEdgesKeepTheFirstLineAcrossTargetsOfOneFile(t *testing.T) {
	tree := godot(map[string]string{"model/a.gd": "class_name A\nextends RefCounted\nfunc f():\n\tGame.y()\n\nfunc g():\n\tGame.x()\n"})
	got := edgeFindings(t, tree, layersFor(withBases(withMayUse(checked("core", "model/**")), "RefCounted"))...)
	if len(got) != 1 || !strings.HasPrefix(got[0], "model/a.gd:4 core uses autoload (autoload/game.gd)") {
		t.Fatalf("%q", got)
	}
}

func TestEdgesSortByPathThenLine(t *testing.T) {
	body := "class_name A\nextends RefCounted\nfunc f():\n\tpass\n\tpass\n\tpass\n\tpass\n\tpass\n\tMarketLines.has_market()\n\tGame.x()\n"
	tree := godot(map[string]string{"model/a.gd": body, "model/a.gd2.gd": body})
	got := edgeFindings(t, tree, layersFor(withBases(withMayUse(checked("core", "model/**")), "RefCounted"))...)
	var heads []string
	for _, g := range got {
		heads = append(heads, g[:strings.Index(g, " core")])
	}
	want := []string{"model/a.gd:9", "model/a.gd:10", "model/a.gd2.gd:9", "model/a.gd2.gd:10"}
	if !slices.Equal(heads, want) {
		t.Fatalf("%q", heads)
	}
}

func TestEdgesPythonBases(t *testing.T) {
	tree := map[string]string{"core/a.py": "class A(object):\n    pass\n\n\nclass C(A, dict):\n    pass\n"}
	core := checked("core", "core/**")
	got := edgeFindings(t, tree, withBases(core, "object"))
	if !slices.Equal(got, []string{"core/a.py:5 core extends dict — not in bases of core"}) {
		t.Fatalf("%q", got)
	}
	if got := edgeFindings(t, tree, withBases(core, "object", "dict")); len(got) != 0 {
		t.Fatalf("%q", got)
	}
}

func TestFindingString(t *testing.T) {
	if got := (Finding{"a", 3, "t"}).String(); got != "a:3 t" {
		t.Fatal(got)
	}
	if got := (Finding{"a", 0, "t"}).String(); got != "a t" {
		t.Fatal(got)
	}
}
