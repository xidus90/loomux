package java

import (
	"slices"
	"strings"

	gts "github.com/odvcencio/gotreesitter"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// importDecl records one import declaration as an import and an imports
// edge. Its path is the dotted name as written, ".*" included. Import has no
// field for `import static`, so its alias marks it, as it marks C#'s
// `using static`, and `import module` is marked the same way with "module";
// no alias of a Java import can be either keyword. An import holding any
// other rejected text is left out.
func (x *extractor) importDecl(n *gts.Node) {
	var imp extract.Import
	star := false
	for _, c := range n.Children() {
		switch x.doc.Type(c) {
		case "static":
			imp.Alias = "static"
		case "asterisk":
			star = true
		case "ERROR":
			if x.doc.Text(c) != "module" {
				return
			}
			imp.Alias = "module"
		case "identifier", "scoped_identifier":
			imp.Path, _ = x.chain(c)
		}
	}
	if imp.Path == "module" && imp.Alias == "" {
		// No import of a type can name "module" alone: Java imports nothing
		// from the unnamed package. The grammar ended a module import here
		// and reads its name as the next statement.
		imp.Alias = "module"
		imp.Path, _ = x.chain(n.NextSibling().Child(0))
		if imp.Path == "" {
			return
		}
	}
	if star {
		imp.Path += ".*"
	}
	x.imports = append(x.imports, imp)
	x.importEdges = append(x.importEdges, extract.RawEdge{
		Source: x.fileID, Relation: model.RelationImports, Specifier: imp.Path, File: x.doc.Rel, Line: line(n),
	})
}

// bases records every type a superclass, super_interfaces or
// extends_interfaces clause names as an extends edge; whether a base is a
// class or an interface only the resolver can tell. Type arguments are
// dropped: B<T> names B.
func (x *extractor) bases(id model.NodeID, clause *gts.Node) {
	for _, c := range clause.Children() {
		if x.doc.Type(c) == "type_list" {
			x.bases(id, c)
			continue
		}
		if path, ok := x.chain(c); ok {
			x.extends = append(x.extends, x.edge(id, model.RelationExtends, path, c))
		}
	}
}

// collect walks a subtree that holds no declaration with a node and records
// its calls for source. An ERROR node is skipped whole: a call found there
// sits in text the grammar rejected.
func (x *extractor) collect(n *gts.Node, source model.NodeID) {
	treesitter.Walk(n, func(c *gts.Node) bool {
		switch x.doc.Type(c) {
		case "ERROR":
			return false
		case "method_invocation":
			x.invocation(c, source)
		case "object_creation_expression":
			if path, ok := x.chain(x.doc.Field(c, "type")); ok {
				e := x.edge(source, model.RelationCalls, path, c)
				e.Member = "new"
				x.calls = append(x.calls, e)
			}
		}
		return true
	})
}

// invocation records a method call as written: m(), this.m(), super.m(),
// X.m(), a.b.X.m(), obj.m(). A call on a receiver the file declares as a
// variable -- a field, parameter or local of any member, an enum constant
// -- or on a field of this or super is marked ".": its receiver names an
// object, not a type. Java writes both alike, and a variable obscures a
// type of the same name. A receiver that is no chain of names -- get().m()
// -- names nothing to resolve.
func (x *extractor) invocation(n *gts.Node, source model.NodeID) {
	path := x.doc.Text(x.doc.Field(n, "name"))
	recv := ""
	if obj := x.doc.Field(n, "object"); obj != nil {
		var ok bool
		if recv, ok = x.chain(obj); !ok {
			return
		}
		path = recv + "." + path
	}
	e := x.edge(source, model.RelationCalls, path, n)
	head, _, field := strings.Cut(recv, ".")
	if x.vars[head] || (field && (head == "this" || head == "super")) {
		e.Member = "."
	}
	x.calls = append(x.calls, e)
}

// variables is every name the file declares as a variable, at any depth:
// fields, parameters, locals, resources, catch, for, lambda and pattern
// variables, and enum constants. One set for the file and not a scope per
// member: a receiver named like a variable anywhere in the file is far more
// likely that variable than a type.
func (x *extractor) variables() map[string]bool {
	vars := map[string]bool{}
	treesitter.Walk(x.doc.Root, func(n *gts.Node) bool {
		switch x.doc.Type(n) {
		case "variable_declarator", "formal_parameter", "catch_formal_parameter", "enhanced_for_statement",
			"resource", "instanceof_expression", "enum_constant":
			if name := x.doc.Field(n, "name"); name != nil {
				vars[x.doc.Text(name)] = true
			}
		case "type_pattern":
			vars[x.doc.Text(n.Child(n.ChildCount()-1))] = true
		case "lambda_expression":
			params := x.doc.Field(n, "parameters")
			if x.doc.Type(params) == "identifier" {
				vars[x.doc.Text(params)] = true
			}
		case "inferred_parameters":
			for _, p := range n.Children() {
				if x.doc.Type(p) == "identifier" {
					vars[x.doc.Text(p)] = true
				}
			}
		}
		return true
	})
	return vars
}

// edge is a raw edge from source to the name path as written at n, split
// into the receiver chain and the last name: "a.b.T.m" calls m on "a.b.T".
func (x *extractor) edge(source model.NodeID, rel model.Relation, path string, n *gts.Node) extract.RawEdge {
	e := extract.RawEdge{Source: source, Relation: rel, Name: path, File: x.doc.Rel, Line: line(n)}
	if i := strings.LastIndexByte(path, '.'); i >= 0 {
		e.Receiver, e.Name = path[:i], path[i+1:]
	}
	return e
}

// chain is a dotted chain of plain names as written, "this.a.b", with type
// arguments dropped; false when anything else stands in it -- a call, an
// index, a literal, a primitive type. this and super count as names: they
// are a receiver the resolver knows.
func (x *extractor) chain(n *gts.Node) (string, bool) {
	switch x.doc.Type(n) {
	case "identifier", "type_identifier", "this", "super":
		return x.doc.Text(n), true
	case "generic_type":
		return x.chain(n.Child(0))
	case "scoped_identifier":
		return x.pair(x.doc.Field(n, "scope"), x.doc.Field(n, "name"))
	case "field_access":
		return x.pair(x.doc.Field(n, "object"), x.doc.Field(n, "field"))
	case "scoped_type_identifier":
		return x.pair(n.Child(0), n.Child(n.ChildCount()-1))
	}
	return "", false
}

// pair is the chain of head, a dot and the chain of tail.
func (x *extractor) pair(head, tail *gts.Node) (string, bool) {
	h, okH := x.chain(head)
	t, okT := x.chain(tail)
	return h + "." + t, okH && okT
}

// dotted is the name a package declaration names, "a.b": its identifier or
// scoped_identifier child, after any annotations; "" without one.
func (x *extractor) dotted(n *gts.Node) string {
	i := slices.IndexFunc(n.Children(), func(c *gts.Node) bool {
		t := x.doc.Type(c)
		return t == "identifier" || t == "scoped_identifier"
	})
	name, _ := x.chain(n.Child(i)) // Child(-1) is nil
	return name
}

// line is the 1-based line a node starts on.
func line(n *gts.Node) int { return int(n.StartPoint().Row) + 1 }
