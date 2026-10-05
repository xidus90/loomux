package detect

import (
	"errors"
	"strings"
	"testing"
)

func TestHooksPathReportsWhatGitHasSet(t *testing.T) {
	var asked []string
	run := func(dir string, argv ...string) (string, error) {
		asked = append(asked, dir+": "+strings.Join(argv, " "))
		return ".githooks\n", nil
	}
	got, err := HooksPath(run, "/project")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != ".githooks" {
		t.Fatalf("hooks path = %q, want %q", got, ".githooks")
	}
	// As a path: git expands ~/.githooks where a raw read would not.
	want := "/project: git config --type=path --get core.hooksPath"
	if len(asked) != 1 || asked[0] != want {
		t.Fatalf("asked = %v, want [%q]", asked, want)
	}
}

// The common case: nothing is set, and that is an answer rather than a fault.
func TestHooksPathIsEmptyWhenUnset(t *testing.T) {
	got, err := HooksPath(func(string, ...string) (string, error) { return "", nil }, ".")
	if err != nil || got != "" {
		t.Fatalf("hooks path = %q, err = %v, want empty and nil", got, err)
	}
}

func TestHooksPathPassesGitsFailureOn(t *testing.T) {
	broken := errors.New("git not found")
	got, err := HooksPath(func(string, ...string) (string, error) { return "", broken }, ".")
	if !errors.Is(err, broken) {
		t.Fatalf("err = %v, want %v", err, broken)
	}
	if got != "" {
		t.Fatalf("hooks path = %q, want empty", got)
	}
}
