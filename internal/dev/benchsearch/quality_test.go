package benchsearch

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// touch creates an empty file, so the file system can vouch for it.
func touch(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	writeFile(t, path, "")
	return path
}

// tick is a clock that advances by step on every reading.
func tick(step time.Duration) func() time.Time {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	return func() time.Time {
		now = now.Add(step)
		return now
	}
}

func TestRankIsThePositionOfTheExpectedFile(t *testing.T) {
	dir := t.TempDir()
	a, b := touch(t, dir, "a.md"), touch(t, dir, "b.md")
	qs := []Question{{ID: "1", Kind: Exact, Expect: b}, {ID: "2", Kind: Exact, Expect: a}}
	answers := map[string][]string{"": {a, b}}
	outcomes, err := RunQuality(qs, func(string) ([]string, error) { return answers[""], nil }, tick(5*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	if outcomes[0].Rank != 2 || !outcomes[0].Hit || outcomes[1].Rank != 1 || outcomes[0].ElapsedMS != 5 {
		t.Fatalf("%+v", outcomes)
	}
}

func TestAMissBeyondTheTopThreeKeepsItsRank(t *testing.T) {
	dir := t.TempDir()
	var ranked []string
	for _, name := range []string{"a.md", "b.md", "c.md", "d.md"} {
		ranked = append(ranked, touch(t, dir, name))
	}
	out, err := RunQuality([]Question{{Kind: Exact, Expect: ranked[3]}},
		func(string) ([]string, error) { return ranked, nil }, tick(0))
	if err != nil || out[0].Rank != 4 || out[0].Hit {
		t.Fatalf("%+v %v", out, err)
	}
}

func TestNotFoundIsRankZero(t *testing.T) {
	dir := t.TempDir()
	a := touch(t, dir, "a.md")
	// Neither the missing candidate nor the missing expectation may count as
	// the same file: a path nobody can stat is simply a different one.
	missing := filepath.Join(dir, "gone.md")
	out, err := RunQuality([]Question{{Kind: Exact, Expect: a}, {Kind: Exact, Expect: missing}},
		func(string) ([]string, error) { return []string{filepath.Join(dir, "other.md")}, nil }, tick(0))
	if err != nil || out[0].Rank != 0 || out[0].Hit || out[1].Rank != 0 {
		t.Fatalf("%+v %v", out, err)
	}
}

func TestRankMatchesAcrossSpelling(t *testing.T) {
	dir := t.TempDir()
	a := touch(t, dir, "Note.md")
	spelled := filepath.Join(dir, "sub", "..", "Note.md")
	if runtime.GOOS == "windows" {
		spelled = strings.ToLower(spelled)
	}
	out, _ := RunQuality([]Question{{Kind: Exact, Expect: spelled}},
		func(string) ([]string, error) { return []string{a}, nil }, tick(0))
	if out[0].Rank != 1 {
		t.Fatalf("rank = %d", out[0].Rank)
	}
}

// Through a junction the two spellings share no prefix; only the file
// system can tell they are one file (8.3 names on the runner likewise).
func TestRankMatchesThroughAJunction(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("junctions are a Windows form")
	}
	real := t.TempDir()
	a := touch(t, real, "a.md")
	link := filepath.Join(t.TempDir(), "j")
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, real).CombinedOutput(); err != nil {
		t.Fatalf("mklink: %v %s", err, out)
	}
	// The junction must go before its target's temp dir is removed.
	t.Cleanup(func() { _ = os.Remove(link) })
	out, _ := RunQuality([]Question{{Kind: Exact, Expect: filepath.Join(link, "a.md")}},
		func(string) ([]string, error) { return []string{a}, nil }, tick(0))
	if out[0].Rank != 1 {
		t.Fatalf("rank = %d", out[0].Rank)
	}
}

func TestASearchErrorEndsTheRun(t *testing.T) {
	boom := errors.New("qmd is gone")
	out, err := RunQuality([]Question{{Kind: Exact}},
		func(string) ([]string, error) { return nil, boom }, tick(0))
	if !errors.Is(err, boom) || out != nil {
		t.Fatalf("out = %+v, err = %v", out, err)
	}
}
