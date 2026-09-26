package python

import (
	gts "github.com/odvcencio/gotreesitter"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// scope is what a call at one place in the file belongs to.
type scope struct {
	source model.NodeID // the innermost definition with a node, or the file
	self   string       // the class self and cls stand for, inside one of its methods; else ""
}

// collect walks a subtree that holds no definition with a node and records
// its calls for sc and its imports for the file.
//
// An ERROR node is skipped whole, and so is a definition the parser rebuilt
// around one (header), decorators included. Both are text the grammar
// rejected, and a call found there sits in a definition that got no node;
// handing it to the scope around would invent a caller.
func (x *extractor) collect(n *gts.Node, sc scope) {
	treesitter.Walk(n, func(c *gts.Node) bool {
		if c.IsError() {
			return false
		}
		switch x.doc.Type(c) {
		case "call":
			x.call(c, sc)
		case "import_statement":
			x.importStatement(c)
		case "import_from_statement":
			x.importFrom(c)
		case "decorated_definition", "function_definition", "class_definition":
			return x.nested(c, sc)
		}
		return true
	})
}

// nested decides how collect goes on at a definition without a node, nested
// in a function or a class or fallen through from statements and class: not
// at all when the parser rebuilt it (header), and into its children for a
// function. For a class, its decorators run where the class stands and keep
// sc; its own children get sc with self forgotten -- their calls stay with
// sc's source, but in its methods self is that class, not the one sc knows.
func (x *extractor) nested(stmt *gts.Node, sc scope) bool {
	def := x.definition(stmt)
	if _, ok := x.header(stmt, def); !ok {
		return false
	}
	if x.doc.Type(def) != "class_definition" {
		return true
	}
	x.decorators(stmt, sc)
	inner := scope{source: sc.source}
	for _, k := range def.Children() {
		x.collect(k, inner)
	}
	return false
}

// call records one call site. Three shapes reach the resolver:
//
//   - foo()                 -- Name alone
//   - self.foo(), cls.foo() -- Name and Owner, the class of the method the
//     call is in
//   - x.foo(), a.b.foo()    -- Name and Receiver, a chain of plain names the
//     resolver may find to be a module or a class
//
// Anything else has no receiver this package can name -- super().foo(),
// f()(), x[0].foo() -- and no edge; a call inside it, super() or f(), is a
// call of its own.
func (x *extractor) call(c *gts.Node, sc scope) {
	fn := x.doc.Field(c, "function")
	e := extract.RawEdge{Source: sc.source, Relation: model.RelationCalls, File: x.doc.Rel}
	switch x.doc.Type(fn) {
	case "identifier":
		e.Name = x.doc.Text(fn)
	case "attribute":
		e.Name = x.doc.Text(x.doc.Field(fn, "attribute"))
		recv, ok := x.chain(x.doc.Field(fn, "object"))
		switch {
		case !ok:
			return
		case sc.self != "" && (recv == "self" || recv == "cls"):
			e.Owner = sc.self
		default:
			e.Receiver = recv
		}
	default:
		return
	}
	x.calls = append(x.calls, e)
}

// chain is an attribute chain of plain names as written, "a.b"; false when
// anything else stands in it -- a call, a subscript, a literal.
func (x *extractor) chain(n *gts.Node) (string, bool) {
	switch x.doc.Type(n) {
	case "identifier":
		return x.doc.Text(n), true
	case "attribute":
		head, ok := x.chain(x.doc.Field(n, "object"))
		return head + "." + x.doc.Text(x.doc.Field(n, "attribute")), ok
	}
	return "", false
}
