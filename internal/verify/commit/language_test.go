package commit_test

import (
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
