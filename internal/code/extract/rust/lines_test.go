package rust_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
)

func TestSpansAndBindingLines(t *testing.T) {
	src := "/// doc\n#[derive(GodotClass)]\nstruct S {\n    a: i32,\n}\n\n#[godot_api]\nimpl S {\n    /// doc\n    #[func]\n    fn f(&self) {\n    }\n}\n"
	r := extractSrc(t, src)
	spans := map[model.NodeID]model.Span{}
	for _, n := range r.Nodes {
		spans[n.ID] = n.Span
	}
	for id, want := range map[model.NodeID]model.Span{"m.rs#S": "L3-L5", "m.rs#S.f": "L11-L12"} {
		if spans[id] != want {
			t.Errorf("%s span = %s, want %s", id, spans[id], want)
		}
	}
	bindings := 0
	for _, e := range r.Edges {
		if e.Relation != model.RelationReferences {
			continue
		}
		bindings++
		if e.Line != 11 {
			t.Errorf("binding line = %d, want 11", e.Line)
		}
	}
	if bindings != 1 {
		t.Errorf("bindings = %d, want 1", bindings)
	}
}
