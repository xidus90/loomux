package html

import (
	"errors"
	"strings"
	"testing"

	htmlgrammar "github.com/odvcencio/gotreesitter/grammars/html"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
)

func TestFileReportsAParseFailure(t *testing.T) {
	// The grammar builds a tree from any input; only a missing grammar makes
	// the parse itself fail.
	_, err := file(nil, "p.html", "<p></p>\n", readPart)
	if err == nil || !strings.Contains(err.Error(), "parse p.html") {
		t.Errorf("file(nil grammar) = %v, want an error naming the file", err)
	}
}

func TestFileReportsAFailingPart(t *testing.T) {
	// The extractors of the parts read with a fixed grammar and never fail;
	// a reader that does fails the page.
	failing := func(string, string, string) (extract.Result, error) { return extract.Result{}, errors.New("broken") }
	_, err := file(htmlgrammar.Language(), "p.html", "<script>x</script>\n", failing)
	if err == nil || err.Error() != "broken" {
		t.Errorf("file = %v, want the part's error", err)
	}
}

func TestTagNameIsEmptyWithoutATagName(t *testing.T) {
	// The parser gives every tag a tag_name; a node without one, the
	// document here, must not stop the page.
	doc, err := treesitter.Parse(htmlgrammar.Language(), "p.html", []byte("<P></P>\n"))
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	x := &extractor{doc: doc}
	tag := doc.Root.Child(0).Child(0)
	if got := x.tagName(tag); got != "p" {
		t.Errorf("tagName(<P>) = %q, want p", got)
	}
	if got := x.tagName(doc.Root); got != "" {
		t.Errorf("tagName(document) = %q, want none", got)
	}
}

func TestMask(t *testing.T) {
	src := "<s>ab\r\ncd</s>\n<s>e</s>"
	for _, c := range []struct {
		sep  byte
		want string
	}{
		{';', "   ab\r\ncd;   \n   e;   "},
		{0, "   ab\r\ncd    \n   e    "},
	} {
		if got := mask(src, [][2]int{{3, 9}, {17, 18}}, c.sep); got != c.want {
			t.Errorf("mask(sep %q) = %q, want %q", c.sep, got, c.want)
		}
	}
	// Line breaks outside the spans stay; a span that runs to the end has
	// no byte after it for sep.
	if got := mask("a\r\nb<s>cd", [][2]int{{7, 9}}, ';'); got != " \r\n    cd" {
		t.Errorf("mask(to the end) = %q", got)
	}
}
