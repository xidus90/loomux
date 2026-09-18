package commit_test

import (
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/verify/commit"
)

func TestValidateCommitMessage(t *testing.T) {
	validMessages := []string{
		"feat(verify): add native gofmt checker in Go",
		"fix: resolve null pointer dereference in runner",
		"docs: update readme with bilingual details",
		"refactor(guard): simplify path matching logic",
		"chore: bump dependencies and update lockfile",
	}

	for _, msg := range validMessages {
		if err := commit.ValidateCommitMessage(msg); err != nil {
			t.Errorf("expected valid for %q, got error: %v", msg, err)
		}
	}

	invalidMessages := []string{
		"feat: füge neue sprachprüfung hinzu",
		"korrigiere fehler in der verifikation",
		"aktualisiere dokumentation und beispiele",
		"WIP: ändere dateien",
		"entferne ungenutzte importe",
		"Verbessere Performance für Windows",
		"",
		"   \n\t  ",
	}

	for _, msg := range invalidMessages {
		if err := commit.ValidateCommitMessage(msg); err == nil {
			t.Errorf("expected error for invalid/German message %q, got nil", msg)
		}
	}
}

func TestValidateCommitMessageHeader(t *testing.T) {
	for _, msg := range []string{
		"# comment\nfix: x\n\nbody\n",
		"Merge branch 'x' into y",
		"fixup! feat: x",
		`Revert "feat: x"`,
	} {
		if err := commit.ValidateCommitMessage(msg); err != nil {
			t.Errorf("%q: %v", msg, err)
		}
	}
	for _, msg := range []string{"Add a thing", "feature: x", "feat(): x", "# fix: x\nAdd a thing"} {
		err := commit.ValidateCommitMessage(msg)
		if err == nil || !strings.Contains(err.Error(), "<type>") {
			t.Errorf("%q: got %v", msg, err)
		}
	}
}
