package gdscript_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
)

func lineOf(t *testing.T, path, src string, r model.Relation, name, spec string) int {
	t.Helper()
	for _, e := range extractSrc(t, path, src).Edges {
		if e.Relation == r && e.Name == name && e.Specifier == spec {
			return e.Line
		}
	}
	t.Fatalf("%s: no %s edge %q %q", path, r, name, spec)
	return 0
}

func TestEdgeLines(t *testing.T) {
	src := "extends RefCounted\n\nconst X = preload(\"res://x.gd\")\n\nfunc a():\n\tvar n = 1\n\tn += 1\n\n\tMarketLines.has_market()\n\nfunc b():\n\tpass\n\nclass Inner extends Node:\n\tpass\n"
	for _, c := range []struct {
		r          model.Relation
		name, spec string
		want       int
	}{
		{model.RelationExtends, "RefCounted", "", 1},
		{model.RelationImports, "", "res://x.gd", 3},
		{model.RelationCalls, "has_market", "", 9},
		{model.RelationReferences, "MarketLines", "", 9},
		{model.RelationExtends, "Node", "", 14},
	} {
		if got := lineOf(t, "a/s.gd", src, c.r, c.name, c.spec); got != c.want {
			t.Errorf("%s %q %q line = %d, want %d", c.r, c.name, c.spec, got, c.want)
		}
	}

	tscn := "[gd_scene format=3]\n[ext_resource type=\"Script\" path=\"res://a.gd\" id=\"1\"]\n"
	if got := lineOf(t, "a/s.tscn", tscn, model.RelationImports, "", "res://a.gd"); got != 2 {
		t.Errorf("ext_resource line = %d, want 2", got)
	}
	proj := "config_version=5\n\n[autoload]\n\nGame=\"*res://game.gd\"\n"
	if got := lineOf(t, "project.godot", proj, model.RelationImports, "", "res://game.gd"); got != 5 {
		t.Errorf("autoload line = %d, want 5", got)
	}
	main := "[application]\n\nrun/main_scene=\"res://main.tscn\"\n"
	if got := lineOf(t, "project.godot", main, model.RelationImports, "", "res://main.tscn"); got != 3 {
		t.Errorf("main_scene line = %d, want 3", got)
	}
}
