package extract_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

func TestResultReadsOldJSON(t *testing.T) {
	var r extract.Result
	if err := json.Unmarshal([]byte(`{"path":"a.gd","language":"gdscript","nodes":[]}`), &r); err != nil {
		t.Fatal(err)
	}
	if r.Exports != nil || r.Bridges != nil || r.Embedded != nil {
		t.Fatalf("an old cache entry read back with bridge fields: %+v", r)
	}
}

func TestResultOmitsEmptyBridgeFields(t *testing.T) {
	b, err := json.Marshal(extract.Result{Path: "a", Language: "x", Nodes: []model.Node{}, Edges: []extract.RawEdge{{File: "a"}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"exports", "bridges", "embedded", "member"} {
		if strings.Contains(string(b), `"`+k+`"`) {
			t.Errorf("%s written although empty: %s", k, b)
		}
	}
}

func TestBridgeFieldsRoundTrip(t *testing.T) {
	in := extract.Result{
		Path: "p.tscn", Language: "gdscript", Nodes: []model.Node{},
		Edges:    []extract.RawEdge{{File: "p.tscn", Member: "signal"}},
		Exports:  []extract.Export{{Namespace: extract.NamespaceGodotClass, Name: "A", Target: "a.gd#A"}},
		Bridges:  []extract.Bridge{{Source: "p.tscn", Relation: model.RelationReferences, Namespace: extract.NamespaceGodotMethod, Name: "m", Scope: []string{"p.cs"}, File: "p.tscn", Line: 3}},
		Embedded: []extract.Result{{Path: "p.tscn", Language: "csharp", Nodes: []model.Node{}}},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out extract.Result
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round trip:\n got %+v\nwant %+v", out, in)
	}
}

func TestFlattenPutsPartsAfterTheFile(t *testing.T) {
	r := extract.Result{Path: "p.html", Language: "html", Embedded: []extract.Result{
		{Path: "p.html", Language: "js"}, {Path: "p.html", Language: "css"},
	}}
	got := r.Flatten()
	if len(got) != 3 || got[0].Language != "html" || got[1].Language != "js" || got[2].Language != "css" {
		t.Fatalf("Flatten = %+v", got)
	}
	for _, p := range got {
		if p.Embedded != nil {
			t.Errorf("%s keeps its parts: %+v", p.Language, p.Embedded)
		}
	}
	if len(r.Embedded) != 2 {
		t.Errorf("Flatten changed its receiver: %+v", r.Embedded)
	}
}

func TestNamespacesListsEveryConstant(t *testing.T) {
	want := []extract.Namespace{"godot.class", "godot.method", "godot.signal"}
	if !reflect.DeepEqual(extract.Namespaces, want) {
		t.Fatalf("Namespaces = %v, want %v", extract.Namespaces, want)
	}
	if extract.NamespaceGodotClass != want[0] || extract.NamespaceGodotMethod != want[1] || extract.NamespaceGodotSignal != want[2] {
		t.Fatal("a constant differs from the spec")
	}
}
