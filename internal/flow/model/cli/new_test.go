package cli

import (
	"reflect"
	"testing"
	"time"
)

func TestNewPicksAdapter(t *testing.T) {
	o := Options{Root: "r", Timeout: time.Minute}
	for provider, want := range map[string]any{
		"claude": claude{o}, "agy": agy{o}, "qoder": qoder{o},
	} {
		got, err := New(provider, o)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: got %#v, %v", provider, got, err)
		}
	}
	if _, err := New("gpt", o); err == nil || err.Error() != "no adapter for provider gpt" {
		t.Fatalf("err = %v", err)
	}
}

func TestWarnings(t *testing.T) {
	tools := []string{"Read", "mcp__x"}
	cases := []struct {
		provider string
		want     []string
	}{
		{"claude", nil},
		{"agy", []string{`warning: node "n" runs on agy, which cannot enforce the tool profile mcp; only loomux's guard policy and the write barrier on file-writing tools limit it`}},
		{"qoder", []string{
			`warning: node "n": qoder has no tool for mcp__x; the node runs without it`,
		}},
	}
	for _, c := range cases {
		if got := Warnings(c.provider, "n", "mcp", tools); !reflect.DeepEqual(got, c.want) {
			t.Fatalf("%s: got %q", c.provider, got)
		}
	}
}
