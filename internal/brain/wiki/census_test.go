package wiki

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/testlock"
)

const censusPage = "---\ntype: %s\ntitle: t\nsources:\n  - id: spec\n    resource: /raw/spec.md\n" +
	"    doc_id: 01J8F2K9XQ7M\n    content_hash: \"sha256:9f2a\"\n    revision: 4\n---\n\nText.\n"

func writeCensusPage(t *testing.T, path, pageType string) {
	t.Helper()
	body := strings.Replace(censusPage, "type: %s\n", "type: "+pageType+"\n", 1)
	if pageType == "" {
		body = strings.Replace(censusPage, "type: %s\n", "", 1)
	}
	writeCensusFile(t, path, body)
}

func writeCensusFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// censusAreas is the two areas of `test_census.py`: alpha with Architecture,
// the old Design Decision, a type of its own that its declaration names, a
// page without a type and a catalog; beta with Architecture and Topic and no
// declaration at all.
func censusAreas(t *testing.T) []config.Area {
	t.Helper()
	base := t.TempDir()
	alpha := filepath.Join(base, "alpha", "wiki")
	writeCensusPage(t, filepath.Join(alpha, "architecture.md"), "Architecture")
	writeCensusPage(t, filepath.Join(alpha, "design-decision.md"), "Design Decision")
	writeCensusPage(t, filepath.Join(alpha, "balancing.md"), "Balancing Rule")
	writeCensusPage(t, filepath.Join(alpha, "no-type.md"), "")
	writeCensusFile(t, filepath.Join(alpha, "index.md"), "# Katalog\n")
	alphaRepo := filepath.Join(base, "alpha", "repo")
	writeCensusFile(t, filepath.Join(alphaRepo, ".brain.toml"),
		"[area]\nscope = \"alpha\"\n\n[wiki]\ntypes = [\"Balancing Rule\"]\n")
	beta := filepath.Join(base, "beta", "wiki")
	writeCensusPage(t, filepath.Join(beta, "architecture.md"), "Architecture")
	writeCensusPage(t, filepath.Join(beta, "topic.md"), "Topic")
	betaRepo := filepath.Join(base, "beta", "repo")
	if err := os.MkdirAll(betaRepo, 0o755); err != nil {
		t.Fatal(err)
	}
	return []config.Area{
		{Scope: "alpha", Path: alphaRepo, WikiPath: alpha},
		{Scope: "beta", Path: betaRepo, WikiPath: beta},
	}
}

func areaPath(area config.Area) string { return area.Path }

func mustCensus(t *testing.T, areas []config.Area) []TypeCount {
	t.Helper()
	counts, err := Census(areas, areaPath)
	if err != nil {
		t.Fatal(err)
	}
	return counts
}

func countFor(t *testing.T, counts []TypeCount, name string) TypeCount {
	t.Helper()
	for _, c := range counts {
		if c.name() == name {
			return c
		}
	}
	t.Fatalf("no count for %q in %v", name, counts)
	return TypeCount{}
}

func TestTheCensusSumsATypeAcrossAreas(t *testing.T) {
	arch := countFor(t, mustCensus(t, censusAreas(t)), "Architecture")
	if !reflect.DeepEqual(arch.PerScope, map[string]int{"alpha": 1, "beta": 1}) || arch.Rank != RankCore {
		t.Fatalf("Architecture = %+v", arch)
	}
}

func TestTheCensusRanksEveryType(t *testing.T) {
	counts := mustCensus(t, censusAreas(t))
	for name, want := range map[string]Rank{
		"Design Decision": RankUnknown,
		"Topic":           RankOrigin,
		"Balancing Rule":  RankDeclared,
		"(no type)":       RankUnknown,
	} {
		if got := countFor(t, counts, name).Rank; got != want {
			t.Errorf("%s ranks %q, want %q", name, got, want)
		}
	}
	if alias := countFor(t, counts, "Design Decision").AliasOf; alias != "Decision" {
		t.Errorf("Design Decision is an alias of %q", alias)
	}
	// One page and not two: the catalog is scaffold, not an untyped page.
	if untyped := countFor(t, counts, "(no type)"); !reflect.DeepEqual(untyped.PerScope, map[string]int{"alpha": 1}) {
		t.Errorf("untyped = %v", untyped.PerScope)
	}
}

func TestTheCensusIsSortedByTotalThenName(t *testing.T) {
	counts := mustCensus(t, censusAreas(t))
	var names []string
	for _, c := range counts {
		names = append(names, c.name())
	}
	want := []string{"Architecture", "(no type)", "Balancing Rule", "Design Decision", "Topic"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("order %v, want %v", names, want)
	}
}

func TestTheCensusRendersOneLinePerType(t *testing.T) {
	got := RenderCensus(mustCensus(t, censusAreas(t)))
	want := "Architecture [core]: 2 (alpha: 1, beta: 1)\n" +
		"? (no type) [unknown]: 1 (alpha: 1)\n" +
		"Balancing Rule [declared]: 1 (alpha: 1)\n" +
		"? Design Decision [unknown] -> Decision: 1 (alpha: 1)\n" +
		"Topic [origin]: 1 (beta: 1)\n"
	if got != want {
		t.Fatalf("render:\n%s\nwant:\n%s", got, want)
	}
	if RenderCensus(nil) != "" {
		t.Fatal("an empty census rendered a line")
	}
}

func TestTheCensusSkipsAnAreaWithoutABundle(t *testing.T) {
	base := t.TempDir()
	file := filepath.Join(base, "file")
	writeCensusFile(t, file, "x")
	for _, area := range []config.Area{
		{Scope: "gamma", Path: base},
		{Scope: "delta", Path: base, WikiPath: filepath.Join(base, "never-initialized")},
		{Scope: "epsilon", Path: base, WikiPath: file},
	} {
		if counts := mustCensus(t, []config.Area{area}); len(counts) != 0 {
			t.Errorf("%s: counted %v", area.Scope, counts)
		}
	}
}

func TestTheWorstRankWinsAcrossAreas(t *testing.T) {
	base := t.TempDir()
	var areas []config.Area
	// The declaring area first and last: the rank must not depend on which
	// area the walk meets first.
	for _, scope := range []string{"declaring", "silent", "declaring2"} {
		wiki := filepath.Join(base, scope, "wiki")
		writeCensusPage(t, filepath.Join(wiki, "rule.md"), "Mystery Type")
		repo := filepath.Join(base, scope, "repo")
		if scope != "silent" {
			writeCensusFile(t, filepath.Join(repo, ".brain.toml"),
				"[area]\nscope = \""+scope+"\"\n\n[wiki]\ntypes = [\"Mystery Type\"]\n")
		}
		areas = append(areas, config.Area{Scope: scope, Path: repo, WikiPath: wiki})
	}
	if rank := countFor(t, mustCensus(t, areas), "Mystery Type").Rank; rank != RankUnknown {
		t.Fatalf("rank %q, want unknown", rank)
	}
}

func TestTheCensusStopsAtADeclarationItCannotRead(t *testing.T) {
	areas := censusAreas(t)
	writeCensusFile(t, filepath.Join(areas[0].Path, ".brain.toml"), "[area\n")
	if _, err := Census(areas, areaPath); err == nil {
		t.Fatal("a broken declaration was taken for none")
	}
}

func TestTheCensusStopsAtAPageItCannotRead(t *testing.T) {
	areas := censusAreas(t)
	// `read_page` raising takes the tally down with it; a page skipped
	// silently would make the count look complete.
	testlock.Lock(t, filepath.Join(areas[1].WikiPath, "topic.md"))
	if _, err := Census(areas, areaPath); err == nil {
		t.Fatal("an unreadable page was counted")
	}
}

func TestATypeSpeltLikeThePlaceholderStaysApart(t *testing.T) {
	base := t.TempDir()
	wiki := filepath.Join(base, "wiki")
	writeCensusPage(t, filepath.Join(wiki, "a.md"), "\"(no type)\"")
	writeCensusPage(t, filepath.Join(wiki, "b.md"), "")
	counts := mustCensus(t, []config.Area{{Scope: "s", Path: base, WikiPath: wiki}})
	if len(counts) != 2 {
		t.Fatalf("counts %v, want two lines", counts)
	}
}

func TestTheWalkSortsAsPythonSortsPathsOnWindows(t *testing.T) {
	// `sorted(rglob(...))` compares paths component by component, each
	// folded to lower case: measured with Python 3.14 on this machine, the
	// six names below came back in exactly this order. A byte-wise walk
	// would put `B` and `Zeta.md` first and `a-c.md` before `a\b.md`.
	root := t.TempDir()
	want := []string{"_z.md", "a/b.md", "a-c.md", "alpha.md", "B/x.md", "Zeta.md"}
	for _, name := range want {
		writeCensusFile(t, filepath.Join(root, filepath.FromSlash(name)), "")
	}
	var got []string
	for _, path := range markdownBelow(root) {
		rel, _ := filepath.Rel(root, path)
		got = append(got, filepath.ToSlash(rel))
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order %v, want %v", got, want)
	}
}
