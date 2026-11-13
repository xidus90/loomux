// Package stylesheet extracts the nodes and raw edges of one CSS or SCSS
// file on the shared tree-sitter core.
//
// Two languages, one extractor: CSS (.css) and SCSS (.scss) differ in
// Result.Language, in their version and in the grammar, and in what only
// SCSS has -- mixins, functions, @use, @include, @extend and calls in a
// value. Plain CSS makes no calls edge: rgba() or url() name nothing in the
// repository.
//
// Nodes are the file and every rule, also inside @media, @supports, @layer
// or a mixin; a keyframe block is none. A rule is named by its selector
// list with whitespace folded to one space, and keeps the list as written
// as its signature; a nested SCSS rule is named "<parent> › <own>". SCSS
// adds @mixin (kind mixin), @function (kind function) and placeholders %p,
// which are rules; a mixin, function or placeholder whose name starts with
// - or _ is private to its module, everything else is exported. Every
// class of a rule's selectors is offered as css.class, every id as css.id,
// and in SCSS &suffix right after & (&__el, &-sm) offers suffix appended to
// every class a parent selector ends on.
//
// Every @import, @use and @forward is one extract.Import per path, the path
// as written; Kind is "" in CSS and "import", "use" or "forward" in SCSS,
// and the namespace of @use … as is its Alias. @include and a call in SCSS
// are calls edges with Member "mixin" or "function", ns.m split into
// Receiver and Name; @extend is an extends edge to ".k" or "%p". Each
// belongs to the innermost rule, mixin or function with a node, else to
// the file.
//
// The SCSS grammar rejects some forms it should take, each a local ERROR:
// @use "x" as y, @extend %p, a range media query (width >= 576px) -- that
// one in CSS too -- and @include ns.m, which it reads as @include ns and a
// selector .m. An @import of a list of paths and an @use … with (…) break
// the statement too. An ERROR is read as text for @use … as, an @import of
// a list and @extend, and nothing else; an @use … with in one is lost. The
// m of @include ns.m is read from the source, and the selector the grammar
// made of it is no rule. A keyword argument f($a: 1) in a value is an
// ERROR as well.
//
// A break can also take a whole file into one ERROR, and its rules with
// it. Three forms did so and are masked before the parse (masked): the %
// of @extend %p, which also closed the block around it early, a map
// literal ($k: v) as an argument, and #{$v} inside a name in a value,
// var(--a-#{$v}). The parser reads a copy of the same length; every text
// is read from the source.
//
// The node types and fields read here, as gotreesitter builds them
// (treesitter.Parser), alike in both grammars unless noted:
//
//	stylesheet                       statements as children
//	rule_set                         a selectors child, a block
//	selectors                        the selectors of the list, "," between them
//	class_selector                   "." and a class_name; in SCSS a nesting_selector and a
//	                                 class_name for &suffix
//	id_selector                      "#" and an id_name
//	block                            "{", declarations and statements, "}"
//	media_statement, supports_statement, at_rule (@layer), if_statement, each_statement
//	                                 a block among their children
//	keyframes_statement              a keyframe_block_list of keyframe_block, no rule_set
//	import_statement                 "@import", a string_value or a call_expression url(...)
//	use_statement, forward_statement (SCSS) "@use" or "@forward", a string_value
//	mixin_statement, function_statement
//	                                 (SCSS) field name, parameters, a block
//	include_statement                (SCSS) "@include", an identifier, arguments, a block
//	extend_statement                 (SCSS) "@extend" and the selector
//	call_expression                  a function_name ("f" or "ns.f"), arguments
//	string_value                     the quotes and, in CSS alone, a string_content
package stylesheet

import (
	"bytes"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"

	gts "github.com/odvcencio/gotreesitter"
	cssgrammar "github.com/odvcencio/gotreesitter/grammars/css"
	scssgrammar "github.com/odvcencio/gotreesitter/grammars/scss"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// The versions are this extractor's identity, one per language. Bump both
// by hand whenever the nodes or edges below change shape or meaning; the
// extract cache and the graph's meta key on them (see golang.Version for
// what a forgotten bump costs).
const (
	cssVersion  = "css/1"
	scssVersion = "scss/1"
)

// The languages as the graph's meta lists them and as a Result carries them.
const (
	cssName  = "css"
	scssName = "scss"
)

// CSS is the CSS extractor as extract.Language describes one.
type CSS struct{}

// Name is "css".
func (CSS) Name() string { return cssName }

// Version is the CSS version plus the tree-sitter runtime: a new runtime
// can build other trees from the same source.
func (CSS) Version() string { return cssVersion + "@" + treesitter.Parser }

// Extensions is ".css" alone.
func (CSS) Extensions() []string { return []string{".css"} }

// File is this package's File.
func (CSS) File(rel, source string) (extract.Result, error) { return File(rel, source) }

// SCSS is the SCSS extractor as extract.Language describes one.
type SCSS struct{}

// Name is "scss".
func (SCSS) Name() string { return scssName }

// Version is the SCSS version plus the tree-sitter runtime.
func (SCSS) Version() string { return scssVersion + "@" + treesitter.Parser }

// Extensions is ".scss" alone; the indented syntax .sass is another grammar.
func (SCSS) Extensions() []string { return []string{".scss"} }

// File is this package's File.
func (SCSS) File(rel, source string) (extract.Result, error) { return File(rel, source) }

// File extracts one CSS or SCSS file; its extension picks the language and
// the grammar. rel is its repo-relative, slash-separated path; source is
// its contents.
func File(rel, source string) (extract.Result, error) {
	if path.Ext(rel) == ".scss" {
		return file(scssgrammar.Language(), scssName, rel, source, false)
	}
	return file(cssgrammar.Language(), cssName, rel, source, false)
}

// Part extracts the CSS embedded in another file, the <style> of an HTML
// page: source is the whole file with everything outside the CSS blanked,
// so spans are lines of that file. It has no file node; rel is the file
// that holds it, whose node contains the rules.
func Part(rel, source string) (extract.Result, error) {
	return file(cssgrammar.Language(), cssName, rel, source, true)
}

// file is File and Part with the grammar handed in: the grammars build a
// tree from any input, and only a missing grammar makes the parse fail.
func file(lang *gts.Language, name, rel, source string, part bool) (extract.Result, error) {
	doc, err := treesitter.Parse(lang, rel, masked([]byte(source)))
	if err != nil {
		return extract.Result{}, err
	}
	defer doc.Close()
	// The tree is cut from the masked copy; every text is read from the
	// source, at the same offsets.
	doc.Src = []byte(source)

	x := &extractor{doc: doc, fileID: model.NodeID(rel), scss: name == scssName}
	x.walk(doc.Root, scope{id: x.fileID})
	nodes := x.nodes
	if !part {
		// walk has run every Symbol by now, as FileNode requires.
		nodes = append([]model.Node{doc.FileNode()}, nodes...)
	}
	return extract.Result{
		Path:        rel,
		Language:    name,
		Imports:     x.imports,
		Nodes:       nodes,
		Edges:       slices.Concat(x.contains, x.importEdges, x.extends, x.calls),
		ParseErrors: doc.ParseErrors(),
		Exports:     x.exports,
	}, nil
}

// extractor gathers one file's nodes and edges in source order.
type extractor struct {
	doc    *treesitter.Doc
	fileID model.NodeID
	scss   bool

	nodes    []model.Node
	contains []extract.RawEdge
	extends  []extract.RawEdge
	calls    []extract.RawEdge
	exports  []extract.Export

	imports     []extract.Import
	importEdges []extract.RawEdge
}

// scope is where a statement stands: id is the innermost node, the source
// of its edges; rule and tails are the innermost rule's name and the
// classes its selectors end on, both empty outside a rule.
type scope struct {
	id    model.NodeID
	rule  string
	tails []string
}

// walk reads n's children in scope s.
func (x *extractor) walk(n *gts.Node, s scope) {
	kids := n.Children()
	for i := 0; i < len(kids); i++ {
		c := kids[i]
		switch x.doc.Type(c) {
		case "ERROR":
			x.rejected(c, s)
		case "rule_set":
			x.rule(c, s)
		case "import_statement", "use_statement", "forward_statement":
			x.importStatement(x.doc.Text(c), c.StartByte())
		case "mixin_statement":
			x.callable(c, "mixin", s)
		case "function_statement":
			x.callable(c, "function", s)
		case "include_statement":
			if x.include(c, s) && i+1 < len(kids) && x.doc.Type(kids[i+1]) == "rule_set" {
				// The grammar read ns.m { … } as @include ns and a rule .m:
				// the rule is the include's content block.
				i++
				x.walk(kids[i], s)
			}
			x.walk(c, s)
		case "extend_statement":
			x.extend(x.doc.Text(c), c.StartByte(), s)
		case "call_expression":
			x.call(c, s)
			x.walk(c, s)
		default:
			x.walk(c, s)
		}
	}
}

// rule makes the node of one rule_set in s and reads its block. A rule the
// parser rebuilt around rejected text in its selectors, or whose selectors
// it had to invent in part -- {} or #{} --, gets no node; its block then
// reads in s.
func (x *extractor) rule(def *gts.Node, s scope) {
	var sel *gts.Node
	for _, c := range def.Children() {
		if x.doc.Type(c) == "selectors" {
			sel = c
		}
	}
	if sel == nil || treesitter.HoldsError(sel) || x.invented(sel) {
		x.walk(def, s)
		return
	}
	own := strings.Join(strings.Fields(x.doc.Text(sel)), " ")
	name := own
	if s.rule != "" {
		name = s.rule + " › " + own
	}
	n := x.define(def, sel.StartByte(), sel.EndByte(), name, "rule", !strings.HasPrefix(name, "%-") && !strings.HasPrefix(name, "%_"), s)
	// The signature is the selector list as written, line breaks included.
	n.Signature = x.doc.Text(sel)
	x.nodes[len(x.nodes)-1] = n
	// An ERROR between the selectors and the block is body the grammar
	// rejected, an @extend %p say: it reads in the rule.
	x.walk(def, scope{id: n.ID, rule: name, tails: x.offers(sel, s.tails, n.ID)})
}

// invented is whether the selectors n name nothing the source holds: no
// text at all, or a class or id of no text. The & the grammar adds in
// front of a nested > .b has no text either and is no reason.
func (x *extractor) invented(n *gts.Node) bool {
	found := n.StartByte() == n.EndByte()
	treesitter.Walk(n, func(c *gts.Node) bool {
		t := x.doc.Type(c)
		found = found || (t == "class_name" || t == "id_name") && c.StartByte() == c.EndByte()
		return !found
	})
	return found
}

// callable makes the node of one @mixin or @function in s and reads its
// block, with no rule around for the rules inside it.
func (x *extractor) callable(def *gts.Node, kind model.Kind, s scope) {
	var head []*gts.Node
	for _, c := range def.Children() {
		if x.doc.Type(c) == "block" {
			break
		}
		head = append(head, c)
	}
	if slices.ContainsFunc(head, treesitter.HoldsError) {
		x.walk(def, s)
		return
	}
	name := x.doc.Text(x.doc.Field(def, "name"))
	n := x.define(def, def.StartByte(), head[len(head)-1].EndByte(), name, kind, !strings.HasPrefix(name, "-") && !strings.HasPrefix(name, "_"), s)
	x.walk(def, scope{id: n.ID})
}

// define makes the node of one definition and the contains edge to it from s.
func (x *extractor) define(def *gts.Node, head, headerEnd uint32, name string, kind model.Kind, exported bool, s scope) model.Node {
	n := x.doc.Symbol(treesitter.Sym{
		Outer: def, Head: head, HeaderEnd: headerEnd,
		Name: name, Qualified: name, Kind: kind, Exported: exported,
	})
	x.nodes = append(x.nodes, n)
	x.contains = append(x.contains, extract.RawEdge{
		Source: s.id, Relation: model.RelationContains, TargetID: n.ID, File: x.doc.Rel,
	})
	return n
}

// trailingClass is the class a selector ends on.
var trailingClass = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`\.([\w-]+)$`) })

// suffixed is an SCSS selector that appends to its parent's class: &__el.
var suffixed = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^&([\w-]+)$`) })

// offers records every class and id of sel as offered by target, once per
// name, and returns the classes sel's selectors end on. tails are those of
// the rule around, which an &suffix appends to.
func (x *extractor) offers(sel *gts.Node, tails []string, target model.NodeID) []string {
	seen := map[extract.Export]bool{}
	add := func(ns extract.Namespace, name string) {
		e := extract.Export{Namespace: ns, Name: name, Target: target}
		if !strings.Contains(name, "#{") && !seen[e] {
			seen[e] = true
			x.exports = append(x.exports, e)
		}
	}
	treesitter.Walk(sel, func(n *gts.Node) bool {
		kids := n.Children()
		switch x.doc.Type(n) {
		case "class_selector":
			for i := 1; i < len(kids); i++ {
				if x.doc.Type(kids[i]) != "class_name" {
					continue
				}
				switch x.doc.Type(kids[i-1]) {
				case ".":
					add(extract.NamespaceCSSClass, x.doc.Text(kids[i]))
				case "nesting_selector":
					for _, k := range tails {
						add(extract.NamespaceCSSClass, k+x.doc.Text(kids[i]))
					}
				}
			}
		case "id_selector":
			add(extract.NamespaceCSSID, x.doc.Text(kids[len(kids)-1]))
		}
		return true
	})
	var out []string
	for _, c := range sel.Children() {
		text := x.doc.Text(c)
		if m := suffixed().FindStringSubmatch(text); m != nil {
			for _, k := range tails {
				out = append(out, k+m[1])
			}
		} else if m := trailingClass().FindStringSubmatch(text); m != nil {
			out = append(out, m[1])
		}
	}
	return out
}

// quoted is a path in an import: a quoted string, or the bare argument of
// url().
var quoted = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`url\(\s*([^"'\s)]+)\s*\)|["']([^"']+)["']`) })

// atKeyword is the keyword an import statement opens with: @import, @use or
// @forward.
var atKeyword = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^@[a-z]+`) })

// useAs is the namespace of an @use, absent without as.
var useAs = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^@use\s+["'][^"']+["']\s+as\s+([\w-]+|\*)`) })

// importStatement records every path of an @import, @use or @forward text
// starting at off. It is read as text: the SCSS grammar breaks on @use … as
// and on a list of paths, and its strings hold no string_content.
func (x *extractor) importStatement(text string, off uint32) {
	kind := ""
	if x.scss {
		kind = atKeyword().FindString(text)[1:]
	}
	alias := ""
	if m := useAs().FindStringSubmatch(text); m != nil {
		alias = m[1]
	}
	for _, m := range quoted().FindAllStringSubmatch(text, -1) {
		x.importOf(extract.Import{Path: m[1] + m[2], Alias: alias, Kind: kind}, x.line(off))
		if kind != "import" {
			break // only SCSS's @import lists paths; with (…) and show follow one
		}
	}
}

// importOf records imp as an import and an imports edge at line.
func (x *extractor) importOf(imp extract.Import, line int) {
	x.imports = append(x.imports, imp)
	x.importEdges = append(x.importEdges, extract.RawEdge{
		Source: x.fileID, Relation: model.RelationImports, Specifier: imp.Path, File: x.doc.Rel, Line: line,
	})
}

// line is the 1-based line of a byte offset.
func (x *extractor) line(off uint32) int {
	return 1 + bytes.Count(x.doc.Src[:off], []byte("\n"))
}
