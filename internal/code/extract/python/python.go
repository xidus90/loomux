// Package python extracts the nodes and raw edges of one Python file on the
// shared tree-sitter core.
//
// Raw, as in the Go extractor: a call or a base class names its target, and
// resolving the name needs every file of the repository, which is
// internal/code/resolve's job.
//
// Nodes are the file, every class and function at module level, and every
// method directly in the body of such a class; a decorated definition counts
// as the definition it decorates. Module level is the module itself and the
// blocks of an if, try or with in it, however nested: a def there binds its
// name in the module, as `try: from x import f` / `except ImportError: def
// f(): ...` relies on. Nothing else gets a node -- not a def inside a
// function or a loop, not a class inside a class. Such a definition belongs
// to the body around it, and so do its calls: every call belongs to the
// innermost definition that has a node, or to the file.
//
// A definition the parser rebuilt around text it rejected gets no node, and
// nothing inside it counts (header).
//
// The node types and fields read here, as gotreesitter v0.55.0 builds them
// (treesitter.Parser):
//
//	module                 the statements as children
//	decorated_definition   decorator children; field definition
//	function_definition    fields name, parameters, return_type, body; a ":" child closes the header
//	class_definition       fields name, superclasses (an argument_list), body; a ":" child closes the header
//	block                  the statements as children
//	call                   fields function, arguments
//	attribute              fields object, attribute
//	import_statement       field name, once per module: dotted_name or aliased_import
//	aliased_import         fields name (a dotted_name), alias (an identifier)
//	import_from_statement  field module_name (dotted_name or relative_import); field name once per
//	                       name, or a wildcard_import child
//	relative_import        an import_prefix (the dots), then an optional dotted_name
//	dotted_name            identifier and "." children, and a line_continuation where a backslash breaks it
//
// future_import_statement, `from __future__ import x`, is a node type of its
// own and is not read: it names a compiler feature, never a file.
package python

import (
	"slices"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
	pygrammar "github.com/odvcencio/gotreesitter/grammars/python"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// version is this extractor's identity. Bump it by hand whenever the nodes or
// edges below change shape or meaning; the extract cache and the graph's meta
// key on it (see golang.Version for what a forgotten bump costs).
const version = "python/1"

// langName is the language as the graph's meta lists it and as a Result carries it.
const langName = "python"

// Language is the Python extractor as extract.Language describes one.
type Language struct{}

// Name is "python".
func (Language) Name() string { return langName }

// Version is this package's version plus the tree-sitter runtime: a new
// runtime can build other trees from the same source.
func (Language) Version() string { return version + "@" + treesitter.Parser }

// Extensions is ".py" alone. A .pyi stub would repeat every definition of the
// module it describes.
func (Language) Extensions() []string { return []string{".py"} }

// File is this package's File.
func (Language) File(rel, source string) (extract.Result, error) { return File(rel, source) }

// File extracts one Python file. rel is its repo-relative, slash-separated
// path; source is its contents.
func File(rel, source string) (extract.Result, error) {
	return file(pygrammar.Language(), rel, source)
}

// file is File with the grammar handed in: the Python grammar builds a tree
// from any input, and only a missing grammar makes the parse fail.
func file(lang *gts.Language, rel, source string) (extract.Result, error) {
	doc, err := treesitter.Parse(lang, rel, []byte(source))
	if err != nil {
		return extract.Result{}, err
	}
	defer doc.Close()

	x := &extractor{doc: doc, fileID: model.NodeID(rel)}
	x.statements(doc.Root)
	return extract.Result{
		Path:     rel,
		Language: langName,
		Imports:  x.imports,
		// statements has run every Symbol by now, as FileNode requires: its
		// body text is what no symbol covers.
		Nodes:       append([]model.Node{doc.FileNode()}, x.nodes...),
		Edges:       slices.Concat(x.contains, x.importEdges, x.extends, x.calls),
		ParseErrors: doc.ParseErrors(),
	}, nil
}

// extractor gathers one file's nodes and edges in source order.
type extractor struct {
	doc    *treesitter.Doc
	fileID model.NodeID

	nodes       []model.Node
	contains    []extract.RawEdge
	extends     []extract.RawEdge
	calls       []extract.RawEdge
	imports     []extract.Import
	importEdges []extract.RawEdge
}

// statements reads statements at module level: the children of the module,
// or of an if, try or with in it, of one of their clauses, or of a block of
// either. A definition becomes a node; an if, try or with, a clause and a
// block are read the same way; anything else -- a condition, an except
// type, a with item, any other statement -- is walked for the file's calls
// and imports.
//
// No definition reached here or in class lies inside an ERROR node, so
// treesitter.InError need not be asked: an ERROR node is none of the types
// read here, and a definition the parser recovered inside one is never a
// child of them. Its calls are dropped with it (collect).
func (x *extractor) statements(n *gts.Node) {
	for _, stmt := range n.Children() {
		switch def := x.definition(stmt); x.doc.Type(def) {
		case "function_definition", "class_definition":
			x.define(stmt, def)
		case "if_statement", "elif_clause", "else_clause",
			"try_statement", "except_clause", "finally_clause",
			"with_statement", "block":
			x.statements(stmt)
		default:
			x.collect(stmt, scope{source: x.fileID})
		}
	}
}

// define makes the node of a function or class at module level, and the
// nodes of the class's methods.
func (x *extractor) define(outer, def *gts.Node) {
	end, ok := x.header(outer, def)
	switch {
	case !ok:
		// Rebuilt around rejected text (header): no node, and its calls
		// are not walked either -- collect would credit them to the file.
	case x.doc.Type(def) == "function_definition":
		n := x.symbol(outer, def, end, "function", "", x.fileID)
		x.body(outer, def, scope{source: n.ID})
	default:
		x.class(outer, def, end)
	}
}

// definition is what a statement defines: the definition a
// decorated_definition wraps, or the statement itself.
func (x *extractor) definition(stmt *gts.Node) *gts.Node {
	if x.doc.Type(stmt) == "decorated_definition" {
		return x.doc.Field(stmt, "definition")
	}
	return stmt
}

// class is a class at module level whose header ends at end: its node, its
// bases, and a method node for every def directly in its body.
func (x *extractor) class(outer, def *gts.Node, end uint32) {
	c := x.symbol(outer, def, end, "class", "", x.fileID)
	here := scope{source: c.ID}
	x.decorators(outer, here)
	if bases := x.doc.Field(def, "superclasses"); bases != nil {
		for _, b := range bases.Children() {
			x.base(c.ID, b)
		}
		x.collect(bases, here)
	}
	for _, stmt := range x.doc.Field(def, "body").Children() {
		fn := x.definition(stmt)
		if x.doc.Type(fn) != "function_definition" {
			// A statement of the class body, or a class inside the class:
			// its calls belong to the class.
			x.collect(stmt, here)
			continue
		}
		end, ok := x.header(stmt, fn)
		if !ok {
			// Rebuilt around rejected text: no node, and no call of it
			// reaches the class.
			continue
		}
		m := x.symbol(stmt, fn, end, "method", c.Name, c.ID)
		x.body(stmt, fn, scope{source: m.ID, self: c.Name})
	}
}

// base records one entry of a class's superclasses as an extends edge, when
// it is a name or an attribute chain of names. A keyword (metaclass=M), a
// subscript (Generic[T]), a splat or a call names no class this package could
// point at, and neither does the punctuation between the entries.
func (x *extractor) base(class model.NodeID, n *gts.Node) {
	path, ok := x.chain(n)
	if !ok {
		return
	}
	e := extract.RawEdge{Source: class, Relation: model.RelationExtends, Name: path, File: x.doc.Rel}
	if i := strings.LastIndexByte(path, '.'); i >= 0 {
		e.Receiver, e.Name = path[:i], path[i+1:]
	}
	x.extends = append(x.extends, e)
}

// symbol mints the node of one definition, whose header ends at end, and the
// contains edge from parent to it, and returns the node. Its id is the one
// every edge must use: a repeated name gets an ordinal, and recomputing the
// id from the name would name a node that does not exist.
func (x *extractor) symbol(outer, def *gts.Node, end uint32, kind model.Kind, owner string, parent model.NodeID) model.Node {
	name := x.doc.Text(x.doc.Field(def, "name"))
	qualified := name
	if owner != "" {
		qualified = owner + "." + name
	}
	n := x.doc.Symbol(treesitter.Sym{
		Outer: outer,
		// The definition node starts at def, or at async before it; a
		// decorator lies outside it, in outer.
		Head:      def.StartByte(),
		HeaderEnd: end,
		Name:      name,
		Qualified: qualified,
		Kind:      kind,
		Owner:     owner,
		Exported:  exported(name),
	})
	x.nodes = append(x.nodes, n)
	x.contains = append(x.contains, extract.RawEdge{
		Source: parent, Relation: model.RelationContains, TargetID: n.ID, File: x.doc.Rel,
	})
	return n
}

// header is where the signature of def ends, the end of its first direct ":"
// child, and false when the parser rebuilt the definition around text it
// rejected. outer is def, or the decorated_definition around it. A colon in
// an annotation, a default or a lambda lies deeper, inside the parameters or
// the return type, and a comment after the colon is a later child.
//
// Rebuilt means an ERROR anywhere in the header -- in or under the children
// of def ahead of that colon, or of outer ahead of def -- or no such colon at
// all. The parser recovers by swallowing what follows: `def n(self)` without
// its colon and the next `def o(self):` parse as one definition named o with
// n's text in an ERROR ahead of the name; `def n(self` without its ")" takes
// the next def into its parameters; a decorated def without its colon lands
// as an ERROR beside the next def in one decorated_definition. Each time the
// span, signature and body hash would claim lines of another definition, and
// no node beats a wrong one. The ERROR can sit at any depth, so any depth
// counts. An ERROR in the body is not the header's, and a MISSING token is
// no ERROR: the parser invented what the source lacks. Either way the extent
// stays as written.
func (x *extractor) header(outer, def *gts.Node) (uint32, bool) {
	if x.doc.Type(outer) == "decorated_definition" {
		for i := range outer.ChildCount() {
			// def itself is read below, where only its header counts.
			if outer.FieldNameForChild(i, x.doc.Lang) != "definition" && holdsError(outer.Child(i)) {
				return 0, false
			}
		}
	}
	for _, c := range def.Children() {
		if holdsError(c) {
			break
		}
		if x.doc.Type(c) == ":" {
			return c.EndByte(), true
		}
	}
	return 0, false
}

// holdsError reports whether n is an ERROR node or has one below it.
func holdsError(n *gts.Node) bool {
	found := false
	treesitter.Walk(n, func(c *gts.Node) bool {
		found = found || c.IsError()
		return !found
	})
	return found
}

// body collects the calls of a function or method with a node: those in its
// decorators, its defaults and annotations, and its body.
func (x *extractor) body(outer, def *gts.Node, sc scope) {
	x.decorators(outer, sc)
	for _, c := range def.Children() {
		x.collect(c, sc)
	}
}

// decorators collects the calls in a definition's decorators. A decorator
// belongs to the definition -- its span and body hash hold it -- and so does
// a call in it. An undecorated definition is its own outer and has no
// decorator child.
func (x *extractor) decorators(outer *gts.Node, sc scope) {
	for _, c := range outer.Children() {
		if x.doc.Type(c) == "decorator" {
			x.collect(c, sc)
		}
	}
}

// exported is whether a name is public: it does not start with an
// underscore, or it is a dunder like __init__ -- protocol, not private.
// "__" and "____" are underscores and nothing else, no dunders.
func exported(name string) bool {
	return !strings.HasPrefix(name, "_") ||
		(strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__") && len(name) > 4)
}
