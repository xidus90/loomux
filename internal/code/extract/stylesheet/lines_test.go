package stylesheet_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/extract/stylesheet"
	"github.com/xidus90/loomux/internal/code/model"
)

func TestEdgeLines(t *testing.T) {
	src := "@use \"x\" as y;\n@import \"a\";\n.k {\n  @include m;\n  width: f(1);\n  @extend .j;\n  @extend %p;\n}\n"
	r, err := stylesheet.File("m.scss", src)
	if err != nil {
		t.Fatal(err)
	}
	type key struct {
		rel  model.Relation
		name string
	}
	got := map[key]int{}
	for _, e := range r.Edges {
		n := e.Name
		if e.Relation == model.RelationImports {
			n = e.Specifier
		}
		got[key{e.Relation, n}] = e.Line
	}
	for k, want := range map[key]int{
		{model.RelationImports, "x"}:  1,
		{model.RelationImports, "a"}:  2,
		{model.RelationCalls, "m"}:    4,
		{model.RelationCalls, "f"}:    5,
		{model.RelationExtends, ".j"}: 6,
		{model.RelationExtends, "%p"}: 7,
	} {
		if got[k] != want {
			t.Errorf("%v line = %d, want %d", k, got[k], want)
		}
	}
}
