package php

import (
	"strings"

	gts "github.com/odvcencio/gotreesitter"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// use records every name a `use` declaration imports as an import and an
// imports edge: the group prefix and the clause joined by a backslash,
// without a leading one, the alias, and the kind -- "function" or "const"
// -- of the clause, else of the whole declaration.
func (x *extractor) use(n *gts.Node) {
	kind, prefix := x.doc.Text(x.doc.Field(n, "type")), ""
	var clauses []*gts.Node
	for _, c := range n.Children() {
		switch x.doc.Type(c) {
		case "namespace_name":
			prefix = x.doc.Text(c) + `\`
		case "namespace_use_group":
			clauses = append(clauses, c.Children()...)
		case "namespace_use_clause":
			clauses = append(clauses, c)
		}
	}
	for _, c := range clauses {
		if x.doc.Type(c) != "namespace_use_clause" {
			continue // the braces and commas of a group
		}
		imp := extract.Import{Alias: x.doc.Text(x.doc.Field(c, "alias")), Kind: kind}
		if k := x.doc.Field(c, "type"); k != nil {
			imp.Kind = x.doc.Text(k)
		}
		for _, p := range c.Children() {
			if t := x.doc.Type(p); (t == "name" || t == "qualified_name") && imp.Path == "" {
				imp.Path = strings.TrimPrefix(prefix+x.doc.Text(p), `\`)
			}
		}
		x.importOf(imp, c)
	}
}

// importOf records imp as an import and an imports edge at n.
func (x *extractor) importOf(imp extract.Import, n *gts.Node) {
	x.imports = append(x.imports, imp)
	x.importEdges = append(x.importEdges, extract.RawEdge{
		Source: x.fileID, Relation: model.RelationImports, Specifier: imp.Path, File: x.doc.Rel, Line: line(n),
	})
}

// bases records every name a base_clause, class_interface_clause or trait
// use names as an extends edge; whether the target is a class, an
// interface or a trait only the resolver can tell. The body of a trait use
// -- insteadof and as -- names methods and is skipped.
func (x *extractor) bases(id model.NodeID, clause *gts.Node) {
	for _, c := range clause.Children() {
		if path, ok := x.name(c); ok {
			x.extends = append(x.extends, x.edge(id, model.RelationExtends, path, c))
		}
	}
}

// requires are the expressions that load another file.
var requires = map[string]bool{
	"require_expression": true, "require_once_expression": true, "include_expression": true, "include_once_expression": true,
}

// expression records what n itself names, for source: a call, a
// constructor call or a loaded file. Its children are the caller's to walk.
func (x *extractor) expression(n *gts.Node, source model.NodeID) {
	switch t := x.doc.Type(n); {
	case t == "function_call_expression":
		if path, ok := x.name(x.doc.Field(n, "function")); ok {
			x.calls = append(x.calls, x.edge(source, model.RelationCalls, path, n))
		}
	case t == "scoped_call_expression":
		// self, static and parent are a relative_scope; $obj::m() calls
		// through an object.
		scope := x.doc.Field(n, "scope")
		kind := x.doc.Type(scope)
		recv, ok := x.name(scope)
		if kind == "relative_scope" || kind == "variable_name" {
			recv, ok = x.doc.Text(scope), true
		}
		mark := model.Kind("::")
		if kind == "variable_name" {
			mark = "."
		}
		x.member(n, recv, ok, mark, source)
	case t == "member_call_expression", t == "nullsafe_member_call_expression":
		recv, ok := x.object(x.doc.Field(n, "object"))
		mark := model.Kind(".")
		if recv == "$this" {
			mark = ""
		}
		x.member(n, recv, ok, mark, source)
	case t == "object_creation_expression":
		for _, c := range n.Children() {
			if path, ok := x.name(c); ok {
				e := x.edge(source, model.RelationCalls, path, n)
				e.Member = "new"
				x.calls = append(x.calls, e)
			}
		}
	case requires[t]:
		if path, ok := x.required(n.Child(n.ChildCount() - 1)); ok {
			x.importOf(extract.Import{Path: path, Kind: "require"}, n)
		}
	}
}

// member records the call of method n on the receiver recv, when ok and
// the method is named as written, marked as RawEdge.Member says: "." for a
// receiver that is an object, "::" for a class.
func (x *extractor) member(n *gts.Node, recv string, ok bool, mark model.Kind, source model.NodeID) {
	name := x.doc.Field(n, "name")
	if !ok || x.doc.Type(name) != "name" {
		return
	}
	x.calls = append(x.calls, extract.RawEdge{
		Source: source, Relation: model.RelationCalls, Name: x.doc.Text(name), Receiver: recv,
		File: x.doc.Rel, Line: line(n), Member: mark,
	})
}

// object is a member call's object as a chain of plain names, "$a->b";
// false for anything else, a call or a new among them.
func (x *extractor) object(n *gts.Node) (string, bool) {
	switch x.doc.Type(n) {
	case "variable_name":
		return x.doc.Text(n), true
	case "member_access_expression", "nullsafe_member_access_expression":
		head, ok := x.object(x.doc.Field(n, "object"))
		name := x.doc.Field(n, "name")
		return head + "->" + x.doc.Text(name), ok && x.doc.Type(name) == "name"
	}
	return "", false
}

// name is a class or function name as written, qualification and leading
// backslash included; false for anything else, a variable among them.
func (x *extractor) name(n *gts.Node) (string, bool) {
	switch x.doc.Type(n) {
	case "name", "qualified_name", "relative_name":
		return x.doc.Text(n), true
	}
	return "", false
}

// required is the file a require or include operand names relative to the
// file that runs it: a plain string literal, or one appended to __DIR__ or
// dirname(__FILE__). False for anything built at run time.
func (x *extractor) required(n *gts.Node) (string, bool) {
	switch x.doc.Type(n) {
	case "parenthesized_expression":
		return x.required(n.Child(1))
	case "binary_expression":
		dir := strings.Join(strings.Fields(x.doc.Text(x.doc.Field(n, "left"))), "")
		if dir != "__DIR__" && dir != "dirname(__FILE__)" {
			return "", false
		}
		lit, ok := x.literal(x.doc.Field(n, "right"))
		return strings.TrimPrefix(lit, "/"), ok
	}
	return x.literal(n)
}

// literal is the text of a string literal with nothing to expand in it:
// its quotes and one string_content between them.
func (x *extractor) literal(n *gts.Node) (string, bool) {
	if t := x.doc.Type(n); (t != "string" && t != "encapsed_string") || n.ChildCount() != 3 ||
		x.doc.Type(n.Child(1)) != "string_content" {
		return "", false
	}
	return x.doc.Text(n.Child(1)), true
}

// edge is a raw edge from source to the name path as written at n, split
// at its last backslash: the receiver is what comes before, "\" for a name
// in the global namespace written \f.
func (x *extractor) edge(source model.NodeID, rel model.Relation, path string, n *gts.Node) extract.RawEdge {
	e := extract.RawEdge{Source: source, Relation: rel, Name: path, File: x.doc.Rel, Line: line(n)}
	if i := strings.LastIndexByte(path, '\\'); i >= 0 {
		e.Receiver, e.Name = path[:max(i, 1)], path[i+1:]
	}
	return e
}

// line is the 1-based line a node starts on.
func line(n *gts.Node) int { return int(n.StartPoint().Row) + 1 }
