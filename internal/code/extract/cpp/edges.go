package cpp

import (
	"slices"
	"strings"

	gts "github.com/odvcencio/gotreesitter"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// include records one #include as an import and an imports edge, the path
// without its quotes or angle brackets. An include of a macro names no
// file and is left out.
func (x *extractor) include(n *gts.Node) {
	p := x.doc.Field(n, "path")
	var imp extract.Import
	switch x.doc.Type(p) {
	case "string_literal":
		imp.Path = strings.Trim(x.doc.Text(p), `"`)
	case "system_lib_string":
		imp.Path, imp.System = strings.Trim(x.doc.Text(p), "<>"), true
	default:
		return
	}
	x.imports = append(x.imports, imp)
	x.importEdges = append(x.importEdges, extract.RawEdge{
		Source: x.fileID, Relation: model.RelationImports, Specifier: imp.Path, File: x.doc.Rel, Line: line(n),
	})
}

// using records `using namespace a::b;` as an import with the alias
// "namespace" for the resolver's type lookup; it names no file and gets no
// imports edge. `using a::T;` brings in one name and is left out.
func (x *extractor) using(n *gts.Node) {
	kids := n.Children()
	if !slices.ContainsFunc(kids, func(c *gts.Node) bool { return x.doc.Type(c) == "namespace" }) {
		return
	}
	for _, c := range kids {
		if segs := x.segments(c); segs != nil {
			x.imports = append(x.imports, extract.Import{Path: strings.Join(segs, "::"), Alias: "namespace"})
			return
		}
	}
}

// bases records every class a base clause names as an extends edge;
// access specifiers and virtual are skipped, and Base<T> names Base.
func (x *extractor) bases(id model.NodeID, clause *gts.Node) {
	for _, c := range clause.Children() {
		if segs := x.segments(c); segs != nil {
			x.extends = append(x.extends, x.edge(id, model.RelationExtends, segs, c))
		}
	}
}

// call records a call as written: f(), a::b::f(), obj.f(), p->f(),
// this->f(). A member call on any object but this is marked ".": its
// receiver names an object, never a class or namespace. A callee that is
// no name -- get().f(), a lambda -- names nothing to resolve.
func (x *extractor) call(n *gts.Node, s scope) {
	if s.source == "" {
		return
	}
	fn := x.doc.Field(n, "function")
	segs := x.callee(fn)
	if segs == nil {
		return
	}
	e := x.edge(s.source, model.RelationCalls, segs, n)
	if x.doc.Type(fn) == "field_expression" && e.Receiver != "this" {
		e.Member = "."
	}
	x.calls = append(x.calls, e)
	if s.binds {
		x.bind(n, segs, s)
	}
}

// construct records new T() as a call marked "new"; new int() names no type.
func (x *extractor) construct(n *gts.Node, s scope) {
	if s.source == "" {
		return
	}
	if segs := x.segments(x.doc.Field(n, "type")); segs != nil {
		e := x.edge(s.source, model.RelationCalls, segs, n)
		e.Member = "new"
		x.calls = append(x.calls, e)
	}
}

// callee is a call's function as receiver parts and name: a member access
// is its object's path as one part, "a.b" for a.b->f(), so it never reads
// as a qualifier.
func (x *extractor) callee(fn *gts.Node) []string {
	if x.doc.Type(fn) != "field_expression" {
		return x.segments(fn)
	}
	recv, ok := x.path(x.doc.Field(fn, "argument"))
	field := x.segments(x.doc.Field(fn, "field"))
	if !ok || field == nil {
		return nil
	}
	return []string{recv, field[0]}
}

// path is an object as a dotted chain of names, "this" and "a::b" among
// them; false for anything else.
func (x *extractor) path(n *gts.Node) (string, bool) {
	switch x.doc.Type(n) {
	case "identifier", "this":
		return x.doc.Text(n), true
	case "qualified_identifier":
		segs := x.segments(n)
		return strings.Join(segs, "::"), segs != nil
	case "field_expression":
		head, ok := x.path(x.doc.Field(n, "argument"))
		field := x.segments(x.doc.Field(n, "field"))
		if !ok || field == nil {
			return "", false
		}
		return head + "." + field[0], true
	}
	return "", false
}

// bindMethods are ClassDB's calls that bind a method under a Godot name.
var bindMethods = map[string]bool{"bind_method": true, "bind_static_method": true, "bind_vararg_method": true}

// bind records a Godot binding in _bind_methods as a references edge:
// ClassDB::bind_method(D_METHOD("name", …), &X::m) and its static and
// vararg kin bind X::m as "name", ADD_SIGNAL(MethodInfo("name", …))
// declares a signal of the class. A binding whose name or method is no
// literal is left out.
func (x *extractor) bind(n *gts.Node, segs []string, s scope) {
	args := x.args(n)
	e := extract.RawEdge{Source: s.source, Relation: model.RelationReferences, File: x.doc.Rel, Line: line(n)}
	switch name := segs[len(segs)-1]; {
	case name == "ADD_SIGNAL" && len(segs) == 1:
		if len(args) == 0 || !x.callOf(args[0], "MethodInfo") {
			return
		}
		e.Member, e.Receiver = model.Kind(extract.NamespaceGodotSignal), s.binder
		e.Specifier = x.firstString(x.args(args[0]))
	case bindMethods[name]:
		e.Member = model.Kind(extract.NamespaceGodotMethod)
		dmethod := false
		for _, a := range args {
			switch {
			case x.callOf(a, "D_METHOD") && !dmethod:
				e.Specifier, dmethod = x.firstString(x.args(a)), true
			case x.doc.Type(a) == "pointer_expression":
				if m := x.segments(x.doc.Field(a, "argument")); len(m) > 1 && e.Name == "" {
					e.Receiver, e.Name = strings.Join(m[:len(m)-1], "::"), m[len(m)-1]
				}
			}
		}
		if !dmethod {
			// bind_vararg_method names the method without D_METHOD. With
			// one, a string beside it is the class of bind_static_method.
			e.Specifier = x.firstString(args)
		}
		if e.Name == "" {
			return
		}
	default:
		return
	}
	if e.Specifier != "" {
		x.bindings = append(x.bindings, e)
	}
}

// args are the arguments of a call, without parentheses and commas.
func (x *extractor) args(call *gts.Node) []*gts.Node {
	var out []*gts.Node
	if list := x.doc.Field(call, "arguments"); list != nil {
		for _, c := range list.Children() {
			if c.IsNamed() && x.doc.Type(c) != "comment" {
				out = append(out, c)
			}
		}
	}
	return out
}

// callOf is whether n is a call of the plain name fn.
func (x *extractor) callOf(n *gts.Node, fn string) bool {
	return x.doc.Type(n) == "call_expression" && slices.Equal(x.segments(x.doc.Field(n, "function")), []string{fn})
}

// firstString is the text of the first plain string literal among args, ""
// without one.
func (x *extractor) firstString(args []*gts.Node) string {
	for _, a := range args {
		if x.doc.Type(a) == "string_literal" {
			return strings.Trim(x.doc.Text(a), `"`)
		}
	}
	return ""
}

// offerClass claims every GDCLASS line in body no nested class has claimed,
// and offers id as the Godot class each names; a class without a node, id
// "", offers nothing.
func (x *extractor) offerClass(body *gts.Node, id model.NodeID) {
	for _, m := range x.marks {
		if m.claimed || uint32(m.at) < body.StartByte() || uint32(m.at) >= body.EndByte() {
			continue
		}
		m.claimed = true
		if id != "" {
			x.exports = append(x.exports, extract.Export{Namespace: extract.NamespaceGodotClass, Name: m.name, Target: id})
		}
	}
}

// edge is a raw edge from source to the name segs as written at n: the last
// segment is the name, the ones before it the receiver.
func (x *extractor) edge(source model.NodeID, rel model.Relation, segs []string, n *gts.Node) extract.RawEdge {
	return extract.RawEdge{
		Source: source, Relation: rel, File: x.doc.Rel, Line: line(n),
		Receiver: strings.Join(segs[:len(segs)-1], "::"), Name: segs[len(segs)-1],
	}
}

// line is the 1-based line a node starts on.
func line(n *gts.Node) int { return int(n.StartPoint().Row) + 1 }
