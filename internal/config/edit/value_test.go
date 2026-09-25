package edit

import (
	"reflect"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/config/schema"
)

func TestRenderMakesALiteralPerKind(t *testing.T) {
	cases := []struct {
		kind  schema.Kind
		in    string
		want  string
		fails bool
	}{
		{schema.String, `docs/wiki`, `"docs/wiki"`, false},
		{schema.Enum, `de`, `"de"`, false},
		{schema.Int, `3`, `3`, false},
		{schema.Int, `three`, ``, true},
		{schema.Int, `+600`, `600`, false},
		{schema.Int, `007`, `7`, false},
		{schema.Bool, `true`, `true`, false},
		{schema.Bool, `false`, `false`, false},
		{schema.Bool, `no`, ``, true},
		{schema.StringList, `a, b`, `["a", "b"]`, false},
		{schema.StringList, ``, `[]`, false},
		{schema.StringList, `a,`, `["a"]`, false},
		{schema.StringList, `a, , b`, `["a", "b"]`, false},
		{schema.StringList, ` , `, `[]`, false},
		{schema.StringList, `"a,b", c`, `["a,b", "c"]`, false},
		{schema.StringList, `""`, `[""]`, false},
		{schema.StringList, `"a`, ``, true},
		{schema.Table, `x`, ``, true},
	}
	for _, c := range cases {
		got, err := Render(c.kind, c.in)
		if (err != nil) != c.fails || got != c.want {
			t.Errorf("Render(%v, %q) = %q, %v", c.kind, c.in, got, err)
		}
	}
}

// TestRenderRoundTripsThroughTheDecoder holds Render and the input form of
// schema.Current together: what Render writes decodes back to what was typed.
func TestRenderRoundTripsThroughTheDecoder(t *testing.T) {
	for _, c := range []struct {
		kind schema.Kind
		in   string
		want any
	}{
		{schema.String, `docs\wiki "x"`, `docs\wiki "x"`},
		{schema.StringList, "a, b", []any{"a", "b"}},
	} {
		literal, err := Render(c.kind, c.in)
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		if _, err := toml.Decode("v = "+literal, &doc); err != nil {
			t.Fatalf("%s does not decode: %v", literal, err)
		}
		if !reflect.DeepEqual(doc["v"], c.want) {
			t.Errorf("%q → %s → %#v", c.in, literal, doc["v"])
		}
	}
}

// A comma inside a {…} brace group belongs to a glob, not to the list: what
// `config list` shows for worktree.mirror, typed back, is the same list.
func TestRenderKeepsBraceGroupsWhole(t *testing.T) {
	cases := map[string]string{
		"docs/**/*.{md,txt}":             `["docs/**/*.{md,txt}"]`,
		"a/{b,{c,d}}, e":                 `["a/{b,{c,d}}", "e"]`,
		"x}, y{":                         `["x}", "y{"]`,
		"docs/**/*.{md,txt}, src, {a,b}": `["docs/**/*.{md,txt}", "src", "{a,b}"]`,
	}
	for in, want := range cases {
		if got, err := Render(schema.StringList, in); err != nil || got != want {
			t.Errorf("Render(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	text := "[worktree]\nmirror = [\"docs/**/*.{md,txt}\", \"a/{b,{c,d}}\", \"src\"]\n"
	entries, err := schema.Current(text)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Key.ID() != "worktree.mirror" {
			continue
		}
		literal, err := Render(schema.StringList, e.Input)
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		if _, err := toml.Decode("v = "+literal, &doc); err != nil {
			t.Fatal(err)
		}
		if want := []any{"docs/**/*.{md,txt}", "a/{b,{c,d}}", "src"}; !reflect.DeepEqual(doc["v"], want) {
			t.Fatalf("%q → %s → %#v", e.Input, literal, doc["v"])
		}
		return
	}
	t.Fatal("worktree.mirror not listed")
}
