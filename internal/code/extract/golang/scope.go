package golang

import "go/ast"

// scope is the set of names a position has in view, plus the local type binding
// of each.
//
// Function-scoped and deliberately over-shadowing: a frame is pushed per
// function and not per block, so a name declared inside an `if` shadows for the
// rest of the function. That direction is the safe one -- it drops call edges
// rather than inventing them -- and a per-block scope is a later refinement,
// not a correctness fix.
//
// It exists because `ast.File.Unresolved` would answer the one question this
// package asks of it -- is this name declared in the file? -- and is
// deprecated together with ast.Object as of Go 1.22. A new package does not
// build on a field on its way out.
//
// The question matters: Go shadows package names constantly.
//
//	graph := graph.New(g)   // from here `graph` is a variable
//
// Reading the second selector as a package would wire a call into a package the
// line never touches.
type scope struct {
	frames []map[string]string // name -> bound type, "" when unknown
}

func newScope() *scope {
	return &scope{frames: []map[string]string{{}}}
}

func (s *scope) push() { s.frames = append(s.frames, map[string]string{}) }

func (s *scope) pop() { s.frames = s.frames[:len(s.frames)-1] }

// declare records a name in the innermost frame, with the type it is bound to
// ("" when the form is one this package does not read).
func (s *scope) declare(name, typ string) {
	if name == "" || name == "_" {
		return
	}
	s.frames[len(s.frames)-1][name] = typ
}

// lookup answers both the shadowing question and the type question in one
// frame walk: ok reports whether the name is in view at all, and typ is the
// type it is bound to ("" when the name is unknown, or when its binding form
// was not one of the four boundType reads).
//
// Kept as one method and not two ("declared" plus a separate "lookup"): the
// only production caller (callEdge) always asks the shadowing question and
// then, in the same breath, the type question, over the identical frame
// stack and the identical key. A second walk would find nothing a first
// walk's ok already didn't decide, so it existed only to be tested, not to
// be reached.
func (s *scope) lookup(name string) (typ string, ok bool) {
	for i := len(s.frames) - 1; i >= 0; i-- {
		if typ, ok := s.frames[i][name]; ok {
			return typ, true
		}
	}
	return "", false
}

// boundType is the type a right-hand side binds, in exactly the four forms
// the original's own Go collector reads (bindings.ts handleGo):
//
//	var x T   /  var x *T   -> T
//	x := T{}                -> T
//	x := &T{}               -> T
//	x := NewT(...)          -> T   (a convention, not a resolution)
//
// Everything else binds nothing: a plain function's return type, a field
// access, a range variable, a channel receive. Widening this set widens the set
// of call edges, and that owes its own reason.
func boundType(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.CompositeLit:
		return typeName(e.Type)
	case *ast.UnaryExpr:
		if lit, ok := e.X.(*ast.CompositeLit); ok {
			return typeName(lit.Type)
		}
	case *ast.CallExpr:
		if id, ok := peelIndex(e.Fun).(*ast.Ident); ok {
			if name, ok := constructorType(id.Name); ok {
				return name
			}
		}
	}
	return ""
}

// constructorType reads "NewCache" as "Cache". Go's own convention, and the
// only heuristic in this file: it is what lets `c := NewCache()` bind.
func constructorType(fn string) (string, bool) {
	const prefix = "New"
	if len(fn) <= len(prefix) || fn[:len(prefix)] != prefix {
		return "", false
	}
	rest := fn[len(prefix):]
	if !exported(rest) {
		return "", false
	}
	return rest, true
}

// typeName is the bare name of a type expression, pointer and type arguments
// peeled.
func typeName(expr ast.Expr) string {
	switch t := peelIndex(expr).(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return typeName(t.X)
	}
	return ""
}

// peelIndex strips a type argument list, so `Of[int]` reads as `Of`.
func peelIndex(expr ast.Expr) ast.Expr {
	switch e := expr.(type) {
	case *ast.IndexExpr:
		return e.X
	case *ast.IndexListExpr:
		return e.X
	}
	return expr
}
