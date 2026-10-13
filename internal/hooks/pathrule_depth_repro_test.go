package hooks

import (
	"path/filepath"
	"testing"
)

// configuration.md, [policy.paths] rules[].match: "`**` means any depth as
// the ending `/**` (`.aws/**`)". A rule whose ending /** follows a wildcard
// segment must then hold below the first level too.
func TestAnEndingDoubleStarAfterAWildcardHoldsAtAnyDepth(t *testing.T) {
	root, state := world(t, "[policy.paths]\nrules = [\n  { match = [\"services/*/secrets/**\"], reason = \"service secrets are human-managed\" },\n]\n")
	for _, rel := range []string{
		"services/api/secrets/key.txt",
		"services/api/secrets/prod/key.txt",
	} {
		code, _, errOut := call(t, root, state, payload("Write", "file_path", filepath.Join(root, filepath.FromSlash(rel))))
		if code != 2 {
			t.Errorf("%s: exit %d (%q); the rule's ending /** should hold at any depth", rel, code, errOut)
		}
	}
}
