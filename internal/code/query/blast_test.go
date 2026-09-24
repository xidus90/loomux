package query

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/diff"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/gitenv"
)

// copyTree copies the files under src to dst.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// git runs git in root and fails the test when git does.
func git(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = gitenv.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// gitRepo is the graph case repository with one commit and a graph.
func gitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copyTree(t, filepath.Join("..", "..", "..", "testdata", "cases", "graph", "repo"), root)
	for _, args := range [][]string{
		{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "t"},
		{"config", "core.autocrlf", "false"}, {"config", "commit.gpgsign", "false"},
		{"add", "."}, {"commit", "-qm", "init"},
	} {
		git(t, root, args...)
	}
	if _, _, err := Build(root, ignore); err != nil {
		t.Fatal(err)
	}
	return root
}

// editAdd swaps the operands in Add's body, line 5 of calc/calc.go.
func editAdd(t *testing.T, root string) {
	t.Helper()
	p := filepath.Join(root, "calc", "calc.go")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	next := strings.Replace(string(data), "return a + b", "return b + a", 1)
	if next == string(data) {
		t.Fatal("calc.go no longer holds `return a + b`")
	}
	if err := os.WriteFile(p, []byte(next), 0o644); err != nil {
		t.Fatal(err)
	}
}

func areaPaths(a BlastAnswer) []string {
	var out []string
	for _, ar := range a.Areas {
		out = append(out, ar.Path)
	}
	return out
}

func hitNames(a BlastAnswer) []string {
	var out []string
	for _, h := range a.Hits {
		out = append(out, h.Node.Name)
	}
	return out
}

func TestBlastOnTheWorkingTree(t *testing.T) {
	root := gitRepo(t)
	editAdd(t, root)
	a, _, err := Blast(root, BlastOptions{NoRefresh: true})
	if err != nil {
		t.Fatal(err)
	}
	if a.Range != "working tree against HEAD" {
		t.Errorf("range = %q", a.Range)
	}
	if got := areaPaths(a); !slices.Equal(got, []string{"calc/calc.go"}) {
		t.Errorf("areas = %v", got)
	}
	if got := hitNames(a); !slices.Equal(got, []string{"TestAdd", "main"}) {
		t.Errorf("hits = %v", got)
	}
}

// A clean tree in a one-commit repository has nothing to fall back to: an
// error of its own, not git's complaint about an ambiguous argument.
func TestBlastOnACleanTreeWithOneCommitHasNoHistory(t *testing.T) {
	root := gitRepo(t)
	_, _, err := Blast(root, BlastOptions{NoRefresh: true})
	if !errors.Is(err, ErrNoHistory) || err.Error() != "nothing to compare: HEAD has no parent commit" {
		t.Fatalf("err = %v, want ErrNoHistory", err)
	}
}

// Without a HEAD no form of blast has anything to compare with.
func TestBlastWithoutAHEADHasNoHistory(t *testing.T) {
	root := t.TempDir()
	copyTree(t, filepath.Join("..", "..", "..", "testdata", "cases", "graph", "repo"), root)
	git(t, root, "init", "-q")
	if _, _, err := Build(root, ignore); err != nil {
		t.Fatal(err)
	}
	for name, opts := range map[string]BlastOptions{
		"working tree": {NoRefresh: true},
		"cached":       {NoRefresh: true, Cached: true},
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := Blast(root, opts)
			if !errors.Is(err, ErrNoHistory) || err.Error() != "nothing to compare: no HEAD to compare with" {
				t.Fatalf("err = %v, want ErrNoHistory", err)
			}
		})
	}
}

func TestBlastOnACleanTreeTakesTheLastCommit(t *testing.T) {
	root := gitRepo(t)
	editAdd(t, root)
	git(t, root, "commit", "-qam", "swap")
	a, _, err := Blast(root, BlastOptions{NoRefresh: true})
	if err != nil {
		t.Fatal(err)
	}
	if a.Range != "HEAD~1...HEAD" {
		t.Errorf("range = %q", a.Range)
	}
	if got := areaPaths(a); !slices.Equal(got, []string{"calc/calc.go"}) {
		t.Errorf("areas = %v", got)
	}
}

func TestBlastWithABase(t *testing.T) {
	root := gitRepo(t)
	editAdd(t, root)
	git(t, root, "commit", "-qam", "swap")
	a, _, err := Blast(root, BlastOptions{Base: "HEAD~1", NoRefresh: true, Depth: blast.All})
	if err != nil {
		t.Fatal(err)
	}
	if a.Range != "HEAD~1...HEAD" {
		t.Errorf("range = %q", a.Range)
	}
	if got := areaPaths(a); !slices.Equal(got, []string{"calc/calc.go"}) {
		t.Errorf("areas = %v", got)
	}
}

func TestBlastCachedSeesOnlyTheIndex(t *testing.T) {
	root := gitRepo(t)
	editAdd(t, root)
	a, _, err := Blast(root, BlastOptions{Cached: true, NoRefresh: true})
	if err != nil {
		t.Fatal(err)
	}
	if a.Range != "index against HEAD" || len(a.Areas) != 0 || len(a.Hits) != 0 {
		t.Errorf("unstaged: %+v", a)
	}
	git(t, root, "add", ".")
	a, _, err = Blast(root, BlastOptions{Cached: true, NoRefresh: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := hitNames(a); !slices.Equal(got, []string{"TestAdd", "main"}) {
		t.Errorf("staged hits = %v", got)
	}
}

func TestBlastRefusesBaseAndCached(t *testing.T) {
	_, _, err := Blast(t.TempDir(), BlastOptions{Base: "HEAD~1", Cached: true})
	if !errors.Is(err, ErrBaseAndCached) {
		t.Fatalf("err = %v", err)
	}
}

func TestBlastWithoutAGraph(t *testing.T) {
	_, _, err := Blast(t.TempDir(), BlastOptions{})
	if !errors.Is(err, ErrNoGraph) {
		t.Fatalf("err = %v", err)
	}
}

func TestBlastHidesWhatKeepRefuses(t *testing.T) {
	root := gitRepo(t)
	editAdd(t, root)
	a, _, err := Blast(root, BlastOptions{NoRefresh: true, Keep: func(p string) bool { return p != "main.go" }})
	if err != nil {
		t.Fatal(err)
	}
	if got := hitNames(a); !slices.Equal(got, []string{"TestAdd"}) || a.Hidden != 1 {
		t.Errorf("hits = %v, hidden = %d", got, a.Hidden)
	}
	a, _, err = Blast(root, BlastOptions{NoRefresh: true, Keep: func(p string) bool { return p != "calc/calc.go" }})
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Areas) != 0 || len(a.Hits) != 0 || a.Hidden != 1 {
		t.Errorf("areas = %v, hits = %v, hidden = %d", areaPaths(a), hitNames(a), a.Hidden)
	}
	// A refused test file is neither a hit nor named as the area's test.
	a, _, err = Blast(root, BlastOptions{NoRefresh: true, Keep: func(p string) bool { return p != "calc/calc_test.go" }})
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Areas) != 1 || len(a.Areas[0].Tests) != 0 || slices.Contains(hitNames(a), "TestAdd") || a.Hidden != 2 {
		t.Errorf("areas = %+v, hits = %v, hidden = %d", a.Areas, hitNames(a), a.Hidden)
	}
}

// fakeGit answers the calls in order; a nil error with no output is an
// empty answer.
func fakeGit(t *testing.T, answers ...func() ([]byte, error)) {
	t.Helper()
	calls := 0
	old := gitOutput
	gitOutput = func(string, ...string) ([]byte, error) {
		f := answers[calls]
		calls++
		return f()
	}
	t.Cleanup(func() { gitOutput = old })
}

func answer(out string, err error) func() ([]byte, error) {
	return func() ([]byte, error) { return []byte(out), err }
}

func TestBlastReportsGitsFailures(t *testing.T) {
	boom := errors.New("boom")
	// head is the probe that finds the revision: the failure stays git's.
	head := answer("0123abc", nil)
	cases := map[string][]func() ([]byte, error){
		"first call":        {answer("", boom), head},
		"second call":       {answer("M\x00calc/calc.go\x00", nil), answer("", boom), head},
		"bad name-status":   {answer("X\x00a\x00", nil), head},
		"bad hunk header":   {answer("M\x00calc/calc.go\x00", nil), answer("diff --git a/calc/calc.go b/calc/calc.go\n--- a/calc/calc.go\n+++ b/calc/calc.go\n@@ nonsense @@\n", nil), head},
		"fallback failures": {answer("", nil), head, answer("", boom)},
		// A probe that fails without git's "no such revision" is git failing.
		"fallback probe fails": {answer("", nil), answer("", boom)},
	}
	for name, answers := range cases {
		t.Run(name, func(t *testing.T) {
			root := built(t)
			fakeGit(t, answers...)
			_, _, err := Blast(root, BlastOptions{NoRefresh: true})
			if err == nil || errors.Is(err, ErrNoHistory) {
				t.Fatalf("err = %v, want git's failure", err)
			}
		})
	}
}

// The index of a stopped cherry-pick holds the conflict as unmerged, and git
// diff --cached names it U: blast says so instead of guessing its lines.
func TestBlastOnAnUnresolvedConflictNamesIt(t *testing.T) {
	_, _, err := Blast(stopped(t, "cherry-pick"), BlastOptions{NoRefresh: true, Cached: true})
	if !errors.Is(err, diff.ErrUnmerged) || !strings.Contains(err.Error(), "unresolved conflict in calc/calc.go") {
		t.Fatalf("err = %v, want the unresolved conflict", err)
	}
}

// Outside a repository there is no HEAD either, but the reason is git's:
// its complaint stands, not "nothing to compare".
func TestBlastOutsideARepositoryKeepsGitsComplaint(t *testing.T) {
	for name, opts := range map[string]BlastOptions{
		"working tree": {NoRefresh: true},
		"cached":       {NoRefresh: true, Cached: true},
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := Blast(built(t), opts)
			if err == nil || errors.Is(err, ErrNoHistory) || !strings.Contains(err.Error(), "not a git repository") {
				t.Fatalf("err = %v, want git's complaint", err)
			}
		})
	}
}

func TestRunGitKeepsGitsComplaint(t *testing.T) {
	_, err := runGit(t.TempDir(), "rev-parse", "HEAD")
	if err == nil || !strings.Contains(err.Error(), "not a git repository") {
		t.Fatalf("err = %v", err)
	}
}

func TestBlastReport(t *testing.T) {
	add := &model.Node{Name: "Add", Kind: "function", Path: "calc/calc.go", Span: "L4-L6"}
	file := &model.Node{Name: "calc.go", Kind: model.KindFile, Path: "calc/doc.go"}
	main := &model.Node{Name: "main", Kind: "function", Path: "main.go", Span: "L5-L7"}
	cases := []struct {
		name string
		in   BlastAnswer
		want string
	}{
		{"nothing", BlastAnswer{Range: "working tree against HEAD"},
			"blast radius: working tree against HEAD\nno change\n"},
		{"nothing visible", BlastAnswer{Range: "r", Hidden: 2},
			"blast radius: r\nno change\nhidden: 2\n"},
		{"everything", BlastAnswer{
			Range: "HEAD~1...HEAD",
			Report: blast.Report{
				Areas: []blast.Area{
					{Path: "calc/calc.go", Status: diff.Modified, Signal: blast.SignalStale,
						Seeds: []blast.Seed{{Node: add, InDegree: 2}}, Tests: []string{"calc/calc_test.go", "x_test.go"}},
					{Path: "calc/doc.go", Status: diff.Added, Signal: blast.SignalNA,
						Seeds: []blast.Seed{{Node: file}}},
				},
				Hits: []blast.RadiusHit{{
					Hit:  blast.Hit{ID: "main.go#main", Node: main, Relation: "calls", Depth: 1},
					From: []string{"calc/calc.go", "calc/doc.go"},
				}},
				Deleted:      []string{"old.go"},
				Unindexed:    []string{"README.md"},
				Evidence:     []blast.Evidence{{Node: add, Lines: []string{"-\treturn a + b", "+\treturn b + a"}, More: 3}},
				MoreEvidence: 1,
			},
			Hidden: 1,
		}, "blast radius: HEAD~1...HEAD\n" +
			"calc/calc.go [stale]\n" +
			"  seed Add (function) L4-L6 in-degree 2\n" +
			"  tests: calc/calc_test.go, x_test.go\n" +
			"calc/doc.go [na]\n" +
			"  seed calc/doc.go (file) in-degree 0\n" +
			"reached:\n" +
			"  main (function, depth 1, calls) in main.go:L5-L7 from calc/calc.go, calc/doc.go\n" +
			"not indexed: README.md\n" +
			"evidence:\n" +
			"  Add in calc/calc.go\n" +
			"    -\treturn a + b\n" +
			"    +\treturn b + a\n" +
			"    … +3 lines\n" +
			"  … 1 more symbols\n" +
			"deleted: old.go\n" +
			"hidden: 1\n"},
		{"only a deletion", BlastAnswer{Range: "r", Report: blast.Report{Deleted: []string{"old.go"}}},
			"blast radius: r\ndeleted: old.go\n"},
		{"only unindexed", BlastAnswer{Range: "r", Report: blast.Report{Unindexed: []string{"a.txt"}}},
			"blast radius: r\nnot indexed: a.txt\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := BlastReport(c.in); got != c.want {
				t.Errorf("got\n%s\nwant\n%s", got, c.want)
			}
		})
	}
}

func TestBlastRefusesABaseThatReadsAsAnOption(t *testing.T) {
	root := built(t)
	old := gitOutput
	gitOutput = func(string, ...string) ([]byte, error) {
		t.Fatal("git ran with a base that reads as an option")
		return nil, nil
	}
	t.Cleanup(func() { gitOutput = old })
	_, _, err := Blast(root, BlastOptions{Base: "--output=x", NoRefresh: true})
	if !errors.Is(err, ErrBadBase) || !strings.Contains(err.Error(), "--output=x") {
		t.Fatalf("err = %v", err)
	}
}

func TestBlastEndsTheOptionsBeforeTheRevision(t *testing.T) {
	cases := []struct {
		opts BlastOptions
		tail []string
	}{
		{BlastOptions{Base: "main"}, []string{"--end-of-options", "main...HEAD"}},
		{BlastOptions{Cached: true}, []string{"--cached", "--end-of-options", "HEAD"}},
		{BlastOptions{}, []string{"--end-of-options", "HEAD"}},
	}
	for _, c := range cases {
		root := built(t)
		var calls [][]string
		old := gitOutput
		gitOutput = func(_ string, args ...string) ([]byte, error) {
			calls = append(calls, args)
			if len(calls) == 1 {
				return []byte("M\x00lib/lib.go\x00"), nil
			}
			return nil, nil
		}
		c.opts.NoRefresh = true
		_, _, err := Blast(root, c.opts)
		gitOutput = old
		if err != nil {
			t.Fatal(err)
		}
		for _, args := range calls {
			if !slices.Equal(args[len(args)-len(c.tail):], c.tail) {
				t.Errorf("%+v: git %v does not end in %v", c.opts, args, c.tail)
			}
		}
	}
}

// The patch is git's own text diff: a textconv filter from the user's
// attributes would hand the hunk parser lines of some other format.
func TestBlastAsksForThePatchWithoutTextconv(t *testing.T) {
	root := built(t)
	var calls [][]string
	fake := func(_ string, args ...string) ([]byte, error) {
		calls = append(calls, args)
		if len(calls) == 1 {
			return []byte("M\x00lib/lib.go\x00"), nil
		}
		return nil, nil
	}
	old := gitOutput
	gitOutput = fake
	t.Cleanup(func() { gitOutput = old })
	if _, _, err := Blast(root, BlastOptions{NoRefresh: true}); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || !slices.Contains(calls[1], "--unified=0") || !slices.Contains(calls[1], "--no-textconv") {
		t.Fatalf("git calls %q, want the patch with --no-textconv", calls)
	}
}
