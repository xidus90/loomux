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

func TestSplitRefusesAnOpenQuote(t *testing.T) {
	for _, in := range []string{`a "b`, `a 'b`, `a \`} {
		if _, err := Split(in); err == nil {
			t.Errorf("Split(%q): want error", in)
		}
	}
}
