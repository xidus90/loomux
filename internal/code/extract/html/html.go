// Package html extracts the nodes and raw edges of one HTML page on the
// shared tree-sitter core, with its inline scripts and styles as embedded
// parts (docs/.superpowers/specs/2026-11-11-web-design.md).
//
// Nodes are the file and every element with a non-empty id, kind element,
// named #id and offered as html.id; a second element of the same id gets
// the next ordinal. <script src> is an extract.Import of Kind "script" and
// <link href> one of Kind "stylesheet" when a token of its rel is
// stylesheet, both with the path as written; <a href> is navigation and
// none. Every class of a class attribute is handed over as a references
// edge with Member css.class, and an on* attribute whose value starts with
// a plain call, go(…) or return go(…), as a calls edge with Member
// js.function; both come from the file, once per name, on the first line
// that names it. Attribute names, tag names and the type and lang of a
// script are read in any case.
//
// The bodies of the inline scripts of one language, and of all <style>
// elements, are each read as one part (extract.Result.Embedded) by that
// language's extractor, from a copy of the page with everything else
// blanked (mask). A script is JavaScript without a type, with an empty
// one, with module or a JavaScript MIME type, and TypeScript with
// lang="ts" or type="text/typescript"; any other type is data or a
// template. The elements take their ids first and each part takes its own
// in the same set after them, so <div id="app"> and <style>#app{} are
// p.html##app and p.html##app~2. A top-level function of a classic script
// -- not a module -- is a global of the page and offered as js.function.
//
// The grammar breaks elements up where a page is not well-formed: <% x %>
// of a template takes the elements around it into an ERROR that holds
// their start tags bare, and an unclosed <p> before a <style> or a
// <script> without an end tag leaves the start_tag and the raw_text side
// by side, with no script_element or style_element around them. So every
// tag is read wherever it stands, an element whose tag has no element
// around it spans its tag alone, and a body belongs to the start_tag
// before it. {{ x }} of a template is text.
//
// The node types read here, as gotreesitter builds them
// (treesitter.Parser):
//
//	document                         doctype, elements, text, comments as children
//	element                          a start_tag, children and an end_tag, or a self_closing_tag
//	script_element, style_element    a start_tag, a raw_text when the body holds anything, an
//	                                 end_tag
//	start_tag, self_closing_tag      "<", a tag_name, attribute children, ">" or "/>"
//	attribute                        an attribute_name, then "=" and an attribute_value or a
//	                                 quoted_attribute_value; a name alone for a boolean
//	quoted_attribute_value           the quotes around one attribute_value, none when empty
//	raw_text                         the body of a script or a style
//	ERROR                            what the grammar rejected, tags and elements inside it
package html

import (
	"regexp"
	"slices"
	"strings"
	"sync"

	gts "github.com/odvcencio/gotreesitter"
	htmlgrammar "github.com/odvcencio/gotreesitter/grammars/html"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/ecmascript"
	"github.com/xidus90/loomux/internal/code/extract/stylesheet"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// version is this extractor's identity. Bump it by hand whenever the nodes
// or edges below change shape or meaning; the extract cache and the graph's
// meta key on it (see golang.Version for what a forgotten bump costs). A
// page's cache entry carries its parts as well, so it is bumped with the
// version of every extractor it embeds; TestEmbeddedVersionsArePinned
// fails when one moves.
const version = "html/1"

// Language is the HTML extractor as extract.Language describes one.
type Language struct{}

// Name is "html".
func (Language) Name() string { return "html" }

// Version is the HTML version plus the tree-sitter runtime: a new runtime
// can build other trees from the same source.
func (Language) Version() string { return version + "@" + treesitter.Parser }

// Extensions are both spellings of a page.
func (Language) Extensions() []string { return []string{".html", ".htm"} }

// File is this package's File.
func (Language) File(rel, source string) (extract.Result, error) { return File(rel, source) }

// File extracts one HTML page. rel is its repo-relative, slash-separated
// path; source is its contents.
func File(rel, source string) (extract.Result, error) {
	return file(htmlgrammar.Language(), rel, source, readPart)
}

// readPart reads one part of a page, in language lang, with that
// language's extractor.
func readPart(lang, rel, source string) (extract.Result, error) {
	switch lang {
	case "typescript":
		return ecmascript.Part(rel, source, true)
	case "css":
		return stylesheet.Part(rel, source)
	}
	return ecmascript.Part(rel, source, false)
}

// file is File with the grammar and the reader of the parts handed in:
// the grammars build a tree from any input, and only a missing grammar
// makes a parse fail.
func file(lang *gts.Language, rel, source string, read func(lang, rel, source string) (extract.Result, error)) (extract.Result, error) {
	doc, err := treesitter.Parse(lang, rel, []byte(source))
	if err != nil {
		return extract.Result{}, err
	}
	defer doc.Close()

	x := &extractor{doc: doc, fileID: model.NodeID(rel), handed: map[[2]string]bool{}}
	treesitter.Walk(doc.Root, x.visit)
	minted := map[string]bool{}
	for _, n := range x.nodes {
		minted[string(n.ID)] = true
	}
	parts, err := embed(rel, source, []part{
		{lang: "javascript", spans: x.js, sep: ';', classic: x.classic},
		{lang: "typescript", spans: x.ts, sep: ';'},
		{lang: "css", spans: x.css},
	}, minted, read)
	if err != nil {
		return extract.Result{}, err
	}
	// A part's definitions keep their bodies on their own nodes; the
	// elements, which may span the whole page, do not count.
	covered := map[int]bool{}
	for _, p := range parts {
		for _, n := range p.Nodes {
			from, to, _ := n.Span.Lines()
			for i := from; i <= to; i++ {
				covered[i] = true
			}
		}
	}
	return extract.Result{
		Path:        rel,
		Language:    "html",
		Imports:     x.imports,
		Nodes:       append([]model.Node{extract.FileNode(rel, source, covered)}, x.nodes...),
		Edges:       slices.Concat(x.contains, x.importEdges, x.handoffs),
		ParseErrors: doc.ParseErrors(),
		Exports:     x.exports,
		Embedded:    parts,
	}, nil
}

// extractor gathers one page's nodes, edges and part bodies in document
// order.
type extractor struct {
	doc    *treesitter.Doc
	fileID model.NodeID

	nodes       []model.Node
	contains    []extract.RawEdge
	exports     []extract.Export
	imports     []extract.Import
	importEdges []extract.RawEdge
	handoffs    []extract.RawEdge
	handed      map[[2]string]bool // (Member, Name) of every handoff so far

	js, ts, css [][2]int // byte spans of the bodies of each part
	classic     [][2]int // 1-based lines of the bodies of classic scripts
}

// visit reads one node of the tree and goes on below it.
func (x *extractor) visit(n *gts.Node) bool {
	switch x.doc.Type(n) {
	case "start_tag", "self_closing_tag":
		x.tag(n)
	case "raw_text":
		x.body(n)
	}
	return true
}

// elementTypes are the parents that make a tag's element.
var elementTypes = []string{"element", "script_element", "style_element"}

// tag reads one start or self-closing tag: the element its id names, the
// file it imports, and what its class and on* attributes hand over.
func (x *extractor) tag(t *gts.Node) {
	as := x.attributes(t)
	line := int(t.StartPoint().Row) + 1
	if id := get(as, "id"); id != "" {
		outer := t
		if p := t.Parent(); p != nil && slices.Contains(elementTypes, x.doc.Type(p)) {
			outer = p
		}
		x.element(outer, t, id)
	}
	switch x.tagName(t) {
	case "script":
		x.importOf(get(as, "src"), "script", line)
	case "link":
		if slices.Contains(strings.Fields(strings.ToLower(get(as, "rel"))), "stylesheet") {
			x.importOf(get(as, "href"), "stylesheet", line)
		}
	}
	for _, a := range as {
		switch {
		case a.name == "class":
			for _, c := range strings.Fields(templated().ReplaceAllString(a.value, "{")) {
				if !strings.ContainsAny(c, "{}<%") {
					x.handoff(model.RelationReferences, c, "css.class", line)
				}
			}
		case strings.HasPrefix(a.name, "on"):
			if m := handler().FindStringSubmatch(a.value); m != nil {
				x.handoff(model.RelationCalls, m[1], "js.function", line)
			}
		}
	}
}

// tagName is a tag's name in lower case, "" for a node without one.
func (x *extractor) tagName(t *gts.Node) string {
	for _, c := range t.Children() {
		if x.doc.Type(c) == "tag_name" {
			return strings.ToLower(x.doc.Text(c))
		}
	}
	return ""
}

// templated is a placeholder of a template language, {{ … }}, {% … %} or
// <% … %>. A class attribute's placeholders become a { each, so that the
// names inside them and every class glued to one, btn-{{ size }}, are
// skipped with the stray markers.
var templated = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`(?s)\{\{.*?\}\}|\{%.*?%\}|<%.*?%>`) })

// handler is an on* value that starts with a plain call.
var handler = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^\s*(?:return\s+)?([A-Za-z_$][\w$]*)\s*\(`) })

// element makes the node of the element outer with start tag t, and offers
// it by its id.
func (x *extractor) element(outer, t *gts.Node, id string) {
	n := x.doc.Symbol(treesitter.Sym{
		Outer: outer, Head: t.StartByte(), HeaderEnd: t.EndByte(),
		Name: "#" + id, Qualified: "#" + id, Kind: "element", Exported: true,
	})
	x.nodes = append(x.nodes, n)
	x.contains = append(x.contains, extract.RawEdge{Source: x.fileID, Relation: model.RelationContains, TargetID: n.ID, File: x.doc.Rel})
	x.exports = append(x.exports, extract.Export{Namespace: extract.NamespaceHTMLID, Name: id, Target: n.ID})
}

// importOf records path, unless empty, as an import of kind and an imports
// edge at line.
func (x *extractor) importOf(path, kind string, line int) {
	if path == "" {
		return
	}
	x.imports = append(x.imports, extract.Import{Kind: kind, Path: path})
	x.importEdges = append(x.importEdges, extract.RawEdge{
		Source: x.fileID, Relation: model.RelationImports, Specifier: path, File: x.doc.Rel, Line: line,
	})
}

// handoff records a name handed over from the file, once per member and
// name.
func (x *extractor) handoff(rel model.Relation, name string, member model.Kind, line int) {
	key := [2]string{string(member), name}
	if x.handed[key] {
		return
	}
	x.handed[key] = true
	x.handoffs = append(x.handoffs, extract.RawEdge{Source: x.fileID, Relation: rel, Name: name, Member: member, File: x.doc.Rel, Line: line})
}

// jsTypes are the types of a script that runs as JavaScript.
var jsTypes = []string{"", "module", "text/javascript", "application/javascript", "text/ecmascript", "application/ecmascript"}

// body files the body of a script or a style under its part. Its tag is
// the start_tag before it: inside the element, or beside it where the
// grammar broke the element up -- an unclosed <p> before it, or no end tag
// after it, which leaves the start_tag last in an ERROR before the body.
func (x *extractor) body(n *gts.Node) {
	t := n.PrevSibling()
	if t != nil && t.IsError() && t.ChildCount() > 0 {
		t = t.Child(t.ChildCount() - 1)
	}
	if t != nil && x.doc.Type(t) == "start_tag" {
		switch x.tagName(t) {
		case "script":
			x.script(x.attributes(t), n)
		case "style":
			x.css = append(x.css, span(n))
		}
	}
}

// script files the body of one inline script under its language; a script
// with src runs the file and never its body.
func (x *extractor) script(as []attribute, body *gts.Node) {
	if slices.ContainsFunc(as, func(a attribute) bool { return a.name == "src" }) {
		return
	}
	typ := strings.ToLower(strings.TrimSpace(get(as, "type")))
	js := slices.Contains(jsTypes, typ)
	switch {
	case typ == "text/typescript" || js && strings.ToLower(strings.TrimSpace(get(as, "lang"))) == "ts":
		x.ts = append(x.ts, span(body))
	case js:
		x.js = append(x.js, span(body))
		if typ != "module" {
			x.classic = append(x.classic, [2]int{int(body.StartPoint().Row) + 1, int(body.EndPoint().Row) + 1})
		}
	}
}

// span is a node's byte range.
func span(n *gts.Node) [2]int { return [2]int{int(n.StartByte()), int(n.EndByte())} }

// attribute is one attribute of a tag, its name in lower case and its
// value as written, "" for a boolean.
type attribute struct{ name, value string }

// attributes are a tag's attributes in order.
func (x *extractor) attributes(t *gts.Node) []attribute {
	var out []attribute
	for _, a := range t.Children() {
		if x.doc.Type(a) != "attribute" {
			continue
		}
		var at attribute
		for _, c := range a.Children() {
			switch x.doc.Type(c) {
			case "attribute_name":
				at.name = strings.ToLower(x.doc.Text(c))
			case "attribute_value":
				at.value = x.doc.Text(c)
			case "quoted_attribute_value":
				if c.ChildCount() == 3 {
					at.value = x.doc.Text(c.Child(1))
				}
			}
		}
		out = append(out, at)
	}
	return out
}

// get is the value of a tag's attribute name; a browser takes the first of
// two alike.
func get(as []attribute, name string) string {
	for _, a := range as {
		if a.name == name {
			return a.value
		}
	}
	return ""
}

// part is one embedded language of a page: the byte spans of its bodies,
// the byte mask writes after each, and the lines of its classic scripts.
type part struct {
	lang    string
	spans   [][2]int
	sep     byte
	classic [][2]int
}

// embed reads each part with bodies from its mask of source with read, in
// order, minting its ids in minted after those already there.
func embed(rel, source string, parts []part, minted map[string]bool, read func(lang, rel, source string) (extract.Result, error)) ([]extract.Result, error) {
	var out []extract.Result
	for _, p := range parts {
		if len(p.spans) == 0 {
			continue
		}
		r, err := read(p.lang, rel, mask(source, p.spans, p.sep))
		if err != nil {
			return nil, err
		}
		remint(&r, minted)
		r.Exports = append(r.Exports, globals(r.Nodes, p.classic)...)
		out = append(out, r)
	}
	return out, nil
}

// remint gives every node of part an id not yet in minted, and points its
// edges and offers at the new ids.
func remint(part *extract.Result, minted map[string]bool) {
	ids := map[model.NodeID]model.NodeID{}
	for i, n := range part.Nodes {
		ids[n.ID] = model.NodeID(extract.MintID(string(n.ID), minted))
		part.Nodes[i].ID = ids[n.ID]
	}
	to := func(id model.NodeID) model.NodeID {
		if n, ok := ids[id]; ok {
			return n
		}
		return id
	}
	for i := range part.Edges {
		part.Edges[i].Source, part.Edges[i].TargetID = to(part.Edges[i].Source), to(part.Edges[i].TargetID)
	}
	for i := range part.Exports {
		part.Exports[i].Target = to(part.Exports[i].Target)
	}
}

// globals offers every top-level function that starts within the lines of
// a classic script as js.function.
func globals(nodes []model.Node, classic [][2]int) []extract.Export {
	var out []extract.Export
	for _, n := range nodes {
		from, _, _ := n.Span.Lines()
		if n.Kind == "function" && n.Owner == "" && slices.ContainsFunc(classic, func(c [2]int) bool { return c[0] <= from && from <= c[1] }) {
			out = append(out, extract.Export{Namespace: extract.NamespaceJSFunction, Name: n.Name, Target: n.ID})
		}
	}
	return out
}
