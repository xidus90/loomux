package commit_test

import (
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/verify/commit"
)

func TestCheckValidMessages(t *testing.T) {
	validMessages := []string{
		"feat(verify): add native gofmt checker in Go",
		"fix: resolve null pointer dereference in runner",
		"docs: update readme with bilingual details",
		"refactor(guard): simplify path matching logic",
		"chore: bump dependencies and update lockfile",
	}

	policy := commit.DefaultPolicy()
	for _, msg := range validMessages {
		if err := commit.Check(msg, policy); err != nil {
			t.Errorf("expected valid for %q, got error: %v", msg, err)
		}
	}
}

func TestCheckProbesRegression(t *testing.T) {
	policy := commit.DefaultPolicy()

	// 5 probes fail language
	for _, msg := range []string{
		"feat: füge neue sprachprüfung hinzu",
		"korrigiere fehler in der verifikation",
		"aktualisiere dokumentation und beispiele",
		"WIP: ändere dateien",
		"Verbessere Performance für Windows",
	} {
		err := commit.Check(msg, policy)
		if err == nil {
			t.Errorf("expected error for %q, got nil", msg)
		}
		if !strings.Contains(err.Error(), "this message reads as German") {
			t.Errorf("expected German refusal for %q, got: %v", msg, err)
		}
	}

	// probe 6 fails conventional header
	probe6 := "entferne ungenutzte importe"
	err := commit.Check(probe6, policy)
	if err == nil || !strings.Contains(err.Error(), "<type>") {
		t.Errorf("expected conventional header error for %q, got: %v", probe6, err)
	}

	// empty messages
	for _, empty := range []string{"", "   \n\t  ", "# only a comment\n"} {
		err := commit.Check(empty, policy)
		if err == nil || err.Error() != "loomux check commit-msg: commit message cannot be empty" {
			t.Errorf("expected empty error for %q, got: %v", empty, err)
		}
	}
}

func TestCheckConventionalHeaders(t *testing.T) {
	policy := commit.DefaultPolicy()

	for _, msg := range []string{
		"# comment\nfix: x\n\nbody\n",
		"Merge branch 'x' into y",
		"fixup! feat: x",
		`Revert "feat: x"`,
	} {
		if err := commit.Check(msg, policy); err != nil {
			t.Errorf("%q: %v", msg, err)
		}
	}

	for _, msg := range []string{"Add a thing", "feature: x", "feat(): x", "# fix: x\nAdd a thing"} {
		err := commit.Check(msg, policy)
		if err == nil || !strings.HasPrefix(err.Error(), "loomux check commit-msg: ") || !strings.Contains(err.Error(), "<type>") {
			t.Errorf("%q: got %v", msg, err)
		}
	}
}

func TestCheckBothLanguageAndHeaderErrors(t *testing.T) {
	policy := commit.DefaultPolicy()
	// Invalid header format and German prose
	msg := "feat(): Fehler beim Laden der Datei behoben"
	err := commit.Check(msg, policy)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msgText := err.Error()
	if !strings.Contains(msgText, "this message reads as German") {
		t.Errorf("expected language report, got: %s", msgText)
	}
	if !strings.Contains(msgText, "\nloomux check commit-msg: ") || !strings.Contains(msgText, "<type>") {
		t.Errorf("expected prefixed header error, got: %s", msgText)
	}
	if !strings.Contains(msgText, "hits: fehler, beim, der, datei") {
		t.Errorf("expected hits line, got: %s", msgText)
	}
	if !strings.Contains(msgText, "Rewrite it, or use `git commit --no-verify`") {
		t.Errorf("expected way out footer, got: %s", msgText)
	}
}

func TestCheckGermanTargetLanguage(t *testing.T) {
	policy := commit.Policy{
		Language:     "de",
		Threshold:    2,
		Conventional: false,
	}

	// German message passes
	german := "Fehler beim Laden der Datei behoben"
	if err := commit.Check(german, policy); err != nil {
		t.Errorf("expected German message to pass under de, got: %v", err)
	}

	// English message fails
	english := "Fix the parser because something broke into pieces"
	err := commit.Check(english, policy)
	if err == nil {
		t.Fatal("expected English message to fail under de, got nil")
	}
	if !strings.Contains(err.Error(), "this message reads as English, and commits here are German.") {
		t.Errorf("expected English refusal, got: %v", err)
	}
}

func TestCheckConventionalDisabled(t *testing.T) {
	policy := commit.Policy{
		Language:     "en",
		Threshold:    2,
		Conventional: false,
	}
	// Non-conventional header passes when conventional is false
	msg := "Add a thing to the project"
	if err := commit.Check(msg, policy); err != nil {
		t.Errorf("expected pass with conventional=false, got: %v", err)
	}
}

// Python refuses to decode a message that is not UTF-8 and exits 1; Go reads
// the bytes and judges the text that is there.
func TestCheckReadsInvalidUTF8(t *testing.T) {
	policy := commit.DefaultPolicy()
	if err := commit.Check("fix: caf\xe9 the parser", policy); err != nil {
		t.Errorf("expected the message to pass, got: %v", err)
	}
	err := commit.Check("fix: der und das \xff", policy)
	if err == nil || !strings.Contains(err.Error(), "hits:") {
		t.Errorf("expected the German line to be refused, got: %v", err)
	}
}
