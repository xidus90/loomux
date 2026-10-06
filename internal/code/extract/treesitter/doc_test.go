package treesitter_test

import (
	"reflect"
	"strings"
	"testing"

	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars/python"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// Python is the test language only: the core knows no grammar, and the
// Python grammar is the one the module already pins.

func parse(t *testing.T, rel, src string) *treesitter.Doc {
	t.Helper()
	d, err := treesitter.Parse(python.Language(), rel, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(d.Close)
	return d
}

// first is the first node of a type in pre-order, or nil.
func first(d *treesitter.Doc, typ string) *gts.Node {
	var hit *gts.Node
	treesitter.Walk(d.Root, func(n *gts.Node) bool {
		if hit == nil && d.Type(n) == typ {
			hit = n
		}
		return hit == nil
	})
	return hit
}

// funcSym describes a definition the way a language package would: the
// extent is outer (the decorated definition, when there is one), the
// signature runs from the def or class keyword to the end of the header's
// colon.
func funcSym(d *treesitter.Doc, outer, fn *gts.Node, name string) treesitter.Sym {
	return treesitter.Sym{
		Outer: outer, Head: fn.StartByte(), HeaderEnd: colonEnd(d, fn),
		Name: name, Qualified: name, Kind: "function", Exported: true,
	}
}

// colonEnd is the end of a definition's first direct `:` child: the colon
// closing its header. A colon in an annotation or a default lies deeper, in
// the parameters or the return type.
func colonEnd(d *treesitter.Doc, def *gts.Node) uint32 {
	for _, c := range def.Children() {
		if d.Type(c) == ":" {
			return c.EndByte()
		}
	}
	return def.EndByte()
}

func TestSymbolSpanSignatureHash(t *testing.T) {
	const src = "@dec\ndef f(a, b) -> int:\n    return a\n"
	d := parse(t, "m.py", src)
	dec := first(d, "decorated_definition")
	fn := d.Field(dec, "definition")
	if fn == nil || d.Type(fn) != "function_definition" {
		t.Fatalf("no function_definition under the decorated definition in %q", src)
	}
	if got := d.Text(d.Field(fn, "name")); got != "f" {
		t.Fatalf("Text(name) = %q, want f", got)
	}

	// The node ends at the last character of `return a`, before the final
	// newline: the decorator is in the span and the hash, the newline is not.
	body := src[:len(src)-1]
	want := model.Node{
		ID: "m.py#f", Name: "f", Kind: "function", Path: "m.py",
		Span: "L1-L3", Signature: "def f(a, b) -> int", Exported: true,
		BodyHash: extract.Hash(body), BodyText: extract.Collapse(body),
	}
	if got := d.Symbol(funcSym(d, dec, fn, "f")); !reflect.DeepEqual(got, want) {
		t.Errorf("Symbol =\n %+v\nwant\n %+v", got, want)
	}
}

func TestSignatureDropsCR(t *testing.T) {
	const src = "@dec\r\ndef f(a, b) -> int:\r\n    return a\r\n"
	d := parse(t, "m.py", src)
	dec := first(d, "decorated_definition")
	n := d.Symbol(funcSym(d, dec, d.Field(dec, "definition"), "f"))
	if n.Signature != "def f(a, b) -> int" {
		t.Errorf("Signature = %q, want the LF form without a carriage return", n.Signature)
	}
	if n.Span != "L1-L3" {
		t.Errorf("Span = %q, want L1-L3 as with LF", n.Span)
	}
	if strings.Contains(n.BodyText, "\r") {
		t.Errorf("BodyText = %q, want no carriage return", n.BodyText)
	}
}

func TestSignatureLeavesACommentAfterTheColonOut(t *testing.T) {
	// A comment between the header's colon and the first statement is a child
	// of the definition before its block, so the body starts after it. The
	// signature ends at the colon, for a def and a class alike.
	for src, want := range map[string]string{
		"def g(a):\n    # leading comment\n    return a\n": "def g(a)",
		"def f(a):  # noqa: E501\n    return a\n":          "def f(a)",
		"class C:  # pragma: no cover\n    pass\n":         "class C",
	} {
		d := parse(t, "m.py", src)
		n := d.Root.Child(0)
		if got := d.Symbol(funcSym(d, n, n, "x")).Signature; got != want {
			t.Errorf("Signature of %q = %q, want %q", src, got, want)
		}
	}
}

func TestSymbolSpanStopsAtTheLineItsNewlineEnds(t *testing.T) {
	// The module node ends after the file's last newline, at column 0 of a
	// line that holds nothing of it; that line is not part of the span.
	d := parse(t, "m.py", "def f():\n    return 1\n")
	if got := d.Symbol(treesitter.Sym{Outer: d.Root, Qualified: "all"}).Span; got != "L1-L2" {
		t.Errorf("Span of a node ending in a newline = %q, want L1-L2", got)
	}

	// An empty node at column 0 ends on the line it starts on; stepping back
	// would put its end before its start.
	e := parse(t, "e.py", "")
	if got := e.Symbol(treesitter.Sym{Outer: e.Root, Qualified: "all"}).Span; got != "L1-L1" {
		t.Errorf("Span of an empty node = %q, want L1-L1", got)
	}
}

func TestSymbolCutsNoSignatureOutsideTheSource(t *testing.T) {
	d := parse(t, "m.py", "def f():\n    pass\n")
	fn := first(d, "function_definition")
	for _, s := range []treesitter.Sym{
		{Outer: fn, Head: 5, HeaderEnd: 2, Qualified: "reversed"},
		{Outer: fn, Head: 0, HeaderEnd: 400, Qualified: "past-the-end"},
	} {
		if got := d.Symbol(s).Signature; got != "" {
			t.Errorf("%s: Signature = %q, want empty", s.Qualified, got)
		}
	}
}

func TestSymbolMintsOrdinals(t *testing.T) {
	d := parse(t, "m.py", "def f():\n    pass\n\ndef f():\n    pass\n")
	var ids []model.NodeID
	for _, n := range d.Root.Children() {
		ids = append(ids, d.Symbol(funcSym(d, n, n, "f")).ID)
	}
	if want := []model.NodeID{"m.py#f", "m.py#f~2"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("ids = %v, want %v", ids, want)
	}

	m := d.Symbol(treesitter.Sym{Outer: d.Root.Child(0), Name: "m", Qualified: "C.m", Kind: "method", Owner: "C"})
	if m.ID != "m.py#C.m" || m.Name != "m" || m.Owner != "C" || m.Kind != "method" || m.Exported {
		t.Errorf("method = %+v, want id m.py#C.m, name m, owner C, kind method, unexported", m)
	}
}

func TestFileNodeKeepsResidual(t *testing.T) {
	const src = "import os\n\ndef f():\n    return inside\n\nNEEDLE = 1\n"
	d := parse(t, "pkg/m.py", src)
	fn := first(d, "function_definition")
	d.Symbol(funcSym(d, fn, fn, "f"))

	want := model.Node{
		ID: "pkg/m.py", Name: "m.py", Kind: model.KindFile, Path: "pkg/m.py",
		Span: "L1-L7", Exported: true, BodyHash: extract.Hash(src),
		// Lines 3 and 4 belong to f; the import and the constant are what is
		// left, so the file is findable by a word that lives in no function.
		BodyText: "import os NEEDLE = 1",
	}
	if got := d.FileNode(); !reflect.DeepEqual(got, want) {
		t.Errorf("FileNode =\n %+v\nwant\n %+v", got, want)
	}
}

func TestParseErrorsCounts(t *testing.T) {
	cases := []struct {
		name, src string
		errors    int
		errorNode bool
	}{
		{"clean", "def f():\n    pass\n", 0, false},
		// One ERROR node and nothing missing.
		{"error node", "class :\n", 1, true},
		// One MISSING `)` and no ERROR node: the definition parses around it.
		{"missing node", "def broken(:\n", 1, false},
	}
	for _, c := range cases {
		d := parse(t, "m.py", c.src)
		if got := d.Root.HasError(); got != (c.errors > 0) {
			t.Errorf("%s: HasError = %v, want %v", c.name, got, c.errors > 0)
		}
		if got := first(d, "ERROR") != nil; got != c.errorNode {
			t.Errorf("%s: an ERROR node = %v, want %v", c.name, got, c.errorNode)
		}
		if got := d.ParseErrors(); got != c.errors {
			t.Errorf("%s: ParseErrors = %d, want %d", c.name, got, c.errors)
		}
	}
}

// A parse that stops early returns what it got to, and the rest of the text
// need not show up as an ERROR at all: the stop itself counts as one.
func TestParseErrorsCountsAnEarlyStop(t *testing.T) {
	const src = "def f():\n    pass\n"
	lang := python.Language()
	p := gts.NewParser(lang)
	cancelled := uint32(1)
	p.SetCancellationFlag(&cancelled)
	d, err := treesitter.ParseWith(p, lang, "m.py", []byte(src))
	if err != nil {
		t.Fatalf("a cancelled parse still returns its tree, got %v", err)
	}
	t.Cleanup(d.Close)
	inTree := 0
	treesitter.Walk(d.Root, func(n *gts.Node) bool {
		if n.IsError() || n.IsMissing() {
			inTree++
		}
		return true
	})
	if got := d.ParseErrors(); got != inTree+1 {
		t.Errorf("ParseErrors = %d, want the %d error nodes of the tree and the stop", got, inTree)
	}
	// The same source parsed to the end has nothing to count.
	if got := parse(t, "m.py", src).ParseErrors(); got != 0 {
		t.Errorf("ParseErrors of the whole parse = %d, want 0", got)
	}
}

func TestInError(t *testing.T) {
	// The stray `)` turns the whole class into an ERROR node, and the method
	// still parses as a function_definition inside it.
	d := parse(t, "m.py", "class C:\n    def m(self):\n        pass\n    )\n")
	errNode := first(d, "ERROR")
	fn := first(d, "function_definition")
	if errNode == nil || fn == nil {
		t.Fatal("want an ERROR node with a function_definition inside it")
	}
	if !treesitter.InError(errNode) {
		t.Error("InError(the ERROR node) = false, want true")
	}
	if !treesitter.InError(fn) {
		t.Error("InError(a definition inside an ERROR node) = false, want true")
	}
	if treesitter.InError(d.Root) {
		t.Error("InError(root) = true, want false: nothing above it is an ERROR node")
	}

	clean := parse(t, "c.py", "def f():\n    pass\n")
	if treesitter.InError(first(clean, "function_definition")) {
		t.Error("InError(a clean definition) = true, want false")
	}
	// A MISSING node is no ERROR node: the definition around it stays.
	missing := parse(t, "b.py", "def broken(:\n")
	if treesitter.InError(first(missing, "function_definition")) {
		t.Error("InError(a definition with only a MISSING node) = true, want false")
	}
}

func TestHoldsError(t *testing.T) {
	d := parse(t, "m.py", "class C:\n    def m(self):\n        pass\n    )\n")
	errNode := first(d, "ERROR")
	if !treesitter.HoldsError(errNode) || !treesitter.HoldsError(d.Root) {
		t.Error("HoldsError(an ERROR node, or the root above one) = false, want true")
	}
	// The method sits inside the ERROR node but holds none: the test looks down.
	if treesitter.HoldsError(first(d, "function_definition")) {
		t.Error("HoldsError(a definition below an ERROR node) = true, want false")
	}
	clean := parse(t, "c.py", "def f():\n    pass\n")
	if treesitter.HoldsError(clean.Root) {
		t.Error("HoldsError(a clean tree) = true, want false")
	}
	// Clean nodes after the ERROR node, in walk order, do not clear the finding.
	after := parse(t, "a.py", "x = (\n\ndef g():\n    pass\n\ndef h():\n    pass\n")
	if first(after, "ERROR") == nil || !treesitter.HoldsError(after.Root) {
		t.Error("HoldsError(an ERROR node followed by clean definitions) = false, want true")
	}
	// A MISSING node is no ERROR node.
	if treesitter.HoldsError(parse(t, "b.py", "def broken(:\n").Root) {
		t.Error("HoldsError(a tree with only a MISSING node) = true, want false")
	}
}

func TestWalkSkipsChildren(t *testing.T) {
	d := parse(t, "m.py", "class C:\n    def m(self):\n        pass\n\ndef f():\n    pass\n")
	var order, names []string
	treesitter.Walk(d.Root, func(n *gts.Node) bool {
		order = append(order, d.Type(n))
		if d.Type(n) == "identifier" {
			names = append(names, d.Text(n))
		}
		return d.Type(n) != "class_definition"
	})
	if len(order) < 2 || order[0] != "module" || order[1] != "class_definition" {
		t.Errorf("order = %v, want module then class_definition first: pre-order", order)
	}
	if want := []string{"f"}; !reflect.DeepEqual(names, want) {
		t.Errorf("identifiers = %v, want %v: C and m lie under the skipped class", names, want)
	}

	// Some grammars return no root for an empty input.
	treesitter.Walk(nil, func(*gts.Node) bool {
		t.Error("Walk(nil) visited a node")
		return true
	})
}

func TestParseRefusesWithoutALanguage(t *testing.T) {
	_, err := treesitter.Parse(nil, "m.py", []byte("x = 1\n"))
	if err == nil || !strings.Contains(err.Error(), "parse m.py") {
		t.Errorf("Parse(nil language) = %v, want an error naming the file", err)
	}
}
