package python_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/extract/python"
	"github.com/xidus90/loomux/internal/code/model"
)

func TestEdgeLines(t *testing.T) {
	src := "import a\nfrom b import c\n\nclass K(Base):\n    x = f()\n"
	r, err := python.File("m.py", src)
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
		{model.RelationImports, "a"}:    1,
		{model.RelationImports, "b"}:    2,
		{model.RelationImports, "b.c"}:  2,
		{model.RelationExtends, "Base"}: 4,
		{model.RelationCalls, "f"}:      5,
	} {
		if got[k] != want {
			t.Errorf("%v line = %d, want %d", k, got[k], want)
		}
	}
}
