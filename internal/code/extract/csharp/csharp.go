// Package csharp extracts the nodes and raw edges of one C# file on the
// shared tree-sitter core.
//
// Raw, as in the Go and Python extractors: a call or a base type names its
// target, and resolving the name needs every file of the repository, which
// is internal/code/resolve's job.
//
// Nodes are the file, every type -- class, struct, record, interface, enum,
// nested ones as Outer.Inner -- and the members of a type: methods,
// constructors, operators, properties and indexers with an accessor body or
// an expression body, and delegates marked [Signal]. A member of a C# 14
// extension block belongs to the static class around it. Namespaces name no
// node and stay out of ids and owners; Result.Enclosing carries each type's
// namespace instead. Every call belongs to the innermost member with a node,
// else to its type, else to the file.
//
// A definition the parser rebuilt around text it rejected gets no node, and
// nothing inside it counts (header).
//
// The node types and fields read here, as gotreesitter builds them
// (treesitter.Parser):
//
//	compilation_unit                   usings, namespaces, types and global statements as children
//	using_directive                    "global" and "static" tokens; field name (the alias only); the path
//	                                   as an identifier, qualified_name or generic_name child
//	file_scoped_namespace_declaration  field name; the declarations after it are its siblings
//	namespace_declaration              field name; a declaration_list child
//	class_declaration, struct_declaration, record_declaration,
//	interface_declaration, enum_declaration
//	                                   attribute_list and modifier children, the name as the last identifier
//	                                   of the header, a base_list child, a declaration_list or
//	                                   enum_member_declaration_list body, or ";"
//	method_declaration, constructor_declaration, delegate_declaration,
//	property_declaration               the same header; body block, arrow_expression_clause or
//	                                   accessor_list (accessor_declaration, field body), or ";"
//	operator_declaration               field operator, the operator token
//	conversion_operator_declaration    field type, the type converted to
//	indexer_declaration                as a property, named this
//	extension_declaration              an extension_body child holding members
//	preproc_if, preproc_elif, preproc_else, preproc_region, ...
//	                                   declarations as children
//	attribute_list                     attribute children, each with field name
//	invocation_expression              field function
//	object_creation_expression         field type
//	member_access_expression           fields expression, name
//	qualified_name                     fields qualifier, name
//	generic_name                       the identifier as first child, then a type_argument_list
//
// Field names are read only where a missing one costs a single name: a
// recovered tree can drop the fields of a type it rebuilt and keep its
// children, so the header and the name are found by child type.
package csharp

import (
	"path"
	"slices"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
	csgrammar "github.com/odvcencio/gotreesitter/grammars/c_sharp"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// version is this extractor's identity. Bump it by hand whenever the nodes or
// edges below change shape or meaning; the extract cache and the graph's meta
// key on it (see golang.Version for what a forgotten bump costs).
const version = "csharp/1"

// langName is the language as the graph's meta lists it and as a Result carries it.
const langName = "csharp"

// Language is the C# extractor as extract.Language describes one.
type Language struct{}

// Name is "csharp".
func (Language) Name() string { return langName }

// Version is this package's version plus the tree-sitter runtime: a new
// runtime can build other trees from the same source.
func (Language) Version() string { return version + "@" + treesitter.Parser }

// Extensions is ".cs" alone.
func (Language) Extensions() []string { return []string{".cs"} }

// File is this package's File.
func (Language) File(rel, source string) (extract.Result, error) { return File(rel, source) }

// File extracts one C# file. rel is its repo-relative, slash-separated path;
// source is its contents.
func File(rel, source string) (extract.Result, error) {
	return file(csgrammar.Language(), rel, source)
}

// file is File with the grammar handed in: the C# grammar builds a tree from
// any input, and only a missing grammar makes the parse fail.
func file(lang *gts.Language, rel, source string) (extract.Result, error) {
	doc, err := treesitter.Parse(lang, rel, []byte(source))
	if err != nil {
		return extract.Result{}, err
	}
	defer doc.Close()

	x := &extractor{
		doc: doc, fileID: model.NodeID(rel),
		// Godot binds a C# script to the class named like its file.
		script:  strings.TrimSuffix(path.Base(rel), ".cs"),
		offered: map[extract.Export]bool{},
	}
	x.decls(doc.Root, "", nil)
	return extract.Result{
		Path:      rel,
		Language:  langName,
		Package:   x.pkg,
		Enclosing: x.enclosing,
		Imports:   x.imports,
		// decls has run every Symbol by now, as FileNode requires: its body
		// text is what no symbol covers.
		Nodes:       append([]model.Node{doc.FileNode()}, x.nodes...),
		Edges:       slices.Concat(x.contains, x.importEdges, x.extends, x.calls),
		ParseErrors: doc.ParseErrors(),
		Exports:     x.exports,
	}, nil
}

// extractor gathers one file's nodes and edges in source order.
type extractor struct {
	doc    *treesitter.Doc
	fileID model.NodeID
	script string // the name the file's script class must have

	pkg       string                  // the namespace of the first type
	enclosing map[model.NodeID]string // nil until the first type
	nodes     []model.Node
	contains  []extract.RawEdge
	extends   []extract.RawEdge
	calls     []extract.RawEdge

	imports     []extract.Import
	importEdges []extract.RawEdge

	exports []extract.Export
	offered map[extract.Export]bool // offers made, keyed without their target
}

// typ is a type with a node, as its members see it.
type typ struct {
	id        model.NodeID
	qualified string // "Outer.Inner"
	iface     bool   // an interface: its members are public by default
	script    bool   // the file's script class: its methods and signals are offered to Godot
}

// typeKinds are the declarations that make a type node.
var typeKinds = map[string]bool{
	"class_declaration": true, "struct_declaration": true, "record_declaration": true,
	"interface_declaration": true, "enum_declaration": true,
}

// memberKinds are the declarations that may make a member node.
var memberKinds = map[string]bool{
	"method_declaration": true, "constructor_declaration": true, "operator_declaration": true,
	"conversion_operator_declaration": true,
	"property_declaration":            true, "indexer_declaration": true, "delegate_declaration": true,
}

// decls reads the declarations among n's children in namespace ns, inside
// the type owner or, when owner is nil, at namespace level. A container
// with declarations of its own -- a namespace block, a declaration list,
// an extension block, a preprocessor branch -- is read the same way;
// anything else is walked for the calls of the innermost owner.
func (x *extractor) decls(n *gts.Node, ns string, owner *typ) {
	for _, c := range n.Children() {
		switch t := x.doc.Type(c); {
		case t == "ERROR":
			// Text the grammar rejected: a definition in it would be a
			// guess, and a call in it would be credited to the scope around.
		case t == "using_directive":
			x.using(c)
		case t == "file_scoped_namespace_declaration":
			ns = join(ns, x.dotted(x.doc.Field(c, "name")))
		case t == "namespace_declaration":
			x.decls(c, join(ns, x.dotted(x.doc.Field(c, "name"))), nil)
		case typeKinds[t]:
			x.typeDecl(c, ns, owner)
		case memberKinds[t] && owner != nil:
			x.member(c, owner)
		case t == "declaration_list", t == "extension_declaration", t == "extension_body",
			strings.HasPrefix(t, "preproc_"):
			x.decls(c, ns, owner)
		default:
			x.collect(c, x.source(owner))
		}
	}
}

// source is the node a call directly inside owner belongs to.
func (x *extractor) source(owner *typ) model.NodeID {
	if owner == nil {
		return x.fileID
	}
	return owner.id
}

// typeDecl makes the node of one type in namespace ns, nested in outer or
// at namespace level when outer is nil, and reads its body.
func (x *extractor) typeDecl(def *gts.Node, ns string, outer *typ) {
	head, body, ok := x.header(def)
	if !ok {
		// Rebuilt around rejected text: no node, and nothing inside it
		// reaches the scope around.
		return
	}
	name := x.lastIdentifier(head)
	qualified, owner, parent, dflt := name, "", x.fileID, true
	if outer != nil {
		// A nested type is a member, private unless it says otherwise.
		qualified, owner, parent, dflt = outer.qualified+"."+name, outer.qualified, outer.id, outer.iface
	}
	kind := model.Kind("class")
	if x.doc.Type(def) == "interface_declaration" {
		kind = "interface"
	}
	n := x.symbol(def, head, name, qualified, kind, owner, x.exported(head, dflt), parent)
	if x.enclosing == nil {
		// The first type: its namespace is the file's package.
		x.pkg, x.enclosing = ns, map[model.NodeID]string{}
	}
	x.enclosing[n.ID] = ns
	if x.attributed(head, "GlobalClass") {
		x.offer(extract.NamespaceGodotClass, name, n.ID)
	}
	derived := false
	for _, c := range head {
		if x.doc.Type(c) == "base_list" {
			x.bases(n.ID, c)
			derived = true
		}
		x.collect(c, n.ID)
	}
	// Godot binds a script to the class named like its file, and every such
	// class derives from a Godot type; one without a base is plain C#.
	x.decls(body, ns, &typ{
		id: n.ID, qualified: qualified, iface: kind == "interface",
		script: outer == nil && x.doc.Type(def) == "class_declaration" && name == x.script && derived,
	})
}

// member makes the node of one member of t. A delegate without [Signal]
// and a property or indexer whose accessors have no body make none; their
// calls belong to t.
func (x *extractor) member(def *gts.Node, t *typ) {
	head, body, ok := x.header(def)
	if !ok {
		// Rebuilt around rejected text: no node, and no call of it
		// reaches the type.
		return
	}
	kind, name := model.Kind("method"), x.lastIdentifier(head)
	switch x.doc.Type(def) {
	case "operator_declaration":
		name = "operator" + x.doc.Text(x.doc.Field(def, "operator"))
	case "conversion_operator_declaration":
		// Named by its target type, as C# itself names it: "operator int".
		name = "operator " + x.doc.Text(x.doc.Field(def, "type"))
	case "indexer_declaration":
		name = "this"
	}
	switch x.doc.Type(def) {
	case "delegate_declaration":
		if !x.attributed(head, "Signal") {
			x.collect(def, t.id)
			return
		}
		kind = "signal"
	case "property_declaration", "indexer_declaration":
		if !x.bodied(body) {
			x.collect(def, t.id)
			return
		}
	}
	n := x.symbol(def, head, name, t.qualified+"."+name, kind, t.qualified, x.exported(head, t.iface), t.id)
	x.collect(def, n.ID)
	if !t.script {
		return
	}
	if x.doc.Type(def) == "method_declaration" {
		x.offer(extract.NamespaceGodotMethod, name, n.ID)
	}
	// Godot finds a signal by its delegate's name without the suffix, and
	// a delegate without the suffix not at all.
	if signal, ok := strings.CutSuffix(name, "EventHandler"); kind == "signal" && ok && signal != "" {
		x.offer(extract.NamespaceGodotSignal, signal, n.ID)
	}
}

// bodied is whether a property's or indexer's body holds code: an
// expression body, or an accessor with a body. `{ get; set; }` holds none.
func (x *extractor) bodied(body *gts.Node) bool {
	if x.doc.Type(body) == "arrow_expression_clause" {
		return true
	}
	for _, acc := range body.Children() {
		if x.doc.Field(acc, "body") != nil {
			return true
		}
	}
	return false
}

// bodyKinds are the children that end a declaration's header: its body, or
// the ";" of a declaration without one.
var bodyKinds = map[string]bool{
	"declaration_list": true, "enum_member_declaration_list": true, "block": true,
	"arrow_expression_clause": true, "accessor_list": true, ";": true,
}

// header splits a declaration into its header -- every child before the
// body -- and the body, nil for a declaration that ends in ";" or has none.
// It is false when the parser rebuilt the declaration around text it
// rejected: an ERROR at any depth of the header. The extent then claims
// text of another definition, and no node beats a wrong one. An ERROR in the
// body is not the header's, and the extent stays as written.
func (x *extractor) header(def *gts.Node) ([]*gts.Node, *gts.Node, bool) {
	kids := def.Children()
	end, body := len(kids), (*gts.Node)(nil)
	for i, c := range kids {
		if t := x.doc.Type(c); bodyKinds[t] {
			end = i
			if t != ";" {
				body = c
			}
			break
		}
		if treesitter.HoldsError(c) {
			return nil, nil, false
		}
	}
	return kids[:end], body, true
}

// lastIdentifier is the name a header declares: its last identifier child.
// A return type, a modifier or a base list comes before it or is no
// identifier; type parameters and parameters are lists of their own.
func (x *extractor) lastIdentifier(head []*gts.Node) string {
	name := ""
	for _, c := range head {
		if x.doc.Type(c) == "identifier" {
			name = x.doc.Text(c)
		}
	}
	return name
}

// exported is whether a header's modifiers make it visible outside its
// assembly's own code: public or internal, protected internal included.
// Without an access modifier, dflt is the C# default of the place.
func (x *extractor) exported(head []*gts.Node, dflt bool) bool {
	explicit := false
	for _, c := range head {
		if x.doc.Type(c) != "modifier" {
			continue
		}
		switch x.doc.Text(c) {
		case "public", "internal":
			return true
		case "private", "protected":
			explicit = true
		}
	}
	return dflt && !explicit
}

// symbol mints the node of one declaration and the contains edge from parent
// to it, and returns the node. Its id is the one every edge must use: a
// repeated name -- an overload -- gets an ordinal, and recomputing the id
// from the name would name a node that does not exist.
func (x *extractor) symbol(def *gts.Node, head []*gts.Node, name, qualified string, kind model.Kind, owner string, exported bool, parent model.NodeID) model.Node {
	// The signature starts after the attributes, which belong to the span.
	start := head[0]
	for _, c := range head {
		if x.doc.Type(c) != "attribute_list" {
			start = c
			break
		}
	}
	n := x.doc.Symbol(treesitter.Sym{
		Outer:     def,
		Head:      start.StartByte(),
		HeaderEnd: head[len(head)-1].EndByte(),
		Name:      name,
		Qualified: qualified,
		Kind:      kind,
		Owner:     owner,
		Exported:  exported,
	})
	x.nodes = append(x.nodes, n)
	x.contains = append(x.contains, extract.RawEdge{
		Source: parent, Relation: model.RelationContains, TargetID: n.ID, File: x.doc.Rel,
	})
	return n
}

// join is namespace inner inside outer, "" being the global one.
func join(outer, inner string) string {
	if outer == "" {
		return inner
	}
	return outer + "." + inner
}
