package gdscript

import (
	"strings"

	gts "github.com/odvcencio/gotreesitter"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// line is the 1-based line a node starts on.
func line(n *gts.Node) int { return int(n.StartPoint().Row) + 1 }

// extendsOf records the base an extends_statement names, for the class or
// script src: a class name or a chain of them (Name, Receiver), a path
// (Specifier), or an inner class of a path (Specifier, and the chain below it
// in Name). ext may be nil -- a class without a base.
func (x *extractor) extendsOf(ext *gts.Node, src model.NodeID) {
	for _, c := range ext.Children() {
		e := extract.RawEdge{Source: src, Relation: model.RelationExtends, File: x.doc.Rel, Line: line(c)}
		switch x.doc.Type(c) {
		case "string":
			e.Specifier = unquote(x.doc.Text(c))
		case "type":
			if spec, inner, ok := x.pathChain(c.Child(0)); ok {
				e.Specifier, e.Name = spec, inner
				break
			}
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

// pathChain reads `"res://a.gd".Inner.Deeper`: the path and the inner
// classes below it, as written.
func (x *extractor) pathChain(n *gts.Node) (string, string, bool) {
	if x.doc.Type(n) != "attribute" || x.doc.Type(n.Child(0)) != "string" {
		return "", "", false
	}
	var parts []string
	for _, c := range n.Children()[1:] {
		switch x.doc.Type(c) {
		case "identifier":
			parts = append(parts, x.doc.Text(c))
		case ".":
		default:
			return "", "", false
		}
	}
	return unquote(x.doc.Text(n.Child(0))), strings.Join(parts, "."), true
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

// locals are the names a function or signal binds for itself: every
// parameter, its lambdas' included, and every var and const in its body.
// Loop and match bindings are not among them.
func (x *extractor) locals(def *gts.Node) map[string]bool {
	out := map[string]bool{}
	treesitter.Walk(def, func(c *gts.Node) bool {
		switch x.doc.Type(c) {
		case "parameters":
			for _, p := range c.Children() {
				if id := x.firstIdentifier(p); id != "" {
					out[id] = true
				}
			}
		case "variable_statement", "const_statement":
			out[x.doc.Text(x.doc.Field(c, "name"))] = true
		}
		return true
	})
	return out
}

// firstIdentifier is the text of the first identifier at or below n, in
// pre-order: a parameter's name, typed or with a default. "" for none.
func (x *extractor) firstIdentifier(n *gts.Node) string {
	id := ""
	treesitter.Walk(n, func(c *gts.Node) bool {
		if id == "" && x.doc.Type(c) == "identifier" {
			id = x.doc.Text(c)
		}
		return id == ""
	})
	return id
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
		if p, ok := x.firstArg(v); ok {
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
		case "base_call":
			// Godot 3's `.f()`: the base's version of f.
			x.addCall(sc, x.firstIdentifier(c), "super", c)
			for _, k := range c.Children() {
				if x.doc.Type(k) == "arguments" {
					x.collect(k, sc)
				}
			}
			return false
		case "attribute":
			x.attribute(c, sc)
			return false
		case "await_expression":
			x.await(c, sc)
		case "setter", "getter":
			// setget's accessors and Godot 4's `get = f, set = g` name methods
			// of the class; a local cannot hide them.
			x.refName(sc, x.doc.Text(c), line(c))
		case "identifier":
			x.reference(c, sc)
		}
		return true
	})
}

// signalVerbs map the calls that name a signal by a string to the verb the
// resolver reads: emit_signal("s") is s.emit().
var signalVerbs = map[string]string{
	"emit_signal": "emit", "connect": "connect", "disconnect": "disconnect", "is_connected": "is_connected",
}

// byName are the calls that call a method named by a string, with the
// argument the name stands in.
var byName = map[string]int{
	"call": 0, "call_deferred": 0, "callv": 0, "rpc": 0, "rpc_unreliable": 0,
	"rpc_id": 1, "rpc_unreliable_id": 1,
}

// callableVerbs are the members of a Callable that call it now.
var callableVerbs = map[string]bool{"call": true, "call_deferred": true, "callv": true, "rpc": true, "rpc_id": true}

// call records a call of a plain name: foo(), and the forms that are no
// call of a function by that name -- preload and load, whose literal is a
// file the script needs; super(), the base's version of the function it
// stands in; a signal named by a string (emit_signal("x") is x.emit()), or
// awaited by Godot 3's yield(obj, "x"); a method named by a string, called
// (call("m"), rpc_id(1, "m")) or taken as a value (funcref(self, "m"),
// Callable(self, "m")). The callee is no reference; the arguments are walked.
func (x *extractor) call(c *gts.Node, sc scope) {
	name := x.doc.Text(c.Child(0))
	args := x.args(c)
	e := extract.RawEdge{Source: sc.source, Relation: model.RelationCalls, Name: name, File: x.doc.Rel, Line: line(c)}
	switch name {
	case "preload", "load":
		if p, ok := x.firstArg(c); ok {
			x.importEdges = append(x.importEdges, extract.RawEdge{
				Source: x.fileID, Relation: model.RelationImports, Specifier: loadPath(name, p), File: x.doc.Rel, Line: e.Line,
			})
		}
		e.Name = ""
	case "super":
		e.Name, e.Receiver = sc.fn, "super"
	case "emit_signal", "connect", "disconnect", "is_connected":
		x.signalCall(&e, "", args, sc)
		if e.Name == "emit_signal" {
			// No literal: the script defines no emit_signal to land on.
			e.Name = ""
		}
	case "yield":
		e.Name = ""
		if len(args) == 2 {
			if sig, ok := x.literal(args[1]); ok {
				if obj := x.chain(args[0]); obj != "" {
					x.addCall(sc, "await", join(trimSelf(obj), sig), c)
				}
			}
		}
	case "funcref", "Callable":
		e.Name = ""
		x.handler(args, 0, sc)
	default:
		if i, ok := byName[name]; ok && i < len(args) {
			if m, ok := x.literal(args[i]); ok {
				e.Name = m
			}
		}
	}
	if e.Name != "" {
		x.calls = append(x.calls, e)
	}
	for _, k := range c.Children()[1:] {
		x.collect(k, sc)
	}
}

// signalCall turns a call e on recv ("" for the script itself) that names a
// signal by its first argument into the signal verb on recv's signal, and
// references the handler connect names by `self, "m"`. With no literal it
// stays the call it is.
func (x *extractor) signalCall(e *extract.RawEdge, recv string, args []*gts.Node, sc scope) {
	if len(args) == 0 {
		return
	}
	if e.Name != "emit_signal" {
		x.handler(args, 1, sc)
	}
	if sig, ok := x.literal(args[0]); ok {
		e.Name, e.Receiver = signalVerbs[e.Name], join(recv, sig)
	}
}

// handler references the method `self, "m"` names at args[i] and args[i+1].
func (x *extractor) handler(args []*gts.Node, i int, sc scope) {
	if i+1 >= len(args) || x.doc.Text(args[i]) != "self" {
		return
	}
	if m, ok := x.literal(args[i+1]); ok {
		x.refName(sc, m, line(args[i+1]))
	}
}

// await records Godot 4's `await X.s`: the signal s of X, which the
// resolver reads like X.s.emit(). A bare `await s` is a reference of s and
// needs nothing here.
func (x *extractor) await(c *gts.Node, sc scope) {
	for _, k := range c.Children() {
		if x.doc.Type(k) != "attribute" {
			continue
		}
		if chain := trimSelf(x.chain(k)); strings.Contains(chain, ".") {
			x.addCall(sc, "await", chain, k)
		}
	}
}

// addCall records a call of name on receiver at n.
func (x *extractor) addCall(sc scope, name, receiver string, n *gts.Node) {
	x.calls = append(x.calls, extract.RawEdge{
		Source: sc.source, Relation: model.RelationCalls, Name: name, Receiver: receiver, File: x.doc.Rel, Line: line(n),
	})
}

// attribute reads a.b.c and a.b(): the leading name is a reference, a name
// after a "." is a member and none, but `self.m` names the member m. Each
// attribute_call is a call whose receiver is the chain of plain names before
// it, `self.` cut off, as long as nothing else -- a call, $Node, a subscript
// -- stands there.
func (x *extractor) attribute(n *gts.Node, sc scope) {
	var chain []string
	plain, called := true, false
	for i, c := range n.Children() {
		switch x.doc.Type(c) {
		case ".":
		case "identifier":
			if i == 0 {
				x.reference(c, sc)
			}
			chain = append(chain, x.doc.Text(c))
		case "attribute_call":
			called = true
			if plain && len(chain) > 0 {
				x.member(c, trimSelf(strings.Join(chain, ".")), chain[0], sc)
			} else if verb := x.doc.Text(c.Child(0)); verb != "emit_signal" && signalVerbs[verb] != "" {
				// On $Node or get_node("x") the signal has no owner to bind,
				// but `self, "m"` still names a handler of this script.
				x.handler(x.args(c), 1, sc)
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
	if plain && !called && len(chain) == 2 && chain[0] == "self" {
		x.refName(sc, chain[1], line(n))
	}
}

// member records the call c of a member on the receiver chain recv, whose
// first name as written is head: recv.f(), and the forms call reads on a
// script, here on recv. A Callable's own call -- m.call(), self.m.rpc() --
// also calls m, unless m is a local of the function.
func (x *extractor) member(c *gts.Node, recv, head string, sc scope) {
	name := x.doc.Text(c.Child(0))
	args := x.args(c)
	e := extract.RawEdge{Source: sc.source, Relation: model.RelationCalls, Name: name, Receiver: recv, File: x.doc.Rel, Line: line(c)}
	if _, ok := signalVerbs[name]; ok {
		x.signalCall(&e, recv, args, sc)
	} else if i, ok := byName[name]; ok && i < len(args) {
		if m, ok := x.literal(args[i]); ok {
			e.Name = m
		}
	}
	x.calls = append(x.calls, e)
	local := head != "self" && sc.locals[head]
	if callableVerbs[name] && recv != "" && !strings.Contains(recv, ".") && !local {
		x.addCall(sc, recv, "", c)
	}
}

// reference records a name a definition uses, once per definition, unless a
// parameter or local of the function holds it. The resolver keeps those
// that name a class, or a method or signal of the class.
func (x *extractor) reference(c *gts.Node, sc scope) {
	if name := x.doc.Text(c); !sc.locals[name] {
		x.refName(sc, name, line(c))
	}
}

// refName records a reference of name from sc, once per definition.
func (x *extractor) refName(sc scope, name string, at int) {
	key := [2]string{string(sc.source), name}
	if x.referenced[key] {
		return
	}
	x.referenced[key] = true
	x.refs = append(x.refs, extract.RawEdge{Source: sc.source, Relation: model.RelationReferences, Name: name, File: x.doc.Rel, Line: at})
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

// args are the arguments of a call or attribute_call, without the
// parentheses and commas around them.
func (x *extractor) args(c *gts.Node) []*gts.Node {
	var out []*gts.Node
	for _, a := range x.doc.Field(c, "arguments").Children() {
		switch x.doc.Type(a) {
		case "(", ")", ",":
		default:
			out = append(out, a)
		}
	}
	return out
}

// firstArg is the first argument of a call, unquoted, when it is a string
// literal: the path of a preload or load. `preload()` has none.
func (x *extractor) firstArg(c *gts.Node) (string, bool) {
	args := x.args(c)
	if len(args) == 0 || x.doc.Type(args[0]) != "string" {
		return "", false
	}
	return unquote(x.doc.Text(args[0])), true
}

// literal is a string or StringName literal, unquoted: the name of a signal
// or a method.
func (x *extractor) literal(a *gts.Node) (string, bool) {
	switch x.doc.Type(a) {
	case "string":
		return unquote(x.doc.Text(a)), true
	case "string_name":
		return unquote(strings.TrimPrefix(x.doc.Text(a), "&")), true
	}
	return "", false
}

// trimSelf cuts a leading self from a receiver chain: self.a is a.
func trimSelf(chain string) string {
	if chain == "self" {
		return ""
	}
	return strings.TrimPrefix(chain, "self.")
}

// join is name on the receiver chain recv: "recv.name", or name alone.
func join(recv, name string) string {
	if recv == "" {
		return name
	}
	return recv + "." + name
}

// unquote strips the quotes of a string literal: "a", 'a'.
func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}
