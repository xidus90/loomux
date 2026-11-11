package golang

import (
	"go/ast"
	"go/token"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

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
	frames []map[string]binding
}

// binding is what a name is bound to: a type this package reads, or the call
// whose first result it holds, which only resolve can type. Both empty when
// the form is one this package does not read.
type binding struct {
	typ string
	via *extract.RawEdge
	fn  bool // a package-level function of this file
}

func newScope() *scope {
	return &scope{frames: []map[string]binding{{}}}
}

func (s *scope) push() { s.frames = append(s.frames, map[string]binding{}) }

func (s *scope) pop() { s.frames = s.frames[:len(s.frames)-1] }

// declare records a name in the innermost frame, with the type it is bound to
// ("" when the form is one this package does not read).
func (s *scope) declare(name, typ string) {
	s.bind(name, binding{typ: typ})
}

// bind records a name in the innermost frame with its whole binding.
func (s *scope) bind(name string, b binding) {
	if name == "" || name == "_" {
		return
	}
	s.frames[len(s.frames)-1][name] = b
}

// local reports whether a name is declared inside a function: a variable, a
// parameter or a result, and never one of the file's package-level names.
func (s *scope) local(name string) bool {
	for i := len(s.frames) - 1; i > 0; i-- {
		if _, ok := s.frames[i][name]; ok {
			return true
		}
	}
	return false
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
func (s *scope) lookup(name string) (b binding, ok bool) {
	for i := len(s.frames) - 1; i >= 0; i-- {
		if b, ok := s.frames[i][name]; ok {
			return b, true
		}
	}
	return binding{}, false
}

// boundType is the type a right-hand side binds, in exactly the four forms
// the original's own Go collector reads (bindings.ts handleGo):
//
//	var x T   /  var x *T   -> T
//	x := T{}                -> T
//	x := &T{}               -> T
//	x := NewT(...)          -> T   (a convention, not a resolution)
//
// Everything else binds no type here: a plain function's return type, a field
// access, a range variable, a channel receive. A call still binds itself
// (bindingOf), and resolve types its result; that is the one widening, and
// its reason is a chain like `x := blast.New(g); x.Reach()`.
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

// bindingOf is what `x := rhs` binds x to: a type boundType reads, or else
// the call whose first result x holds, for resolve to type.
func bindingOf(rhs ast.Expr, fset *token.FileSet, rel string, owner model.NodeID, sc *scope) binding {
	if typ := boundType(rhs); typ != "" {
		return binding{typ: typ}
	}
	if call, ok := rhs.(*ast.CallExpr); ok {
		if via, ok := callEdge(call, fset, rel, owner, sc); ok {
			return binding{via: &via}
		}
	}
	return binding{}
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
