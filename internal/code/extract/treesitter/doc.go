// Package treesitter is the part of every tree-sitter extractor that does not
// depend on the language: parse one file, walk its tree, turn a definition
// into a node, and count what the parser had to skip.
//
// It knows no grammar and no node type. Which node types are definitions,
// calls or imports, how a method finds its owner, what an import binds -- all
// of that stays in the language's own package, as a table and the few rules
// the table cannot express. The core would otherwise grow a switch per
// language, and a change for one language could move the nodes of another.
// What is shared is what must be identical for all of them: spans, signatures
// and body hashes cut from byte offsets the way the Go extractor cuts them,
// and ids minted through extract.MintID, so two definitions of one name in a
// file never share an id.
//
// One parser per file. A gotreesitter Parser must not be used by two
// goroutines at once, and one kept in a package variable would be state that
// every caller shares; building one costs little against the parse itself.
//
// This package and the language packages are the only ones that import
// gotreesitter. The hook path extracts Go files alone and never reaches it.
package treesitter

import (
	"fmt"
	"strings"

	gts "github.com/odvcencio/gotreesitter"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// Parser is the tree-sitter runtime every tree was built with, the version
// exactly as go.mod pins it. A tree-sitter language appends it to its own
// version.
//
// A language's version changes only when loomux's extractor does, but a new
// runtime can build different trees from the same source. Without Parser in
// the version, the extract cache would keep handing out entries the old
// runtime built, and the graph's meta would look fresh. TestParserMatchesGoMod
// fails when the pin moves and this constant does not.
const Parser = "gotreesitter/v0.55.1"

// Doc is one parsed file. Close releases its tree; no node of it may be used
// after that.
type Doc struct {
	Rel  string
	Src  []byte
	Lang *gts.Language
	Root *gts.Node

	tree    *gts.Tree
	minted  map[string]bool // ids handed out so far, for extract.MintID
	covered map[int]bool    // 1-based lines a symbol span covers
}

// Sym describes one definition to Symbol.
//
// The extent and the signature start at different places on purpose: a
// decorator belongs to the definition's span and body hash -- editing it
// changes what the definition does -- but not to its signature, which starts
// at the keyword.
//
// The signature ends at the end of the token that closes the header -- in
// Python its ':' -- and not where the body starts. A comment between the two
// (`def f(a):  # noqa`, or one on its own line before the first statement)
// is a child of the definition ahead of its block, so the body field starts
// after it. Which token closes the header is the language package's to say;
// Symbol trims a trailing ':' from what it is given.
type Sym struct {
	Outer     *gts.Node // extent: span, body hash and body text, decorators included
	HeaderEnd uint32    // where the signature ends: the end of the header's closing token
	Head      uint32    // where the signature starts: the def or class keyword
	Name      string
	Qualified string // the id's suffix after '#': "f", "C", "C.m"
	Kind      model.Kind
	Owner     string
	Exported  bool
}

// Parse parses one file. rel is its repo-relative, slash-separated path.
func Parse(lang *gts.Language, rel string, src []byte) (*Doc, error) {
	return parse(gts.NewParser(lang), lang, rel, src)
}

// parse is Parse on a given parser. The build hands every file a fresh one
// with no limit set; a test sets one to make a parse stop early.
func parse(p *gts.Parser, lang *gts.Language, rel string, src []byte) (*Doc, error) {
	tree, err := p.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", rel, err)
	}
	return &Doc{
		Rel: rel, Src: src, Lang: lang, Root: tree.RootNode(),
		tree: tree, minted: map[string]bool{}, covered: map[int]bool{},
	}, nil
}

// Close releases the tree.
func (d *Doc) Close() { d.tree.Release() }

// Type is a node's type in the document's grammar: "function_definition".
func (d *Doc) Type(n *gts.Node) string { return n.Type(d.Lang) }

// Field is a node's child in a named field, or nil.
func (d *Doc) Field(n *gts.Node, name string) *gts.Node { return n.ChildByFieldName(name, d.Lang) }

// Text is a node's source text.
func (d *Doc) Text(n *gts.Node) string { return n.Text(d.Src) }

// Symbol turns one definition into its node and marks its lines as covered,
// so FileNode leaves them out of the file's residual.
func (d *Doc) Symbol(s Sym) model.Node {
	from, to := lines(s.Outer)
	for i := from; i <= to; i++ {
		d.covered[i] = true
	}
	body := d.slice(s.Outer.StartByte(), s.Outer.EndByte())
	// The header of `def f(a) -> int:` ends with its colon; the colon is
	// syntax, not signature. Collapse also drops every carriage return.
	sig := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(extract.Collapse(d.slice(s.Head, s.HeaderEnd))), ":"))
	return model.Node{
		ID:        model.NodeID(extract.MintID(d.Rel+"#"+s.Qualified, d.minted)),
		Name:      s.Name,
		Kind:      s.Kind,
		Owner:     s.Owner,
		Path:      d.Rel,
		Span:      model.Span(fmt.Sprintf("L%d-L%d", from, to)),
		Signature: sig,
		Exported:  s.Exported,
		BodyHash:  extract.Hash(body),
		BodyText:  extract.Collapse(body),
	}
}

// lines is the 1-based first and last line of a node.
//
// A node that ends with a newline ends at column 0 of the next line, which
// holds nothing of it; the last line is then the one before. The Go extractor
// has the same answer for free: a declaration's End is the character after
// its closing brace, on the brace's own line.
func lines(n *gts.Node) (int, int) {
	start, end := n.StartPoint(), n.EndPoint()
	from, to := int(start.Row)+1, int(end.Row)+1
	if end.Column == 0 && end.Row > start.Row {
		to--
	}
	return from, to
}

// slice is the source text between two byte offsets, or "" for a range the
// source does not hold.
func (d *Doc) slice(from, to uint32) string {
	if from > to || int(to) > len(d.Src) {
		return ""
	}
	return string(d.Src[from:to])
}

// FileNode is the node that stands for the whole file, the same one the Go
// extractor builds (extract.FileNode). Call it after every Symbol: its body
// text is the residual, the lines no symbol covers.
func (d *Doc) FileNode() model.Node { return extract.FileNode(d.Rel, string(d.Src), d.covered) }

// ParseErrors counts the ERROR and MISSING nodes in the tree: what the parser
// skipped over or had to invent, and extracted around. A parse that stopped
// early -- a limit, a timeout, a cancellation -- counts one more: its tree
// holds what the parser got to, and the text after that need not show up as
// an ERROR. The count is a lower bound all the same: a recovered tree can
// drop text without a node to show for it.
func (d *Doc) ParseErrors() int {
	count := 0
	if d.tree.ParseStoppedEarly() {
		count++
	}
	Walk(d.Root, func(n *gts.Node) bool {
		if n.IsError() || n.IsMissing() {
			count++
		}
		return true
	})
	return count
}

// InError reports whether n or one of its ancestors is an ERROR node. A
// definition there was recovered from text the grammar rejected, and its
// extent is a guess; a MISSING node alone does not count.
func InError(n *gts.Node) bool {
	for p := n; p != nil; p = p.Parent() {
		if p.IsError() {
			return true
		}
	}
	return false
}

// HoldsError reports whether n is an ERROR node or has one below it; unlike
// InError it looks down, not up.
func HoldsError(n *gts.Node) bool {
	found := false
	Walk(n, func(c *gts.Node) bool {
		found = found || c.IsError()
		return !found
	})
	return found
}

// Walk visits n and everything below it in pre-order. When visit returns
// false, the node's children are skipped. A nil n -- the root some grammars
// return for an empty input -- visits nothing.
func Walk(n *gts.Node, visit func(*gts.Node) bool) {
	if n == nil || !visit(n) {
		return
	}
	for i := range n.ChildCount() {
		Walk(n.Child(i), visit)
	}
}
