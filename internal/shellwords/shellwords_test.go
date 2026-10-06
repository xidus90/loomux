package shellwords

import (
	"reflect"
	"testing"
)

func TestSplit(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{`go vet ./...`, []string{"go", "vet", "./..."}},
		{`npx stylelint "**/*.{css,scss}"`, []string{"npx", "stylelint", "**/*.{css,scss}"}},
		{`a 'b c' "d\"e"`, []string{"a", "b c", `d"e`}},
		{`x ""`, []string{"x", ""}},
		{`  spaced   out  `, []string{"spaced", "out"}},
		{`a\ b`, []string{"a b"}},
	}
	for _, c := range cases {
		got, err := Split(c.in)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("Split(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
}

// Inside double quotes a backslash escapes only $ ` " \ and a newline; before
// any other character it is a plain character, as in a Windows path.
func TestSplitKeepsABackslashInsideDoubleQuotesBeforeAnOrdinaryCharacter(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{`gofmt -l "src\main.go"`, []string{"gofmt", "-l", `src\main.go`}},
		{`echo "a\nb"`, []string{"echo", `a\nb`}},
		{`rg "\bfoo"`, []string{"rg", `\bfoo`}},
		{`x "a\"b" "c\\d" "e\$f" "g\` + "`" + `h"`, []string{"x", `a"b`, `c\d`, `e$f`, "g`h"}},
		{"x \"a\\\nb\"", []string{"x", "a\nb"}},
		// Outside double quotes every backslash still escapes.
		{`x a\nb 'c\d'`, []string{"x", "anb", `c\d`}},
	}
	for _, c := range cases {
		got, err := Split(c.in)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("Split(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
}

func TestSplitRefusesAnOpenQuote(t *testing.T) {
	for _, in := range []string{`a "b`, `a 'b`, `a \`, `a "b\`} {
		if _, err := Split(in); err == nil {
			t.Errorf("Split(%q): want error", in)
		}
	}
}
