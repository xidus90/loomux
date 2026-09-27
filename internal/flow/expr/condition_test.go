package expr_test

import (
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/expr"
)

var names = expr.Names{
	Fields: map[string]flow.Type{
		"verdict": flow.String,
		"count":   flow.Int,
		"done":    flow.Bool,
		"notes":   flow.StringList,
	},
	Params: map[string]flow.Type{
		"max_rounds": flow.Int,
		"target":     flow.String,
		"strict":     flow.Bool,
		"tags":       flow.StringList,
	},
	Predicates: map[string]flow.Predicate{
		"green": func(state flow.State, _ flow.Params) bool { return state.Fields["verdict"] == "green" },
	},
}

var params = flow.Params{"max_rounds": 3, "target": "done", "strict": true, "tags": []string{"a"}}

func state(verdict string, count int, done bool, notes []string) flow.State {
	return flow.State{Fields: map[string]flow.Value{
		"verdict": verdict, "count": count, "done": done, "notes": notes,
	}}
}

func TestConditionsHoldAsWritten(t *testing.T) {
	cases := []struct {
		source string
		state  flow.State
		want   bool
	}{
		{`verdict == "done"`, state("done", 0, false, nil), true},
		{`verdict == "done"`, state("open", 0, false, nil), false},
		{`verdict != "done"`, state("open", 0, false, nil), true},
		{`verdict == target`, state("done", 0, false, nil), true},
		{`verdict == "say \"hi\" \\ bye"`, state(`say "hi" \ bye`, 0, false, nil), true},
		{`count < max_rounds`, state("", 2, false, nil), true},
		{`count < max_rounds`, state("", 3, false, nil), false},
		{`count <= 3`, state("", 3, false, nil), true},
		{`count > -1`, state("", 0, false, nil), true},
		{`count >= 4`, state("", 3, false, nil), false},
		{`count == 3`, state("", 3, false, nil), true},
		{`count != 3`, state("", 3, false, nil), false},
		{`  count==3  `, state("", 3, false, nil), true},
		{`done == true`, state("", 0, true, nil), true},
		{`done != false`, state("", 0, false, nil), false},
		{`done == strict`, state("", 0, true, nil), true},
		{`notes == []`, state("", 0, false, nil), true},
		{`notes == []`, state("", 0, false, []string{"x"}), false},
		{`notes != []`, state("", 0, false, []string{"x"}), true},
		{`green`, state("green", 0, false, nil), true},
		{`green`, state("red", 0, false, nil), false},
		{`green | count > 5`, state("red", 6, false, nil), true},
		{`green | count > 5`, state("red", 1, false, nil), false},
		{`green & count > 5`, state("green", 6, false, nil), true},
		{`green & count > 5`, state("green", 1, false, nil), false},
	}
	for _, c := range cases {
		condition, err := expr.ParseCondition(c.source, names)
		if err != nil {
			t.Fatalf("%s: %v", c.source, err)
		}
		if got := condition.Holds(c.state, params); got != c.want {
			t.Errorf("%s: Holds = %v, want %v", c.source, got, c.want)
		}
	}
}

// Every refusal names what it found and, where there is a list to choose from,
// the list. The loader puts the file and the edge in front.
func TestConditionsAreRefusedWithAReason(t *testing.T) {
	cases := []struct{ source, want string }{
		{``, "the condition is empty"},
		{`   `, "the condition is empty"},
		{`verdit == "done"`, `reads "verdit"; known fields: count, done, notes, verdict`},
		{`green | done == true & count > 1`, "the condition mixes | and &; use only one of them"},
		{`| green`, "| has no term before it"},
		{`green | | green`, "| has no term before it"},
		{`green &`, "& has no term after it"},
		{`nothing`, `names "nothing", which is neither a field nor a predicate; known predicates: green`},
		{`count`, `field "count" needs a comparison`},
		{`count == 3 3`, `cannot read "count == 3 3"; expected <field> <op> <value> or a predicate name`},
		{`"a" == verdict`, `cannot read "\"a\" == verdict"`},
		{`count = 3`, "unknown operator at column 7; operators are ==, !=, <, <=, >, >="},
		{`count ! 3`, "unknown operator at column 7"},
		{`count == (3)`, "unexpected '(' at column 10"},
		{`count == -`, "a lone - at column 10 is not a number"},
		{`verdict == "open`, "has no closing quote"},
		{`verdict == "open\`, "has no closing quote"},
		{`count == 99999999999999999999`, "99999999999999999999 is too large for an int"},
		{`count == nope`, `compares "count" with "nope", which is no parameter; known parameters: max_rounds, strict, tags, target`},
		{`count == "3"`, `compares int field "count" with a string`},
		{`notes == tags`, `list field "notes" compares only with []`},
		{`verdict < "b"`, `< needs an int, but "verdict" is string`},
		{`notes <= []`, `<= needs an int, but "notes" is list[string]`},
		{`count == ==`, `cannot compare "count" with ==`},
	}
	for _, c := range cases {
		_, err := expr.ParseCondition(c.source, names)
		if err == nil {
			t.Errorf("%q: want an error containing %q, got none", c.source, c.want)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: error %q does not contain %q", c.source, err, c.want)
		}
	}
}

// With nothing declared the list is not left empty: "known predicates: " with
// nothing behind it reads like a message that was cut off.
func TestAnEmptyListOfNamesSaysNone(t *testing.T) {
	_, err := expr.ParseCondition(`anything`, expr.Names{})
	if err == nil || !strings.Contains(err.Error(), "known predicates: none") {
		t.Fatalf("want known predicates: none, got %v", err)
	}
}
