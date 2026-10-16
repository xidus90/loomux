package golang_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/extract/golang"
	"github.com/xidus90/loomux/internal/code/model"
)

func TestEdgeLines(t *testing.T) {
	src := "package p\n\nimport \"fmt\"\n\nfunc f() {}\n\nfunc h(x T) {\n\tf()\n\tx.g()\n\tfmt.Println()\n}\n"
	r, err := golang.File("p.go", src)
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
	want := map[key]int{
		{model.RelationImports, "fmt"}:   3,
		{model.RelationCalls, "f"}:       8,
		{model.RelationCalls, "g"}:       9,
		{model.RelationCalls, "Println"}: 10,
	}
	for k, line := range want {
		if got[k] != line {
			t.Errorf("%v line = %d, want %d", k, got[k], line)
		}
	}
}
