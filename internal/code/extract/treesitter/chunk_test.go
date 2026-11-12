package treesitter_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract/treesitter"
)

func TestChunksStartWhereTheLanguageSays(t *testing.T) {
	src := []byte("a\nb\n c\nd\n\ne")
	letter := func(pos int) bool { return src[pos] != ' ' }
	// The line holding end never counts; every later one the predicate
	// takes does, an empty one included.
	if got, want := treesitter.Chunks(src, 0, 10, letter), []int{2, 7, 9, 10}; !reflect.DeepEqual(got, want) {
		t.Errorf("Chunks = %v, want %v", got, want)
	}
	if got := treesitter.Chunks([]byte("a\nb\n"), 2, 10, letter); got != nil {
		t.Errorf("Chunks after the last line = %v, want none", got)
	}
	many := []byte("x\n" + strings.Repeat("f;\n", 7))
	if got, want := treesitter.Chunks(many, 0, 3, func(int) bool { return true }), []int{2, 11, 20}; !reflect.DeepEqual(got, want) {
		t.Errorf("thinned Chunks = %v, want every third start %v", got, want)
	}
}

func TestReparseChunkSeesItsChunkAlone(t *testing.T) {
	src := []byte("def a():\n    pass\ndef b():\n    pass\ndef c():\n    pass\n")
	d := parse(t, "m.py", string(src))
	starts := []int{18, 36}
	for i, want := range []string{"b", "c"} {
		if err := d.ReparseChunk(src, starts, i); err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, n := range d.Root.Children() {
			names = append(names, d.Text(d.Field(n, "name")))
		}
		if !reflect.DeepEqual(names, []string{want}) {
			t.Errorf("chunk %d holds %v, want %s alone", i, names, want)
		}
	}
}

func TestBlankKeepsLineBreaks(t *testing.T) {
	b := []byte("ab\r\ncd\n")
	treesitter.Blank(b)
	if string(b) != "  \r\n  \n" {
		t.Errorf("Blank = %q", b)
	}
}

func TestLexicalEnds(t *testing.T) {
	src := []byte(`x "a\"b" 'c` + "\n" + `/* d */ e`)
	if got := treesitter.LineEnd(src, 0); got != 11 {
		t.Errorf("LineEnd = %d, want 11", got)
	}
	if got := treesitter.LineEnd(src, 12); got != len(src) {
		t.Errorf("LineEnd of the last line = %d, want %d", got, len(src))
	}
	if got := treesitter.Closing(src, 14, "*/"); got != 19 {
		t.Errorf("Closing = %d, want 19", got)
	}
	if got := treesitter.Closing(src, 0, "@@"); got != len(src) {
		t.Errorf("Closing without end = %d, want %d", got, len(src))
	}
	if got := treesitter.QuotedEnd(src, 2); got != 8 {
		t.Errorf("QuotedEnd over an escape = %d, want 8", got)
	}
	if got := treesitter.QuotedEnd(src, 9); got != 11 {
		t.Errorf("QuotedEnd of an open quote = %d, want its line's end 11", got)
	}
	if got := treesitter.QuotedEnd([]byte(`"ab`), 0); got != 3 {
		t.Errorf("QuotedEnd at the end of src = %d, want 3", got)
	}
}
