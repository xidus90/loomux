// Package golang extracts the nodes and raw edges of one Go file.
//
// Raw, because a call's target is a name here and not yet a node: resolving it
// needs every file of the repository, and that is internal/code/resolve's job.
// The split is the original's (src/graph/extract.ts against
// src/graph/resolve.ts) and it is what keeps this package free of any
// knowledge about the repository.
//
// go/parser alone, no go/types: cross-package resolution belongs to the
// optional --lsp stage against gopls. See section 3.5 of the G2 spec (in the
// working papers of the archive release `archive/parity-recordings`).
//
// Ported from src/graph/extract.ts (describeGo) (MIT; origin under "Ported
// sources" in NOTICE.md).
package golang

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// Version is this extractor's identity. Bump it by hand whenever the nodes or
// edges below change shape or meaning.
//
// It reaches the graph's meta and the freshness record, and `check` compares it
// before diffing a single node. Without it a changed extractor matches the tree
// byte for byte, reports fresh, and keeps answering from nodes the old
// extractor built. cli.Version cannot serve: every development build says
// 0.0.0-dev.
//
// The extract cache keys every file's entry on it too. A forgotten bump
// therefore survives even an explicit `graph build` -- the cached entries
// still match and are reused -- and only `graph build --no-reuse` clears
// them. So every change to the nodes or edges comes with a bump.
const Version = "go/1"

// Language is the Go extractor as extract.Language describes one.
type Language struct{}

// Name is "go".
func (Language) Name() string { return "go" }

// Version is this package's Version.
func (Language) Version() string { return Version }

// Extensions is the one extension go/parser reads.
func (Language) Extensions() []string { return []string{".go"} }

// File is this package's File.
func (Language) File(rel, source string) (extract.Result, error) { return File(rel, source) }

// File extracts one Go file. rel is its repo-relative, slash-separated path;
// source is its contents, which the caller has already read in order to hash
// it.
func File(rel, source string) (extract.Result, error) {
	fset := token.NewFileSet()
	// SkipObjectResolution: ast.Object and File.Unresolved are deprecated as of
	// Go 1.22, and this package tracks scopes itself (scope.go). Skipping also
	// costs less -- 44-46ms against 58-59ms over this repository.
	file, err := parser.ParseFile(fset, rel, source, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return extract.Result{}, fmt.Errorf("parse %s: %w", rel, err)
	}

	r := extract.Result{Path: rel, Language: "go", Package: file.Name.Name, Imports: importsOf(file)}
	minted := map[string]bool{}
	fileID := model.NodeID(rel)

	covered := map[int]bool{} // 1-based lines a symbol span covers
	// owners maps a function declaration to the id its node was MINTED with.
	// The call walk must not recompute that id: the mint may have appended an
	// ordinal, and a second computation would produce an id no node has.
	owners := map[*ast.FuncDecl]model.NodeID{}
	for _, decl := range file.Decls {
		nodes := declNodes(fset, rel, source, decl, minted)
		if fn, ok := decl.(*ast.FuncDecl); ok && len(nodes) == 1 {
			owners[fn] = nodes[0].ID
		}
		for _, n := range nodes {
			r.Nodes = append(r.Nodes, n)
			r.Edges = append(r.Edges, extract.RawEdge{
				Source: fileID, Relation: model.RelationContains,
				TargetID: n.ID, File: rel,
			})
			markCovered(covered, n.Span)
		}
	}

	r.Nodes = append([]model.Node{extract.FileNode(rel, source, covered)}, r.Nodes...)
	r.Edges = append(r.Edges, importEdges(rel, file)...)
	r.Edges = append(r.Edges, callEdges(rel, file, owners)...)
	return r, nil
}

// importsOf is every import of a file, alias exactly as written.
//
// "_" and "." are recorded with their alias as written: they bind no selector,
// and the resolver has to be able to tell "this import binds nothing" from
// "this import binds its package clause".
func importsOf(file *ast.File) []extract.Import {
	var out []extract.Import
	for _, spec := range file.Imports {
		p, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		imp := extract.Import{Path: p}
		if spec.Name != nil {
			imp.Alias = spec.Name.Name
		}
		out = append(out, imp)
	}
	return out
}

// markCovered records the lines of a span as belonging to a symbol.
//
// The span is parsed by model.Span.Lines and not by a copy here: the extractor
// writes that form and the query reads it, and two parsers for one string are
// one too many.
func markCovered(covered map[int]bool, span model.Span) {
	from, to, ok := span.Lines()
	if !ok {
		return
	}
	for i := from; i <= to; i++ {
		covered[i] = true
	}
}

// declNodes turns one top-level declaration into its nodes.
//
// A grouped `type ( ... )` yields one node per spec and never one for the
// declaration: hashing the GenDecl would make a change to one type change the
// hash of every other in the group, and `check` would report them all.
func declNodes(fset *token.FileSet, rel, source string, decl ast.Decl, minted map[string]bool) []model.Node {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		return []model.Node{funcNode(fset, rel, source, d, minted)}
	case *ast.GenDecl:
		if d.Tok != token.TYPE {
			// No const and no var nodes: the original emits none for Go, and a const
			// block reaches a query through the file node's residual instead.
			return nil
		}
		var out []model.Node
		for _, spec := range d.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			out = append(out, typeNode(fset, rel, source, ts, minted))
		}
		return out
	default:
		return nil
	}
}

// funcNode is a function or a method.
func funcNode(fset *token.FileSet, rel, source string, d *ast.FuncDecl, minted map[string]bool) model.Node {
	name := d.Name.Name
	kind := model.Kind("function")
	owner := ""
	qualified := name
	if d.Recv != nil {
		kind = "method"
		owner = receiverType(d.Recv)
		if owner != "" {
			// Methods do not nest in Go, so the id qualifies by the receiver or
			// two receivers' same-named methods would collide. The name stays
			// bare, so a call `u.Save()` can hit it.
			qualified = owner + "." + name
		}
	}
	// The header ends where the body opens; a body-less declaration is its own
	// header.
	headerEnd := d.End()
	if d.Body != nil {
		headerEnd = d.Body.Pos()
	}
	return model.Node{
		ID:        model.NodeID(extract.MintID(rel+"#"+qualified, minted)),
		Name:      name,
		Kind:      kind,
		Owner:     owner,
		Path:      rel,
		Span:      span(fset, d.Pos(), d.End()),
		Signature: extract.Collapse(slice(fset, source, d.Pos(), headerEnd)),
		Exported:  exported(name),
		BodyHash:  extract.Hash(slice(fset, source, d.Pos(), d.End())),
		BodyText:  extract.Collapse(slice(fset, source, d.Pos(), d.End())),
	}
}

// typeNode is one TypeSpec: a struct, an interface or a named type.
//
// The signature deviates from the original's code and follows the original's
// comment. The original's type_spec starts at the NAME and its header ends at
// the `struct` keyword, so its signature for `type Cache struct { ... }` is
// "type Cache struct" -- a value no test of the reference pins. See 5.2.1 of
// the G2 spec (in the working papers of the archive release
// `archive/parity-recordings`).
func typeNode(fset *token.FileSet, rel, source string, ts *ast.TypeSpec, minted map[string]bool) model.Node {
	kind := model.Kind("type")
	sig := "type " + extract.Collapse(slice(fset, source, ts.Pos(), ts.End()))
	switch ts.Type.(type) {
	case *ast.StructType:
		kind = "struct"
		sig = "type " + ts.Name.Name + " struct"
	case *ast.InterfaceType:
		kind = "interface"
		sig = "type " + ts.Name.Name + " interface"
	}
	return model.Node{
		ID:        model.NodeID(extract.MintID(rel+"#"+ts.Name.Name, minted)),
		Name:      ts.Name.Name,
		Kind:      kind,
		Path:      rel,
		Span:      span(fset, ts.Pos(), ts.End()),
		Signature: sig,
		Exported:  exported(ts.Name.Name),
		BodyHash:  extract.Hash(slice(fset, source, ts.Pos(), ts.End())),
		BodyText:  extract.Collapse(slice(fset, source, ts.Pos(), ts.End())),
	}
}

// span is "L<from>-L<to>", 1-based, over a node's whole extent.
func span(fset *token.FileSet, from, to token.Pos) model.Span {
	return model.Span(fmt.Sprintf("L%d-L%d", fset.Position(from).Line, fset.Position(to).Line))
}

// slice is the source text between two positions.
func slice(fset *token.FileSet, source string, from, to token.Pos) string {
	a, b := fset.Position(from).Offset, fset.Position(to).Offset
	if a < 0 || b > len(source) || a > b {
		return ""
	}
	return source[a:b]
}

// importEdges is one edge per import specifier, from the file node.
//
// A blank import binds no selector and is still a dependency of the file, so it
// keeps its edge. A dot import binds no selector either; the same holds.
func importEdges(rel string, file *ast.File) []extract.RawEdge {
	var out []extract.RawEdge
	for _, imp := range importsOf(file) {
		out = append(out, extract.RawEdge{
			Source: model.NodeID(rel), Relation: model.RelationImports,
			Specifier: imp.Path, File: rel,
		})
	}
	return out
}

// callEdges walks every function body and the package-level initialisers and
// emits one raw edge per call site.
//
// Three shapes reach resolve, and the difference is what resolve is allowed to
// assume:
//
//   - a bare name          -- resolve tries the same file, then a unique match
//   - a name plus an owner -- a member call on a known local type
//   - a name plus a specifier -- a package selector; resolve looks in that
//     package alone
func callEdges(rel string, file *ast.File, owners map[*ast.FuncDecl]model.NodeID) []extract.RawEdge {
	var out []extract.RawEdge
	// A call outside every function -- in a var initialiser -- is owned by the
	// file, the same node that owns the imports.
	walkCalls(file, rel, model.NodeID(rel), owners, &out)
	return out
}

// walkCalls descends the file, keeping a scope stack and the symbol a call
// belongs to.
func walkCalls(file *ast.File, rel string, fileID model.NodeID, owners map[*ast.FuncDecl]model.NodeID, out *[]extract.RawEdge) {
	sc := newScope()
	// The file's own package-level names: a call may target one of them, and a
	// local of the same name must shadow it.
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil {
				sc.declare(d.Name.Name, "")
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					sc.declare(s.Name.Name, "")
				case *ast.ValueSpec:
					for _, n := range s.Names {
						sc.declare(n.Name, typeName(s.Type))
					}
				}
			}
		}
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			// A package-level initialiser: its calls belong to the file.
			collect(decl, rel, fileID, sc, out)
			continue
		}
		owner, ok := owners[fn]
		if !ok {
			// No node was minted for this declaration, so nothing can own its
			// calls. Attributing them to the file would invent a caller.
			continue
		}
		sc.push()
		// The receiver variable is bound to its own type: that is how `c.Get()`
		// inside a method of *Cache finds Cache.
		if fn.Recv != nil {
			sc.declare(receiverVar(fn.Recv), receiverType(fn.Recv))
		}
		declareParams(sc, fn.Type)
		collect(fn, rel, owner, sc, out)
		sc.pop()
	}
}

// declareParams puts a function's parameters and named results in view. Their
// types are recorded, so a member call on a parameter resolves.
func declareParams(sc *scope, ft *ast.FuncType) {
	for _, list := range []*ast.FieldList{ft.Params, ft.Results, ft.TypeParams} {
		if list == nil {
			continue
		}
		for _, field := range list.List {
			for _, name := range field.Names {
				sc.declare(name.Name, typeName(field.Type))
			}
		}
	}
}

// collect walks one declaration's statements, tracking declarations as it goes
// and emitting a raw edge per call.
func collect(node ast.Node, rel string, owner model.NodeID, sc *scope, out *[]extract.RawEdge) {
	ast.Inspect(node, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.AssignStmt:
			if s.Tok == token.DEFINE {
				for i, lhs := range s.Lhs {
					id, ok := lhs.(*ast.Ident)
					if !ok {
						continue
					}
					typ := ""
					if i < len(s.Rhs) {
						typ = boundType(s.Rhs[i])
					}
					sc.declare(id.Name, typ)
				}
			}
		case *ast.ValueSpec:
			for _, name := range s.Names {
				sc.declare(name.Name, typeName(s.Type))
			}
		case *ast.RangeStmt:
			// A range variable binds no type this package can read, but it must
			// shadow: `for store := range m` hides the package `store`.
			for _, e := range []ast.Expr{s.Key, s.Value} {
				if id, ok := e.(*ast.Ident); ok {
					sc.declare(id.Name, "")
				}
			}
		case *ast.FuncLit:
			// A closure's parameters and named results must shadow too, the same
			// as a top-level function's: `func(model *Model) { model.Decode() }`
			// must not read `model` as the package it shadows. Declared into the
			// current frame and never popped -- this package is function-scoped
			// throughout, and over-shadowing past the literal's own end is the
			// same safe trade RangeStmt and AssignStmt already make: it can only
			// drop a later edge, never invent one.
			declareParams(sc, s.Type)
		case *ast.CallExpr:
			if e, ok := callEdge(s, rel, owner, sc); ok {
				*out = append(*out, e)
			}
		}
		return true
	})
}

// callEdge reads one call site.
//
// Shadowing is decided HERE, because only this package sees the file's scopes;
// whether an unshadowed receiver names a package is decided in resolve, because
// only that sees the target's package clause. Splitting the question along that
// line is what keeps `import "gopkg.in/yaml.v3"` from binding `v3`.
func callEdge(call *ast.CallExpr, rel string, owner model.NodeID, sc *scope) (extract.RawEdge, bool) {
	switch fn := peelIndex(call.Fun).(type) {
	case *ast.Ident:
		return extract.RawEdge{
			Source: owner, Relation: model.RelationCalls,
			Name: fn.Name, File: rel,
		}, true
	case *ast.SelectorExpr:
		recv, ok := peelIndex(fn.X).(*ast.Ident)
		if !ok {
			// A chained or computed receiver -- `a.b().c()`, `m[k].c()`. The original
			// drops these too: without a receiver type a bare method name says
			// nothing about what it belongs to.
			return extract.RawEdge{}, false
		}
		typ, ok := sc.lookup(recv.Name)
		if !ok {
			// Declared nowhere in view. It may be a package, and resolve is the
			// only side that can say so.
			return extract.RawEdge{
				Source: owner, Relation: model.RelationCalls,
				Name: fn.Sel.Name, Receiver: recv.Name, File: rel,
			}, true
		}
		if typ == "" {
			// Declared, but bound to nothing this package reads. Dropping beats
			// guessing: a unique bare method name says nothing about its
			// receiver.
			return extract.RawEdge{}, false
		}
		return extract.RawEdge{
			Source: owner, Relation: model.RelationCalls,
			Name: fn.Sel.Name, Owner: typ, File: rel,
		}, true
	default:
		return extract.RawEdge{}, false
	}
}
