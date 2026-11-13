package ecmascript

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestFileReportsAParseFailure(t *testing.T) {
	// The grammars build a tree from any input; only a missing grammar
	// makes the parse itself fail.
	_, err := file(nil, "javascript", "m.js", "function f() {}\n", false)
	if err == nil || !strings.Contains(err.Error(), "parse m.js") {
		t.Errorf("file(nil grammar) = %v, want an error naming the file", err)
	}
}

func TestOpenerTakesDeclarationLinesOutsideText(t *testing.T) {
	src := []byte("a\n  b\n}\n$x\n@d\n_y\nZ\n9\n\n// c\n" +
		"/* k\nblock */\n" +
		"s = '`' + \"`\" // `\n" +
		"t = `\\`\ntpl ${ { o: `\ninner\n` } } ${a}\nmore`\n" +
		"f({ g() {} })\nafter\n" +
		"open = `\nrest")
	opens := opener(src)
	var got []string
	for pos := 0; pos < len(src); pos++ {
		if (pos == 0 || src[pos-1] == '\n') && pos < len(src) && opens(pos) {
			line := src[pos:]
			if i := bytes.IndexByte(line, '\n'); i >= 0 {
				line = line[:i]
			}
			got = append(got, string(line))
		}
	}
	// Indented lines, braces, digits, blank lines and comments open
	// nothing; nor do the lines of a block comment, a template (nested
	// ones and escapes included) or a template left open.
	want := []string{"a", "$x", "@d", "_y", "Z", "s = '`' + \"`\" // `", "t = `\\`", "f({ g() {} })", "after", "open = `"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("openers =\n %q\nwant\n %q", got, want)
	}
}
