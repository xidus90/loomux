package cpp_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/extract/cpp"
	"github.com/xidus90/loomux/internal/code/model"
)

func TestEdgeLines(t *testing.T) {
	src := "#include \"a.h\"\n\nclass K : Base {\n    int x = F();\n    void _bind_methods() {\n        new T();\n        ADD_SIGNAL(MethodInfo(\"s\"));\n    }\n};\n"
	r, err := cpp.File("m.cpp", src)
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
		if e.Relation == model.RelationImports || e.Relation == model.RelationReferences {
			n = e.Specifier
		}
		got[key{e.Relation, n}] = e.Line
	}
	for k, want := range map[key]int{
		{model.RelationImports, "a.h"}:  1,
		{model.RelationExtends, "Base"}: 3,
		{model.RelationCalls, "F"}:      4,
		{model.RelationCalls, "T"}:      6,
		{model.RelationReferences, "s"}: 7,
	} {
		if got[k] != want {
			t.Errorf("%v line = %d, want %d", k, got[k], want)
		}
	}
}
