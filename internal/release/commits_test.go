package release

import (
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/conventional"
)

func TestCheckCommits(t *testing.T) {
	msgs := []string{"feat: x", "fix: y", "chore: z", "Merge branch 'a'", "docs: w\n\nBREAKING CHANGE: gone"}
	got := CheckCommits("patch", msgs)
	want := []string{
		`commit "feat: x" needs at least release:minor, the label is release:patch`,
		`commit "docs: w" needs at least release:major, the label is release:patch`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
	if got := CheckCommits("major", msgs); got != nil {
		t.Fatalf("major: %q", got)
	}
	if got := CheckCommits("none", []string{"chore: a", ""}); got != nil {
		t.Fatalf("none: %q", got)
	}
}

func TestCheckCommitsRejectsAHeaderOutsideTheForm(t *testing.T) {
	// A header that is not Conventional Commits asks for no level, so without
	// its own check it would pass under every label -- the one commit title
	// the local hook would have refused slips through a pull request.
	msgs := []string{"Add a thing", "feature: y", "fix(): z", "chore: fine", "Merge branch 'a'", `Revert "feat: x"`}
	got := CheckCommits("major", msgs)
	want := []string{
		`commit "Add a thing": header "Add a thing" is not of the form ` + conventional.Form,
		`commit "feature: y": type "feature" is unknown; expected ` + conventional.Form,
		`commit "fix(): z": scope "" must be non-empty and without spaces; expected ` + conventional.Form,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
}
