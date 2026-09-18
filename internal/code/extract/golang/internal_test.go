package golang

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
)

// These tests reach into unexported helpers directly. Each covers a branch
// that a well-formed Go source, parsed by go/parser, never drives: go/parser
// only ever hands this package ASTs it built itself, so the defensive arms
// below guard shapes File never actually produces.

func TestItoaZero(t *testing.T) {
	// mintID only ever calls itoa with k starting at 2, so the n == 0 arm is
	// unreachable through File. Direct call is the only way to it.
	if got := itoa(0); got != "0" {
		t.Errorf("itoa(0) = %q, want %q", got, "0")
	}
}

func TestExportedSplitsOnTheLastDot(t *testing.T) {
	// funcNode always calls exported with the bare name, never the qualified
	// "Owner.Name" id, so this arm is unreachable through File.
	if !exported("User.Save") {
		t.Error("exported(\"User.Save\") = false, want true: the part after the dot is exported")
	}
	if exported("User.save") {
		t.Error("exported(\"User.save\") = true, want false: the part after the dot is unexported")
	}
}

func TestExportedOnEmptyName(t *testing.T) {
	if exported("") {
		t.Error("exported(\"\") = true, want false")
	}
}

func TestReceiverTypeOnAnUnnamedOrMissingReceiver(t *testing.T) {
	if got := receiverType(nil); got != "" {
		t.Errorf("receiverType(nil) = %q, want empty", got)
	}
	if got := receiverType(&ast.FieldList{}); got != "" {
		t.Errorf("receiverType(empty list) = %q, want empty", got)
	}
}

func TestReceiverTypeOnGenericReceiver(t *testing.T) {
	// A single type parameter parses to an IndexExpr receiver type.
	one := &ast.FieldList{List: []*ast.Field{{
		Type: &ast.StarExpr{X: &ast.IndexExpr{
			X:     ast.NewIdent("Cache"),
			Index: ast.NewIdent("K"),
		}},
	}}}
	if got := receiverType(one); got != "Cache" {
		t.Errorf("receiverType(one type param) = %q, want %q", got, "Cache")
	}

	// Two or more type parameters parse to an IndexListExpr receiver type.
	two := &ast.FieldList{List: []*ast.Field{{
		Type: &ast.StarExpr{X: &ast.IndexListExpr{
			X:       ast.NewIdent("Cache"),
			Indices: []ast.Expr{ast.NewIdent("K"), ast.NewIdent("V")},
		}},
	}}}
	if got := receiverType(two); got != "Cache" {
		t.Errorf("receiverType(two type params) = %q, want %q", got, "Cache")
	}
}

func TestReceiverTypeOnUnreadableExpr(t *testing.T) {
	// A receiver type go/parser never actually produces (a selector, say),
	// so receiverType falls through to its empty-string default.
	sel := &ast.FieldList{List: []*ast.Field{{
		Type: &ast.SelectorExpr{X: ast.NewIdent("pkg"), Sel: ast.NewIdent("Foo")},
	}}}
	if got := receiverType(sel); got != "" {
		t.Errorf("receiverType(unreadable) = %q, want empty", got)
	}
}

func TestReceiverVar(t *testing.T) {
	// callEdges (Task 4) is the only production caller; nothing in this task
	// invokes receiverVar, so its arms are covered here directly.
	if got := receiverVar(nil); got != "" {
		t.Errorf("receiverVar(nil) = %q, want empty", got)
	}
	if got := receiverVar(&ast.FieldList{}); got != "" {
		t.Errorf("receiverVar(empty list) = %q, want empty", got)
	}
	unnamed := &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("User")}}}
	if got := receiverVar(unnamed); got != "" {
		t.Errorf("receiverVar(unnamed) = %q, want empty", got)
	}
	named := &ast.FieldList{List: []*ast.Field{{
		Names: []*ast.Ident{ast.NewIdent("u")},
		Type:  ast.NewIdent("User"),
	}}}
	if got := receiverVar(named); got != "u" {
		t.Errorf("receiverVar(named) = %q, want %q", got, "u")
	}
}

func TestImportsOfSkipsAnUnquotableLiteral(t *testing.T) {
	// go/parser never hands back an ImportSpec whose Path is not a validly
	// quoted string; this arm guards a shape it never actually produces.
	file := &ast.File{Imports: []*ast.ImportSpec{
		{Path: &ast.BasicLit{Kind: token.STRING, Value: "not-quoted"}},
	}}
	if got := importsOf(file); len(got) != 0 {
		t.Errorf("importsOf(unquotable) = %v, want none", got)
	}
}

func TestImportsOfKeepsAnAlias(t *testing.T) {
	file := &ast.File{Imports: []*ast.ImportSpec{
		{
			Name: ast.NewIdent("f"),
			Path: &ast.BasicLit{Kind: token.STRING, Value: `"fmt"`},
		},
	}}
	got := importsOf(file)
	if len(got) != 1 || got[0].Alias != "f" || got[0].Path != "fmt" {
		t.Errorf("importsOf(aliased) = %v, want one Import{Alias: f, Path: fmt}", got)
	}
}

func TestMarkCoveredOnAnUnparsableSpan(t *testing.T) {
	// The extractor only ever writes spans through span(), which always
	// parses; a caller passing anything else hits this defensive return.
	covered := map[int]bool{}
	markCovered(covered, model.Span("not-a-span"))
	if len(covered) != 0 {
		t.Errorf("markCovered(bad span) covered %v, want none", covered)
	}
}

func TestDeclNodesOnADeclKindParserRecoveryProduces(t *testing.T) {
	// File returns before the decl loop whenever go/parser reports an error,
	// so a *ast.BadDecl -- which only appears alongside such an error -- can
	// never reach declNodes through File. Reached here directly instead.
	if got := declNodes(token.NewFileSet(), "p.go", "", &ast.BadDecl{}, map[string]bool{}); got != nil {
		t.Errorf("declNodes(BadDecl) = %v, want nil", got)
	}
}

func TestDeclNodesSkipsANonTypeSpecInATypeGenDecl(t *testing.T) {
	// go/parser never puts anything but a *ast.TypeSpec in a TYPE GenDecl's
	// Specs; this arm guards a shape it never actually produces.
	decl := &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{&ast.ImportSpec{
		Path: &ast.BasicLit{Kind: token.STRING, Value: `"x"`},
	}}}
	if got := declNodes(token.NewFileSet(), "p.go", "", decl, map[string]bool{}); got != nil {
		t.Errorf("declNodes(non-TypeSpec in TYPE GenDecl) = %v, want nil", got)
	}
}

func TestWalkCallsSkipsAFuncDeclAbsentFromOwners(t *testing.T) {
	// File's own node pass mints exactly one node per FuncDecl today, so
	// owners always has an entry for a real one when walkCalls runs after it.
	// That is an invariant BETWEEN the two functions in this package, not a
	// grammar-level impossibility: a future dedup pass or an ordinal
	// collision in mintID's caller could leave a FuncDecl unminted, and this
	// guard is what keeps that case from inventing a caller for its calls.
	src := "package p\n\nfunc a() { b() }\n\nfunc b() {}\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	var out []RawEdge
	walkCalls(file, "p.go", model.NodeID("p.go"), map[*ast.FuncDecl]model.NodeID{}, &out)
	if len(out) != 0 {
		t.Errorf("walkCalls(no owners) = %+v, want no edges", out)
	}
}

func TestCollectSkipsANonIdentLhsOfADefineAssign(t *testing.T) {
	// go/parser only ever accepts plain identifiers on the left of `:=`; a
	// selector there is a shape it never actually produces.
	assign := &ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: []ast.Expr{&ast.SelectorExpr{X: ast.NewIdent("o"), Sel: ast.NewIdent("f")}},
		Rhs: []ast.Expr{ast.NewIdent("v")},
	}
	sc := newScope()
	var out []RawEdge
	collect(assign, "p.go", model.NodeID("p.go"), sc, &out)
	if len(out) != 0 {
		t.Errorf("collect(non-ident define lhs) = %+v, want no edges", out)
	}
}

func TestSliceOnAReversedRange(t *testing.T) {
	fset := token.NewFileSet()
	f := fset.AddFile("p.go", -1, 10)
	f.SetLinesForContent([]byte("0123456789"))
	// A reversed range never happens with real AST positions -- End() is
	// never before Pos() -- but slice guards against it defensively.
	if got := slice(fset, "0123456789", f.Pos(9), f.Pos(0)); got != "" {
		t.Errorf("slice(reversed) = %q, want empty", got)
	}
}
