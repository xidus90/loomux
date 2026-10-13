package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The guard hooks writes and commands, not reads, and `never` is no rule of
// it: a Read or a `cat` of a `never` path exits 0. docs/{en,de}/configuration.md
// therefore bound [privacy] to loomux's own channels; this test holds the
// premise and the wording together, so that a guard that starts refusing
// reads has to change the docs with it.
func TestThePrivacyDocPromisesNothingTheGuardDoesNotKeep(t *testing.T) {
	root, state := world(t, "[area]\nscope = \"project/x\"\n\n[privacy]\nmode = \"manual_cloud\"\nnever = [\"private/**\"]\n")
	secret := filepath.Join(root, "private", "notes.md")
	os.MkdirAll(filepath.Dir(secret), 0o755)
	os.WriteFile(secret, []byte("api key"), 0o644)
	for _, input := range []string{
		payload("Read", "file_path", secret),
		payload("Bash", "command", "cat private/notes.md"),
	} {
		if code, _, errOut := call(t, root, state, input); code != 0 {
			t.Errorf("%s: exit %d (%s); the privacy doc says reads of a never path are not refused", input, code, errOut)
		}
	}

	for lang, promised := range map[string][]string{
		"en": {"Guarantees", "never leaks", "external models", "remote telemetry"},
		"de": {"Garantiert", "unter keinen Umständen", "externe Modelle"},
	} {
		doc, err := os.ReadFile(filepath.Join("..", "..", "docs", lang, "configuration.md"))
		if err != nil {
			t.Fatal(err)
		}
		_, rest, _ := strings.Cut(string(doc), "### `[privacy]`")
		section, _, _ := strings.Cut(rest, "\n---\n")
		if section == "" {
			t.Fatalf("%s: no [privacy] section", lang)
		}
		for _, word := range promised {
			if strings.Contains(section, word) {
				t.Errorf("%s: the [privacy] section still says %q", lang, word)
			}
		}
		for _, word := range []string{"`Read`", "`cat`"} {
			if !strings.Contains(section, word) {
				t.Errorf("%s: the [privacy] section does not name %s among what it does not cover", lang, word)
			}
		}
	}
}
