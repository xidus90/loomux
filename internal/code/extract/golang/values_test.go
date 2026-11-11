package golang_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/golang"
	"github.com/xidus90/loomux/internal/code/model"
)

func mustFile(t *testing.T, src string) extract.Result {
	t.Helper()
	r, err := golang.File("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func names(edges []extract.RawEdge) map[string]int {
	out := map[string]int{}
	for _, e := range edges {
		out[string(e.Source)+">"+e.Receiver+"."+e.Owner+"."+e.Name]++
	}
	return out
}

func TestFileReferencesAFunctionInEveryValuePosition(t *testing.T) {
	r := mustFile(t, `package p

import "example.com/q"

type Deps struct{ F func() }

func target() {}

var hook = q.Run

func use(f func()) {}

func wire() Deps {
	use(target)
	d := Deps{F: target}
	_ = d
	fs := []func(){target}
	_ = fs
	var g = q.Run
	_ = g
	return Deps{F: q.Run}
}
`)
	got := names(edgesOf(t, r, model.RelationReferences))
	for key, n := range map[string]int{
		"p.go>q..Run":        1, // the package-level var
		"p.go#wire>..target": 3, // argument, key-value value, slice element
		"p.go#wire>q..Run":   2, // var spec, return composite
	} {
		if got[key] != n {
			t.Errorf("references %s = %d, want %d; all %v", key, got[key], n, got)
		}
	}
	for key := range got {
		switch key {
		case "p.go#wire>..F", "p.go#wire>..Deps", "p.go#wire>..d", "p.go#wire>..fs", "p.go#wire>..g":
			t.Errorf("%s is no function value and must not be referenced", key)
		}
	}
}

func TestFileSkipsALocalAndAUniverseNameAsAValue(t *testing.T) {
	r := mustFile(t, `package p

func target() {}

func f(target func()) {
	x := 1
	use(x, nil, true, target)
}

func use(...any) {}
`)
	// The parameter shadows the package function: a local is never a
	// function reference. nil and true are builtins, not functions of the
	// package.
	if refs := edgesOf(t, r, model.RelationReferences); len(refs) != 0 {
		t.Errorf("want no references, got %+v", refs)
	}
}

func TestFileReferencesNeitherAPackageVariableNorACallNorALiteral(t *testing.T) {
	r := mustFile(t, `package p

var limit = 3

type T struct{}

func (T) M() {}

func mk() func() { return nil }

func f(p *func() T, s struct{ t T }) {
	use(limit, mk(), func() {})
	(*p)().M()
	s.t.M()
}

func use(...any) {}
`)
	// limit is the file's own variable, mk() is a call (collect emits it as
	// one), a function literal names nothing, and `(*p)()` is an inner call
	// this package cannot read.
	if refs := edgesOf(t, r, model.RelationReferences); len(refs) != 0 {
		t.Errorf("want no references, got %+v", refs)
	}
	for _, e := range edgesOf(t, r, model.RelationCalls) {
		if e.Name == "M" {
			t.Errorf("a member on a computed inner call or a field chain must be dropped; got %+v", e)
		}
	}
}

func TestFileDropsABareCallOfALocal(t *testing.T) {
	r := mustFile(t, `package p

type T struct{}

func (T) Run() {}

func mk() T { return T{} }

func Of[K any]() T { return T{} }

func f(m map[string]func() T, fs []func()) {
	mk := func() T { return T{} }
	mk()
	mk().Run()
	m["k"]()
	m["k"]().Run()
	fs[0]()
	Of[int]()
}
`)
	// mk, m and fs are locals: a bare call through one names no function of
	// the package, and neither does a member on its result. Of[int] is a
	// generic package function and stays.
	calls := edgesOf(t, r, model.RelationCalls)
	for _, e := range calls {
		if e.Name == "mk" || e.Name == "m" || e.Name == "fs" || e.Name == "Run" {
			t.Errorf("a call through a local must be dropped; got %+v via %+v", e, e.Via)
		}
	}
	if !hasEdge(calls, extract.RawEdge{Source: "p.go#f", Name: "Of"}) {
		t.Errorf("a generic package function must stay; got %+v", calls)
	}
}

func TestFileReferencesAMethodValueOnAKnownReceiver(t *testing.T) {
	r := mustFile(t, `package p

type S struct{}

func (s *S) handle() {}

func (s *S) wire() { register(s.handle) }

func register(func()) {}
`)
	if !hasEdge(edgesOf(t, r, model.RelationReferences), extract.RawEdge{Source: "p.go#S.wire", Name: "handle", Owner: "S"}) {
		t.Errorf("a method value on a typed receiver must reference the method; got %+v", r.Edges)
	}
}

func TestFileCarriesTheInnerCallOfAChainedReceiver(t *testing.T) {
	r := mustFile(t, `package p

import "example.com/blast"

func f() { blast.New(1).Reach(2) }
`)
	var chained *extract.RawEdge
	for _, e := range edgesOf(t, r, model.RelationCalls) {
		if e.Name == "Reach" {
			chained = &e
		}
	}
	if chained == nil || chained.Via == nil {
		t.Fatalf("a chained call must carry its inner call; got %+v", r.Edges)
	}
	if chained.Via.Name != "New" || chained.Via.Receiver != "blast" || chained.Owner != "" || chained.Receiver != "" {
		t.Errorf("chained edge = %+v, via %+v; want Reach via blast.New", chained, chained.Via)
	}
}

func TestFileReadsTheRightHandCallBeforeTheNameItBinds(t *testing.T) {
	r := mustFile(t, `package p

import "example.com/graph"

func f(g int) {
	graph := graph.New(g)
	graph.Run()
}
`)
	calls := edgesOf(t, r, model.RelationCalls)
	// The call on the right still sees the package; only what follows sees
	// the variable.
	if !hasEdge(calls, extract.RawEdge{Source: "p.go#f", Name: "New", Receiver: "graph"}) {
		t.Errorf("graph.New must reach resolve as a package selector; got %+v", calls)
	}
	for _, e := range calls {
		if e.Name == "New" && e.Via != nil {
			t.Errorf("graph.New must not be read as a member of its own result; got %+v via %+v", e, e.Via)
		}
		if e.Name == "Run" && (e.Via == nil || e.Via.Name != "New") {
			t.Errorf("graph.Run must carry the binding call New; got %+v", e)
		}
	}
}

func TestFileCarriesTheInnerCallOfABoundResult(t *testing.T) {
	r := mustFile(t, `package p

import "example.com/blast"

func f() {
	x, err := blast.New(1)
	_ = err
	x.Reach(2)
}
`)
	for _, e := range edgesOf(t, r, model.RelationCalls) {
		if e.Name == "Reach" {
			if e.Via == nil || e.Via.Name != "New" || e.Via.Receiver != "blast" {
				t.Errorf("a call on a bound result must carry the binding call; got %+v via %+v", e, e.Via)
			}
			return
		}
	}
	t.Errorf("no call edge for Reach; got %+v", r.Edges)
}
