package schema

import (
	"slices"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestSplitListReadsWhatAHumanTypes(t *testing.T) {
	for in, want := range map[string][]string{
		"":                        nil,
		" , ":                     nil,
		"a,":                      {"a"},
		"a, , b":                  {"a", "b"},
		"a, b":                    {"a", "b"},
		"docs/**/*.{md,txt}, src": {"docs/**/*.{md,txt}", "src"},
		"a/{b,{c,d}}, e":          {"a/{b,{c,d}}", "e"},
		"x}, y{":                  {"x}", "y{"},
		`"a,b", c`:                {"a,b", "c"},
		` "a, {b" , c`:            {"a, {b", "c"},
		`""`:                      {""},
		`"", a`:                   {"", "a"},
		`" padded "`:              {" padded "},
		`"say \"hi\", then", x`:   {`say "hi", then`, "x"},
		`"back\\slash"`:           {`back\slash`},
		`say "hi"`:                {`say "hi"`},
	} {
		got, err := SplitList(in)
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("SplitList(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestSplitListRefusesABrokenQuote(t *testing.T) {
	for _, in := range []string{`"a`, `"a\"`, `"a" b`, `"a""b"`, "\"a\"\nb = 1", `a, "b`, `"a" # note`, `"a" b, c`} {
		if got, err := SplitList(in); err == nil {
			t.Errorf("SplitList(%q) = %q, want an error", in, got)
		}
	}
	// A quote left open says so, not that the item is malformed.
	if _, err := SplitList(`a, "b`); err == nil || !strings.Contains(err.Error(), "a quoted item is not closed") {
		t.Errorf("SplitList of an open quote: %v", err)
	}
}

// What JoinList shows, typed back, is the same list: the interactive form
// offers a list's current value in this form for editing.
func TestJoinListRoundTrips(t *testing.T) {
	lists := [][]string{
		{"a", "b"},
		{"a,b", "c"},
		{"docs/**/*.{md,txt}", "src"},
		{""},
		{" padded", "x "},
		{`"quoted"`, `say "hi"`},
		{`back\slash`, "a, {b"},
		{"x}, y{"},
		{"x{", "y"},
		{"x}{", "y"},
		{"a{b", "c}", "d"},
	}
	for _, k := range Keys() {
		if k.Kind != StringList || k.Default == "" {
			continue
		}
		var doc map[string]any
		if _, err := toml.Decode("v = "+k.Default, &doc); err != nil {
			t.Fatalf("%s: %v", k.ID(), err)
		}
		var items []string
		for _, item := range doc["v"].([]any) {
			items = append(items, item.(string))
		}
		lists = append(lists, items)
	}
	for _, list := range lists {
		typed := JoinList(list)
		got, err := SplitList(typed)
		if err != nil || !slices.Equal(got, list) {
			t.Errorf("%q → %q → %q, %v", list, typed, got, err)
		}
	}
}

func TestJoinListQuotesOnlyWhereNeeded(t *testing.T) {
	if got := JoinList([]string{"docs/**/*.{md,txt}", "a,b", ""}); got != `docs/**/*.{md,txt}, "a,b", ""` {
		t.Errorf("JoinList = %s", got)
	}
}
