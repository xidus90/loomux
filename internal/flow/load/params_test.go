package load

import (
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/flow"
)

func declaredParams() *flow.Graph {
	return &flow.Graph{Name: "example", Params: map[string]flow.Field{
		"rounds": {Type: flow.Int, Default: 5},
		"strict": {Type: flow.Bool, Default: false},
		"topic":  {Type: flow.String, Default: "tests"},
		"tags":   {Type: flow.StringList, Default: []string{}},
	}}
}

func TestParamsAreTheDefaultsWithoutOptions(t *testing.T) {
	got, err := Params(declaredParams(), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := flow.Params{"rounds": 5, "strict": false, "topic": "tests", "tags": []string{}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

// An option arrives as text, from the command line or from a marker, and
// leaves in the Go type its parameter declares.
func TestAnOptionTakesTheDeclaredType(t *testing.T) {
	got, err := Params(declaredParams(), map[string]string{
		"rounds": "7", "strict": "true", "topic": "a b", "tags": `["x","y"]`,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := flow.Params{"rounds": 7, "strict": true, "topic": "a b", "tags": []string{"x", "y"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

// false is read, not merely accepted: over a default of true it has to win.
func TestAFalseOptionOverridesATrueDefault(t *testing.T) {
	graph := &flow.Graph{Name: "example", Params: map[string]flow.Field{
		"strict": {Type: flow.Bool, Default: true},
	}}
	got, err := Params(graph, map[string]string{"strict": "false"})
	if err != nil {
		t.Fatal(err)
	}
	if want := (flow.Params{"strict": false}); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

// Every option it cannot use, one line each and in name order, the way a load
// stage reports every finding it has.
func TestParamsRefuseEveryOptionTheyCannotRead(t *testing.T) {
	_, err := Params(declaredParams(), map[string]string{
		"round": "7", "rounds": "seven", "strict": "yes", "tags": "x,y",
	})
	want := `option round is no parameter of flow "example"; known parameters: rounds, strict, tags, topic
option rounds: cannot read "seven" as int
option strict: cannot read "yes" as bool
option tags: cannot read "x,y" as list[string]`
	if err == nil || err.Error() != want {
		t.Fatalf("err =\n%v\nwant\n%s", err, want)
	}
}

func TestParamsOfAFlowWithoutParameters(t *testing.T) {
	_, err := Params(&flow.Graph{Name: "example"}, map[string]string{"x": "1"})
	want := `option x is no parameter of flow "example"; known parameters: none`
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v", err)
	}
}

// null is valid JSON and no list: read as one, it would put a nil into a
// state that promises a []string.
func TestParamsRefuseNullForAList(t *testing.T) {
	_, err := Params(declaredParams(), map[string]string{"tags": "null"})
	want := `option tags: cannot read "null" as list[string]`
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v", err)
	}
}
