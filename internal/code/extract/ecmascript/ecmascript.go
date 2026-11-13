// Package ecmascript extracts the nodes and raw edges of one JavaScript or
// TypeScript file on the shared tree-sitter core.
//
// Two languages, one extractor: JavaScript (.js, .mjs, .cjs, .jsx) and
// TypeScript (.ts, .mts, .cts, .tsx) differ in Result.Language, in their
// version and in the grammar an extension reads with, nothing else. .jsx
// reads with the tsx grammar, which takes the JSX attribute class= that the
// JavaScript grammar breaks on; .js keeps the JavaScript grammar
// (docs/.superpowers/specs/2026-11-11-ecmascript-design.md). A declaration
// file (.d.ts, .d.mts, .d.cts) repeats what its sources declare and keeps
// its file node alone, unparsed.
//
// Raw, as in the other extractors: a call, a base or an import names its
// target, and resolving the name needs every file of the repository, which
// is internal/code/resolve's job.
//
// Nodes are the file and what a module declares at its top level, behind
// export and export default too: functions, classes with their methods
// (the constructor as "constructor"), and a const, let or var bound to a
// function or an arrow; in TypeScript also interfaces, type aliases and
// enums (kind class), and the declarations of a namespace, qualified by it
// (N.f). CommonJS exports.f = function … and module.exports.f = … make a
// function f, and an anonymous default export -- export default function,
// export default class, module.exports = function -- is named default.
// Nested functions, object methods and ambient declarations (declare …)
// get no node. Every call belongs to the innermost function or method with
// a node, else to the file.
//
// Exported is set for a declaration behind export and for a name an export
// list without a path, export default or CommonJS names; a method is
// exported with its class unless #private, private or protected. Every
// binding of an import is one extract.Import (see its doc for Name, Alias
// and Kind); an export list without a path is one too, of Kind "export",
// and gets no imports edge.
//
// A definition the parser rebuilt around text it rejected gets no node, and
// nothing inside an ERROR counts. A break can also take the rest of the
// file with it: the JavaScript grammar reads the / of </p> after a JSX
// attribute class= as a regex, which runs to the next / and wraps every
// definition up to there in one ERROR, or ends the tree when no / follows.
// From the first such ERROR, or from where the tree ends, the rest is read
// again chunk by chunk (treesitter.Chunks): a chunk starts at a line that
// opens a declaration in column 0 outside a template literal or a block
// comment, and a regex cannot run past its chunk.
//
// The node types and fields read here, as gotreesitter builds them
// (treesitter.Parser), alike in all three grammars unless noted:
//
//	program                          statements as children; ERROR when the tree ends early
//	import_statement                 an import_clause or an import_require_clause (TypeScript), field source
//	import_clause                    an identifier (the default), namespace_import or named_imports
//	namespace_import                 "*", "as" and the identifier
//	import_specifier, export_specifier
//	                                 fields name (identifier or string), alias
//	import_require_clause            the identifier, field source
//	export_statement                 field declaration, or field value after "default", or an
//	                                 export_clause, namespace_export, "*" or (export =) an identifier;
//	                                 field source for a re-export
//	function_declaration, generator_function_declaration,
//	function_expression, generator_function, arrow_function
//	                                 fields name, parameters, body
//	class_declaration, abstract_class_declaration, class
//	                                 field name (identifier in JavaScript, type_identifier in
//	                                 TypeScript), a class_heritage child, field body (class_body)
//	class_heritage                   JavaScript: "extends" and the expression; TypeScript:
//	                                 extends_clause (field value) and implements_clause
//	method_definition                accessibility_modifier children, field name, field body
//	abstract_method_signature        field name, no body; method_signature, an overload, is skipped
//	interface_declaration            field name, an extends_type_clause (fields type), field body
//	type_alias_declaration           field name, "=", field value
//	enum_declaration                 field name, field body
//	internal_module, module          fields name (identifier or nested_identifier), body; a
//	                                 namespace arrives inside an expression_statement
//	lexical_declaration,
//	variable_declaration             variable_declarator children: fields name, value
//	assignment_expression            fields left, right
//	call_expression                  fields function (identifier, member_expression, import), arguments
//	member_expression                fields object, property
//	new_expression                   field constructor
//	generic_type, nested_type_identifier
//	                                 a base in a type position: field name; fields module, name
//	jsx_attribute                    the attribute's name, "=" and a string or a jsx_expression
//	jsx_opening_element,
//	jsx_self_closing_element         field name (identifier, member_expression or jsx_namespace_name)
//	string                           the quotes around one string_fragment
package ecmascript

import (
	"bytes"
	"path"
	"slices"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
	jsgrammar "github.com/odvcencio/gotreesitter/grammars/javascript"
	tsxgrammar "github.com/odvcencio/gotreesitter/grammars/tsx"
	tsgrammar "github.com/odvcencio/gotreesitter/grammars/typescript"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// The versions are this extractor's identity, one per language. Bump both
// by hand whenever the nodes or edges below change shape or meaning; the
// extract cache and the graph's meta key on them (see golang.Version for
// what a forgotten bump costs).
const (
	jsVersion = "javascript/1"
	tsVersion = "typescript/1"
)

// The languages as the graph's meta lists them and as a Result carries them.
const (
	jsName = "javascript"
	tsName = "typescript"
)

// JavaScript is the JavaScript extractor as extract.Language describes one.
type JavaScript struct{}

// Name is "javascript".
func (JavaScript) Name() string { return jsName }

// Version is the JavaScript version plus the tree-sitter runtime: a new
// runtime can build other trees from the same source.
func (JavaScript) Version() string { return jsVersion + "@" + treesitter.Parser }

// Extensions are the scripts and modules of JavaScript, JSX among them.
func (JavaScript) Extensions() []string { return []string{".js", ".mjs", ".cjs", ".jsx"} }

// File is this package's File.
func (JavaScript) File(rel, source string) (extract.Result, error) { return File(rel, source) }

// TypeScript is the TypeScript extractor as extract.Language describes one.
type TypeScript struct{}

// Name is "typescript".
func (TypeScript) Name() string { return tsName }

// Version is the TypeScript version plus the tree-sitter runtime.
func (TypeScript) Version() string { return tsVersion + "@" + treesitter.Parser }

// Extensions are the sources of TypeScript, TSX among them.
func (TypeScript) Extensions() []string { return []string{".ts", ".mts", ".cts", ".tsx"} }

// File is this package's File.
func (TypeScript) File(rel, source string) (extract.Result, error) { return File(rel, source) }

// File extracts one JavaScript or TypeScript file; its extension picks the
// language and the grammar. rel is its repo-relative, slash-separated path;
// source is its contents.
func File(rel, source string) (extract.Result, error) {
	ext, lang := path.Ext(rel), jsName
	switch ext {
	case ".ts", ".mts", ".cts", ".tsx":
		lang = tsName
	}
	stem := strings.TrimSuffix(rel, ext)
	if lang == tsName && ext != ".tsx" && strings.HasSuffix(stem, ".d") {
		return extract.Result{Path: rel, Language: lang, Nodes: []model.Node{extract.FileNode(rel, source, nil)}}, nil
	}
	return file(grammar(ext), lang, rel, source, false)
}

// Part extracts the script embedded in another file, the inline scripts of
// an HTML page: source is the whole file with everything outside the
// script blanked, so spans are lines of that file. ts reads it as
// TypeScript. It has no file node; rel is the file that holds it, whose
// node contains the definitions.
func Part(rel, source string, ts bool) (extract.Result, error) {
	if ts {
		return file(tsgrammar.Language(), tsName, rel, source, true)
	}
	return file(jsgrammar.Language(), jsName, rel, source, true)
}

// grammar is the grammar an extension reads with.
func grammar(ext string) *gts.Language {
	switch ext {
	case ".jsx", ".tsx":
		return tsxgrammar.Language()
	case ".ts", ".mts", ".cts":
		return tsgrammar.Language()
	}
	return jsgrammar.Language()
}

// maxChunks bounds how many parses read the rest of a file its first tree
// left behind. A rest with more declarations is read in fewer, larger
// chunks.
const maxChunks = 256

// file is File and Part with the grammar handed in: the grammars build a
// tree from any input, and only a missing grammar makes the parse fail.
func file(lang *gts.Language, name, rel, source string, part bool) (extract.Result, error) {
	src := []byte(source)
	doc, err := treesitter.Parse(lang, rel, src)
	if err != nil {
		return extract.Result{}, err
	}
	defer doc.Close()

	x := &extractor{doc: doc, fileID: model.NodeID(rel)}
	opens := opener(src)
	cut := x.trusted(opens)
	for _, c := range doc.Root.Children() {
		if c.StartByte() < cut {
			x.statement(c, "")
		}
	}
	errs := doc.ParseErrors()
	starts := treesitter.Chunks(src, cut, maxChunks, opens)
	if int(cut) < len(src) && (cut == 0 || src[cut-1] == '\n') && opens(int(cut)) {
		starts = append([]int{int(cut)}, starts...) // the ERROR starts a line of its own
	}
	for i := 0; i < len(starts) && doc.ReparseChunk(src, starts, i) == nil; i++ {
		x.statements(doc.Root, "")
		errs += doc.ParseErrors()
	}
	x.finish()
	nodes := x.nodes
	if !part {
		// statements has run every Symbol by now, as FileNode requires: its
		// body text is what no symbol covers.
		nodes = append([]model.Node{doc.FileNode()}, nodes...)
	}
	return extract.Result{
		Path:        rel,
		Language:    name,
		Imports:     x.imports,
		Nodes:       nodes,
		Edges:       slices.Concat(x.contains, x.importEdges, x.bases, x.calls, x.refs),
		ParseErrors: errs,
		Exports:     x.exports,
		Bridges:     x.bridges,
	}, nil
}

// trusted is where the first tree stops being trusted: the start of the
// first broken top-level node that runs across a line opening a
// declaration, else the end of the tree. Broken is a node holding an ERROR,
// or any node when the root itself is one: the regex a stray / opens then
// sits beside its ERROR, not in it.
func (x *extractor) trusted(opens func(pos int) bool) uint32 {
	broken := x.doc.Root.IsError()
	for _, c := range x.doc.Root.Children() {
		if (broken || treesitter.HoldsError(c)) && len(treesitter.Chunks(x.doc.Src[:c.EndByte()], c.StartByte(), 1, opens)) > 0 {
			return c.StartByte()
		}
	}
	return x.doc.Root.EndByte()
}

// opener reports whether the line starting at pos of src can open a
// declaration: a keyword, a name or a decorator, and not text of a template
// literal or a block comment. What is text is worked out once, at the first
// question.
func opener(src []byte) func(pos int) bool {
	var text []bool
	return func(pos int) bool {
		if text == nil {
			text = make([]bool, len(src))
			code(src, 0, false, text)
		}
		c := src[pos]
		return !text[pos] && (c == '_' || c == '$' || c == '@' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z')
	}
}

// code scans code from i and marks in text every byte of a block comment
// or a template literal. Nested, it is a template's ${…} and ends after the
// brace that closes it. A regex literal is read as code: a quote or a
// backtick in one is taken for the start of a literal, which can only hide
// lines from a restart.
func code(src []byte, i int, nested bool, text []bool) int {
	depth := 0
	for i < len(src) {
		switch c := src[i]; {
		case bytes.HasPrefix(src[i:], []byte("//")):
			i = treesitter.LineEnd(src, i)
		case bytes.HasPrefix(src[i:], []byte("/*")):
			end := treesitter.Closing(src, i+2, "*/")
			mark(text, i, end)
			i = end
		case c == '"' || c == '\'':
			i = treesitter.QuotedEnd(src, i)
		case c == '`':
			end := template(src, i+1, text)
			mark(text, i, end)
			i = end
		case c == '}' && nested && depth == 0:
			return i + 1
		default:
			switch c {
			case '{':
				depth++
			case '}':
				depth--
			}
			i++
		}
	}
	return len(src)
}

// template scans a template literal from i, after its opening backtick, to
// the offset after its closing one.
func template(src []byte, i int, text []bool) int {
	for i < len(src) {
		switch {
		case src[i] == '\\':
			i += 2
		case src[i] == '`':
			return i + 1
		case bytes.HasPrefix(src[i:], []byte("${")):
			i = code(src, i+2, true, text)
		default:
			i++
		}
	}
	return len(src)
}

// mark sets text from from to to.
func mark(text []bool, from, to int) {
	for i := from; i < to; i++ {
		text[i] = true
	}
}

// extractor gathers one file's nodes and edges in source order.
type extractor struct {
	doc    *treesitter.Doc
	fileID model.NodeID
	hidden bool // inside a namespace that is not exported

	nodes    []model.Node
	members  []member
	contains []extract.RawEdge
	bases    []extract.RawEdge
	calls    []extract.RawEdge
	refs     []extract.RawEdge

	imports     []extract.Import
	importEdges []extract.RawEdge

	exports []extract.Export
	bridges []extract.Bridge

	declared []string // the top-level names an export keyword exports by their own name
}

// member is a method whose Exported follows its class's, which an export
// list after the class can still set: indexes into extractor.nodes.
type member struct {
	node, class int
	private     bool
}

// finish sets what only the whole file tells: Exported for every name an
// export list names and for the methods of exported classes, and the scope
// of every class handover. It also drops every Alias equal to its Name.
//
// A declaration export names directly that a list also exports under
// another name gets an export of Kind "export" under its own name: the
// list's entry alone would say the file exports it renamed only.
func (x *extractor) finish() {
	named, own := map[string]bool{}, map[string]bool{}
	for i, imp := range x.imports {
		if imp.Alias == imp.Name {
			x.imports[i].Alias = "" // an alias is written only where it differs
		}
		if imp.Kind == "export" {
			named[imp.Name] = true
			own[imp.Name] = own[imp.Name] || x.imports[i].Alias == ""
		}
	}
	for _, name := range x.declared {
		if named[name] && !own[name] {
			own[name] = true
			x.imports = append(x.imports, extract.Import{Name: name, Kind: "export"})
		}
	}
	for i := range x.nodes {
		if x.nodes[i].Owner == "" && named[x.nodes[i].Name] {
			x.nodes[i].Exported = true
		}
	}
	for _, m := range x.members {
		x.nodes[m.node].Exported = x.nodes[m.class].Exported && !m.private
	}
	scope := x.styles()
	for i := range x.bridges {
		x.bridges[i].Scope = scope
	}
}

// statements reads the statements of a module, or of a namespace's body
// when prefix names the namespace.
func (x *extractor) statements(n *gts.Node, prefix string) {
	for _, c := range n.Children() {
		x.statement(c, prefix)
	}
}

// statement reads one statement of a module or a namespace's body.
func (x *extractor) statement(c *gts.Node, prefix string) {
	switch x.doc.Type(c) {
	case "ERROR":
		// Text the grammar rejected: a definition in it would be a
		// guess, and a call in it would be credited to the file.
	case "ambient_declaration":
		// declare …: a declaration of what another file defines.
	case "export_statement":
		x.export(c, prefix)
	case "import_statement":
		x.importStatement(c)
	case "expression_statement":
		if e := c.Child(0); x.doc.Type(e) == "internal_module" {
			x.namespace(e, prefix, false)
		} else if prefix != "" || x.doc.Type(e) != "assignment_expression" || !x.commonJS(e) {
			x.walk(c, x.fileID)
		}
	default:
		if !x.declaration(c, prefix, false) {
			x.walk(c, x.fileID)
		}
	}
}

// functionValues are the expressions that make a function.
var functionValues = map[string]bool{"function_expression": true, "generator_function": true, "arrow_function": true}

// declaration makes the nodes of a declaration in the namespace prefix,
// exported or not as its export says; false for a statement that is none.
func (x *extractor) declaration(n *gts.Node, prefix string, exported bool) bool {
	switch x.doc.Type(n) {
	case "function_declaration", "generator_function_declaration":
		x.function(n, n, x.doc.Text(x.doc.Field(n, "name")), prefix, exported)
	case "class_declaration", "abstract_class_declaration":
		x.class(n, x.doc.Text(x.doc.Field(n, "name")), prefix, exported)
	case "interface_declaration":
		if node, ok := x.symbol(n, n, x.doc.Text(x.doc.Field(n, "name")), prefix, "interface", exported); ok {
			for _, c := range n.Children() {
				if x.doc.Type(c) == "extends_type_clause" {
					x.typeBases(node.ID, model.RelationExtends, c)
				}
			}
		}
	case "type_alias_declaration":
		x.symbol(n, n, x.doc.Text(x.doc.Field(n, "name")), prefix, "type", exported)
	case "enum_declaration":
		x.symbol(n, n, x.doc.Text(x.doc.Field(n, "name")), prefix, "class", exported)
	case "lexical_declaration", "variable_declaration":
		for _, d := range n.Children() {
			if x.doc.Type(d) != "variable_declarator" {
				continue // the keyword, the commas and the semicolon
			}
			name, value := x.doc.Field(d, "name"), x.doc.Field(d, "value")
			if x.doc.Type(name) == "identifier" && functionValues[x.doc.Type(value)] {
				x.function(d, value, x.doc.Text(name), prefix, exported)
			} else {
				x.walk(d, x.fileID)
			}
		}
	case "module":
		x.namespace(n, prefix, exported)
	default:
		return false
	}
	return true
}

// namespace reads the body of a TypeScript namespace under its name; one
// named by a string is an ambient module and declares nothing here. A
// member is exported only when every namespace around it is.
func (x *extractor) namespace(n *gts.Node, prefix string, exported bool) {
	name := x.doc.Field(n, "name")
	if x.doc.Type(name) == "string" {
		return
	}
	hidden := x.hidden
	x.hidden = hidden || !exported
	x.statements(x.doc.Field(n, "body"), qualify(prefix, x.doc.Text(name)))
	x.hidden = hidden
}

// function makes the node of a function def, spanning outer, and reads its
// calls.
func (x *extractor) function(outer, def *gts.Node, name, prefix string, exported bool) {
	if node, ok := x.symbol(outer, def, name, prefix, "function", exported); ok {
		x.walk(def, node.ID)
	}
}

// class makes the node of a class, its bases, its methods and their calls.
// Code in the body outside a method belongs to the file.
func (x *extractor) class(def *gts.Node, name, prefix string, exported bool) {
	node, ok := x.symbol(def, def, name, prefix, "class", exported)
	if !ok {
		return
	}
	class := len(x.nodes) - 1
	for _, c := range def.Children() {
		if x.doc.Type(c) == "class_heritage" {
			x.heritage(node.ID, c)
			x.walk(c, x.fileID)
		}
	}
	for _, c := range x.doc.Field(def, "body").Children() {
		switch x.doc.Type(c) {
		case "method_definition", "abstract_method_signature":
			x.method(c, node.ID, qualify(prefix, name), class)
		case "method_signature":
			// An overload: the implementation after it gets the node.
		default:
			x.walk(c, x.fileID)
		}
	}
}

// method makes the node of a method of the class class, named owner and at
// index in extractor.nodes.
func (x *extractor) method(def *gts.Node, class model.NodeID, owner string, index int) {
	name := x.doc.Text(x.doc.Field(def, "name"))
	head, ok := x.header(def)
	if !ok {
		return
	}
	private := strings.HasPrefix(name, "#")
	for _, c := range head {
		if t := x.doc.Text(c); x.doc.Type(c) == "accessibility_modifier" && (t == "private" || t == "protected") {
			private = true
		}
	}
	n := x.doc.Symbol(treesitter.Sym{
		Outer:     def,
		Head:      head[0].StartByte(),
		HeaderEnd: head[len(head)-1].EndByte(),
		Name:      name,
		Qualified: owner + "." + name,
		Kind:      "method",
		Owner:     owner,
	})
	x.add(n, class)
	x.members = append(x.members, member{node: len(x.nodes) - 1, class: index, private: private})
	x.walk(def, n.ID)
}

// symbol makes the node of a definition def in the namespace prefix and the
// contains edge from the file to it. outer is its extent: def itself, or the
// declarator or assignment that names a function, whose signature then
// starts with the name. It is false, and makes nothing, when the parser
// rebuilt the header around rejected text.
func (x *extractor) symbol(outer, def *gts.Node, name, prefix string, kind model.Kind, exported bool) (model.Node, bool) {
	head, ok := x.header(def)
	if !ok {
		return model.Node{}, false
	}
	start := outer.StartByte()
	if outer == def {
		// The signature starts after the decorators, which belong to the span.
		i := slices.IndexFunc(head, func(c *gts.Node) bool { return x.doc.Type(c) != "decorator" })
		start = head[i].StartByte()
	}
	n := x.doc.Symbol(treesitter.Sym{
		Outer:     outer,
		Head:      start,
		HeaderEnd: head[len(head)-1].EndByte(),
		Name:      name,
		Qualified: qualify(prefix, name),
		Kind:      kind,
		Owner:     prefix,
		Exported:  exported && !x.hidden,
	})
	x.add(n, x.fileID)
	return n, true
}

// add records a node and the contains edge from parent to it.
func (x *extractor) add(n model.Node, parent model.NodeID) {
	x.nodes = append(x.nodes, n)
	x.contains = append(x.contains, extract.RawEdge{
		Source: parent, Relation: model.RelationContains, TargetID: n.ID, File: x.doc.Rel,
	})
}

// header is every child of a definition before its body, or before the "="
// of a type alias. It is false when the parser rebuilt the definition around
// text it rejected: an ERROR at any depth of the header. The extent then
// claims text of another definition, and no node beats a wrong one. An
// ERROR in the body is not the header's.
func (x *extractor) header(def *gts.Node) ([]*gts.Node, bool) {
	head, body := def.Children(), x.doc.Field(def, "body")
	if i := slices.IndexFunc(head, func(c *gts.Node) bool {
		return body != nil && c.StartByte() == body.StartByte() || x.doc.Type(c) == "="
	}); i >= 0 {
		head = head[:i]
	}
	return head, !slices.ContainsFunc(head, treesitter.HoldsError)
}

// qualify is name in the namespace prefix.
func qualify(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}
