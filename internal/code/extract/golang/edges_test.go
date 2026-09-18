package golang_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/extract/golang"
	"github.com/xidus90/loomux/internal/code/model"
)

func edgesOf(t *testing.T, r golang.Result, rel model.Relation) []golang.RawEdge {
	t.Helper()
	var out []golang.RawEdge
	for _, e := range r.Edges {
		if e.Relation == rel {
			out = append(out, e)
		}
	}
	return out
}

func hasEdge(edges []golang.RawEdge, want golang.RawEdge) bool {
	for _, e := range edges {
		if e.Source == want.Source && e.Name == want.Name &&
			e.Owner == want.Owner && e.Receiver == want.Receiver {
			return true
		}
	}
	return false
}

func TestFileEmitsContainsFromTheFileToEverySymbol(t *testing.T) {
	r, err := golang.File("pkg/edges.go", fixture(t, "edges"))
	if err != nil {
		t.Fatal(err)
	}
	contains := edgesOf(t, r, model.RelationContains)
	// One per symbol node, never one for the file itself.
	if len(contains) != len(r.Nodes)-1 {
		t.Fatalf("got %d contains edges for %d symbol nodes", len(contains), len(r.Nodes)-1)
	}
	for _, e := range contains {
		if e.Source != "pkg/edges.go" {
			t.Errorf("contains edge from %q, want the file node", e.Source)
		}
		if e.TargetID == "" {
			t.Error("a contains edge carries a resolved target id, not a name")
		}
	}
}

func TestFileEmitsOneImportEdgePerSpecifier(t *testing.T) {
	r, err := golang.File("pkg/edges.go", fixture(t, "edges"))
	if err != nil {
		t.Fatal(err)
	}
	imports := edgesOf(t, r, model.RelationImports)
	want := map[string]bool{
		"fmt":                     false,
		"example.com/repo/blast":  false,
		"example.com/repo/store":  false,
		"example.com/repo/driver": false,
	}
	for _, e := range imports {
		if e.Source != "pkg/edges.go" {
			t.Errorf("import edge from %q, want the file node", e.Source)
		}
		if _, ok := want[e.Specifier]; !ok {
			t.Errorf("unexpected import %q", e.Specifier)
			continue
		}
		want[e.Specifier] = true
	}
	for spec, seen := range want {
		if !seen {
			// A blank import binds no selector but is still a dependency of the
			// file, so it keeps its edge.
			t.Errorf("import %q missing", spec)
		}
	}
}

func TestFileResolvesACallOnTheMethodsOwnReceiver(t *testing.T) {
	r, err := golang.File("pkg/edges.go", fixture(t, "edges"))
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	// `c.Get()` inside a method of *Cache: the receiver variable is known, so
	// the edge carries the owner and resolve can find the right method.
	if !hasEdge(calls, golang.RawEdge{Source: "pkg/edges.go#Cache.Warm", Name: "Get", Owner: "Cache"}) {
		t.Errorf("a call on the own receiver must carry its owner; got %+v", calls)
	}
}

func TestFileEmitsABareCallWithoutAnOwner(t *testing.T) {
	r, err := golang.File("pkg/edges.go", fixture(t, "edges"))
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	if !hasEdge(calls, golang.RawEdge{Source: "pkg/edges.go#caller", Name: "local"}) {
		t.Errorf("a bare call must be a raw edge with a name alone; got %+v", calls)
	}
}

func TestFileCarriesAnUnshadowedSelectorReceiverUnresolved(t *testing.T) {
	r, err := golang.File("pkg/edges.go", fixture(t, "edges"))
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	// The extractor does NOT decide that `b` is a package: the name a plain
	// import binds is the TARGET's package clause, and one file cannot see it.
	// `import "gopkg.in/yaml.v3"` binds `yaml`, not `v3`, and guessing here
	// would put the guess where nothing can correct it.
	for _, want := range []golang.RawEdge{
		{Source: "pkg/edges.go#caller", Name: "New", Receiver: "b"},
		{Source: "pkg/edges.go#caller", Name: "Open", Receiver: "store"},
		{Source: "pkg/edges.go#caller", Name: "Println", Receiver: "fmt"},
	} {
		if !hasEdge(calls, want) {
			t.Errorf("selector %s.%s must reach resolve unresolved; got %+v",
				want.Receiver, want.Name, calls)
		}
	}
	for _, e := range calls {
		if e.Specifier != "" {
			t.Errorf("a call edge carries no specifier; that is the resolver's job: %+v", e)
		}
	}
}

func TestFileRecordsThePackageClauseAndEveryImport(t *testing.T) {
	r, err := golang.File("pkg/edges.go", fixture(t, "edges"))
	if err != nil {
		t.Fatal(err)
	}
	if r.Package != "edges" {
		t.Errorf("Package = %q, want the file's clause", r.Package)
	}
	want := map[string]string{
		"fmt":                     "",
		"example.com/repo/blast":  "b",
		"example.com/repo/store":  "",
		"example.com/repo/driver": "_",
	}
	if len(r.Imports) != len(want) {
		t.Fatalf("got %+v, want %d imports", r.Imports, len(want))
	}
	for _, imp := range r.Imports {
		alias, ok := want[imp.Path]
		if !ok {
			t.Errorf("unexpected import %+v", imp)
			continue
		}
		// The alias is raw, "_" included: the resolver has to tell "binds
		// nothing" from "binds its package clause".
		if imp.Alias != alias {
			t.Errorf("import %q alias = %q, want %q", imp.Path, imp.Alias, alias)
		}
	}
}

func TestFileTreatsAShadowedPackageNameAsAVariable(t *testing.T) {
	r, err := golang.File("pkg/edges.go", fixture(t, "edges"))
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	// After `store := Cache{}` the name is a local variable. Go code does this
	// constantly (`model := model.Decode(r)`), and letting the selector reach
	// resolve as a receiver-with-no-owner would wire a call into a package the
	// line never touches.
	for _, e := range calls {
		if e.Name == "Get" && e.Receiver != "" {
			t.Errorf("a shadowed name is a value, not a receiver to resolve; got %+v", e)
		}
	}
	// It is a member call on a known local type instead.
	if !hasEdge(calls, golang.RawEdge{Source: "pkg/edges.go#caller", Name: "Get", Owner: "Cache"}) {
		t.Errorf("the shadowed call must resolve against the local binding; got %+v", calls)
	}
}

func TestFilePeelsATypeArgumentList(t *testing.T) {
	r, err := golang.File("pkg/edges.go", fixture(t, "edges"))
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	// `b.Of[int](3)` must take the same path as `b.Of(3)`.
	if !hasEdge(calls, golang.RawEdge{
		Source: "pkg/edges.go#generic", Name: "Of", Receiver: "b",
	}) {
		t.Errorf("a generic call must peel its type arguments; got %+v", calls)
	}
}

func TestFileBindsALocalVariableOnlyInTheFourFormsGraftKnows(t *testing.T) {
	// var x T, x := T{}, x := &T{}, x := NewT(...) -- and nothing else. The
	// fourth is a convention, not a resolution. Widening this set widens the
	// set of call edges and owes its own reason.
	const src = `package p

type T struct{}

func (t T) M() {}

func NewT() T { return T{} }

func other() T { return T{} }

func f() {
	var a T
	a.M()
	b := T{}
	b.M()
	c := &T{}
	c.M()
	d := NewT()
	d.M()
	e := other()
	e.M()
}
`
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	owned := 0
	for _, e := range calls {
		if e.Name == "M" && e.Owner == "T" {
			owned++
		}
	}
	// a, b, c, d carry the owner; e does not -- a plain function's return type
	// is not something this extractor knows.
	if owned != 4 {
		t.Errorf("got %d owned calls of M, want 4; edges %+v", owned, calls)
	}
}

func TestFileAttributesACallToTheEnclosingSymbol(t *testing.T) {
	const src = "package p\n\nfunc a() {}\n\nfunc b() { a() }\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	if len(calls) != 1 || calls[0].Source != "p.go#b" {
		t.Fatalf("got %+v, want one call whose source is the calling function", calls)
	}
}

func TestFileAttributesAPackageLevelCallToTheFile(t *testing.T) {
	// A call in a var initialiser sits inside no function, so the file owns it.
	const src = "package p\n\nfunc a() int { return 1 }\n\nvar x = a()\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	if len(calls) != 1 || calls[0].Source != "p.go" {
		t.Fatalf("got %+v, want one call owned by the file node", calls)
	}
}

func TestFileEmitsNeitherHeritageNorReferenceEdges(t *testing.T) {
	const src = `package p

type Reader interface{ Read() string }

type Impl struct{ Reader }

func (i Impl) Read() string { return "" }
`
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	// Go has no explicit implements, and an embedded interface is not one
	// either. Graft emits heritage edges for kind:class and the JVM/Swift
	// types alone, and reference edges only where an import binds a symbol --
	// a collector it has for TypeScript and PHP, not for Go.
	for _, e := range r.Edges {
		switch e.Relation {
		case model.RelationExtends, model.RelationImplements, model.RelationReferences:
			t.Errorf("this extractor emits no %q edges; got %+v", e.Relation, e)
		}
	}
}

func TestFileDropsACallOnAChainedReceiver(t *testing.T) {
	const src = "package p\n\nfunc a() T { return T{} }\n\ntype T struct{}\n\nfunc (T) M() {}\n\nfunc f() { a().M() }\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	// `a().M()` has a call, not an identifier, as its receiver expression: a
	// bare method name off it says nothing about what it belongs to.
	for _, e := range r.Edges {
		if e.Relation == model.RelationCalls && e.Name == "M" {
			t.Errorf("a chained receiver must yield no call edge for M; got %+v", e)
		}
	}
}

func TestFileDropsACallThroughAFunctionLiteral(t *testing.T) {
	const src = "package p\n\nfunc f() { func() { }() }\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	// An immediately invoked function literal is neither a bare name nor a
	// selector; callEdge's default arm drops it.
	if len(calls) != 0 {
		t.Errorf("got %+v, want no call edges for an invoked function literal", calls)
	}
}

func TestFileResolvesACallOnAPointerParameter(t *testing.T) {
	const src = "package p\n\ntype Cache struct{}\n\nfunc (c *Cache) Get() {}\n\nfunc f(p *Cache) { p.Get() }\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	if !hasEdge(calls, golang.RawEdge{Source: "p.go#f", Name: "Get", Owner: "Cache"}) {
		t.Errorf("a call on a pointer parameter must carry its owner; got %+v", calls)
	}
}

func TestFileShadowsAPackageThroughARangeVariable(t *testing.T) {
	const src = "package p\n\nfunc local() int { return 1 }\n\nfunc f(store map[string]int) {\n\tfor store := range store {\n\t\t_ = local()\n\t}\n}\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	if !hasEdge(calls, golang.RawEdge{Source: "p.go#f", Name: "local"}) {
		t.Errorf("got %+v, want the bare call inside the range body", calls)
	}
}

func TestFilePeelsATwoArgumentTypeList(t *testing.T) {
	const src = "package p\n\nfunc f() { b.Of2[int, string](3) }\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	if !hasEdge(calls, golang.RawEdge{Source: "p.go#f", Name: "Of2", Receiver: "b"}) {
		t.Errorf("a two-argument type list must be peeled too; got %+v", calls)
	}
}

func TestFileDropsAMemberCallOnAnUnexportedConstructorConvention(t *testing.T) {
	const src = "package p\n\ntype T struct{}\n\nfunc Newfoo() T { return T{} }\n\nfunc f() {\n\tx := Newfoo()\n\tx.M()\n}\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	// "Newfoo" does not fit the New<Exported> convention -- its rest, "foo", is
	// unexported -- so x binds no type, and the later call on it is dropped.
	for _, e := range calls {
		if e.Name == "M" {
			t.Errorf("a call bound through an unexported constructor name must be dropped; got %+v", e)
		}
	}
}

func TestFileDropsACallOnAComputedReceiver(t *testing.T) {
	const src = "package p\n\nfunc f(m map[string]T) { m[\"k\"].M() }\n\ntype T struct{}\n\nfunc (T) M() {}\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	// Without a receiver type a bare method name says nothing about what it
	// belongs to. Graft drops it; so does this.
	for _, e := range r.Edges {
		if e.Relation == model.RelationCalls && e.Name == "M" {
			t.Errorf("a computed receiver must yield no call edge; got %+v", e)
		}
	}
}

func TestFileDoesNotDeclareAMethodsBareNameInPackageScope(t *testing.T) {
	// A package-level var named the same as a later method must survive the
	// method's own declaration walk: only walkCalls' package-scope pass over
	// non-method FuncDecls may touch that name, and a method (Recv != nil) is
	// not one of those.
	const src = "package p\n\ntype Cache struct{}\n\nvar Get *Cache\n\nfunc (c *Cache) Get() {}\n\nfunc caller() { Get.Foo() }\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	if !hasEdge(calls, golang.RawEdge{Source: "p.go#caller", Name: "Foo", Owner: "Cache"}) {
		t.Errorf("Get must still resolve to the package var's type Cache, not be overwritten by the method's own bare name; got %+v", calls)
	}
}

func TestFileDeclaresAPlainFunctionsBareNameSoASelectorOnItIsDropped(t *testing.T) {
	// Store is a plain function (Recv == nil): the package-scope pass records
	// it with an unknown ("") type, so a later selector on that exact name is
	// recognised as "declared, bound to nothing readable" and dropped -- not
	// kept as an unresolved receiver that might be a package.
	const src = "package p\n\nfunc Store() {}\n\nfunc caller() { Store.Load() }\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	for _, e := range calls {
		if e.Name == "Load" {
			t.Errorf("a selector on a declared package-level function name must be dropped, not kept as an unresolved receiver; got %+v", e)
		}
	}
}

func TestFileRebindsALocalOnlyOnADefiningAssignment(t *testing.T) {
	// `x = Impl{}` is a plain assignment, not `:=`: it must not re-bind x's
	// declared type, or `x.Do()` would carry the wrong Owner.
	const src = "package p\n\ntype I struct{}\n\ntype Impl struct{}\n\nfunc (i Impl) Do() {}\n\nfunc caller() {\n\tvar x I\n\tx = Impl{}\n\tx.Do()\n}\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	if !hasEdge(calls, golang.RawEdge{Source: "p.go#caller", Name: "Do", Owner: "I"}) {
		t.Errorf("a plain assignment must not rebind x's declared type; got %+v", calls)
	}
}

func TestFileDoesNotReadPastTheRightHandSideOfAMultiValueDefine(t *testing.T) {
	// `a, b := f()` has one Rhs expression for two Lhs names: reading Rhs[1]
	// for b would run past the slice.
	const src = "package p\n\nfunc f() (int, int) { return 1, 2 }\n\nfunc caller() {\n\ta, b := f()\n\t_ = a\n\t_ = b\n}\n"
	if _, err := golang.File("p.go", src); err != nil {
		t.Fatal(err)
	}
}

func TestFileResolvesAPackageLevelReceiverFromTheOutermostScopeFrame(t *testing.T) {
	// c is declared once, at package level (frame 0). caller pushes its own
	// frame (frame 1) and declares nothing named c, so the lookup must walk
	// all the way back down to frame 0.
	const src = "package p\n\ntype Cache struct{}\n\nfunc (c *Cache) Get() {}\n\nvar c *Cache\n\nfunc caller() { c.Get() }\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	if !hasEdge(calls, golang.RawEdge{Source: "p.go#caller", Name: "Get", Owner: "Cache"}) {
		t.Errorf("a name declared only at package level must still resolve from inside a function; got %+v", calls)
	}
}

func TestFileBindsAConstructorOnlyWhenTheWholeNewPrefixMatches(t *testing.T) {
	// FooBar does not start with "New": binding "Bar" off a length check alone,
	// without checking the prefix itself, would be wrong.
	const src = "package p\n\nfunc FooBar() T { return T{} }\n\ntype T struct{}\n\nfunc (T) Do() {}\n\nfunc caller() {\n\tx := FooBar()\n\tx.Do()\n}\n"
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	for _, e := range calls {
		if e.Name == "Do" {
			t.Errorf("FooBar is not a New<Type> constructor; x must stay unbound and the call dropped, got %+v", e)
		}
	}
}

func TestFileDoesNotReadPastAShortConstructorCandidate(t *testing.T) {
	// "Ne" is shorter than "New": reading its would-be suffix, or slicing it
	// against "New"'s own length, must not run past the string.
	const src = "package p\n\nfunc Ne() T { return T{} }\n\ntype T struct{}\n\nfunc caller() {\n\tx := Ne()\n\t_ = x\n}\n"
	if _, err := golang.File("p.go", src); err != nil {
		t.Fatal(err)
	}
}
