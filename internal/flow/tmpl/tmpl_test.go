package tmpl_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/flow/tmpl"
)

func parse(t *testing.T, text string) tmpl.Template {
	t.Helper()
	template, err := tmpl.Parse(text)
	if err != nil {
		t.Fatalf("%q: %v", text, err)
	}
	return template
}

func TestRenderFillsEveryKindOfValue(t *testing.T) {
	template := parse(t, "Review {{topic}} in round {{round}} (strict: {{strict}}):\n{{notes}}")
	got, err := template.Render(map[string]any{
		"topic": "the spec", "round": 2, "strict": true, "notes": []string{"one", "two"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "Review the spec in round 2 (strict: true):\n- one\n- two"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAnEmptyListRendersAsNothing(t *testing.T) {
	got, err := parse(t, "[{{notes}}]").Render(map[string]any{"notes": []string{}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "[]" {
		t.Fatalf("got %q, want []", got)
	}
}

// One pass: a value that looks like a placeholder is written as it is. A second
// pass would let a model's answer pull other fields into the next prompt.
func TestRenderIsOnePass(t *testing.T) {
	got, err := parse(t, "{{answer}}").Render(map[string]any{"answer": "{{secret}}"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "{{secret}}" {
		t.Fatalf("got %q, want the value unread", got)
	}
}

func TestEscapedBracesAreLiteral(t *testing.T) {
	template := parse(t, `JSON: \{{"a": 1}} and {{name}}`)
	if names := template.Names(); !slices.Equal(names, []string{"name"}) {
		t.Fatalf("names = %v, want [name]", names)
	}
	got, err := template.Render(map[string]any{"name": "x"})
	if err != nil {
		t.Fatal(err)
	}
	if got != `JSON: {{"a": 1}} and x` {
		t.Fatalf("got %q", got)
	}
}

func TestNamesAreSortedAndUnique(t *testing.T) {
	names := parse(t, "{{b}} {{a_1}} {{b}}").Names()
	if !slices.Equal(names, []string{"a_1", "b"}) {
		t.Fatalf("names = %v, want [a_1 b]", names)
	}
}

func TestATextWithoutPlaceholdersStaysAsItIs(t *testing.T) {
	template := parse(t, "plain } { text")
	if len(template.Names()) != 0 {
		t.Fatalf("names = %v, want none", template.Names())
	}
	got, err := template.Render(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "plain } { text" {
		t.Fatalf("got %q", got)
	}
}

func TestParseRefusesWhatIsNoPlaceholder(t *testing.T) {
	cases := []struct{ text, want string }{
		{"Hello {{name", "unclosed {{ at byte 6"},
		{"{{ name }}", "{{ name }} is not a placeholder"},
		{"{{}}", "{{}} is not a placeholder"},
		{"{{1st}}", "{{1st}} is not a placeholder"},
		{`{"a": {{"b": 1}}}`, "is not a placeholder"},
	}
	for _, c := range cases {
		_, err := tmpl.Parse(c.text)
		if err == nil {
			t.Errorf("%q: want an error containing %q, got none", c.text, c.want)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: error %q does not contain %q", c.text, err, c.want)
		}
	}
}

func TestRenderNamesAMissingValue(t *testing.T) {
	_, err := parse(t, "{{topic}}").Render(map[string]any{})
	if err == nil || err.Error() != "no value for {{topic}}" {
		t.Fatalf("got %v", err)
	}
}

func TestRenderRefusesAValueNoTextCanShow(t *testing.T) {
	_, err := parse(t, "{{n}}").Render(map[string]any{"n": int64(3)})
	if err == nil || err.Error() != "{{n}} holds a int64, which a text cannot show" {
		t.Fatalf("got %v", err)
	}
}
