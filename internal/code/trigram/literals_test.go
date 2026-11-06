package trigram

import (
	"reflect"
	"strings"
	"testing"
)

// TestLiterals pins the translation from a grep pattern to the literal
// conditions the index can use. A pattern that yields no condition is not an
// error: it only means the search falls back to the scan.
func TestLiterals(t *testing.T) {
	cases := []struct {
		name    string
		pattern string
		fixed   bool
		ignore  bool
		want    [][]string
		ok      bool
	}{
		// One literal longer than a trigram is a usable condition.
		{"literal", "abcdef", false, false, [][]string{{"abcdef"}}, true},
		// A root alternation of short branches cannot constrain anything.
		{"short alternation", "ab|cd", false, false, nil, false},
		// A root alternation of usable branches is a disjunction.
		{"alternation", "abc|def", false, false, [][]string{{"abc"}, {"def"}}, true},
		// A class alone has no literal.
		{"class only", "[a-z]+", false, false, nil, false},
		// A wildcard before a literal keeps only the literal, and the literal is
		// still necessary for a match.
		{"wildcard before literal", "a.*bcd", false, false, [][]string{{"bcd"}}, true},
		{"literal after newline class", `abc[\s\S]+def`, false, false, [][]string{{"abc", "def"}}, true},
		// --fixed: the pattern is one literal, dots and brackets included.
		{"fixed", "a.b", true, false, [][]string{{"a.b"}}, true},
		{"fixed short", "a.b"[0:2], true, false, nil, false},
		// -i folds ASCII; the index is folded, so the condition is lowercase.
		{"case folded", "ABC", false, true, [][]string{{"abc"}}, true},
		{"fixed case folded", "AbC.d", true, true, [][]string{{"abc.d"}}, true},
		// Folding a non-ASCII letter is outside the index's contract: no error,
		// no condition, the caller scans.
		{"non-ascii fold", "Ä", false, true, nil, false},
		{"non-ascii fold in literal", "xÄy", false, true, nil, false},
		// A case-sensitive non-ASCII literal is not a trigram the folded index
		// carries; the conservative answer is the scan, not a wrong candidate set.
		{"non-ascii literal", "Äbc", false, false, nil, false},
		// A branch without a literal makes the whole disjunction unconstrained.
		{"alternation with short branch", "abc|de", false, false, nil, false},
		// The trailing literal of a concatenation is enough.
		// The parser factors the common prefix of the alternation, and that
		// prefix is the only literal a match needs.
		{"trailing literal", `[0-9]+\.(cache|cacheDir)`, false, false, [][]string{{"cache"}}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sets, ok, err := Literals(c.pattern, c.fixed, c.ignore)
			if err != nil {
				t.Fatalf("Literals(%q) error: %v", c.pattern, err)
			}
			if ok != c.ok {
				t.Fatalf("Literals(%q) ok = %v, want %v (sets %v)", c.pattern, ok, c.ok, sets)
			}
			if c.ok && !reflect.DeepEqual(sets, c.want) {
				t.Fatalf("Literals(%q) = %v, want %v", c.pattern, sets, c.want)
			}
			if !c.ok && sets != nil {
				t.Fatalf("Literals(%q) returned sets %v with ok=false", c.pattern, sets)
			}
		})
	}
}

// TestLiteralsErrors pins the two pattern errors the grep path already
// refuses: syntax RE2 cannot compile, and forms it does not support at all.
func TestLiteralsErrors(t *testing.T) {
	if _, _, err := Literals("(", false, false); err == nil || !strings.Contains(err.Error(), "invalid regex") {
		t.Fatalf("Literals(%q) error = %v, want invalid regex", "(", err)
	}
	if _, _, err := Literals("(?=x)", false, false); err == nil || !strings.Contains(err.Error(), "lookaround not supported in RE2 regex") {
		t.Fatalf("Literals(%q) error = %v, want lookaround", "(?=x)", err)
	}
	if _, _, err := Literals(`a\1`, false, false); err == nil || !strings.Contains(err.Error(), "backreference not supported in RE2 regex") {
		t.Fatalf("Literals(%q) error = %v, want backreference", `a\1`, err)
	}
	// An escaped parenthesis is a literal parenthesis; the "?" after it is the
	// quantifier of the literal, so "?=" is not the lookaround operator.
	if _, _, err := Literals(`\(?=`, false, false); err != nil {
		t.Fatalf("Literals(%q) error = %v, want none", `\(?=`, err)
	}
}

// TestLiteralsEmpty pins the same refusal the scan gives an empty pattern.
func TestLiteralsEmpty(t *testing.T) {
	if _, _, err := Literals("", false, false); err == nil {
		t.Fatal("Literals(\"\") returned no error")
	}
}

// TestLiteralsFixedNonASCII pins the fixed path's refusal to fold a
// non-ASCII pattern: -i on it is a fold the index cannot do.
func TestLiteralsFixedNonASCII(t *testing.T) {
	if sets, ok, err := Literals("Äbc.d", true, true); err != nil || ok || sets != nil {
		t.Fatalf("Literals(fixed, -i, non-ASCII) = %v, %v, %v; want nil, false, nil", sets, ok, err)
	}
}

// TestLiteralsDeduplicatesAlternation pins that a repeated branch yields one
// conjunction.
func TestLiteralsDeduplicatesAlternation(t *testing.T) {
	sets, ok, err := Literals("abc|abc", false, false)
	if err != nil || !ok {
		t.Fatalf("Literals(abc|abc) = %v, %v; want ok", sets, err)
	}
	if want := [][]string{{"abc"}}; !reflect.DeepEqual(sets, want) {
		t.Fatalf("Literals(abc|abc) = %v, want %v", sets, want)
	}
}

// TestLiteralsBounds pins the two caps: a conjunction with too many literals
// and a disjunction with too many branches are not worth narrowing, so both
// answer "no condition".
func TestLiteralsBounds(t *testing.T) {
	// Nine literals in one conjunction exceed maxLiterals.
	long := "aaa.*bbb.*ccc.*ddd.*eee.*fff.*ggg.*hhh.*iii"
	if sets, ok, err := Literals(long, false, false); err != nil || ok || sets != nil {
		t.Fatalf("Literals(9 literals) = %v, %v, %v; want nil, false, nil", sets, ok, err)
	}
	// Seventeen branches exceed maxSets.
	branches := []string{"aax", "bbx", "ccx", "ddx", "eex", "ffx", "ggx", "hhx", "iix", "jjx", "kkx", "llx", "mmx", "nnx", "oox", "ppx", "qqx"}
	if sets, ok, err := Literals(strings.Join(branches, "|"), false, false); err != nil || ok || sets != nil {
		t.Fatalf("Literals(17 branches) = %v, %v, %v; want nil, false, nil", sets, ok, err)
	}
	// A cross product above maxSets is dropped too.
	if sets, ok, err := Literals("(aaa|bbb|ccc|ddd|eee)(fff|ggg|hhh|iii)", false, false); err != nil || ok || sets != nil {
		t.Fatalf("Literals(20 sets) = %v, %v, %v; want nil, false, nil", sets, ok, err)
	}
}
