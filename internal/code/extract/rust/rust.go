// Package rust extracts the nodes and raw edges of one Rust file on the
// shared tree-sitter core.
//
// Nodes are the file, the functions, structs, enums, unions, traits, type
// aliases and macro_rules! definitions at module level, and the methods of
// an impl or trait. Modules written inline name no node; their path
// qualifies the ids of what is inside them (a::b::f), and a module in a file
// of its own is that file. An impl has no node of its own: its methods are
// owned by the type it names, which may be declared in any file of the crate,
// so a method hangs from its type only when the type is in the same file.
//
// Tests are marked on the node (model.Node.Test), since Rust keeps them in
// the file of the code they test: #[test], any attribute whose path ends in
// test, and everything under #[cfg(test)].
//
// The godot-rust registrations are read as text from the attributes in front
// of an item: a struct deriving GodotClass is offered as a Godot class, and
// #[func], #[rpc] and #[signal] in a #[godot_api] impl leave a raw
// references edge for the resolver to turn into a binding.
//
// Uses, mod x;, calls, macro invocations, impl Trait for and supertraits are
// read in edges.go. A call belongs to the innermost function or method that
// has a node, and to the file where none does; the raw edges are described at
// extract.RawEdge and extract.Import.
//
// A definition the parser rebuilt around text it rejected gets no node
// (header).
//
// The node types and fields read here, as gotreesitter builds them
// (treesitter.Parser):
//
//	source_file, declaration_list      items as children, each preceded by its attribute_item siblings
//	attribute_item                     an attribute child: a path, and the field arguments, a token_tree
//	function_item                      visibility_modifier, field name, field body (a block)
//	function_signature_item            a method without a body, ending in ";"
//	struct_item, union_item            field name, a field_declaration_list or ordered_field_declaration_list body
//	enum_item                          field name, an enum_variant_list body
//	trait_item                         field name, a declaration_list body
//	type_item                          field name
//	macro_definition                   field name; the rules follow the name
//	mod_item                           field name, a declaration_list body unless the module is its own file
//	impl_item                          field type, field trait, a declaration_list body
//	generic_type, reference_type       field type
//	scoped_type_identifier             field name
package rust

import (
	"slices"

	gts "github.com/odvcencio/gotreesitter"
	rustgrammar "github.com/odvcencio/gotreesitter/grammars/rust"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// version is this extractor's identity. Bump it by hand whenever the nodes or
// edges below change shape or meaning; the extract cache and the graph's meta
// key on it (see golang.Version for what a forgotten bump costs).
const version = "rust/1"

// langName is the language as the graph's meta lists it and as a Result carries it.
const langName = "rust"

// Language is the Rust extractor as extract.Language describes one.
type Language struct{}

// Name is "rust".
func (Language) Name() string { return langName }

// Version is this package's version plus the tree-sitter runtime: a new
// runtime can build other trees from the same source.
func (Language) Version() string { return version + "@" + treesitter.Parser }

// Extensions is ".rs" alone.
func (Language) Extensions() []string { return []string{".rs"} }

// File is this package's File.
func (Language) File(rel, source string) (extract.Result, error) { return File(rel, source) }

// File extracts one Rust file. rel is its repo-relative, slash-separated path;
// source is its contents.
func File(rel, source string) (extract.Result, error) {
	return file(rustgrammar.Language(), rel, source)
}

// file is File with the grammar handed in: the Rust grammar builds a tree from
// any input, and only a missing grammar makes the parse fail.
func file(lang *gts.Language, rel, source string) (extract.Result, error) {
	doc, err := treesitter.Parse(lang, rel, []byte(source))
	if err != nil {
		return extract.Result{}, err
	}
	defer doc.Close()

	x := &extractor{doc: doc, fileID: model.NodeID(rel), types: map[string]model.NodeID{}, owned: map[int]string{}, impls: map[int]string{}}
	x.items(doc.Root, "", false)
	x.hang()
	return extract.Result{
		Path:     rel,
		Language: langName,
		// items has run every Symbol by now, as FileNode requires: its body
		// text is what no symbol covers.
		Nodes:       append([]model.Node{doc.FileNode()}, x.nodes...),
		Edges:       slices.Concat(x.contains, x.bindings, x.importEdges, x.relations, x.calls),
		Imports:     x.imports,
		ParseErrors: doc.ParseErrors(),
		Exports:     x.exports,
	}, nil
}

// extractor gathers one file's nodes and edges in source order.
type extractor struct {
	doc    *treesitter.Doc
	fileID model.NodeID

	nodes    []model.Node
	contains []extract.RawEdge
	bindings []extract.RawEdge
	exports  []extract.Export

	imports     []extract.Import
	importEdges []extract.RawEdge // mod x; as an edge
	relations   []extract.RawEdge // implements and extends
	calls       []extract.RawEdge

	types map[string]model.NodeID // the type nodes of this file by module path and name
	owned map[int]string          // contains edges of methods by index: the type that should hold them
	impls map[int]string          // implements edges by index in relations: the type that should start them
}

// owner is the type whose members are read: an impl's type, or a trait.
type owner struct {
	name     string // as written, without generics
	qual     string // module path and name: the key of its node
	mod      string // the module path alone, "a::b::" or ""
	exported bool   // the members are visible without a pub of their own
	test     bool
	godot    bool // a #[godot_api] impl
}

// each hands every item among n's children to visit with the attributes in
// front of it. Comments sit between an attribute and its item and change
// nothing.
func (x *extractor) each(n *gts.Node, visit func(item *gts.Node, attrs []attr)) {
	var attrs []attr
	for _, c := range n.Children() {
		switch x.doc.Type(c) {
		case "attribute_item":
			attrs = append(attrs, x.attrsOf(c)...)
		case "line_comment", "block_comment":
		default:
			visit(c, attrs)
			attrs = nil
		}
	}
}

// kinds are the item types that make a node of their own, with its kind.
var kinds = map[string]model.Kind{
	"function_item": "function", "struct_item": "struct", "enum_item": "enum", "union_item": "union",
	"trait_item": "trait", "type_item": "type", "macro_definition": "macro",
}

// items reads the items among n's children in module mod ("" or "a::b::"),
// test being whether the module is compiled for tests alone.
func (x *extractor) items(n *gts.Node, mod string, test bool) {
	x.each(n, func(c *gts.Node, attrs []attr) {
		t := x.doc.Type(c)
		test := test || cfgTest(attrs)
		switch {
		case t == "mod_item":
			// A module in a file of its own has no body here.
			if body := x.doc.Field(c, "body"); body != nil {
				x.items(body, mod+x.doc.Text(x.doc.Field(c, "name"))+"::", test)
			} else {
				x.module(c, attrs, mod)
			}
		case t == "impl_item":
			x.impl(c, attrs, mod, test)
		case kinds[t] != "":
			x.item(c, t, attrs, mod, test)
		default:
			x.refs(c, scope{source: x.fileID, mod: mod})
		}
	})
}

// item makes the node of one function or type and reads what is inside it.
func (x *extractor) item(def *gts.Node, t string, attrs []attr, mod string, test bool) {
	head, name, ok := x.header(def)
	if !ok {
		return
	}
	exported := x.exported(head)
	n := x.symbol(def, head, name, mod+name, kinds[t], "", exported, test || (t == "function_item" && isTest(attrs)))
	x.contains = append(x.contains, extract.RawEdge{
		Source: x.fileID, Relation: model.RelationContains, TargetID: n.ID, File: x.doc.Rel,
	})
	if t == "function_item" {
		x.refs(x.doc.Field(def, "body"), scope{source: n.ID, mod: mod})
	}
	switch t {
	case "struct_item":
		x.types[mod+name] = n.ID
		if class := godotClass(attrs, name); class != "" {
			x.exports = append(x.exports, extract.Export{Namespace: extract.NamespaceGodotClass, Name: class, Target: n.ID})
		}
	case "enum_item", "union_item", "type_item":
		x.types[mod+name] = n.ID
	case "trait_item":
		x.types[mod+name] = n.ID
		x.supertraits(def, n.ID)
		if body := x.doc.Field(def, "body"); body != nil {
			x.members(body, owner{name: name, qual: mod + name, mod: mod, exported: exported, test: test})
		}
	}
}

// impl reads the methods of one impl block. The block has no node: its
// methods belong to the type it names, trait impl or not.
func (x *extractor) impl(def *gts.Node, attrs []attr, mod string, test bool) {
	_, _, ok := x.header(def)
	body := x.doc.Field(def, "body")
	if !ok || body == nil {
		return
	}
	name := x.typeName(x.doc.Field(def, "type"))
	_, godot := find(attrs, "godot_api")
	o := owner{name: name, qual: mod + name, mod: mod, test: test, godot: godot}
	x.implements(def, o)
	x.members(body, o)
}

// typeName is the type an impl names, as written without its generics, and
// without the reference or the path in front of it: Foo for Foo<T>, &Foo and
// a::Foo.
func (x *extractor) typeName(n *gts.Node) string {
	switch x.doc.Type(n) {
	case "generic_type", "reference_type":
		return x.typeName(x.doc.Field(n, "type"))
	case "scoped_type_identifier":
		return x.doc.Text(x.doc.Field(n, "name"))
	}
	return x.doc.Text(n)
}

// members reads the functions in the body of an impl or trait.
func (x *extractor) members(body *gts.Node, o owner) {
	x.each(body, func(c *gts.Node, attrs []attr) {
		if t := x.doc.Type(c); t != "function_item" && t != "function_signature_item" {
			x.refs(c, scope{source: x.fileID, owner: o.name, mod: o.mod})
			return
		}
		head, name, ok := x.header(c)
		if !ok {
			return
		}
		n := x.symbol(c, head, name, o.qual+"."+name, "method", o.name, o.exported || x.exported(head), o.test || cfgTest(attrs) || isTest(attrs))
		x.owned[len(x.contains)] = o.qual
		x.contains = append(x.contains, extract.RawEdge{
			Source: x.fileID, Relation: model.RelationContains, TargetID: n.ID, File: x.doc.Rel,
		})
		x.refs(x.doc.Field(c, "body"), scope{source: n.ID, owner: o.name, mod: o.mod})
		if o.godot {
			x.bind(c, n, o.name, attrs)
		}
	})
}

// hang moves the contains edge of each method, and the implements edge of
// each trait impl, from the file to its type when the file declares the
// type. It runs at the end: an impl may come before its type.
func (x *extractor) hang() {
	for i, qual := range x.owned {
		if id, ok := x.types[qual]; ok {
			x.contains[i].Source = id
		}
	}
	for i, qual := range x.impls {
		if id, ok := x.types[qual]; ok {
			x.relations[i].Source = id
		}
	}
}

// bind records the Godot registration of a method in a #[godot_api] impl as a
// references edge: #[func] and #[rpc] expose it under its own name or the
// one of rename =, #[signal] declares a signal of that name.
func (x *extractor) bind(def *gts.Node, n model.Node, recv string, attrs []attr) {
	e := extract.RawEdge{
		Source: n.ID, Relation: model.RelationReferences, File: x.doc.Rel,
		Line: int(def.StartPoint().Row) + 1, Receiver: recv, Name: n.Name, Specifier: n.Name,
	}
	// Every attribute counts: #[rpc] in front of #[func(rename = x)] is one
	// binding named x.
	for _, a := range attrs {
		switch a.name {
		case "func", "rpc":
			e.Member = model.Kind(extract.NamespaceGodotMethod)
			if rename := a.value("rename"); rename != "" {
				e.Specifier = rename
			}
		case "signal":
			if e.Member == "" {
				e.Member = model.Kind(extract.NamespaceGodotSignal)
			}
		}
	}
	if e.Member != "" {
		x.bindings = append(x.bindings, e)
	}
}

// bodyKinds are the children that end a definition's header: its body, the
// ";" of one without, or the first token of a macro's rules.
var bodyKinds = map[string]bool{
	"block": true, "declaration_list": true, "field_declaration_list": true,
	"ordered_field_declaration_list": true, "enum_variant_list": true,
	";": true, "{": true, "(": true, "[": true,
}

// header is a definition's header -- every child before the body -- and its
// name. It is false when the parser rebuilt the definition around text it
// rejected: an ERROR at any depth of the header. The extent then claims text
// of another definition, and no node beats a wrong one. An ERROR in the body
// is not the header's, and the extent stays as written.
func (x *extractor) header(def *gts.Node) ([]*gts.Node, string, bool) {
	kids := def.Children()
	end := len(kids)
	for i, c := range kids {
		if bodyKinds[x.doc.Type(c)] {
			end = i
			break
		}
		if treesitter.HoldsError(c) {
			return nil, "", false
		}
	}
	return kids[:end], x.doc.Text(x.doc.Field(def, "name")), true
}

// exported is whether a header carries a visibility: pub, pub(crate), ...
func (x *extractor) exported(head []*gts.Node) bool {
	return slices.ContainsFunc(head, func(c *gts.Node) bool { return x.doc.Type(c) == "visibility_modifier" })
}

// symbol mints the node of one definition and returns it. Its id is the one
// every edge must use: a repeated name -- two trait impls of one method --
// gets an ordinal, and recomputing the id from the name would name a node
// that does not exist.
func (x *extractor) symbol(def *gts.Node, head []*gts.Node, name, qualified string, kind model.Kind, of string, exported, test bool) model.Node {
	n := x.doc.Symbol(treesitter.Sym{
		Outer:     def,
		Head:      head[0].StartByte(),
		HeaderEnd: head[len(head)-1].EndByte(),
		Name:      name,
		Qualified: qualified,
		Kind:      kind,
		Owner:     of,
		Exported:  exported,
	})
	n.Test = test
	x.nodes = append(x.nodes, n)
	return n
}
