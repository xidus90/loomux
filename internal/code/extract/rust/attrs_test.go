package rust

import (
	"reflect"
	"testing"
)

func TestAttrValue(t *testing.T) {
	for _, tc := range []struct {
		args, key, want string
	}{
		{"(rename=X)", "rename", "X"},
		{"(init, base = Node, rename = Foo )", "rename", "Foo"},
		{"(init)", "rename", ""},
		{"", "rename", ""},
		{"(renamed = X)", "rename", ""},
	} {
		if got := (attr{args: tc.args}).value(tc.key); got != tc.want {
			t.Errorf("value(%q, %q) = %q, want %q", tc.args, tc.key, got, tc.want)
		}
	}
}

func TestAttrWords(t *testing.T) {
	got := attr{args: "(Debug,  godot::prelude::GodotClass )"}.words()
	if want := []string{"Debug", "godot::prelude::GodotClass"}; !reflect.DeepEqual(got, want) {
		t.Errorf("words = %q, want %q", got, want)
	}
	if got := (attr{}).words(); got != nil {
		t.Errorf("words of no arguments = %q, want none", got)
	}
}

func TestGodotClassNeedsTheDerive(t *testing.T) {
	derive := attr{name: "derive", args: "(GodotClass)"}
	class := attr{name: "class", args: "(rename = X)"}
	for _, tc := range []struct {
		attrs []attr
		want  string
	}{
		{[]attr{derive}, "S"},
		{[]attr{derive, class}, "X"},
		{[]attr{class}, ""},
		{[]attr{{name: "derive", args: "(Debug)"}}, ""},
		{[]attr{{name: "serde", args: "(GodotClass)"}}, ""},
		{nil, ""},
	} {
		if got := godotClass(tc.attrs, "S"); got != tc.want {
			t.Errorf("godotClass(%v) = %q, want %q", tc.attrs, got, tc.want)
		}
	}
}

func TestLastSegment(t *testing.T) {
	for in, want := range map[string]string{"test": "test", "tokio::test": "test", "a::b::c": "c", "": ""} {
		if got := lastSegment(in); got != want {
			t.Errorf("lastSegment(%q) = %q, want %q", in, got, want)
		}
	}
}
