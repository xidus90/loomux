package php_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/extract/php"
	"github.com/xidus90/loomux/internal/code/model"
)

func TestEdgeLines(t *testing.T) {
	src := "<?php\nuse A\\B;\nrequire 'x.php';\nclass K extends Base implements I {\n    use T;\n    function m() {\n        f();\n        X::s();\n        $this->o();\n        new N();\n    }\n}\n"
	r, err := php.File("m.php", src)
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
		{model.RelationImports, `A\B`}:   2,
		{model.RelationImports, "x.php"}: 3,
		{model.RelationExtends, "Base"}:  4,
		{model.RelationExtends, "I"}:     4,
		{model.RelationExtends, "T"}:     5,
		{model.RelationCalls, "f"}:       7,
		{model.RelationCalls, "s"}:       8,
		{model.RelationCalls, "o"}:       9,
		{model.RelationCalls, "N"}:       10,
	} {
		if got[k] != want {
			t.Errorf("%v line = %d, want %d", k, got[k], want)
		}
	}
}
