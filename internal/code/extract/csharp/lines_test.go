package csharp_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/extract/csharp"
	"github.com/xidus90/loomux/internal/code/model"
)

func TestEdgeLines(t *testing.T) {
	src := "using A.B;\n\nclass K : Base {\n    int x = F();\n    void M() {\n        new T();\n    }\n}\n"
	r, err := csharp.File("m.cs", src)
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
		{model.RelationImports, "A.B"}:  1,
		{model.RelationExtends, "Base"}: 3,
		{model.RelationCalls, "F"}:      4,
		{model.RelationCalls, "T"}:      6,
	} {
		if got[k] != want {
			t.Errorf("%v line = %d, want %d", k, got[k], want)
		}
	}
}
