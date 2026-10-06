package gdscript

import (
	"strings"
	"testing"
)

// The grammar yields no string node without both quotes, so unquote's own
// guards are reached only here.
func TestUnquoteStripsOnlyAMatchedPair(t *testing.T) {
	for in, want := range map[string]string{
		`"a"`: "a", `'a'`: "a", `""`: "", `"`: `"`, `'`: `'`, `"a'`: `"a'`, `a"`: `a"`, "": "", `a`: "a", "`a`": "`a`",
	} {
		if got := unquote(in); got != want {
			t.Errorf("unquote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestScriptReportsAParseFailure(t *testing.T) {
	// The grammar builds a tree from any input; only a missing grammar makes
	// the parse itself fail.
	_, err := script(nil, "a.gd", "func f():\n\tpass\n")
	if err == nil || !strings.Contains(err.Error(), "parse a.gd") {
		t.Errorf("script(nil grammar) = %v, want an error naming the file", err)
	}
}

func TestResourceReportsAParseFailure(t *testing.T) {
	_, err := resource(nil, "x.tscn", "[gd_scene]\n")
	if err == nil || !strings.Contains(err.Error(), "parse x.tscn") {
		t.Errorf("resource(nil grammar) = %v, want an error naming the file", err)
	}
}
