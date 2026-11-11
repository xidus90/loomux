// Package php extracts the nodes and raw edges of one PHP file on the
// shared tree-sitter core.
//
// Raw, as in the other extractors: a call or a base type names its target,
// and resolving the name needs every file of the repository, which is
// internal/code/resolve's job.
//
// Nodes are the file, every class, interface, trait and enum, the methods
// of each, and the free functions. PHP declares a function or a class in
// its namespace wherever the declaration runs -- inside an if, a function
// or a method -- so each gets its node at the file whatever code is around
// it. Namespaces name no node and stay out of ids and owners;
// Result.Enclosing carries the namespace of each type and function
// instead. Every call belongs to the innermost method or function with a
// node, else to its type, else to the file. An anonymous class has no
// node; its calls belong to the code around it.
//
// A name keeps its qualification as written: a raw edge splits it at the
// last backslash, and the receiver is everything before it, so \A\B names
// B in "\A", \B names B in "\", and B names B with no receiver. Only the
// path of a `use` import drops the leading backslash, since PHP reads that
// name fully qualified either way.
//
// The grammar reads the text outside <?php ... ?> as text nodes and
// nothing else: HTML in a file names nothing.
//
// A definition the parser rebuilt around text it rejected gets no node, and
// nothing inside it counts (header).
//
// The node types and fields read here, as gotreesitter builds them
// (treesitter.Parser):
//
//	program                         statements, text and text_interpolation as children
//	namespace_definition            field name (absent for the global block), field body
//	                                (absent for `namespace A;`, which holds for the siblings after it)
//	namespace_use_declaration       field type ("function", "const") for the whole group, a
//	                                namespace_name prefix, field body namespace_use_group, or
//	                                namespace_use_clause children
//	namespace_use_clause            field type, a name or qualified_name child, field alias
//	class_declaration, interface_declaration, trait_declaration,
//	enum_declaration                field attributes, field name, base_clause and
//	                                class_interface_clause children, field body
//	method_declaration, function_definition
//	                                field attributes, visibility_modifier children, field name,
//	                                field body, or ";" for an abstract method
//	use_declaration                 a trait use: name or qualified_name children
//	function_call_expression        field function
//	scoped_call_expression          fields scope, name
//	member_call_expression,
//	nullsafe_member_call_expression fields object, name
//	object_creation_expression      the class as a name or qualified_name child, or an anonymous_class
//	require_expression, require_once_expression,
//	include_expression, include_once_expression
//	                                the operand as the last child
//
// Two forms of PHP 8.5 are beyond the grammar: clone($o, [...]) and the
// (void) cast. Each leaves an ERROR inside the body around it, and only a
// call inside that ERROR is lost.
package php

import (
	"slices"

	gts "github.com/odvcencio/gotreesitter"
	pgrammar "github.com/odvcencio/gotreesitter/grammars/php"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// version is this extractor's identity. Bump it by hand whenever the nodes or
// edges below change shape or meaning; the extract cache and the graph's meta
// key on it (see golang.Version for what a forgotten bump costs).
const version = "php/1"

// langName is the language as the graph's meta lists it and as a Result carries it.
const langName = "php"

// Language is the PHP extractor as extract.Language describes one.
type Language struct{}

// Name is "php".
func (Language) Name() string { return langName }

// Version is this package's version plus the tree-sitter runtime: a new
// runtime can build other trees from the same source.
func (Language) Version() string { return version + "@" + treesitter.Parser }

// Extensions is ".php" alone.
func (Language) Extensions() []string { return []string{".php"} }

// File is this package's File.
func (Language) File(rel, source string) (extract.Result, error) { return File(rel, source) }

// File extracts one PHP file. rel is its repo-relative, slash-separated
// path; source is its contents.
func File(rel, source string) (extract.Result, error) {
	return file(pgrammar.Language(), rel, source)
}

// file is File with the grammar handed in: the PHP grammar builds a tree
// from any input, and only a missing grammar makes the parse fail.
func file(lang *gts.Language, rel, source string) (extract.Result, error) {
	doc, err := treesitter.Parse(lang, rel, []byte(source))
	if err != nil {
		return extract.Result{}, err
	}
	defer doc.Close()

	x := &extractor{doc: doc, fileID: model.NodeID(rel)}
	x.decls(doc.Root, "", nil, x.fileID)
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
	}, nil
}

// extractor gathers one file's nodes and edges in source order.
type extractor struct {
	doc    *treesitter.Doc
	fileID model.NodeID

	pkg       string                  // the namespace of the first declaration
	enclosing map[model.NodeID]string // nil until the first declaration
	nodes     []model.Node
	contains  []extract.RawEdge
	extends   []extract.RawEdge
	calls     []extract.RawEdge

	imports     []extract.Import
	importEdges []extract.RawEdge
}

// typ is a type with a node, as its members see it.
type typ struct {
	id   model.NodeID
	name string
}

// typeKinds are the declarations that make a type node.
var typeKinds = map[string]bool{
	"class_declaration": true, "interface_declaration": true, "trait_declaration": true, "enum_declaration": true,
}

// decls reads n's children in namespace ns. owner is the type n is the
// body of, nil anywhere else; source is the node a call found here belongs
// to. A namespace statement holds for the siblings after it.
func (x *extractor) decls(n *gts.Node, ns string, owner *typ, source model.NodeID) {
	for _, c := range n.Children() {
		switch t := x.doc.Type(c); {
		case t == "ERROR":
			// Text the grammar rejected: a definition in it would be a
			// guess, and a call in it would be credited to the scope around.
		case t == "namespace_definition":
			name := x.doc.Text(x.doc.Field(c, "name"))
			if body := x.doc.Field(c, "body"); body != nil {
				x.decls(body, name, nil, x.fileID)
			} else {
				ns = name
			}
		case t == "namespace_use_declaration":
			x.use(c)
		case typeKinds[t]:
			x.typeDecl(c, ns)
		case t == "function_definition":
			if fn, ok := x.declare(c, "function", nil, ns); ok {
				x.decls(c, ns, nil, fn.ID)
			}
		case owner != nil && t == "method_declaration":
			if m, ok := x.declare(c, "method", owner, ns); ok {
				x.decls(c, ns, nil, m.ID)
			}
		case owner != nil && t == "use_declaration":
			x.bases(owner.id, c)
		default:
			x.expression(c, source)
			x.decls(c, ns, nil, source)
		}
	}
}

// typeDecl makes the node of one type in namespace ns and reads its bases
// and its body.
func (x *extractor) typeDecl(def *gts.Node, ns string) {
	kind := model.Kind("class")
	if x.doc.Type(def) == "interface_declaration" {
		kind = "interface"
	}
	n, ok := x.declare(def, kind, nil, ns)
	if !ok {
		// Rebuilt around rejected text: nothing inside it reaches the
		// scope around.
		return
	}
	for _, c := range def.Children() {
		switch x.doc.Type(c) {
		case "base_clause", "class_interface_clause":
			x.bases(n.ID, c)
		}
	}
	if body := x.doc.Field(def, "body"); body != nil {
		x.decls(body, ns, &typ{id: n.ID, name: n.Name}, n.ID)
	}
}

// declare makes the node of a type or function in namespace ns, or of a
// method of owner, and the contains edge to it. It is false, and makes
// nothing, when the parser rebuilt the declaration around rejected text.
func (x *extractor) declare(def *gts.Node, kind model.Kind, owner *typ, ns string) (model.Node, bool) {
	head, ok := x.header(def)
	if !ok {
		return model.Node{}, false
	}
	name := x.doc.Text(x.doc.Field(def, "name"))
	qualified, ownerName, parent, exported := name, "", x.fileID, true
	if owner != nil {
		qualified, ownerName, parent, exported = owner.name+"."+name, owner.name, owner.id, x.public(head)
	}
	// The signature starts after the attributes, which belong to the span.
	start := head[0]
	if x.doc.Type(start) == "attribute_list" && len(head) > 1 {
		start = head[1]
	}
	n := x.doc.Symbol(treesitter.Sym{
		Outer:     def,
		Head:      start.StartByte(),
		HeaderEnd: head[len(head)-1].EndByte(),
		Name:      name,
		Qualified: qualified,
		Kind:      kind,
		Owner:     ownerName,
		Exported:  exported,
	})
	x.nodes = append(x.nodes, n)
	x.contains = append(x.contains, extract.RawEdge{
		Source: parent, Relation: model.RelationContains, TargetID: n.ID, File: x.doc.Rel,
	})
	if owner == nil {
		if x.enclosing == nil {
			// The first declaration: its namespace is the file's package.
			x.pkg, x.enclosing = ns, map[model.NodeID]string{}
		}
		x.enclosing[n.ID] = ns
	}
	return n, true
}

// bodyKinds are the children that end a declaration's header: its body, or
// the ";" of an abstract method.
var bodyKinds = map[string]bool{
	"declaration_list": true, "enum_declaration_list": true, "compound_statement": true, ";": true,
}

// header is every child of a declaration before its body. It is false when
// the parser rebuilt the declaration around text it rejected: an ERROR at
// any depth of the header. The extent then claims text of another
// definition, and no node beats a wrong one. An ERROR in the body is not
// the header's, and the extent stays as written.
func (x *extractor) header(def *gts.Node) ([]*gts.Node, bool) {
	head := def.Children()
	if i := slices.IndexFunc(head, func(c *gts.Node) bool { return bodyKinds[x.doc.Type(c)] }); i >= 0 {
		head = head[:i]
	}
	return head, !slices.ContainsFunc(head, treesitter.HoldsError)
}

// public is whether a method's header leaves it public: PHP's default,
// unless private or protected says otherwise.
func (x *extractor) public(head []*gts.Node) bool {
	for _, c := range head {
		if x.doc.Type(c) == "visibility_modifier" {
			if v := x.doc.Text(c); v == "private" || v == "protected" {
				return false
			}
		}
	}
	return true
}
