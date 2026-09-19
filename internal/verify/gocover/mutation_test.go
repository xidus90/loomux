package gocover

import (
	"strings"
	"testing"
)

// These tests close gaps the stage 2a mutation round found: each one pins a
// difference the suite could not see before.

func TestParseSkipsABlankLine(t *testing.T) {
	lines, err := Parse(strings.NewReader("a.go:1:\tF\t100.0%\n \t\nb.go:2:\tG\t50.0%\n"))
	if err != nil || len(lines) != 2 {
		t.Fatalf("%+v %v", lines, err)
	}
}

func TestParseRefusesALineWithAFieldTooMany(t *testing.T) {
	if _, err := Parse(strings.NewReader("a.go:1:\tF\t100.0%\textra\n")); err == nil {
		t.Fatal("no error")
	}
}

func TestParseRefusesALocationThatIsOnlyANumber(t *testing.T) {
	if _, err := Parse(strings.NewReader("12\tF\t100.0%\n")); err == nil || !strings.Contains(err.Error(), "unexpected location") {
		t.Fatalf("%v", err)
	}
}

// exemptIn asks exempt about a function at line in a file holding src.
func exemptIn(src string, line int) bool {
	return exempt(func(string) ([]byte, error) { return []byte(src), nil }, "a.go", line)
}

func TestAnExemptionOnTheFirstLineCoversTheSecond(t *testing.T) {
	if !exemptIn("//coverage:exempt reason\nfunc f() {}\n", 2) {
		t.Fatal("not exempt")
	}
}

func TestALineOutsideTheFileIsNeverExempt(t *testing.T) {
	src := "//coverage:exempt reason\nfunc f() {}"
	for _, line := range []int{0, 1, 4} {
		if exemptIn(src, line) {
			t.Fatalf("line %d exempt", line)
		}
	}
}
