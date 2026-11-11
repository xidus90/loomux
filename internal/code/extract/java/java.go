// Package java extracts the nodes and raw edges of one Java file on the
// shared tree-sitter core.
//
// Raw, as in the other extractors: a call or a base type names its target,
// and resolving the name needs every file of the repository, which is
// internal/code/resolve's job.
//
// Nodes are the file, every type -- class, interface, enum, record and
// annotation type, nested ones as Outer.Inner -- the methods and
// constructors of a type, a record's compact constructor named like the
// record, and the methods of a compact source file, kind function. The
// package names no node; Result.Package carries it. Every call belongs to
// the innermost member with a node, else to its type, else to the file. An
// anonymous class, an enum constant's body included, has no node; its calls
// belong to the member or type around it.
//
// A definition the parser rebuilt around text it rejected gets no node, and
// nothing inside it counts (header).
//
// The node types and fields read here, as gotreesitter builds them
// (treesitter.Parser):
//
//	program                          package, imports, types and top-level methods as children
//	package_declaration              the name as an identifier or scoped_identifier child
//	import_declaration               "static" token, the path as an identifier or scoped_identifier
//	                                 child, an asterisk child for ".*"
//	class_declaration, interface_declaration, enum_declaration,
//	record_declaration, annotation_type_declaration
//	                                 a modifiers child, the name as the only identifier child of the
//	                                 header, superclass, super_interfaces or extends_interfaces
//	                                 children, the body as a class_body, interface_body, enum_body
//	                                 or annotation_type_body child
//	enum_body                        enum_constant children, then enum_body_declarations
//	method_declaration, constructor_declaration,
//	compact_constructor_declaration  the same header; a block or constructor_body, or ";"
//	modifiers                        keyword tokens and marker_annotation or annotation children
//	method_invocation                fields object (absent for a bare call), name
//	object_creation_expression       field type
//	scoped_identifier                fields scope, name
//	field_access                     fields object, field
//	scoped_type_identifier           no fields: a type_identifier or scoped_type_identifier, ".",
//	                                 a type_identifier
//	generic_type                     the type as first child, then type_arguments
//
// Java 25 is beyond the grammar in two places. `import module m;` arrives
// as an import holding an ERROR around "module", or, after a static
// on-demand import, as an import of "module" with a missing ";" followed by
// the name as a statement of its own; both become the import of m. A
// statement before super(...) in a constructor is an ERROR inside its body,
// and only the calls in it are lost.
package java

import (
	"slices"

	gts "github.com/odvcencio/gotreesitter"
	jgrammar "github.com/odvcencio/gotreesitter/grammars/java"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// version is this extractor's identity. Bump it by hand whenever the nodes or
// edges below change shape or meaning; the extract cache and the graph's meta
// key on it (see golang.Version for what a forgotten bump costs).
const version = "java/1"

// langName is the language as the graph's meta lists it and as a Result carries it.
const langName = "java"

// Language is the Java extractor as extract.Language describes one.
type Language struct{}

// Name is "java".
func (Language) Name() string { return langName }

// Version is this package's version plus the tree-sitter runtime: a new
// runtime can build other trees from the same source.
func (Language) Version() string { return version + "@" + treesitter.Parser }

// Extensions is ".java" alone.
func (Language) Extensions() []string { return []string{".java"} }

// File is this package's File.
func (Language) File(rel, source string) (extract.Result, error) { return File(rel, source) }

// File extracts one Java file. rel is its repo-relative, slash-separated
// path; source is its contents.
func File(rel, source string) (extract.Result, error) {
	return file(jgrammar.Language(), rel, source)
}

// file is File with the grammar handed in: the Java grammar builds a tree
// from any input, and only a missing grammar makes the parse fail.
func file(lang *gts.Language, rel, source string) (extract.Result, error) {
	doc, err := treesitter.Parse(lang, rel, []byte(source))
	if err != nil {
		return extract.Result{}, err
	}
	defer doc.Close()

	x := &extractor{doc: doc, fileID: model.NodeID(rel)}
	x.vars = x.variables()
	x.decls(doc.Root, nil)
	return extract.Result{
		Path:     rel,
		Language: langName,
		Package:  x.pkg,
		Imports:  x.imports,
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
	vars   map[string]bool // every variable name the file declares

	pkg      string
	nodes    []model.Node
	contains []extract.RawEdge
	extends  []extract.RawEdge
	calls    []extract.RawEdge

	imports     []extract.Import
	importEdges []extract.RawEdge
}

// typ is a type with a node, as its members see it.
type typ struct {
	id        model.NodeID
	qualified string // "Outer.Inner"
	iface     bool   // an interface or annotation type: its members are public by default
}

// typeKinds are the declarations that make a type node.
var typeKinds = map[string]bool{
	"class_declaration": true, "interface_declaration": true, "enum_declaration": true,
	"record_declaration": true, "annotation_type_declaration": true,
}

// memberKinds are the declarations that make a member node.
var memberKinds = map[string]bool{
	"method_declaration": true, "constructor_declaration": true, "compact_constructor_declaration": true,
}

// bodies are the containers of a type's members.
var bodies = map[string]bool{
	"class_body": true, "interface_body": true, "enum_body": true,
	"enum_body_declarations": true, "annotation_type_body": true,
}

// decls reads the declarations among n's children inside the type owner,
// or at file level when owner is nil. A type's body is read the same way;
// anything else is walked for the calls of the innermost owner.
func (x *extractor) decls(n *gts.Node, owner *typ) {
	for _, c := range n.Children() {
		switch t := x.doc.Type(c); {
		case t == "ERROR":
			// Text the grammar rejected: a definition in it would be a
			// guess, and a call in it would be credited to the scope around.
		case t == "package_declaration":
			x.pkg = x.dotted(c)
		case t == "import_declaration":
			x.importDecl(c)
		case typeKinds[t]:
			x.typeDecl(c, owner)
		case memberKinds[t]:
			x.member(c, owner)
		case bodies[t]:
			x.decls(c, owner)
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

// typeDecl makes the node of one type, nested in outer or at file level
// when outer is nil, and reads its body.
func (x *extractor) typeDecl(def *gts.Node, outer *typ) {
	head, body, ok := x.header(def)
	if !ok {
		// Rebuilt around rejected text: no node, and nothing inside it
		// reaches the scope around.
		return
	}
	name := x.lastIdentifier(head)
	qualified, owner, parent, dflt := name, "", x.fileID, false
	if outer != nil {
		// A member of an interface is public unless it says otherwise.
		qualified, owner, parent, dflt = outer.qualified+"."+name, outer.qualified, outer.id, outer.iface
	}
	kind := model.Kind("class")
	if t := x.doc.Type(def); t == "interface_declaration" || t == "annotation_type_declaration" {
		kind = "interface"
	}
	n := x.symbol(def, head, name, qualified, kind, owner, x.exported(head, dflt), parent)
	for _, c := range head {
		switch x.doc.Type(c) {
		case "superclass", "super_interfaces", "extends_interfaces":
			x.bases(n.ID, c)
		}
	}
	x.decls(body, &typ{id: n.ID, qualified: qualified, iface: kind == "interface"})
}

// member makes the node of one method or constructor of t, or of a method
// of a compact source file when t is nil.
func (x *extractor) member(def *gts.Node, t *typ) {
	head, _, ok := x.header(def)
	if !ok {
		// Rebuilt around rejected text: no node, and no call of it
		// reaches the type.
		return
	}
	name := x.lastIdentifier(head)
	var n model.Node
	if t == nil {
		n = x.symbol(def, head, name, name, "function", "", x.exported(head, false), x.fileID)
	} else {
		n = x.symbol(def, head, name, t.qualified+"."+name, "method", t.qualified, x.exported(head, t.iface), t.id)
	}
	x.collect(def, n.ID)
}

// bodyKinds are the children that end a declaration's header: its body, or
// the ";" of a declaration without one.
var bodyKinds = map[string]bool{
	"class_body": true, "interface_body": true, "enum_body": true, "annotation_type_body": true,
	"block": true, "constructor_body": true, ";": true,
}

// header splits a declaration into its header -- every child before the
// body -- and the body, nil for a declaration that ends in ";". It is false
// when the parser rebuilt the declaration around text it rejected: an ERROR
// at any depth of the header. The extent then claims text of another
// definition, and no node beats a wrong one. An ERROR in the body is not the
// header's, and the extent stays as written.
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
// A type is a type_identifier or a type node of its own, and parameters and
// type parameters are lists of their own.
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
// package: public or protected. Without an access modifier, dflt is the
// Java default of the place.
func (x *extractor) exported(head []*gts.Node, dflt bool) bool {
	for _, c := range head {
		if x.doc.Type(c) != "modifiers" {
			continue
		}
		for _, m := range c.Children() {
			switch x.doc.Type(m) {
			case "public", "protected":
				return true
			case "private":
				return false
			}
		}
	}
	return dflt
}

// annotations are the modifiers that belong to a definition's span but not
// to its signature.
var annotations = map[string]bool{"marker_annotation": true, "annotation": true}

// symbol mints the node of one declaration and the contains edge from parent
// to it, and returns the node. Its id is the one every edge must use: a
// repeated name -- an overload -- gets an ordinal, and recomputing the id
// from the name would name a node that does not exist.
func (x *extractor) symbol(def *gts.Node, head []*gts.Node, name, qualified string, kind model.Kind, owner string, exported bool, parent model.NodeID) model.Node {
	// The signature starts after the annotations, which sit among the modifiers.
	var parts []*gts.Node
	for _, c := range head {
		if x.doc.Type(c) == "modifiers" {
			parts = append(parts, c.Children()...)
		} else {
			parts = append(parts, c)
		}
	}
	start := parts[slices.IndexFunc(parts, func(p *gts.Node) bool { return !annotations[x.doc.Type(p)] })]
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
