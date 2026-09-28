package benchsearch

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
)

// questionsExpecting renders the default shape with the n-th question's
// expect taken from expect.
func questionsExpecting(expect func(n int) string) string {
	var b strings.Builder
	i := 0
	for _, k := range kinds {
		for range DefaultShape()[k] {
			i++
			fmt.Fprintf(&b, "- id: q%02d\n  sort: %s\n  query: \"q%d\"\n  expect: %q\n  beleg: \"the evidence\"\n",
				i, k, i, filepath.ToSlash(expect(i)))
		}
	}
	return b.String()
}

// pointedWorld is a knowledge area holding a question set whose expect
// paths lie wherever expect says, beside areas one and two.
func pointedWorld(t *testing.T, d Deps, expect func(one, two string, n int) string) *search.FakePort {
	t.Helper()
	w := everyday(t, d, "knowledge")
	one := addArea(t, d.StateDir, "one")
	two := addArea(t, d.StateDir, "two")
	writeYAML(t, w.measure, questionsExpecting(func(n int) string { return expect(one, two, n) }))
	port := answering("one", 50)
	for _, scope := range []string{"knowledge", "two"} {
		port.Listings[search.CollectionName(scope)] = search.ScriptedIndexed{Paths: []string{"a.md"}}
	}
	d.Daemon = daemon(port)
	return port
}

func TestWithoutScopeTheOneAreaOfTheExpectsIsMeasured(t *testing.T) {
	d := depsWith(t)
	port := pointedWorld(t, d, func(one, _ string, _ int) string { return filepath.Join(one, "a.md") })
	d.Daemon = daemon(port)
	md, err := Bench(Options{Scope: "knowledge"}, d)
	if err != nil {
		t.Fatal(err)
	}
	if c := port.Calls[0]; !slices.Equal(c.Collections, []string{"one"}) {
		t.Fatalf("collections = %v", c.Collections)
	}
	if !strings.Contains(md, "| total | 50/50 |") || !strings.Contains(md, "- scope: one\n") {
		t.Fatalf("markdown:\n%s", md)
	}
}

func TestWithoutScopeExpectsInSeveralAreasMeasureAll(t *testing.T) {
	d := depsWith(t)
	port := pointedWorld(t, d, func(one, two string, n int) string {
		if n%2 == 0 {
			return filepath.Join(two, "a.md")
		}
		return filepath.Join(one, "a.md")
	})
	d.Daemon = daemon(port)
	md, err := Bench(Options{}, d)
	if err != nil {
		t.Fatal(err)
	}
	if c := port.Calls[0]; !slices.Equal(c.Collections, []string{"knowledge", "one", "two"}) {
		t.Fatalf("collections = %v", c.Collections)
	}
	if !strings.Contains(md, "- scope: all\n") {
		t.Fatalf("markdown:\n%s", md)
	}
}

func TestWithoutScopeAnExpectOutsideEveryAreaIsRefused(t *testing.T) {
	d := depsWith(t)
	outside := filepath.Join(t.TempDir(), "a.md")
	writeFile(t, outside, "# A\n\nthe evidence\n")
	port := pointedWorld(t, d, func(one, _ string, n int) string {
		if n == 7 {
			return outside
		}
		return filepath.Join(one, "a.md")
	})
	d.Daemon = daemon(port)
	_, err := Bench(Options{}, d)
	if err == nil || !strings.Contains(err.Error(), "q07") || !strings.Contains(err.Error(), outside) ||
		!strings.Contains(err.Error(), "--scope") {
		t.Fatalf("err = %v", err)
	}
	if len(port.Calls) != 0 {
		t.Fatalf("searched: %+v", port.Calls)
	}
}

func TestAGivenScopeIsNeverDerived(t *testing.T) {
	d := depsWith(t)
	port := pointedWorld(t, d, func(one, _ string, _ int) string { return filepath.Join(one, "a.md") })
	port.Results = nil
	for range 50 {
		port.Results = append(port.Results, answering("knowledge", 1).Results...)
	}
	d.Daemon = daemon(port)
	md, err := Bench(Options{Scope: "knowledge", ScopeSet: true}, d)
	if err != nil {
		t.Fatal(err)
	}
	if c := port.Calls[0]; !slices.Equal(c.Collections, []string{"knowledge"}) {
		t.Fatalf("collections = %v", c.Collections)
	}
	if !strings.Contains(md, "- scope: knowledge\n") {
		t.Fatalf("markdown:\n%s", md)
	}
}

func TestInsideTellsAPathUnderADirectory(t *testing.T) {
	dir := t.TempDir()
	for path, want := range map[string]bool{
		filepath.Join(dir, "x", "a.md"):          true,
		dir:                                      true,
		filepath.Join(filepath.Dir(dir), "a.md"): false,
		dir + "x":                                false,
	} {
		if got := inside(path, dir); got != want {
			t.Errorf("inside(%q) = %v", path, got)
		}
	}
}

func TestAnExpectBelongsToTheDeepestAreaHoldingIt(t *testing.T) {
	outer := t.TempDir()
	inner := filepath.Join(outer, "92 Engineering", "craft")
	// The outer area comes first: registry order must not decide.
	registered := []config.Area{{Scope: "knowledge", Path: outer}, {Scope: "engineering/craft", Path: inner}}
	questions := []Question{{ID: "q01", Expect: filepath.Join(inner, "a.md")}, {ID: "q02", Expect: filepath.Join(inner, "b.md")}}
	scope, err := pointedScope(registered, questions)
	if err != nil || scope != "engineering/craft" {
		t.Fatalf("scope %q, err %v", scope, err)
	}
	slices.Reverse(registered)
	if scope, err := pointedScope(registered, questions); err != nil || scope != "engineering/craft" {
		t.Fatalf("reversed: scope %q, err %v", scope, err)
	}
}
