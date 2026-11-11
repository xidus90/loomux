package java_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/extract/java"
	"github.com/xidus90/loomux/internal/code/model"
)

func TestEdgeLines(t *testing.T) {
	src := "import a.b.C;\n\nclass K extends Base {\n    int x = f();\n    void m() {\n        new T();\n    }\n}\n"
	r, err := java.File("m.java", src)
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
		{model.RelationImports, "a.b.C"}: 1,
		{model.RelationExtends, "Base"}:  3,
		{model.RelationCalls, "f"}:       4,
		{model.RelationCalls, "T"}:       6,
	} {
		if got[k] != want {
			t.Errorf("%v line = %d, want %d", k, got[k], want)
		}
	}
}
