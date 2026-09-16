package model_test

import (
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
)

func TestRelationIsWalk(t *testing.T) {
	walk := []model.Relation{
		model.RelationCalls,
		model.RelationReferences,
		model.RelationImports,
		model.RelationImplements,
		model.RelationExtends,
	}
	for _, r := range walk {
		if !r.IsWalk() {
			t.Errorf("relation %q must be walkable", r)
		}
	}
	// A file contains every symbol defined in it. Walking that edge would make
	// each sibling a neighbour and turn the file into a hub the graph never
	// earned.
	if model.RelationContains.IsWalk() {
		t.Error(`relation "contains" must never be walkable`)
	}
	if model.Relation("nonsense").IsWalk() {
		t.Error("an unknown relation must not be walkable")
	}
}

func TestSymbolsInFile(t *testing.T) {
	g := &model.Graph{Nodes: []model.Node{
		{ID: "src/a.ts", Kind: model.KindFile, Path: "src/a.ts"},
		{ID: "src/a.ts#helper", Kind: "function", Path: "src/a.ts"},
		{ID: "src/a.ts#Other.m", Kind: "method", Path: "src/a.ts"},
		{ID: "src/b.ts#useB", Kind: "function", Path: "src/b.ts"},
	}}

	got := model.SymbolsInFile(g, "src/a.ts")
	want := []model.NodeID{"src/a.ts#helper", "src/a.ts#Other.m"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := model.SymbolsInFile(g, "src/none.ts"); got != nil {
		t.Errorf("got %v, want nil for a path with no symbols", got)
	}
	if got := model.SymbolsInFile(nil, "src/a.ts"); got != nil {
		t.Errorf("got %v, want nil for a nil graph", got)
	}
}
