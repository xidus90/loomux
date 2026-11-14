package target

import (
	"strings"
	"testing"
)

var rulePairs = [][2]string{{"match", `["a"]`}, {"reason", `"r"`}}

func TestAppendRuleAddsABlockWhereThereIsNoArray(t *testing.T) {
	tgt := Project(t.TempDir())
	next, err := tgt.AppendRule("[commit]\nlanguage = \"de\"\n", "policy.paths.rules", rulePairs)
	want := "[commit]\nlanguage = \"de\"\n\n[[policy.paths.rules]]\nmatch = [\"a\"]\nreason = \"r\"\n"
	if err != nil || next != want {
		t.Fatalf("%q %v", next, err)
	}
	// A second rule follows the first in the same form.
	next, err = tgt.AppendRule(next, "policy.paths.rules", rulePairs)
	if err != nil || strings.Count(next, "[[policy.paths.rules]]") != 2 {
		t.Fatalf("%q %v", next, err)
	}
}

func TestAppendRuleExtendsAnInlineArray(t *testing.T) {
	tgt := Project(t.TempDir())
	text := "[policy.paths]\nrules = [\n  { match = [\"b\"], reason = \"s\" }\n]\n"
	next, err := tgt.AppendRule(text, "policy.paths.rules", rulePairs)
	if err != nil || strings.Contains(next, "[[") || strings.Count(next, "match =") != 2 {
		t.Fatalf("%q %v", next, err)
	}
}

func TestAppendRuleRefusesWhatItCannotPlaceOrWhatTheReadersRefuse(t *testing.T) {
	tgt := Project(t.TempDir())
	// An array spelled as a dotted key has no table to put the item in.
	if _, err := tgt.AppendRule("policy.paths.rules = []\n", "policy.paths.rules", rulePairs); err == nil {
		t.Error("a dotted-key array took a rule")
	}
	if _, err := tgt.AppendRule("", "policy.paths.rules", [][2]string{{"match", "nope"}}); err == nil {
		t.Error("a rule the reader refuses was taken")
	}
}

func TestRemoveRuleDropsOneEntryAndChecksTheRest(t *testing.T) {
	tgt := Project(t.TempDir())
	text := "[[policy.paths.rules]]\nmatch = [\"a\"]\nreason = \"r\"\n\n[[policy.paths.rules]]\nmatch = [\"b\"]\nreason = \"s\"\n"
	next, err := tgt.RemoveRule(text, "policy.paths.rules", 0)
	if want := "[[policy.paths.rules]]\nmatch = [\"b\"]\nreason = \"s\"\n"; err != nil || next != want {
		t.Fatalf("%q %v", next, err)
	}
	if _, err := tgt.RemoveRule(text, "policy.paths.rules", 2); err == nil {
		t.Error("an entry that is not there was removed")
	}
	// The reader refuses what is left: the remaining rule has a bad pattern.
	bad := text + "\n[[policy.paths.rules]]\nmatch = \"nope\"\n"
	if _, err := tgt.RemoveRule(bad, "policy.paths.rules", 0); err == nil {
		t.Error("a file the readers refuse was written")
	}
}
