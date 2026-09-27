package model_test

import (
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/flow/model"
)

func TestToolsOfEveryProfile(t *testing.T) {
	cases := map[string][]string{
		"read_only": {"Glob", "Grep", "Read"},
		"edit":      {"Edit", "Glob", "Grep", "Read", "Write"},
		"shell":     {"Bash", "Glob", "Grep", "Read"},
		"mcp":       {"Glob", "Grep", "Read"},
	}
	for profile, want := range cases {
		got, err := model.Tools(profile, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s: got %v, want %v", profile, got, want)
		}
	}
}

func TestOnlyTheMCPProfileTakesServersSortedAndOnce(t *testing.T) {
	got, err := model.Tools("mcp", []string{"github", "brain", "brain"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Glob", "Grep", "Read", "mcp__brain", "mcp__github"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	edit, err := model.Tools("edit", []string{"brain"})
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(edit, "mcp__brain") {
		t.Fatalf("edit takes no servers, got %v", edit)
	}
}

func TestToolsRefusesAnUnknownProfile(t *testing.T) {
	_, err := model.Tools("admin", nil)
	want := `unknown tool profile "admin"; known profiles: edit, mcp, read_only, shell`
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %s", err, want)
	}
}

// The profiles are the only ceiling between an agent node and Bash or Write;
// a caller that edits its copy must not widen the next node's.
func TestToolsHandsOutACopy(t *testing.T) {
	first, err := model.Tools("read_only", nil)
	if err != nil {
		t.Fatal(err)
	}
	first[0] = "Bash"
	again, err := model.Tools("read_only", nil)
	if err != nil {
		t.Fatal(err)
	}
	if again[0] != "Glob" {
		t.Fatalf("a profile was widened through a returned slice: %v", again)
	}
}
