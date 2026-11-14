package edit

import (
	"errors"
	"strconv"
	"testing"

	"github.com/BurntSushi/toml"
)

const ruleSection = "policy.paths.rules"

// itemsOf counts the entries the rule list holds once the text is decoded.
func itemsOf(t *testing.T, text string) int {
	t.Helper()
	doc := map[string]any{}
	if err := toml.Unmarshal([]byte(text), &doc); err != nil {
		t.Fatalf("result is not TOML: %v\n%s", err, text)
	}
	switch v := RuleNode(text, ruleSection).(type) {
	case []map[string]any:
		return len(v)
	case []any:
		return len(v)
	}
	return 0
}

func TestRemoveItemBlocks(t *testing.T) {
	three := "[commit]\nlanguage = \"de\"\n\n" +
		"[[policy.paths.rules]]\nmatch = [\"a\"]\n\n" +
		"# why b\n[[policy.paths.rules]]\nmatch = [\"b\"]\n\n" +
		"# why c\n[[policy.paths.rules]]\nmatch = [\"c\"]\n"
	for _, tc := range []struct {
		name  string
		text  string
		index int
		want  string
	}{
		{"first keeps the comment of the next", three, 0,
			"[commit]\nlanguage = \"de\"\n\n# why b\n[[policy.paths.rules]]\nmatch = [\"b\"]\n\n# why c\n[[policy.paths.rules]]\nmatch = [\"c\"]\n"},
		{"middle takes its comment and one gap", three, 1,
			"[commit]\nlanguage = \"de\"\n\n[[policy.paths.rules]]\nmatch = [\"a\"]\n\n# why c\n[[policy.paths.rules]]\nmatch = [\"c\"]\n"},
		{"last leaves no trailing blank", three, 2,
			"[commit]\nlanguage = \"de\"\n\n[[policy.paths.rules]]\nmatch = [\"a\"]\n\n# why b\n[[policy.paths.rules]]\nmatch = [\"b\"]\n"},
		{"only block leaves no rest", "[[policy.paths.rules]]\nmatch = [\"a\"]\n", 0, ""},
		{"the next header keeps one gap", "[commit]\nx = 1\n\n[[policy.paths.rules]]\nmatch = [\"a\"]\n[other]\ny = 2\n", 0,
			"[commit]\nx = 1\n\n[other]\ny = 2\n"},
		{"a comment glued to the next block stays", "[[policy.paths.rules]]\nmatch = [\"a\"]\n# next\n[[policy.paths.rules]]\nmatch = [\"b\"]\n", 0,
			"# next\n[[policy.paths.rules]]\nmatch = [\"b\"]\n"},
		{"a sub-table goes with its block", "[[policy.paths.rules]]\nmatch = [\"a\"]\n[policy.paths.rules.extra]\nk = 1\n\n[[policy.paths.rules]]\nmatch = [\"b\"]\n", 0,
			"[[policy.paths.rules]]\nmatch = [\"b\"]\n"},
		{"other sections stay as they were", "[[policy.commands.rules]]\nregex = 'x'\n\n[[policy.paths.rules]]\nmatch = [\"a\"]\n\n[[policy.paths.rules]]\nmatch = [\"b\"]\n\n[[policy.commands.rules]]\nregex = 'y'\n", 1,
			"[[policy.commands.rules]]\nregex = 'x'\n\n[[policy.paths.rules]]\nmatch = [\"a\"]\n\n[[policy.commands.rules]]\nregex = 'y'\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RemoveItem(tc.text, ruleSection, tc.index)
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v\nwant %q", got, err, tc.want)
			}
			if itemsOf(t, got) != itemsOf(t, tc.text)-1 {
				t.Errorf("item count did not drop by one:\n%s", got)
			}
		})
	}
}

func TestRemoveItemInline(t *testing.T) {
	one := "[policy.paths]\nrules = [{ match = [\"a\"] }, { match = [\"b\"] }, { match = [\"c\"] }]\n"
	trailing := "[policy.paths]\nrules = [{ match = [\"a\"] }, { match = [\"b\"] }, { match = [\"c\"] },]\n"
	multi := "[policy.paths]\nrules = [\n  { match = [\"a\"] },\n  { match = [\"b\"] },\n  { match = [\"c\"] }\n]\n"
	multiTrailing := "[policy.paths]\nrules = [\n  { match = [\"a\"] },\n  { match = [\"b\"] },\n  { match = [\"c\"] },\n]\n"
	for _, tc := range []struct {
		name  string
		text  string
		index int
		want  string
	}{
		{"line first", one, 0, "[policy.paths]\nrules = [{ match = [\"b\"] }, { match = [\"c\"] }]\n"},
		{"line middle", one, 1, "[policy.paths]\nrules = [{ match = [\"a\"] }, { match = [\"c\"] }]\n"},
		{"line last", one, 2, "[policy.paths]\nrules = [{ match = [\"a\"] }, { match = [\"b\"] }]\n"},
		{"line trailing comma first", trailing, 0, "[policy.paths]\nrules = [{ match = [\"b\"] }, { match = [\"c\"] },]\n"},
		{"line trailing comma last", trailing, 2, "[policy.paths]\nrules = [{ match = [\"a\"] }, { match = [\"b\"] },]\n"},
		{"lines first", multi, 0, "[policy.paths]\nrules = [\n  { match = [\"b\"] },\n  { match = [\"c\"] }\n]\n"},
		{"lines middle", multi, 1, "[policy.paths]\nrules = [\n  { match = [\"a\"] },\n  { match = [\"c\"] }\n]\n"},
		{"lines last", multi, 2, "[policy.paths]\nrules = [\n  { match = [\"a\"] },\n  { match = [\"b\"] }\n]\n"},
		{"lines trailing comma first", multiTrailing, 0, "[policy.paths]\nrules = [\n  { match = [\"b\"] },\n  { match = [\"c\"] },\n]\n"},
		{"lines trailing comma last", multiTrailing, 2, "[policy.paths]\nrules = [\n  { match = [\"a\"] },\n  { match = [\"b\"] },\n]\n"},
		{"only element of a line", "[policy.paths]\nrules = [{ match = [\"a\"] }] # keep\n", 0, "[policy.paths]\nrules = [] # keep\n"},
		{"only element of lines", "[policy.paths]\nrules = [\n  { match = [\"a\"] },\n]\n", 0, "[policy.paths]\nrules = []\n"},
		{"own comments go, the neighbour's stay",
			"[policy.paths]\nrules = [\n  # a\n  { match = [\"a\"] }, # about a\n  # b\n  { match = [\"b\"] }, # about b\n  { match = [\"c\"] } # about c\n]\n", 1,
			"[policy.paths]\nrules = [\n  # a\n  { match = [\"a\"] }, # about a\n  { match = [\"c\"] } # about c\n]\n"},
		{"last without comma keeps the comment of the one before",
			"[policy.paths]\nrules = [\n  { match = [\"a\"] }, # about a\n  # b\n  { match = [\"b\"] }\n]\n", 1,
			"[policy.paths]\nrules = [\n  { match = [\"a\"] } # about a\n]\n"},
		{"an item that shares its line",
			"[policy.paths]\nrules = [{ match = [\"a\"] },\n  { match = [\"b\"] }]\n", 0,
			"[policy.paths]\nrules = [\n  { match = [\"b\"] }]\n"},
		{"strings with brackets, commas and hashes",
			"[policy.paths]\nrules = [{ reason = \"a, [b] # \\\" c\" }, { reason = 'd, ]' }]\n", 0,
			"[policy.paths]\nrules = [{ reason = 'd, ]' }]\n"},
		{"other keys of the table stay",
			"[policy.paths]\nmode = \"x\"\nrules = [{ match = [\"a\"] }, { match = [\"b\"] }]\nmore = 1\n", 0,
			"[policy.paths]\nmode = \"x\"\nrules = [{ match = [\"b\"] }]\nmore = 1\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RemoveItem(tc.text, ruleSection, tc.index)
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v\nwant %q", got, err, tc.want)
			}
			if itemsOf(t, got) != itemsOf(t, tc.text)-1 {
				t.Errorf("item count did not drop by one:\n%s", got)
			}
		})
	}
}

func TestRemoveItemKeepsBOMAndLineEndings(t *testing.T) {
	text := "\uFEFF[[policy.paths.rules]]\r\nmatch = [\"a\"]\r\n\r\n[[policy.paths.rules]]\r\nmatch = [\"b\"]\r\n"
	got, err := RemoveItem(text, ruleSection, 0)
	if want := "\uFEFF[[policy.paths.rules]]\r\nmatch = [\"b\"]\r\n"; err != nil || got != want {
		t.Fatalf("blocks: %q, %v", got, err)
	}
	text = "\uFEFF[policy.paths]\r\nrules = [\r\n  { match = [\"a\"] },\r\n  { match = [\"b\"] }\r\n]\r\n"
	got, err = RemoveItem(text, ruleSection, 1)
	if want := "\uFEFF[policy.paths]\r\nrules = [\r\n  { match = [\"a\"] }\r\n]\r\n"; err != nil || got != want {
		t.Fatalf("inline: %q, %v", got, err)
	}
	// A file with both endings keeps each \r on its own line.
	text = "[policy.paths]\nrules = [\n  { match = [\"a\"] },\r\n  { match = [\"b\"] }\n]\n"
	got, err = RemoveItem(text, ruleSection, 0)
	if want := "[policy.paths]\nrules = [\n  { match = [\"b\"] }\n]\n"; err != nil || got != want {
		t.Fatalf("mixed: %q, %v", got, err)
	}
}

func TestRemoveItemRefusesWhatIsNotThere(t *testing.T) {
	blocks := "[[policy.paths.rules]]\nmatch = [\"a\"]\n"
	inline := "[policy.paths]\nrules = [{ match = [\"a\"] }]\n"
	for _, text := range []string{blocks, inline, "[commit]\nx = 1\n"} {
		for _, index := range []int{1, 3, -1} {
			_, err := RemoveItem(text, ruleSection, index)
			if err == nil || err.Error() != "no item "+strconv.Itoa(index)+" in policy.paths.rules" {
				t.Errorf("%q index %d: %v", text, index, err)
			}
		}
	}
	if _, err := RemoveItem("[policy.paths]\nrules = []\n", ruleSection, 0); err == nil {
		t.Error("an empty list gave up an item")
	}
}

// A value that only looks like a list, by a bracket in a string or a comment,
// is not edited.
func TestRemoveItemLeavesAScalarAlone(t *testing.T) {
	for _, text := range []string{
		"[policy.paths]\nrules = \"a[b, c]\"\n",
		"[policy.paths]\nrules = 5 # [b, c]\n",
	} {
		got, err := RemoveItem(text, ruleSection, 0)
		if err == nil || got != "" || err.Error() != "no item 0 in policy.paths.rules" {
			t.Errorf("%q: %q, %v", text, got, err)
		}
	}
}

// Blocks for one section and an inline list for another: only the named
// section changes, in either direction.
func TestRemoveItemInAMixedFile(t *testing.T) {
	text := "[policy.commands]\nrules = [{ regex = 'x' }, { regex = 'y' }]\n\n" +
		"[[policy.paths.rules]]\nmatch = [\"a\"]\n\n[[policy.paths.rules]]\nmatch = [\"b\"]\n"
	got, err := RemoveItem(text, ruleSection, 0)
	want := "[policy.commands]\nrules = [{ regex = 'x' }, { regex = 'y' }]\n\n[[policy.paths.rules]]\nmatch = [\"b\"]\n"
	if err != nil || got != want {
		t.Fatalf("blocks: %q, %v", got, err)
	}
	got, err = RemoveItem(text, "policy.commands.rules", 0)
	want = "[policy.commands]\nrules = [{ regex = 'y' }]\n\n[[policy.paths.rules]]\nmatch = [\"a\"]\n\n[[policy.paths.rules]]\nmatch = [\"b\"]\n"
	if err != nil || got != want {
		t.Fatalf("inline: %q, %v", got, err)
	}
}

func TestRemoveItemRefusesWhatItCannotPlace(t *testing.T) {
	// A multi-line string hides lines that look like headers.
	blocks := "[[policy.paths.rules]]\nreason = \"\"\"\nx\n\"\"\"\n"
	if _, err := RemoveItem(blocks, ruleSection, 0); !errors.Is(err, ErrAmbiguous) {
		t.Errorf("blocks: %v", err)
	}
	// An array spelled as a dotted key has no table of its own.
	if _, err := RemoveItem("policy.paths.rules = [{ match = [\"a\"] }]\n", ruleSection, 0); !errors.Is(err, ErrAmbiguous) {
		t.Errorf("dotted key: %v", err)
	}
	// The list sits inside an inline table; no line of the file holds its key.
	text := "[policy]\npaths = { rules = [{ match = [\"a\"] }] }\n"
	if _, err := RemoveItem(text, ruleSection, 0); !errors.Is(err, ErrAmbiguous) {
		t.Errorf("inline table: %v", err)
	}
	if _, err := RemoveItem("not toml [", ruleSection, 0); err == nil {
		t.Error("text that is no TOML was edited")
	}
	// Top-level list: no dot in the section.
	got, err := RemoveItem("rules = [{ a = 1 }, { a = 2 }]\n", "rules", 0)
	if err != nil || got != "rules = [{ a = 2 }]\n" {
		t.Errorf("top-level: %q, %v", got, err)
	}
}

func TestArrayOfStopsAtAnUnclosedArray(t *testing.T) {
	if a := arrayOf("k = [ { a = 1 }"); a.close != -1 || len(a.items) != 0 {
		t.Errorf("%+v", a)
	}
}
