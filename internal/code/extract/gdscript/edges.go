package gdscript

import (
	"strings"

	gts "github.com/odvcencio/gotreesitter"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// extendsOf records the base an extends_statement names, for the class or
// script src: a class name or a chain of them (Name, Receiver), or a path
// (Specifier). ext may be nil -- a class without a base.
func (x *extractor) extendsOf(ext *gts.Node, src model.NodeID) {
	for _, c := range ext.Children() {
		e := extract.RawEdge{Source: src, Relation: model.RelationExtends, File: x.doc.Rel}
		switch x.doc.Type(c) {
		case "string":
			e.Specifier = unquote(x.doc.Text(c))
		case "type":
			chain := x.chain(c.Child(0))
			if chain == "" {
				continue
			}
			e.Name = chain
			if i := strings.LastIndexByte(chain, '.'); i >= 0 {
				e.Receiver, e.Name = chain[:i], chain[i+1:]
			}
		default:
			continue
		}
		x.extends = append(x.extends, e)
	}
}

// chain is an identifier, or an attribute made of identifiers alone, as
// written: "Outer.Inner". "" for anything else.
func (x *extractor) chain(n *gts.Node) string {
	switch x.doc.Type(n) {
	case "identifier":
		return x.doc.Text(n)
	case "attribute":
		var parts []string
		for _, c := range n.Children() {
			switch x.doc.Type(c) {
			case "identifier":
				parts = append(parts, x.doc.Text(c))
			case ".":
			default:
				return ""
			}
		}
		return strings.Join(parts, ".")
	}
	return ""
}

// alias records `const X = preload("p")` and `var X = load("p")` at the top
// of a script: X binds the file p for the rest of it. The import edge itself
// comes from collect, which walks the statement after this.
func (x *extractor) alias(stmt *gts.Node) {
	v := x.doc.Field(stmt, "value")
	if x.doc.Type(v) != "call" {
		return
	}
	switch x.doc.Text(v.Child(0)) {
	case "preload", "load":
		if p, ok := x.firstArg(v, false); ok {
			x.imports = append(x.imports, extract.Import{Alias: x.doc.Text(x.doc.Field(stmt, "name")), Path: loadPath(x.doc.Text(v.Child(0)), p)})
		}
	}
}

// collect walks a subtree that holds no definition with a node and records
// its calls and references for sc and its imports for the file. An ERROR
// node is skipped whole: a call found there sits in text the grammar
// rejected.
func (x *extractor) collect(n *gts.Node, sc scope) {
	treesitter.Walk(n, func(c *gts.Node) bool {
		if c.IsError() {
			return false
		}
		switch x.doc.Type(c) {
		case "call":
			x.call(c, sc)
			return false
		case "attribute":
			x.attribute(c, sc)
			return false
		case "identifier":
			x.reference(c, sc)
		}
		return true
	})
}

// call records a call of a plain name: foo(), and the forms that are no
// call of a function by that name -- preload and load, whose literal is a
// file the script needs; super(), the base's version of the function it
// stands in; emit_signal("x"), which is x.emit(). The callee is no
// reference; the arguments are walked.
func (x *extractor) call(c *gts.Node, sc scope) {
	e := extract.RawEdge{Source: sc.source, Relation: model.RelationCalls, Name: x.doc.Text(c.Child(0)), File: x.doc.Rel}
	switch e.Name {
	case "preload", "load":
		if p, ok := x.firstArg(c, false); ok {
			x.importEdges = append(x.importEdges, extract.RawEdge{
				Source: x.fileID, Relation: model.RelationImports, Specifier: loadPath(e.Name, p), File: x.doc.Rel,
			})
		}
		e.Name = ""
	case "super":
		e.Name, e.Receiver = sc.fn, "super"
	case "emit_signal":
		signal, ok := x.firstArg(c, true)
		e.Name, e.Receiver = "emit", signal
		if !ok {
			e.Name = ""
		}
	}
	if e.Name != "" {
		x.calls = append(x.calls, e)
	}
	for _, k := range c.Children()[1:] {
		x.collect(k, sc)
	}
}

// attribute reads a.b.c and a.b(): the leading name is a reference, a name
// after a "." is a member and none. Each attribute_call is a call whose
// receiver is the chain of plain names before it, `self.` cut off, as long
// as nothing else -- a call, $Node, a subscript -- stands there.
func (x *extractor) attribute(n *gts.Node, sc scope) {
	var chain []string
	plain := true
	for i, c := range n.Children() {
		switch x.doc.Type(c) {
		case ".":
		case "identifier":
			if i == 0 {
				x.reference(c, sc)
			}
			chain = append(chain, x.doc.Text(c))
		case "attribute_call":
			if plain {
				recv := strings.Join(chain, ".")
				if recv == "self" {
					recv = ""
				} else {
					recv = strings.TrimPrefix(recv, "self.")
				}
				x.calls = append(x.calls, extract.RawEdge{
					Source: sc.source, Relation: model.RelationCalls,
					Name: x.doc.Text(c.Child(0)), Receiver: recv, File: x.doc.Rel,
				})
			}
			plain = false
			for _, k := range c.Children()[1:] {
				x.collect(k, sc)
			}
		default:
			plain = false
			x.collect(c, sc)
		}
	}
}

// reference records a name a definition uses, once per definition. Most are
// locals and parameters; the resolver keeps those that name a class.
func (x *extractor) reference(c *gts.Node, sc scope) {
	key := [2]string{string(sc.source), x.doc.Text(c)}
	if x.referenced[key] {
		return
	}
	x.referenced[key] = true
	x.refs = append(x.refs, extract.RawEdge{Source: sc.source, Relation: model.RelationReferences, Name: key[1], File: x.doc.Rel})
}

// loadPath is the path a preload or load literal names. preload reads a path
// without a scheme from the script's folder, which the resolver does; load
// reads it from res://, so it is made so here.
func loadPath(callee, p string) string {
	if callee == "load" && !strings.Contains(p, "://") {
		return "res://" + p
	}
	return p
}

// firstArg is the first argument of a call, unquoted, and whether it is a
// string literal, or with name a StringName literal (&"x") too. The
// arguments are "(", the arguments with their commas, ")"; `preload()` has
// none.
func (x *extractor) firstArg(c *gts.Node, name bool) (string, bool) {
	args := x.doc.Field(c, "arguments").Children()
	if len(args) < 3 {
		return "", false
	}
	text := x.doc.Text(args[1])
	if name && x.doc.Type(args[1]) == "string_name" {
		return unquote(strings.TrimPrefix(text, "&")), true
	}
	return unquote(text), x.doc.Type(args[1]) == "string"
}

// unquote strips the quotes of a string literal: "a", 'a'.
func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}
