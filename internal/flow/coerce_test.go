package flow_test

import (
	"encoding/json"
	"testing"

	"github.com/xidus90/loomux/internal/flow"
)

func TestCoerceTakesEveryShapeAValueArrivesIn(t *testing.T) {
	cases := []struct {
		name string
		kind flow.Type
		raw  any
		want flow.Value
	}{
		{"a string stays one", flow.String, "x", "x"},
		{"an int stays one", flow.Int, 1, 1},
		{"TOML hands out int64", flow.Int, int64(2), 2},
		{"JSON without UseNumber hands out float64", flow.Int, float64(3), 3},
		{"the journal hands out json.Number", flow.Int, json.Number("4"), 4},
		{"a json.Number spelled as a float", flow.Int, json.Number("1.0"), 1},
		{"a json.Number in exponent form", flow.Int, json.Number("1e3"), 1000},
		{"a bool stays one", flow.Bool, true, true},
		{"a list of strings stays one", flow.StringList, []string{"a"}, []string{"a"}},
		{"JSON hands out []any", flow.StringList, []any{"a", "b"}, []string{"a", "b"}},
		{"an empty []any is an empty list", flow.StringList, []any{}, []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := flow.Coerce(c.kind, c.raw)
			if err != nil {
				t.Fatal(err)
			}
			if list, ok := c.want.([]string); ok {
				gotList, ok := got.([]string)
				if !ok {
					t.Fatalf("got %T, want []string", got)
				}
				if len(gotList) != len(list) {
					t.Fatalf("got %v, want %v", gotList, list)
				}
				for i := range list {
					if gotList[i] != list[i] {
						t.Fatalf("got %v, want %v", gotList, list)
					}
				}
				return
			}
			if got != c.want {
				t.Fatalf("got %#v, want %#v", got, c.want)
			}
		})
	}
}

// The whole reason Entries reads with UseNumber: 2^53+1 has no float64.
func TestCoerceKeepsAnIntTooLargeForAFloat(t *testing.T) {
	got, err := flow.Coerce(flow.Int, json.Number("9007199254740993"))
	if err != nil {
		t.Fatal(err)
	}
	if got != 9007199254740993 {
		t.Fatalf("got %v, want 9007199254740993", got)
	}
}

func TestCoerceRefusesWhatIsNotTheDeclaredType(t *testing.T) {
	cases := []struct {
		name string
		kind flow.Type
		raw  any
		want string
	}{
		{"an int is no string", flow.String, 1, `cannot read 1 as string`},
		{"a string is no int", flow.Int, "1", `cannot read 1 as int`},
		{"a fraction is no int", flow.Int, float64(3.5), `cannot read 3.5 as int; it is not a whole number`},
		{"a fraction is no int in json.Number either", flow.Int, json.Number("3.5"), `cannot read 3.5 as int; it is not a whole number`},
		{"a number too large for an int", flow.Int, json.Number("99999999999999999999"), `cannot read 99999999999999999999 as int`},
		{"a float too large for an int is no fraction either", flow.Int, float64(1e19), `cannot read 1e+19 as int`},
		{"a json.Number that is no number at all", flow.Int, json.Number("x"), `cannot read x as int`},
		{"an int is no bool", flow.Bool, 1, `cannot read 1 as bool`},
		{"a string is no list", flow.StringList, "a", `cannot read a as list[string]`},
		{"a list of ints is no list of strings", flow.StringList, []any{1}, `cannot read 1 as string`},
		{"an unknown type", flow.Type("date"), "x", `unknown type "date"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := flow.Coerce(c.kind, c.raw)
			if err == nil {
				t.Fatal("want an error")
			}
			if err.Error() != c.want {
				t.Fatalf("got %q, want %q", err, c.want)
			}
		})
	}
}
