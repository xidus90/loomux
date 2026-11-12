package ecmascript_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/extract/ecmascript"
	"github.com/xidus90/loomux/internal/code/model"
)

func TestEdgeLines(t *testing.T) {
	src := "import a from './a';\nconst m = require('./m');\nclass K extends Base implements I {\n  m() {\n    f();\n    ns.g();\n    new N();\n    document.getElementById('x');\n  }\n}\nimport('./d');\n"
	r, err := ecmascript.File("m.ts", src)
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
		{model.RelationImports, "./a"}:  1,
		{model.RelationImports, "./m"}:  2,
		{model.RelationExtends, "Base"}: 3,
		{model.RelationImplements, "I"}: 3,
		{model.RelationCalls, "f"}:      5,
		{model.RelationCalls, "g"}:      6,
		{model.RelationCalls, "N"}:      7,
		{model.RelationReferences, "x"}: 8,
		{model.RelationImports, "./d"}:  11,
	} {
		if got[k] != want {
			t.Errorf("%v line = %d, want %d", k, got[k], want)
		}
	}
}
