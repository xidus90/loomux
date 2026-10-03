package run

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/check"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/testlock"
)

// TestMain points both state places at an empty directory before any test
// runs. The fixtures below set LOOMUX_STATE_DIR where they register areas;
// a test that sets nothing would otherwise read this machine's registry,
// and the old place would be asked whenever the new one holds nothing.
func TestMain(m *testing.M) {
	empty, err := os.MkdirTemp("", "run-state-")
	if err != nil {
		panic(err)
	}
	os.Setenv(config.StateDirEnv, empty)
	os.Setenv("LOOMUX_LEGACY_BRAIN_DIR", empty)
	code := m.Run()
	os.RemoveAll(empty)
	os.Exit(code)
}

// testLookup is the lookup the CLI builds, without the fallback to the old
// place: the fixtures write their registration and their read-only
// declarations under the state directory they set, and nowhere else.
func testLookup() config.ArtifactLookup {
	return config.ArtifactLookup{Primary: config.StateDir()}
}

func checkFile(path string) []check.Finding {
	return CheckFile(path, testLookup())
}

func checkBundle(area config.Area, areas []config.Area) []check.Finding {
	return CheckBundle(area, areas, testLookup())
}

func checkAll(subjects, areas []config.Area) []check.Finding {
	return CheckAll(subjects, areas, testLookup())
}

// The three tests the plan spells out stand below with their wording and
// their intent unchanged; only the package qualification differs, because
// `internal/brain/check/okf` and `internal/brain/check/house` import `internal/brain/check` and a `run.go`
// inside that package would close an import cycle Go refuses to build.

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// page is the smallest body that keeps the rules a test is not about
// quiet: a type the vocabulary knows and one complete source.
func page(title string) string {
	return "---\ntitle: " + title + "\ndescription: d\ntype: Topic\n" +
		"sources:\n  - id: s1\n    resource: brain://x/y\n" +
		"    doc_id: d1\n    content_hash: h1\n    revision: 1\n" +
		"generated:\n  by: test\n---\n\n# " + title + "\n"
}

// ruleOnly is the single finding of a rule, and it fails rather than
// answering a zero value when there is not exactly one: a caller that
// read fields off a zero Finding would report the absence of a finding
// as a wrong severity.
func ruleOnly(findings []check.Finding, rule string) check.Finding {
	var out []check.Finding
	for _, f := range findings {
		if f.Rule == rule {
			out = append(out, f)
		}
	}
	if len(out) != 1 {
		panic(fmt.Sprintf("%d findings of %q, want exactly 1",
			len(out), rule))
	}
	return out[0]
}

func countRule(findings []check.Finding, rule string) int {
	n := 0
	for _, f := range findings {
		if f.Rule == rule {
			n++
		}
	}
	return n
}

// area builds one registered area with a wiki of two linked pages, so
// that neither `orphan` nor `no-index` speaks for reasons the caller did
// not ask about.
func area(t *testing.T, base, scope string) config.Area {
	t.Helper()
	root := filepath.Join(base, strings.ReplaceAll(scope, "/", "-"))
	wiki := filepath.Join(root, "wiki")
	write(t, filepath.Join(wiki, "index.md"),
		"# c\n\n* [a](a.md) -- first\n* [b](b.md) -- second\n")
	write(t, filepath.Join(wiki, "log.md"), "# log\n\n## 2026-09-01\n")
	// Not markdown, and the walk must not read them. `graph.json` is
	// what the indexer writes beside the pages, and judged as one it
	// would report a missing type and a missing title in every area.
	// `_identities.tsv` would not: it is one of the five scaffold names,
	// so every rule steps over it whatever the walk hands in -- which is
	// why the one that decides anything here is the json.
	write(t, filepath.Join(wiki, "_identities.tsv"), "id\tname\n")
	write(t, filepath.Join(wiki, "graph.json"), "{\"edges\": []}\n")
	// A *directory* whose name ends in .md. The walk skips it for being
	// a directory, and were that test struck it would be skipped one
	// step later all the same, because reading a directory as a file
	// fails. That cause was measured directly, not inferred from the
	// mutant surviving: wiki.ReadPage on a directory answers
	// "Unzulaessige Funktion" on this machine, the same error
	// os.ReadFile gives it.
	if err := os.MkdirAll(filepath.Join(wiki, "folder.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(wiki, "a.md"), page("a")+"\n[b](b.md)\n")
	write(t, filepath.Join(wiki, "b.md"), page("b")+"\n[a](a.md)\n")
	return config.Area{Scope: scope, Path: root, WikiPath: wiki}
}

// eightAreas is the fixture size design 7 argues the parallelism from,
// and every one of them emits at least one finding: an unsorted run of
// eight silent areas would render the same output in any order and prove
// nothing.
func eightAreas(t *testing.T) []config.Area {
	t.Helper()
	base := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(base, "state"))
	var areas []config.Area
	for i := 0; i < 8; i++ {
		a := area(t, base, fmt.Sprintf("project/p%d", i))
		// One page without a type, so the area has something to say.
		write(t, filepath.Join(a.WikiPath, "loud.md"),
			"---\ntitle: loud\n---\n\n[a](a.md)\n")
		areas = append(areas, a)
	}
	return areas
}

func TestCheckAllIsDeterministicUnderConcurrency(t *testing.T) {
	// Parallel over areas, sorted on output. A run whose order changes
	// between two runs is not comparable, and comparability is why the
	// indexer sorts.
	areas := eightAreas(t)
	first := check.Render(checkAll(areas, areas), true)
	if first == "" {
		t.Fatal("the fixture reported nothing; it cannot show an order")
	}
	for i := 0; i < 5; i++ {
		if got := check.Render(checkAll(areas, areas), true); got != first {
			t.Fatalf("run %d differs from the first", i+2)
		}
	}
}

// twoAreasOneSignpost is a signpost whose catalog names nobody and one
// other area for it to miss.
func twoAreasOneSignpost(t *testing.T) []config.Area {
	t.Helper()
	base := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(base, "state"))
	post := area(t, base, "knowledge")
	post.Signpost = true
	return []config.Area{post, area(t, base, "project/other")}
}

func TestFederationRulesRunAfterAllBundlesAreRead(t *testing.T) {
	// unlisted-area needs every bundle; running it per area would report
	// every other area as missing.
	areas := twoAreasOneSignpost(t)
	if n := countRule(checkAll(areas, areas), "unlisted-area"); n != 1 {
		t.Fatalf("%d unlisted-area findings, want exactly 1", n)
	}
}

func writeLonelyPage(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(dir, "state"))
	path := filepath.Join(dir, "lonely.md")
	write(t, path, page("lonely")+"\n[nowhere](gone.md)\n")
	return path
}

func TestCheckFileAsksOnlyWhatOneFileCanAnswer(t *testing.T) {
	// The fast path a Stop hook runs on every write. Orphan and dead-link
	// need neighbours and must not appear.
	//
	// Green at no cost, and it says so of itself: all three rules live in
	// `house.Bundle` and `house.Federation`, which this width never
	// calls, so no filter has to hold for them. The rules that do leak
	// through a one-page run are the two notes the next test names.
	path := writeLonelyPage(t)
	for _, f := range checkFile(path) {
		if f.Rule == "orphan" || f.Rule == "dead-link" ||
			f.Rule == "unlisted-area" {
			t.Fatalf("%s needs neighbours and cannot be decided on one file",
				f.Name())
		}
	}
}

func TestCheckFileLeavesTheNeighbourNotesToTheBundle(t *testing.T) {
	// `okf/no-index` and `okf/no-log` do fire on a one-page slice --
	// measured, both were read: `noIndex` reports every directory whose
	// pages carry no catalog, and one page is such a directory, and
	// `noLog` reports any non-empty page set without `log.md`. They are
	// what the filter is actually for.
	//
	// `okf/catalog-malformed` needs no filter line and this test holds
	// that: the only catalog it could judge is the page itself, and a
	// page that is a catalog is scaffold and is skipped before the
	// question is asked.
	found := checkFile(writeLonelyPage(t))
	for _, rule := range []string{"no-index", "no-log", "catalog-malformed"} {
		if n := countRule(found, rule); n != 0 {
			t.Errorf("%d %s findings on one file, want none", n, rule)
		}
	}
}

func TestTheOutputIsOrderedByScopeThenPathThenAxisThenRule(t *testing.T) {
	// Six equal runs can agree by scheduling luck; this asks the order
	// itself, and it is the test that fails when the sort is struck.
	found := checkAll(eightAreas(t), eightAreas(t))
	key := func(f check.Finding) string {
		return f.Scope + "\x00" + f.Relative + "\x00" +
			string(f.Axis) + "\x00" + f.Rule
	}
	for i := 1; i < len(found); i++ {
		if key(found[i-1]) > key(found[i]) {
			t.Fatalf("finding %d (%s %s) precedes %s %s", i,
				found[i-1].Scope, found[i-1].Name(),
				found[i].Scope, found[i].Name())
		}
	}
}

func TestSerialAndParallelRenderTheSameBytes(t *testing.T) {
	// The measurement of 9.5 needs both, and a run that answered
	// differently depending on how it was scheduled would make the
	// measurement meaningless before it started.
	areas := eightAreas(t)
	serial := check.Render(sorted(runAreas(areas, areas, testLookup(), false)), true)
	parallel := check.Render(sorted(runAreas(areas, areas, testLookup(), true)), true)
	if serial != parallel {
		t.Fatalf("serial and parallel differ:\n%s\nvs\n%s",
			serial, parallel)
	}
}

func TestOnlyMarkdownFilesReachTheRules(t *testing.T) {
	// A bundle holds more than pages: the indexer writes `graph.json`
	// and `_identities.tsv` beside them. Judged as a page, the json
	// reports a missing type and a missing title in every area -- the
	// tsv would not, because it carries one of the five scaffold names
	// and every rule steps over it whatever the walk hands in.
	//
	// The directory named `folder.md` is the other half: the walk skips
	// it for being a directory, and this test also shows what happens
	// when that test is struck, because reading a directory as a file
	// fails and the page never reaches a rule either way.
	base := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(base, "state"))
	one := area(t, base, "project/p")
	for _, f := range checkBundle(one, []config.Area{one}) {
		if !strings.HasSuffix(f.Relative, ".md") ||
			strings.HasPrefix(f.Relative, "folder.md") {
			t.Errorf("%s reported on %q, which is not a page",
				f.Name(), f.Relative)
		}
	}
}

func TestStaleIsMeasuredAgainstARealClock(t *testing.T) {
	// The zero time is the danger: it lies before every stale_after, so
	// a run that left `Now` unset would report nothing and look exactly
	// like a clean stock. Python sets the value in the caller
	// (`src/brain/wiki/lint.py:575`), and this width is that caller.
	base := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(base, "state"))
	a := area(t, base, "project/p")
	write(t, filepath.Join(a.WikiPath, "old.md"),
		strings.Replace(page("old"), "type: Topic",
			"type: Topic\nstale_after: 2020-01-01", 1)+"\n[a](a.md)\n")
	if n := countRule(checkBundle(a, []config.Area{a}), "stale"); n != 1 {
		t.Fatalf("%d stale findings, want 1", n)
	}
}

func TestUntouchedIsMeasuredAgainstTheDeclaredThreshold(t *testing.T) {
	// The second field the zero time would silence: with `Now` unset the
	// cutoff lies before every modification time and nothing is old.
	base := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(base, "state"))
	a := area(t, base, "project/p")
	old := filepath.Join(a.WikiPath, "a.md")
	twoYears := time.Now().AddDate(-2, 0, 0)
	if err := os.Chtimes(old, twoYears, twoYears); err != nil {
		t.Fatal(err)
	}
	if n := countRule(checkBundle(a, []config.Area{a}), "untouched"); n != 1 {
		t.Fatalf("%d untouched findings, want 1", n)
	}
	// And the threshold comes from the manifest, not from the constant:
	// a bundle that declares a thousand days has nothing old in it.
	write(t, filepath.Join(a.Path, ".loomux", "config.toml"),
		"[area]\nscope = \"project/p\"\n\n[wiki]\nuntouched_days = 1000\n")
	if n := countRule(checkBundle(a, []config.Area{a}), "untouched"); n != 0 {
		t.Fatalf("%d untouched findings under 1000 days, want 0", n)
	}
}

func TestTheProjectFamilyComesFromTheFirstScopeSegment(t *testing.T) {
	// The two realization checks apply only to `project/<name>` areas
	// (architecture 9.4, decision 29), and Python takes the family from
	// the scope's first path segment (`src/brain/cli.py:1555-1557`).
	// Without this test the field could be set to a constant either way
	// and nothing would notice: no other fixture page carries
	// `realization` at all.
	base := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(base, "state"))
	for _, c := range []struct {
		scope string
		want  int
	}{{"project/p", 1}, {"knowledge", 0}} {
		a := area(t, base, c.scope)
		write(t, filepath.Join(a.WikiPath, "planned.md"),
			strings.Replace(page("planned"), "type: Topic",
				"type: Topic\nrealization: implemented", 1)+
				"\n[a](a.md)\n")
		n := countRule(checkBundle(a, []config.Area{a}), "implemented-without-commit")
		if n != c.want {
			t.Errorf("%s: %d implemented-without-commit, want %d",
				c.scope, n, c.want)
		}
	}
}

func TestTheTypeVocabularyComesFromTheAreaManifest(t *testing.T) {
	// `ctx.DeclaredTypes` had no filler before this width. The manifest
	// sits at the *area* root, not at the bundle root, and
	// `config.ReadManifest` never walks upwards -- Python decides the
	// same way (`src/brain/cli.py:1539`).
	base := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(base, "state"))
	a := area(t, base, "project/p")
	write(t, filepath.Join(a.WikiPath, "odd.md"),
		strings.Replace(page("odd"), "type: Topic", "type: manual", 1)+
			"\n[a](a.md)\n")
	if n := countRule(checkBundle(a, []config.Area{a}), "unknown-type"); n != 1 {
		t.Fatalf("%d unknown-type findings with no declaration, want 1", n)
	}
	write(t, filepath.Join(a.Path, ".loomux", "config.toml"),
		"[area]\nscope = \"project/p\"\n\n[wiki]\ntypes = [\"manual\"]\n")
	if n := countRule(checkBundle(a, []config.Area{a}), "unknown-type"); n != 0 {
		t.Fatalf("%d unknown-type findings after declaring it, want 0", n)
	}
}

// The other way a declaration can be unusable, and until the manifest
// reader turned strict it was the one this width could not see: a file that
// exists and does not read looked like no declaration at all, so the run
// went green over a manifest nobody could use.
func TestADeclarationThatCannotBeOpenedIsReportedToo(t *testing.T) {
	base := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(base, "state"))
	a := area(t, base, "knowledge")
	locked := filepath.Join(a.Path, ".loomux", "config.toml")
	write(t, locked, "[area]\nscope = \"knowledge\"\n")
	testlock.Lock(t, locked)

	found := checkBundle(a, []config.Area{a})
	if n := countRule(found, "manifest-unreadable"); n != 1 {
		t.Fatalf("%d manifest-unreadable findings, want 1", n)
	}
	if check.ExitCode(found) != 1 {
		t.Error("a declaration that cannot be opened left the run green")
	}
}

func TestADeclarationThatStopsARuleIsReported(t *testing.T) {
	// The position earlier tasks handed on: `house.Federation` stops
	// `unlisted-area` when the signpost's manifest is unusable, and it
	// stops silently, because a rule has no channel for it. This width
	// has one.
	base := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(base, "state"))
	a := area(t, base, "knowledge")
	a.Signpost = true
	other := area(t, base, "project/other")
	broken := filepath.Join(a.Path, ".loomux", "config.toml")

	write(t, broken, "[area\nscope = \"knowledge\"\n")
	found := checkAll([]config.Area{a, other}, []config.Area{a, other})
	if n := countRule(found, "manifest-unreadable"); n != 1 {
		t.Fatalf("%d manifest-unreadable findings, want 1", n)
	}
	// The two properties the comment over `declarationFinding` spends
	// twelve lines defending, and neither was asked before: the degree
	// steers the exit code, so a silent demotion to a warning would
	// make the whole new channel toothless without anything falling
	// over; and the finding is about no page, which is what the
	// parenthesised name says.
	only := ruleOnly(found, "manifest-unreadable")
	if only.Severity != check.Error {
		t.Errorf("severity %q, want an error", only.Severity)
	}
	if only.Relative != "(declaration)" {
		t.Errorf("relative %q, want the declaration, not a page",
			only.Relative)
	}
	if only.Scope != a.Scope {
		t.Errorf("scope %q, want %q", only.Scope, a.Scope)
	}
	if check.ExitCode(found) != 1 {
		t.Error("an unusable declaration left the run green")
	}
	if n := countRule(found, "unlisted-area"); n != 0 {
		t.Fatalf("a stopped rule still reported %d times", n)
	}

	write(t, broken, "[area]\nscope = \"k\"\n\n[layout]\nhub = \"../out\"\n")
	found = checkAll([]config.Area{a, other}, []config.Area{a, other})
	if n := countRule(found, "manifest-layout-invalid"); n != 1 {
		t.Fatalf("%d manifest-layout-invalid findings, want 1", n)
	}
	only = ruleOnly(found, "manifest-layout-invalid")
	if only.Severity != check.Error || only.Relative != "(declaration)" {
		t.Errorf("%s carries %q on %q, want an error on the declaration",
			only.Name(), only.Severity, only.Relative)
	}
	// The second cause is a second name on purpose: a file that parses
	// and states an unusable value is not unreadable, and the two send
	// the reader to different repairs.
	if n := countRule(found, "manifest-unreadable"); n != 0 {
		t.Fatalf("a parsable manifest was called unreadable %d times", n)
	}

	write(t, broken, "[area]\nscope = \"k\"\n\n[layout]\nwiki = \".\"\n")
	if n := countRule(checkAll([]config.Area{a, other}, []config.Area{a, other}),
		"manifest-layout-invalid"); n != 1 {
		t.Fatal("a refused [layout] wiki went unreported")
	}
}

func TestASoundDeclarationIsNotReported(t *testing.T) {
	base := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(base, "state"))
	a := area(t, base, "project/p")
	write(t, filepath.Join(a.Path, ".loomux", "config.toml"),
		"[area]\nscope = \"project/p\"\n\n[layout]\nwiki = \"wiki\"\n"+
			"hub = \"h\"\n")
	names := []string{"manifest-unreadable", "manifest-layout-invalid"}
	found := checkBundle(a, []config.Area{a})
	for _, rule := range names {
		if n := countRule(found, rule); n != 0 {
			t.Errorf("%d %s findings on a sound manifest", n, rule)
		}
	}
	// And an area with no manifest at all is no defect either: an area
	// may declare nothing (`config.ErrNoManifest`).
	empty := area(t, base, "project/bare")
	bare := checkBundle(empty, []config.Area{empty})
	for _, rule := range names {
		if n := countRule(bare, rule); n != 0 {
			t.Errorf("%d %s findings on an area with no manifest", n, rule)
		}
	}
}

func TestCheckFileFindsTheBundleRootThroughTheDeclaredLayout(t *testing.T) {
	// The worktree case, and the reason `[layout] wiki` exists: the
	// registration holds one absolute path per area and a linked
	// worktree has another, so the root has to come from the repository
	// itself. ultra-brain's `pkg/guard/guard.go:293` walks the parents for the
	// declaration the same way.
	repo := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(repo, "state"))
	write(t, filepath.Join(repo, ".loomux", "config.toml"),
		"[area]\nscope = \"project/p\"\n\n[layout]\nwiki = \"docs/wiki\"\n")
	path := filepath.Join(repo, "docs", "wiki", "topics", "x.md")
	write(t, path, "---\ntitle: x\n---\n\n# x\n")
	found := checkFile(path)
	if len(found) == 0 {
		t.Fatal("the page reported nothing; the path cannot be read off")
	}
	if got := found[0].Relative; got != "topics/x.md" {
		t.Fatalf("relative path %q, want it read from docs/wiki", got)
	}
	// The scope stays empty: a run over one file knows a path but no
	// area (the comment on `check.Finding.Scope`).
	if found[0].Scope != "" {
		t.Fatalf("scope %q on a single-file run", found[0].Scope)
	}
}

func TestCheckFileTakesTheLongestRegisteredWikiPath(t *testing.T) {
	// The registration on this machine says outright that the `hub`
	// area's wiki encloses the wikis of four project areas. A walk that
	// took the first or the outermost match would judge a page of
	// `project/space` against the enclosing bundle and give every
	// finding the wrong relative path.
	base := t.TempDir()
	state := filepath.Join(base, "state")
	outer := filepath.Join(base, "vault")
	inner := filepath.Join(outer, "space")
	// Seven entries beside the one that must win, and each of them is
	// a different way of getting the answer wrong. Two enclose the file
	// and stand on either side of it, because the three plausible wrong
	// answers -- first match, last match, any match -- each need their
	// own decoy:
	//
	//   - `vault` encloses the file, is *shorter*, and stands **first**,
	//     so an answer that stopped at the first containing entry would
	//     take it.
	//   - `hub` encloses the file, is *shorter*, and stands **last**, so
	//     an answer that let the last containing entry win would take
	//     it. Together the two also kill `>` turned into `>=`.
	//   - `decoy` is longer than the right one and does not contain the
	//     file, so an answer that skipped the containment test would
	//     take it.
	//   - `beside` is longer still and lies next to the file, the
	//     ordinary shape of a path that is simply outside.
	//   - `below` is the longest of all and lies *under* the file, which
	//     is the one case `filepath.Rel` answers with exactly "..".
	//   - `stray` is spelt relative -- neither registry reader forbids
	//     that (`internal/brain/check/house/federation.go:369-375` says so of its
	//     own comparison) -- and against an absolute file
	//     `filepath.Rel` refuses.
	//   - `nowiki` registers no wiki at all.
	write(t, filepath.Join(state, "registry.toml"), fmt.Sprintf(
		"[[area]]\nscope = \"vault\"\npath = %q\nwiki = %q\n\n"+
			"[[area]]\nscope = \"project/space\"\npath = %q\n"+
			"wiki = %q\n\n"+
			"[[area]]\nscope = \"decoy\"\npath = %q\n"+
			"wiki = %q\n\n"+
			"[[area]]\nscope = \"beside\"\npath = %q\n"+
			"wiki = %q\n\n"+
			"[[area]]\nscope = \"below\"\npath = %q\n"+
			"wiki = %q\n\n"+
			"[[area]]\nscope = \"stray\"\npath = %q\n"+
			"wiki = \"rel/ative\"\n\n"+
			"[[area]]\nscope = \"nowiki\"\npath = %q\n\n"+
			"[[area]]\nscope = \"hub\"\npath = %q\nwiki = %q\n",
		filepath.ToSlash(outer), filepath.ToSlash(outer),
		filepath.ToSlash(inner), filepath.ToSlash(inner),
		filepath.ToSlash(outer), filepath.ToSlash(inner+"-decoy"),
		filepath.ToSlash(outer),
		filepath.ToSlash(filepath.Join(base, "elsewhere-and-longer")),
		filepath.ToSlash(outer),
		filepath.ToSlash(filepath.Join(inner, "x.md", "deeper")),
		filepath.ToSlash(outer), filepath.ToSlash(outer),
		filepath.ToSlash(outer), filepath.ToSlash(outer)))
	t.Setenv(config.StateDirEnv, state)
	// A declaration at the inner area that names no place: the chain's
	// first link finds it, cannot use it, and hands the question on
	// rather than climbing past it to the enclosing area.
	write(t, filepath.Join(inner, ".loomux", "config.toml"),
		"[area]\nscope = \"project/space\"\n\n[wiki]\n"+
			"types = [\"manual\"]\n")
	path := filepath.Join(inner, "x.md")
	write(t, path, "---\ntitle: x\ntype: manual\n---\n\n# x\n")
	found := checkFile(path)
	if len(found) == 0 {
		t.Fatal("the page reported nothing")
	}
	if got := found[0].Relative; got != "x.md" {
		t.Fatalf("relative path %q, want it read from the inner wiki", got)
	}
	// And the manifest travels back with the root: the type is unknown
	// to the built-in vocabulary and known to this area's declaration.
	if n := countRule(found, "unknown-type"); n != 0 {
		t.Fatalf("%d unknown-type findings; the declaration went unread", n)
	}
}

func TestADeclaredLayoutIsIgnoredForAFileOutsideIt(t *testing.T) {
	// A repository may hold pages that are not in its bundle -- this one
	// holds a hundred of them. `[layout] wiki` names the bundle, not the
	// repository, so a file beside it must not be measured from there,
	// or every finding on it would carry a path climbing out of the
	// bundle with `../`.
	repo := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(repo, "state"))
	write(t, filepath.Join(repo, ".loomux", "config.toml"),
		"[area]\nscope = \"project/p\"\n\n[layout]\nwiki = \"docs/wiki\"\n")
	write(t, filepath.Join(repo, "docs", "wiki", "index.md"), "# c\n")
	path := filepath.Join(repo, "other", "x.md")
	write(t, path, "---\ntitle: x\n---\n\n# x\n")
	found := checkFile(path)
	if len(found) == 0 || found[0].Relative != "x.md" {
		t.Fatalf("findings %+v, want a page relative to its directory",
			found)
	}
}

func TestCheckFileDoesNotClimbPastADeclarationItCannotUse(t *testing.T) {
	// A manifest is the statement of the area one stands in. Climbing
	// past it because it names no `[layout] wiki` would answer with the
	// area that encloses it -- the containment the write barrier
	// insists on at ultra-brain's `pkg/guard/guard.go:334`. With no registration to
	// fall back on,
	// the file's own directory is the answer.
	repo := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(repo, "state"))
	write(t, filepath.Join(repo, "inner", ".loomux", "config.toml"),
		"[area]\nscope = \"project/p\"\n")
	path := filepath.Join(repo, "inner", "deep", "x.md")
	write(t, path, "---\ntitle: x\n---\n\n# x\n")
	found := checkFile(path)
	if len(found) == 0 || found[0].Relative != "x.md" {
		t.Fatalf("findings %+v, want a page relative to its directory",
			found)
	}
}

func TestCheckFileKeepsTheRootWhenTheDeclarationIsBroken(t *testing.T) {
	// A declaration that does not parse costs the vocabulary, never the
	// root: the registration still says where the bundle is, and
	// `unknown-type` then judges by the built-in set alone. The defect
	// itself is reported by the two wider widths, which read the same
	// file -- this width has no area to report it on.
	base := t.TempDir()
	state := filepath.Join(base, "state")
	inner := filepath.Join(base, "space")
	write(t, filepath.Join(state, "registry.toml"), fmt.Sprintf(
		"[[area]]\nscope = \"project/space\"\npath = %q\nwiki = %q\n",
		filepath.ToSlash(inner), filepath.ToSlash(inner)))
	t.Setenv(config.StateDirEnv, state)
	write(t, filepath.Join(inner, ".loomux", "config.toml"), "[area\n")
	path := filepath.Join(inner, "deep", "x.md")
	write(t, path, "---\ntitle: x\ntype: manual\n---\n\n# x\n")
	found := checkFile(path)
	if len(found) == 0 || found[0].Relative != "deep/x.md" {
		t.Fatalf("findings %+v, want a page relative to the inner wiki",
			found)
	}
	if n := countRule(found, "unknown-type"); n != 1 {
		t.Fatalf("%d unknown-type findings, want the built-in set alone", n)
	}
	for _, rule := range []string{"manifest-unreadable",
		"manifest-layout-invalid"} {
		if n := countRule(found, rule); n != 0 {
			t.Errorf("%d %s findings from the fast path", n, rule)
		}
	}
}

func TestCheckFileFallsBackToTheFilesOwnDirectory(t *testing.T) {
	// Neither a declaration above it nor an entry in the registration:
	// Python answers `file_path.parent` in exactly this state
	// (`src/brain/wiki/lint.py:599`), and so does this.
	dir := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(dir, "state"))
	path := filepath.Join(dir, "x.md")
	write(t, path, "---\ntitle: x\n---\n\n# x\n")
	found := checkFile(path)
	if len(found) == 0 || found[0].Relative != "x.md" {
		t.Fatalf("findings %+v, want a page relative to its directory",
			found)
	}
}

func TestCheckFileAnswersNothingForAFileItCannotRead(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(dir, "state"))
	if found := checkFile(filepath.Join(dir, "gone.md")); found != nil {
		t.Fatalf("%+v for a file that is not there, want nothing", found)
	}
}

func TestAnAreaWithoutAWikiIsNotRead(t *testing.T) {
	// `_lint_targets` lints only areas whose `wiki_path` is set
	// (`src/brain/cli.py:1489`), and an area that declares none has no
	// bundle to read.
	base := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(base, "state"))
	a := config.Area{Scope: "project/p", Path: base}
	// Its declaration is not read either, broken though it is: an area
	// Python never lints must not be reported by this width over a file
	// no rule of it would have opened.
	write(t, filepath.Join(base, ".loomux", "config.toml"), "[area\n")
	if found := checkAll([]config.Area{a}, []config.Area{a}); len(found) != 0 {
		t.Fatalf("%+v for an area with no wiki, want nothing", found)
	}
}

func TestABundleThatCannotBeWalkedIsNotADefect(t *testing.T) {
	// A wiki path that is not there yields no pages and no findings,
	// rather than an error the signature could not carry anyway.
	base := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(base, "state"))
	a := config.Area{Scope: "project/p", Path: base,
		WikiPath: filepath.Join(base, "gone")}
	if found := checkBundle(a, []config.Area{a}); len(found) != 0 {
		t.Fatalf("%+v for a wiki that is not there, want nothing", found)
	}
}

func TestTwoAreasOnOneWikiAreDecidedByTheFirstEntry(t *testing.T) {
	// Two entries may name the same wiki: `read_registry` refuses two
	// areas with one scope (`src/brain/registry.py:29-30`) and says
	// nothing about two areas with one wiki path. The bundle root is
	// then the same string either way, so only the *area* differs -- and
	// with it the manifest the vocabulary comes from.
	//
	// The choice is the reader's, because no spec makes one, and it is
	// made explicit here rather than left to the comparison: the first
	// entry of equal length keeps the answer. That is what `>` says and
	// `>=` would not, and without this case the two are the same.
	base := t.TempDir()
	state := filepath.Join(base, "state")
	wiki := filepath.Join(base, "wiki")
	first := filepath.Join(base, "first")
	second := filepath.Join(base, "secnd")
	write(t, filepath.Join(state, "registry.toml"), fmt.Sprintf(
		"[[area]]\nscope = \"a/one\"\npath = %q\nwiki = %q\n\n"+
			"[[area]]\nscope = \"a/two\"\npath = %q\nwiki = %q\n",
		filepath.ToSlash(first), filepath.ToSlash(wiki),
		filepath.ToSlash(second), filepath.ToSlash(wiki)))
	t.Setenv(config.StateDirEnv, state)
	// Only the first area declares the type. The second is there to be
	// wrongly preferred.
	write(t, filepath.Join(first, ".loomux", "config.toml"),
		"[area]\nscope = \"a/one\"\n\n[wiki]\ntypes = [\"manual\"]\n")
	write(t, filepath.Join(second, ".loomux", "config.toml"),
		"[area]\nscope = \"a/two\"\n")
	path := filepath.Join(wiki, "x.md")
	write(t, path, "---\ntitle: x\ntype: manual\n---\n\n# x\n")
	if n := countRule(checkFile(path), "unknown-type"); n != 0 {
		t.Fatalf("%d unknown-type findings; the second entry won", n)
	}
}

// federationFixture is the registration the four bundle-width federation
// tests share: a signpost that names nobody, a shared area that cites a
// project page, and the project it cites. Every one of the three is a
// subject of at least one of the two rules, so a run that asked the
// wrong subject shows it here.
func federationFixture(t *testing.T) []config.Area {
	t.Helper()
	base := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(base, "state"))
	post := area(t, base, "wegweiser")
	post.Signpost = true
	// The signpost names one of the two other areas and not the other,
	// so a count is a statement: a run that read no catalog at all
	// would report both, and a run that took the subjects for the
	// expected areas would report neither.
	write(t, filepath.Join(post.WikiPath, "index.md"),
		"# c\n\n* [a](a.md) -- first\n* [b](b.md) -- second\n"+
			"* [other](../../project-other/wiki) -- third\n")
	shared := area(t, base, "shared/demo")
	shared.Shared = true
	// The two pages `area` writes cite `brain://x/y`, which no shared
	// scope owns and which the rule would therefore report as well.
	// Rewritten to cite this area itself, so that the one page below is
	// the only subject and a count is a statement about it.
	for _, name := range []string{"a.md", "b.md"} {
		other := "b.md"
		if name == "b.md" {
			other = "a.md"
		}
		write(t, filepath.Join(shared.WikiPath, name),
			strings.Replace(page(name), "brain://x/y",
				"brain://shared/demo/self", 1)+"\n[o]("+other+")\n")
	}
	write(t, filepath.Join(shared.WikiPath, "p.md"),
		"---\ntitle: p\ndescription: d\ntype: Topic\n"+
			"sources:\n  - id: s1\n    resource: brain://project/other/z\n"+
			"    doc_id: d1\n    content_hash: h1\n    revision: 1\n"+
			"---\n\n[a](a.md)\n")
	// A second shared area, cited by a page of the first. It is not a
	// subject of a bundle run over `shared/demo`, so a rule that built
	// its shared set from the subjects rather than from the whole
	// registration would report this citation -- and citing a shared
	// area is the permitted direction.
	write(t, filepath.Join(shared.WikiPath, "q.md"),
		strings.Replace(page("q"), "brain://x/y",
			"brain://shared/second/z", 1)+"\n[a](a.md)\n")
	write(t, filepath.Join(shared.WikiPath, "index.md"),
		"# c\n\n* [a](a.md) -- first\n* [b](b.md) -- second\n"+
			"* [p](p.md) -- third\n* [q](q.md) -- fourth\n")
	second := area(t, base, "shared/second")
	second.Shared = true
	return []config.Area{post, shared, area(t, base, "project/other"),
		second}
}

func TestABundleRunAsksWrongDirectionAboutItsOwnArea(t *testing.T) {
	// Python runs `wrong_direction` in every area's lint, not only in
	// the sweep (`src/brain/cli.py:1524-1556` hands `shared_scopes` in
	// for each target). The rule needs the registration, never a second
	// bundle's pages: it reads `bundles[area.Scope]` alone
	// (`internal/brain/check/house/federation.go:186-208`).
	areas := federationFixture(t)
	found := checkBundle(areas[1], areas)
	if n := countRule(found, "wrong-direction"); n != 1 {
		t.Fatalf("%d wrong-direction findings, want exactly 1", n)
	}
	if got := ruleOnly(found, "wrong-direction").Scope; got != "shared/demo" {
		t.Fatalf("scope %q, want \"shared/demo\"", got)
	}
}

func TestABundleRunAsksTheSignpostWhoIsMissing(t *testing.T) {
	// Python asks it whenever the linted area is the signpost
	// (`src/brain/cli.py:1534-1542`, `expected_targets=... if
	// area.signpost else ()`), and the rule reads only the signpost's
	// own catalog plus the registration.
	areas := federationFixture(t)
	found := checkBundle(areas[0], areas)
	if n := countRule(found, "unlisted-area"); n != 2 {
		t.Fatalf("%d unlisted-area findings, want 2", n)
	}
	for _, f := range found {
		if f.Rule == "unlisted-area" &&
			strings.Contains(f.Message, "project/other") {
			t.Fatalf("the area the catalog links was reported: %s",
				f.Message)
		}
	}
}

func TestABundleRunDoesNotAskAForeignSignpost(t *testing.T) {
	// The input nobody had run: a bundle width that handed every area
	// to the rule would ask the signpost about a catalog it never read,
	// and `unlistedArea` errs towards speaking on a bundle it was not
	// given (`internal/brain/check/house/federation.go:34-42`) -- so it would
	// report every area of the federation while checking one that is
	// not the signpost at all. Python asks the rule only of the area it
	// lints.
	areas := federationFixture(t)
	if n := countRule(checkBundle(areas[1], areas), "unlisted-area"); n != 0 {
		t.Fatalf("%d unlisted-area findings for a non-signpost area", n)
	}
	if n := countRule(checkBundle(areas[2], areas), "wrong-direction"); n != 0 {
		t.Fatalf("%d wrong-direction findings for a project area", n)
	}
}

func TestTheTwoWidthsAgreeOnEveryArea(t *testing.T) {
	// The contract the gap broke: a bundle run must say about one area
	// what the sweep says about it. Rendered rather than compared field
	// by field, because the rendering is what a reader and a hook see.
	areas := federationFixture(t)
	all := checkAll(areas, areas)
	for _, a := range areas {
		var mine []check.Finding
		for _, f := range all {
			if f.Scope == a.Scope {
				mine = append(mine, f)
			}
		}
		want := check.Render(mine, true)
		if got := check.Render(checkBundle(a, areas), true); got != want {
			t.Fatalf("scope %q:\nbundle:\n%s\nall:\n%s",
				a.Scope, got, want)
		}
	}
}

func TestTheWideRunJudgesAgainstTheWholeRegistration(t *testing.T) {
	// `Targets` drops an area whose wiki is missing, and until now the
	// wide run handed that shortened list to the federation rules as
	// both the subjects and the registration. A shared area with no
	// `wiki` entry then vanished from the shared set, and a citation
	// into it turned from allowed into `house/wrong-direction` -- an
	// error, and one the bundle width does not report, so the two
	// widths contradicted each other on the same page.
	//
	// Python never had the question: `shared_scopes` comes from
	// `read_registry`, the whole file (`src/brain/cli.py:1532-1533`).
	base := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(base, "state"))
	a := area(t, base, "knowledge")
	a.Shared = true
	write(t, filepath.Join(a.WikiPath, "cites.md"),
		"---\ntype: topic\ntitle: t\ndescription: d\nsources:\n"+
			"  - id: s\n    resource: brain://engineering/nowiki/x\n"+
			"    doc_id: d\n    content_hash: h\n    revision: 1\n"+
			"---\nb\n")
	// Shared, and with no wiki of its own -- exactly what `Targets`
	// refuses to walk and what the registration still declares.
	nowiki := config.Area{Scope: "engineering/nowiki",
		Path: filepath.Join(base, "nowiki"), Shared: true}

	all := []config.Area{a, nowiki}
	walkable := []config.Area{a}
	// Counted by the scope the message names, not by rule: `page()`
	// cites `brain://x/y` on every fixture page, so the shared area
	// carries two wrong-direction findings of its own either way.
	if n := citing(checkAll(walkable, all), "engineering/nowiki"); n != 0 {
		t.Fatalf("%d findings against a shared area the registration "+
			"declares, want 0", n)
	}
	// And the rule still speaks when the cited area really is a project.
	nowiki.Shared = false
	all = []config.Area{a, nowiki}
	if n := citing(checkAll(walkable, all), "engineering/nowiki"); n != 1 {
		t.Fatalf("%d findings against a project, want 1", n)
	}
}

// citing counts the findings whose message names this scope. The rule
// name alone would not do here: the fixture pages cite a scope of their
// own, so a run has wrong-direction findings that this test is not about.
func citing(found []check.Finding, scope string) int {
	n := 0
	for _, f := range found {
		if strings.Contains(f.Message, scope) {
			n++
		}
	}
	return n
}

// The bundle is read in the order Python's `sorted(rglob("*.md"))` gives on
// Windows -- component by component, each folded to lower case -- and not in
// WalkDir's byte order, which puts `B.md` before `a.md` and `a-c.md` before
// the directory `a`. The sort after the rules is stable, so this order is the
// one findings equal on every key keep.
func TestReadBundleReadsInPythonsOrder(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"B.md", "a.md", "a-c.md", filepath.Join("a", "x.md")} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), []byte("# p\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var got []string
	for _, p := range readBundle(root) {
		got = append(got, p.Relative)
	}
	want := []string{"a/x.md", "a-c.md", "a.md", "B.md"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("read order %q, want %q", got, want)
	}
}
