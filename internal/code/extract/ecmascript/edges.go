package ecmascript

import (
	"slices"
	"unicode"

	gts "github.com/odvcencio/gotreesitter"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// export reads one export statement in the namespace prefix: a declaration
// behind it, a default, a re-export of another file or an export list.
func (x *extractor) export(n *gts.Node, prefix string) {
	decl, value, source := x.doc.Field(n, "declaration"), x.doc.Field(n, "value"), x.doc.Field(n, "source")
	switch {
	case source != nil:
		path, _ := x.literal(source)
		for _, c := range n.Children() {
			switch x.doc.Type(c) {
			case "export_clause":
				for _, imp := range x.specifiers(c) {
					imp.Path, imp.Kind = path, "reexport"
					x.importOf(imp, n)
				}
			case "namespace_export":
				x.importOf(extract.Import{Path: path, Name: "*", Alias: x.doc.Text(c.Child(c.ChildCount() - 1)), Kind: "reexport"}, n)
			case "*":
				x.importOf(extract.Import{Path: path, Name: "*", Kind: "reexport"}, n)
			}
		}
	case decl != nil && x.doc.Type(decl) == "internal_module":
		x.namespace(decl, prefix, true)
	case decl != nil:
		before := len(x.nodes)
		x.declaration(decl, prefix, true)
		isDefault := slices.ContainsFunc(n.Children(), func(c *gts.Node) bool { return x.doc.Type(c) == "default" })
		for _, d := range x.nodes[before:] {
			if !isDefault && prefix == "" && d.Owner == "" {
				x.declared = append(x.declared, d.Name)
			}
		}
		if name := x.doc.Field(decl, "name"); isDefault && name != nil {
			// export default function f() {}: f is also the default.
			x.exportName(x.doc.Text(name), "default")
		}
	case value != nil:
		switch t := x.doc.Type(value); {
		case functionValues[t]:
			x.function(value, value, "default", prefix, true)
		case t == "class":
			x.class(value, "default", prefix, true)
		case t == "identifier":
			x.exportName(x.doc.Text(value), "default")
		default:
			x.walk(value, x.fileID)
		}
	default:
		for _, c := range n.Children() {
			switch t := x.doc.Type(c); {
			case t == "export_clause":
				for _, imp := range x.specifiers(c) {
					imp.Kind = "export"
					x.imports = append(x.imports, imp)
				}
			case t == "identifier":
				// TypeScript's export = x, CommonJS's module.exports = x.
				x.exportName(x.doc.Text(c), "default")
			case functionValues[t]:
				x.function(c, c, "default", prefix, true)
			case t == "class":
				x.class(c, "default", prefix, true)
			case c.IsNamed():
				x.walk(c, x.fileID)
			}
		}
	}
}

// exportName records that the file exports its name local as alias, an
// export list without a path.
func (x *extractor) exportName(local, alias string) {
	x.imports = append(x.imports, extract.Import{Name: local, Alias: alias, Kind: "export"})
}

// specifiers are the names of an import or export list: Name the name in
// the file that exports it, Alias the other side's when one is written.
func (x *extractor) specifiers(list *gts.Node) []extract.Import {
	var out []extract.Import
	for _, c := range list.Children() {
		if t := x.doc.Type(c); t != "import_specifier" && t != "export_specifier" {
			continue // the braces and commas
		}
		out = append(out, extract.Import{Name: x.name(x.doc.Field(c, "name")), Alias: x.doc.Text(x.doc.Field(c, "alias"))})
	}
	return out
}

// name is an identifier, or the text of a string written as a name.
func (x *extractor) name(n *gts.Node) string {
	if s, ok := x.literal(n); ok {
		return s
	}
	return x.doc.Text(n)
}

// importStatement records one import per binding of an import statement,
// or the path alone for an import that binds nothing.
func (x *extractor) importStatement(n *gts.Node) {
	path, _ := x.literal(x.doc.Field(n, "source"))
	before := len(x.imports)
	for _, c := range n.Children() {
		switch x.doc.Type(c) {
		case "import_clause":
			for _, b := range c.Children() {
				switch x.doc.Type(b) {
				case "identifier":
					x.importOf(extract.Import{Path: path, Name: "default", Alias: x.doc.Text(b)}, n)
				case "namespace_import":
					x.importOf(extract.Import{Path: path, Name: "*", Alias: x.doc.Text(b.Child(b.ChildCount() - 1))}, n)
				case "named_imports":
					for _, imp := range x.specifiers(b) {
						imp.Path = path
						x.importOf(imp, n)
					}
				}
			}
		case "import_require_clause":
			req, _ := x.literal(x.doc.Field(c, "source"))
			x.importOf(extract.Import{Path: req, Name: "*", Alias: x.doc.Text(c.Child(0)), Kind: "require"}, n)
		}
	}
	if len(x.imports) == before {
		x.importOf(extract.Import{Path: path}, n)
	}
}

// commonJS reads a module-level assignment to module.exports, exports.f or
// module.exports.f, also a chain such as exports = module.exports = x,
// which exports x under every target it names; false for any other
// assignment, and for a value it makes nothing of.
func (x *extractor) commonJS(a *gts.Node) bool {
	var targets []string
	right := a
	for x.doc.Type(right) == "assignment_expression" {
		if target, ok := x.exportTarget(x.doc.Field(right, "left")); ok {
			targets = append(targets, target)
		}
		right = x.doc.Field(right, "right")
	}
	if len(targets) == 0 {
		return false
	}
	switch t := x.doc.Type(right); {
	case t == "call_expression" && x.doc.Text(x.doc.Field(right, "function")) == "require":
		path, ok := x.onlyLiteral(right)
		if !ok {
			return false
		}
		for _, target := range targets {
			if target == "" {
				// The whole module, its default too: export * passes on none.
				x.importOf(extract.Import{Path: path, Name: "*", Kind: "reexport"}, right)
				x.importOf(extract.Import{Path: path, Name: "default", Kind: "reexport"}, right)
			} else {
				x.importOf(extract.Import{Path: path, Name: "*", Alias: target, Kind: "reexport"}, right)
			}
		}
	case functionValues[t]:
		// One node, under the first target; the others name it.
		x.function(a, right, exportAlias(targets[0]), "", true)
		for _, target := range targets[1:] {
			x.exportName(exportAlias(targets[0]), exportAlias(target))
		}
	case t == "identifier":
		for _, target := range targets {
			x.exportName(x.doc.Text(right), exportAlias(target))
		}
	case t == "object" && slices.Contains(targets, ""):
		for _, c := range right.Children() {
			switch x.doc.Type(c) {
			case "shorthand_property_identifier":
				x.exportName(x.doc.Text(c), "")
			case "pair":
				if v := x.doc.Field(c, "value"); x.doc.Type(v) == "identifier" {
					x.exportName(x.doc.Text(v), x.doc.Text(x.doc.Field(c, "key")))
				} else {
					x.walk(c, x.fileID)
				}
			}
		}
	default:
		return false
	}
	return true
}

// exportAlias is the name a CommonJS target exports under: default for
// module.exports itself.
func exportAlias(target string) string {
	if target == "" {
		return "default"
	}
	return target
}

// exportTarget is the name an assignment's left side exports under: "" for
// module.exports itself, f for exports.f and module.exports.f; false for
// any other left side.
func (x *extractor) exportTarget(left *gts.Node) (string, bool) {
	if x.doc.Type(left) != "member_expression" {
		return "", false
	}
	obj, prop := x.doc.Text(x.doc.Field(left, "object")), x.doc.Text(x.doc.Field(left, "property"))
	switch {
	case obj == "module" && prop == "exports":
		return "", true
	case obj == "exports", obj == "module.exports":
		return prop, true
	}
	return "", false
}

// walk records what n and everything below it names, for source: calls,
// constructor calls, require and import(), and the web's handovers. A
// definition below n makes no node; what it calls belongs to source.
func (x *extractor) walk(n *gts.Node, source model.NodeID) {
	switch x.doc.Type(n) {
	case "ERROR":
		return
	case "call_expression":
		x.call(n, source)
	case "new_expression":
		x.construct(n, source)
	case "jsx_attribute":
		x.attribute(n, source)
	case "jsx_opening_element", "jsx_self_closing_element":
		x.component(n, source)
	}
	for _, c := range n.Children() {
		x.walk(c, source)
	}
}

// call records one call: f(), ns.f(), this.m() or super.m(); require("x")
// and import("x") with a literal are imports instead.
func (x *extractor) call(n *gts.Node, source model.NodeID) {
	fn := x.doc.Field(n, "function")
	switch x.doc.Type(fn) {
	case "import":
		if path, ok := x.onlyLiteral(n); ok {
			x.importOf(extract.Import{Path: path, Kind: "dynamic"}, n)
		}
	case "identifier":
		name := x.doc.Text(fn)
		if path, ok := x.onlyLiteral(n); ok && name == "require" {
			x.require(n, path)
			return
		}
		x.calls = append(x.calls, x.edge(source, model.RelationCalls, name, "", n))
	case "member_expression":
		obj, prop := x.doc.Field(fn, "object"), x.doc.Text(x.doc.Field(fn, "property"))
		x.web(n, obj, prop, source)
		if recv, ok := x.receiver(obj); ok {
			x.calls = append(x.calls, x.edge(source, model.RelationCalls, prop, recv, n))
		}
	}
}

// component records a JSX tag that renders a component, <X/> or <ns.X/>,
// as a call to it: React, Preact and Solid call the component there. A
// lowercase tag is a DOM element and calls nothing.
func (x *extractor) component(n *gts.Node, source model.NodeID) {
	var name, recv string
	switch tag := x.doc.Field(n, "name"); x.doc.Type(tag) {
	case "identifier":
		name = x.doc.Text(tag)
	case "member_expression":
		r, ok := x.receiver(x.doc.Field(tag, "object"))
		if !ok {
			return
		}
		name, recv = x.doc.Text(x.doc.Field(tag, "property")), r
	}
	if name == "" || !unicode.IsUpper([]rune(name)[0]) {
		return
	}
	x.calls = append(x.calls, x.edge(source, model.RelationCalls, name, recv, n))
}

// construct records new X() and new ns.X() as a call marked "new".
func (x *extractor) construct(n *gts.Node, source model.NodeID) {
	c := x.doc.Field(n, "constructor")
	var e extract.RawEdge
	switch x.doc.Type(c) {
	case "identifier":
		e = x.edge(source, model.RelationCalls, x.doc.Text(c), "", n)
	case "member_expression":
		recv, ok := x.receiver(x.doc.Field(c, "object"))
		if !ok {
			return
		}
		e = x.edge(source, model.RelationCalls, x.doc.Text(x.doc.Field(c, "property")), recv, n)
	default:
		return
	}
	e.Member = "new"
	x.calls = append(x.calls, e)
}

// receiver is the object of a member call as a raw edge names it: a plain
// name, this or super; false for anything else, a chain or a call among
// them.
func (x *extractor) receiver(obj *gts.Node) (string, bool) {
	switch x.doc.Type(obj) {
	case "identifier", "this", "super":
		return x.doc.Text(obj), true
	}
	return "", false
}

// require records require(path) with the bindings of the declarator it
// initialises: const m binds the module, const { f, g: h } its names.
func (x *extractor) require(n *gts.Node, path string) {
	before := len(x.imports)
	if p := n.Parent(); x.doc.Type(p) == "variable_declarator" {
		switch name := x.doc.Field(p, "name"); x.doc.Type(name) {
		case "identifier":
			x.importOf(extract.Import{Path: path, Name: "*", Alias: x.doc.Text(name), Kind: "require"}, n)
		case "object_pattern":
			for _, c := range name.Children() {
				switch x.doc.Type(c) {
				case "shorthand_property_identifier_pattern":
					x.importOf(extract.Import{Path: path, Name: x.doc.Text(c), Kind: "require"}, n)
				case "pair_pattern":
					x.importOf(extract.Import{
						Path: path, Name: x.doc.Text(x.doc.Field(c, "key")), Alias: x.doc.Text(x.doc.Field(c, "value")), Kind: "require",
					}, n)
				}
			}
		}
	}
	if len(x.imports) == before {
		x.importOf(extract.Import{Path: path, Kind: "require"}, n)
	}
}

// heritage records the bases of a class: in JavaScript the expression
// after extends, in TypeScript its extends_clause and implements_clause.
func (x *extractor) heritage(id model.NodeID, h *gts.Node) {
	for _, c := range h.Children() {
		switch x.doc.Type(c) {
		case "extends_clause":
			x.base(id, model.RelationExtends, x.doc.Field(c, "value"))
		case "implements_clause":
			x.typeBases(id, model.RelationImplements, c)
		default:
			x.base(id, model.RelationExtends, c)
		}
	}
}

// base records a base written as an expression, B or ns.B; anything else,
// a call among them, names none.
func (x *extractor) base(id model.NodeID, rel model.Relation, n *gts.Node) {
	switch x.doc.Type(n) {
	case "identifier":
		x.bases = append(x.bases, x.edge(id, rel, x.doc.Text(n), "", n))
	case "member_expression":
		if recv, ok := x.receiver(x.doc.Field(n, "object")); ok {
			x.bases = append(x.bases, x.edge(id, rel, x.doc.Text(x.doc.Field(n, "property")), recv, n))
		}
	}
}

// typeBases records the bases of an implements or an interface's extends
// clause, each written as a type: B, B<T> or ns.B.
func (x *extractor) typeBases(id model.NodeID, rel model.Relation, clause *gts.Node) {
	for _, c := range clause.Children() {
		t := c
		if x.doc.Type(c) == "generic_type" {
			t = x.doc.Field(c, "name")
		}
		switch x.doc.Type(t) {
		case "type_identifier":
			x.bases = append(x.bases, x.edge(id, rel, x.doc.Text(t), "", c))
		case "nested_type_identifier":
			x.bases = append(x.bases, x.edge(id, rel, x.doc.Text(x.doc.Field(t, "name")), x.doc.Text(x.doc.Field(t, "module")), c))
		}
	}
}

// importOf records imp as an import and an imports edge at n.
func (x *extractor) importOf(imp extract.Import, n *gts.Node) {
	x.imports = append(x.imports, imp)
	x.importEdges = append(x.importEdges, extract.RawEdge{
		Source: x.fileID, Relation: model.RelationImports, Specifier: imp.Path, File: x.doc.Rel, Line: line(n),
	})
}

// onlyLiteral is the text of a call's argument when it has one and that is
// a plain string literal.
func (x *extractor) onlyLiteral(call *gts.Node) (string, bool) {
	args := x.arguments(call)
	if len(args) != 1 {
		return "", false
	}
	return x.literal(args[0])
}

// arguments are a call's arguments, without the parentheses and commas.
func (x *extractor) arguments(call *gts.Node) []*gts.Node {
	var out []*gts.Node
	for _, c := range x.doc.Field(call, "arguments").Children() {
		if c.IsNamed() {
			out = append(out, c)
		}
	}
	return out
}

// literal is the text of a string literal with nothing to expand in it:
// its quotes and one string_fragment between them.
func (x *extractor) literal(n *gts.Node) (string, bool) {
	if x.doc.Type(n) != "string" || n.ChildCount() != 3 || x.doc.Type(n.Child(1)) != "string_fragment" {
		return "", false
	}
	return x.doc.Text(n.Child(1)), true
}

// edge is a raw edge from source to name, on receiver, at n.
func (x *extractor) edge(source model.NodeID, rel model.Relation, name, receiver string, n *gts.Node) extract.RawEdge {
	return extract.RawEdge{Source: source, Relation: rel, Name: name, Receiver: receiver, File: x.doc.Rel, Line: line(n)}
}

// line is the 1-based line a node starts on.
func line(n *gts.Node) int { return int(n.StartPoint().Row) + 1 }
