package query

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/code/model"
)

// grepWork is the start-up work one grep call does, counted at the seams.
type grepWork struct {
	walks, listings, builds, restamps, reads int
}

// countGrepWork counts the walks, tracked-file listings, rebuilds, restamps
// and record reads of every grep call until the test ends.
func countGrepWork(t *testing.T) *grepWork {
	t.Helper()
	w := &grepWork{}
	oldWalk, oldTracked, oldBuild, oldRestamp, oldRead := walkCorpus, trackedSet, buildWith, restamp, readRecord
	walkCorpus = func(root string, m corpus.Matcher) (corpus.Listing, error) {
		w.walks++
		return oldWalk(root, m)
	}
	trackedSet = func(rec corpus.Record, root string) ([]string, error) {
		w.listings++
		return oldTracked(rec, root)
	}
	buildWith = func(root string, opts BuildOptions, say func(string)) (*model.Graph, Stats, error) {
		w.builds++
		return oldBuild(root, opts, say)
	}
	restamp = func(root string, was corpus.Stamp, rec corpus.Record, index corpus.Stamp) error {
		w.restamps++
		return oldRestamp(root, was, rec, index)
	}
	readRecord = func(root string) (corpus.Record, bool) {
		w.reads++
		return oldRead(root)
	}
	t.Cleanup(func() {
		walkCorpus, trackedSet, buildWith, restamp, readRecord = oldWalk, oldTracked, oldBuild, oldRestamp, oldRead
	})
	return w
}

// budgetEdit is a few tracked files with a new line each.
var budgetEdit = map[string]string{
	"README.md":  "# Repo\n\nzzzBudgetNeedle\n",
	"main.go":    "package main\n\nimport \"example.com/repo/lib\"\n\n// zzzBudgetNeedle\nfunc main() { lib.Run() }\n",
	"lib/lib.go": "package lib\n\n// zzzBudgetNeedle\n// Run does the thing.\nfunc Run() {}\n",
}

// The start-up work of grep is pinned per scenario: a regression shows as an
// extra walk, git listing, rebuild, restamp or record read, not as time. git
// is off the PATH during each call, so any git process fails it.
func TestGrepWorkBudget(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(t *testing.T, root string)
		opts    GrepOptions
		pattern string
		hits    int
		index   string
		want    grepWork
	}{
		{"clean", func(*testing.T, string) {}, GrepOptions{}, "zzzUniqueWord", 2, "used",
			grepWork{walks: 1, reads: 1}},
		{"dirty", func(t *testing.T, root string) { writeTree(t, root, budgetEdit) }, GrepOptions{},
			"zzzBudgetNeedle", 3, "overlay", grepWork{walks: 1, reads: 1}},
		{"committed", func(t *testing.T, root string) {
			writeTree(t, root, budgetEdit)
			commit(t, root, "edit", "README.md", "main.go", "lib/lib.go")
		}, GrepOptions{}, "zzzBudgetNeedle", 3, "overlay", grepWork{walks: 1, listings: 1, restamps: 1, reads: 1}},
		{"trust", func(*testing.T, string) {}, GrepOptions{TrustIndex: true}, "zzzUniqueWord", 2, "trusted",
			grepWork{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := committedRepo(t)
			c.setup(t, root)
			w := countGrepWork(t)
			t.Setenv("PATH", "")
			grepWithin := func(want grepWork) {
				t.Helper()
				*w = grepWork{}
				res, notes, err := Grep(root, c.pattern, c.opts)
				if err != nil {
					t.Fatalf("%v (notes %v)", err, notes)
				}
				if res.Index != c.index || res.TotalHits != c.hits {
					t.Fatalf("index = %q hits = %d, want %q and %d (notes %v)", res.Index, res.TotalHits, c.index, c.hits, notes)
				}
				if *w != want {
					t.Fatalf("work = %+v, want %+v (notes %v)", *w, want, notes)
				}
			}
			grepWithin(c.want)
			// The first call saved what it learned: the second lists and
			// restamps nothing.
			again := c.want
			again.listings, again.restamps = 0, 0
			grepWithin(again)
		})
	}
}

// A clean-tree grep of a literal allocates within a ceiling of about twice
// what it measured: 1122 to 1123 allocs per call on Windows, ceiling 2250.
func TestGrepAllocationBudget(t *testing.T) {
	root := committedRepo(t)
	allocs := testing.AllocsPerRun(5, func() {
		if _, _, err := Grep(root, "zzzUniqueWord", GrepOptions{}); err != nil {
			t.Fatal(err)
		}
	})
	t.Logf("allocs per grep: %.0f", allocs)
	const ceiling = 2250
	if allocs > ceiling {
		t.Fatalf("allocs per grep = %.0f, want at most %.0f", allocs, float64(ceiling))
	}
}
