package release

import (
	"reflect"
	"testing"
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
