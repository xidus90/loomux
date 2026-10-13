package setup

import (
	"strings"
	"testing"
)

// configuration.md ("Your first day") says init adds a rule against
// `pip install` to a uv project, and getting-started.md section 2 shows the
// policy rules written as inline arrays. init must manage that form too.
func TestInitAddsThePipRuleOverAnInlineRulesArray(t *testing.T) {
	before := `[policy.commands]
rules = [
  { regex = '(^|\s)git\s+push(\s|$)', reason = "Pushing to remote requires explicit human decision" }
]
`
	after, err := configText(before, []string{"uv"}, Choice{})
	if err != nil {
		t.Fatalf("configText refuses the documented form: %v", err)
	}
	if !strings.Contains(after, "pip") {
		t.Fatalf("no pip rule added:\n%s", after)
	}
}
