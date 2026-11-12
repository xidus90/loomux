package ecmascript

import (
	"path"
	"slices"
	"strings"
	"unicode"

	gts "github.com/odvcencio/gotreesitter"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// The web side of a script (docs/.superpowers/specs/2026-11-11-ecmascript-design.md):
// an element id or a selector's id is handed over as a references edge the
// resolver scopes by the pages that load the script; a class goes out as a
// bridge scoped by the stylesheets the module imports; a JSX id is offered
// as an element of the component around it.

// web records what a member call obj.prop(…) at n hands over to a page or
// a stylesheet, for source.
func (x *extractor) web(n, obj *gts.Node, prop string, source model.NodeID) {
	switch prop {
	case "getElementById":
		if id, ok := x.onlyLiteral(n); ok {
			x.ref(source, extract.NamespaceHTMLID, id, n)
		}
	case "querySelector", "querySelectorAll":
		if sel, ok := x.onlyLiteral(n); ok {
			x.selectors(sel, source, n)
		}
	case "getElementsByClassName":
		if names, ok := x.onlyLiteral(n); ok {
			x.classes(names, source, n)
		}
	case "add", "remove", "toggle", "contains":
		if x.doc.Type(obj) != "member_expression" || x.doc.Text(x.doc.Field(obj, "property")) != "classList" {
			return
		}
		for _, a := range x.arguments(n) {
			if names, ok := x.literal(a); ok {
				x.classes(names, source, n)
			}
		}
	}
}

// selectors records a selector list made of #id and .class alone, each
// part on its own: an id as an element and a style rule's id, a class as a
// class. A list with any other part -- a type, a descendant, an attribute
// -- names nothing for certain and records nothing.
func (x *extractor) selectors(list string, source model.NodeID, n *gts.Node) {
	parts := strings.Split(list, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
		if len(parts[i]) < 2 || (parts[i][0] != '#' && parts[i][0] != '.') || !isIdent(parts[i][1:]) {
			return
		}
	}
	for _, p := range parts {
		if p[0] == '#' {
			x.ref(source, extract.NamespaceHTMLID, p[1:], n)
			x.ref(source, extract.NamespaceCSSID, p[1:], n)
		} else {
			x.classes(p[1:], source, n)
		}
	}
}

// isIdent is whether s is one CSS name: letters, digits, '-' and '_'.
func isIdent(s string) bool {
	return !strings.ContainsFunc(s, func(r rune) bool {
		return r != '-' && r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// ref records an id handed over in namespace ns as a references edge.
func (x *extractor) ref(source model.NodeID, ns extract.Namespace, name string, n *gts.Node) {
	e := x.edge(source, model.RelationReferences, name, "", n)
	e.Member = model.Kind(ns)
	x.refs = append(x.refs, e)
}

// classes records every class of a space-separated list as a css.class
// bridge; finish sets their scope.
func (x *extractor) classes(names string, source model.NodeID, n *gts.Node) {
	for _, c := range strings.Fields(names) {
		x.bridges = append(x.bridges, extract.Bridge{
			Source: source, Relation: model.RelationReferences, Namespace: extract.NamespaceCSSClass,
			Name: c, File: x.doc.Rel, Line: line(n),
		})
	}
}

// attribute records a JSX attribute with a literal value: className, or
// class as Solid and Preact write it, hands its classes over; id offers an
// element as source, the component around it.
func (x *extractor) attribute(n *gts.Node, source model.NodeID) {
	value, ok := x.literal(n.Child(n.ChildCount() - 1))
	if !ok {
		return
	}
	switch x.doc.Text(n.Child(0)) {
	case "className", "class":
		x.classes(value, source, n)
	case "id":
		x.exports = append(x.exports, extract.Export{Namespace: extract.NamespaceHTMLID, Name: value, Target: source})
	}
}

// styles are the stylesheets the file imports by a relative path, joined
// with its directory: the scope of its class handovers. Nil without one.
func (x *extractor) styles() []string {
	var out []string
	for _, imp := range x.imports {
		ext := path.Ext(imp.Path)
		if (ext != ".css" && ext != ".scss") || !(strings.HasPrefix(imp.Path, "./") || strings.HasPrefix(imp.Path, "../")) {
			continue
		}
		if p := path.Join(path.Dir(x.doc.Rel), imp.Path); !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}
