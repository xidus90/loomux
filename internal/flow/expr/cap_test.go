package expr_test

import (
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/expr"
)

var capParams = map[string]flow.Type{"max_rounds": flow.Int, "topic": flow.String}

func TestCapsReadEveryForm(t *testing.T) {
	cases := []struct {
		raw  any
		want flow.Cap
	}{
		{nil, flow.Cap{Add: 1}},
		{int64(4), flow.Cap{Add: 4}},
		{"max_rounds", flow.Cap{Param: "max_rounds"}},
		{"max_rounds + 1", flow.Cap{Param: "max_rounds", Add: 1}},
		{"max_rounds+2", flow.Cap{Param: "max_rounds", Add: 2}},
	}
	for _, c := range cases {
		got, err := expr.ParseCap(c.raw, capParams)
		if err != nil {
			t.Fatalf("%v: %v", c.raw, err)
		}
		if got != c.want {
			t.Errorf("%v: got %+v, want %+v", c.raw, got, c.want)
		}
	}
}

func TestCapsAreRefusedWithAReason(t *testing.T) {
	cases := []struct {
		raw  any
		want string
	}{
		{int64(0), "max_visits must be at least 1, got 0"},
		{int64(-2), "max_visits must be at least 1, got -2"},
		{1.5, `max_visits must be an integer or "<parameter> + <integer>", got 1.5`},
		{"max_rounds + x", `max_visits "max_rounds + x" is not "<parameter> + <integer>"`},
		{"max_rounds + -1", `max_visits "max_rounds + -1" is not "<parameter> + <integer>"`},
		{"rounds", `max_visits "rounds" names no parameter; known parameters: max_rounds, topic`},
		{"topic + 1", `max_visits parameter "topic" is string, not int`},
	}
	for _, c := range cases {
		_, err := expr.ParseCap(c.raw, capParams)
		if err == nil {
			t.Errorf("%v: want an error containing %q, got none", c.raw, c.want)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%v: error %q does not contain %q", c.raw, err, c.want)
		}
	}
}
