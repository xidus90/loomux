package cli

import (
	"strings"
	"testing"
)

// configuration.md ("Your first day") says `loomux config list` shows how
// many path rules there are; getting-started.md section 2 writes them as an
// inline array. The count must not depend on the TOML form.
func TestConfigListCountsAnInlineRulesArray(t *testing.T) {
	root := configRoot(t, `[policy.paths]
rules = [
  { match = [".env*", "*.pem", "*.key"], reason = "Secrets must not be written by an agent" },
  { match = ["package-lock.json", "go.sum"], reason = "Lock files are managed by package tools" }
]
`)
	_, out, _ := runConfig(t, "", "list", "--root", root)
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "policy.paths.rules") {
			if !strings.Contains(line, "2 entries") {
				t.Fatalf("row: %q; the file holds 2 rules", line)
			}
			return
		}
	}
	t.Fatalf("no policy.paths.rules row:\n%s", out)
}
