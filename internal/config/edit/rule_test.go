package edit

import (
	"errors"
	"strings"
	"testing"
)

func TestAppendRuleFollowsTheFormTheFileAlreadyUses(t *testing.T) {
	pairs := [][2]string{{"regex", `'x'`}}
	got, err := AppendRule("[commit]\n", "policy.commands.rules", pairs)
	if err != nil || got != "[commit]\n\n[[policy.commands.rules]]\nregex = 'x'\n" {
		t.Fatalf("block form = %q, %v", got, err)
	}
	got, err = AppendRule("[policy.commands]\nrules = [{ regex = 'y' }]\n", "policy.commands.rules", pairs)
	if err != nil || strings.Contains(got, "[[") || strings.Count(got, "regex =") != 2 {
		t.Fatalf("inline form = %q, %v", got, err)
	}
	if _, err = AppendRule("policy.commands.rules = []\n", "policy.commands.rules", pairs); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("a dotted-key array took a rule: %v", err)
	}
}

func TestRuleNodeReadsBothFormsAndNothingFromBrokenText(t *testing.T) {
	if _, ok := RuleNode("[[a.b]]\nx = 1\n", "a.b").([]map[string]any); !ok {
		t.Error("a [[section]] list is not a []map")
	}
	if _, ok := RuleNode("[a]\nb = [{ x = 1 }]\n", "a.b").([]any); !ok {
		t.Error("an inline array is not a []any")
	}
	if RuleNode("not toml [", "a.b") != nil {
		t.Error("broken text holds something")
	}
}
