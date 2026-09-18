package conventional_test

import (
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/conventional"
)

func TestParse(t *testing.T) {
	good := []string{"feat: x", "fix(commit-msg): y", "chore(release)!: z", "revert!: w"}
	for _, h := range good {
		if _, err := conventional.Parse(h); err != nil {
			t.Errorf("%q: %v", h, err)
		}
	}
	bad := map[string]string{
		"Add a thing":   "not of the form",
		"feature: x":    `type "feature" is unknown`,
		"feat(): x":     "scope",
		"feat(a b): x":  "scope",
		"feat((a)): x":  "not of the form",
		"feat:  ":       "description is empty",
		"feat:x":        "not of the form",
		"feat(a)(b): x": "not of the form",
	}
	for h, want := range bad {
		_, err := conventional.Parse(h)
		if err == nil || !strings.Contains(err.Error(), want) || !strings.Contains(err.Error(), "<type>") {
			t.Errorf("%q: got %v, want %q", h, err, want)
		}
	}
}

func TestExempt(t *testing.T) {
	for _, h := range []string{"Merge branch 'x'", "fixup! feat: x", "squash! a", "amend! a", `Revert "feat: x"`} {
		if !conventional.Exempt(h) {
			t.Errorf("%q not exempt", h)
		}
	}
	if conventional.Exempt("Reverted stuff") {
		t.Error("Reverted exempt")
	}
}

func TestMessage(t *testing.T) {
	if got := conventional.Message("# only\n\n"); got != nil {
		t.Fatalf("got %q", got)
	}
	got := conventional.Message("# c\r\nfeat: x\r\n\r\nbody\n# c\n")
	if len(got) != 3 || got[0] != "feat: x" || got[2] != "body" {
		t.Fatalf("got %q", got)
	}
}

func TestLevel(t *testing.T) {
	cases := map[string]string{
		"":                                   "none",
		"Merge branch 'x'":                   "none",
		"nonsense":                           "none",
		"chore: x":                           "none",
		"fix: x":                             "patch",
		"feat(a): x":                         "minor",
		"fix!: x":                            "major",
		"fix: x\n\nBREAKING CHANGE: gone":    "major",
		"docs: x\n\nBREAKING-CHANGE: gone":   "major",
		"feat: x\n\nnot BREAKING CHANGE: no": "minor",
	}
	for msg, want := range cases {
		if got := conventional.Level(msg); got != want {
			t.Errorf("%q: got %s, want %s", msg, got, want)
		}
	}
}
