package status

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/testlock"
)

// neverReconciled is the first line of every world without a usable stamp.
const neverReconciled = "last reconcile: never; run `brain reconcile`"

// asked is the moment every test asks at: the stale stamps lie before it and
// the fresh ones after, as in the recorded worlds.
func asked() time.Time { return time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC) }

// world keeps the registry, the legacy state directory and the writable
// areas in three temporary directories of their own, so a test notices when
// Lines reads one of them in place of another.
type world struct {
	t        *testing.T
	registry string
	legacy   string
	repos    string
	entries  strings.Builder
}

func newWorld(t *testing.T) *world {
	t.Helper()
	return &world{t: t, registry: t.TempDir(), legacy: t.TempDir(), repos: t.TempDir()}
}

// write puts content at path and creates the directories above it.
func (w *world) write(path, content string) {
	w.t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		w.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		w.t.Fatal(err)
	}
}

// register appends one [[area]] table. The path goes in with forward
// slashes, the way the real registry spells Windows paths, inside a TOML
// literal string so that no character of it is read as an escape.
func (w *world) register(scope, path string, readonly bool) {
	fmt.Fprintf(&w.entries, "[[area]]\nscope = '%s'\npath = '%s'\nreadonly = %t\n\n",
		scope, filepath.ToSlash(path), readonly)
}

// writable registers scope as a writable area in repos/<name>, writes its
// manifest there and returns the directory, which holds its artefacts too.
func (w *world) writable(scope, name, manifest string) string {
	dir := filepath.Join(w.repos, name)
	w.register(scope, dir, false)
	w.write(filepath.Join(dir, ".loomux", "config.toml"), fmt.Sprintf("[area]\nscope = '%s'\n%s", scope, manifest))
	return dir
}

// readOnly registers scope as a read-only area at path, writes its manifest
// into legacy/areas/<flat> and returns that directory, which holds its
// artefacts too.
func (w *world) readOnly(scope, flat, path, manifest string) string {
	dir := filepath.Join(w.legacy, "areas", flat)
	w.register(scope, path, true)
	w.write(filepath.Join(dir, ".loomux", "config.toml"), fmt.Sprintf("[area]\nscope = '%s'\n%s", scope, manifest))
	return dir
}

func (w *world) stamp(content string) {
	w.write(filepath.Join(w.legacy, "maintenance", "last-run.txt"), content)
}

// graph writes a graph.json without edges; dropped is the inside of the
// reasons object.
func (w *world) graph(dir string, total, resolved int, dropped string) {
	w.write(filepath.Join(dir, "graph.json"), fmt.Sprintf(
		`{"edges": [], "links": {"total": %d, "resolved": %d, "dropped": {%s}}}`, total, resolved, dropped))
}

// identities writes a register: the header, then the rows.
func (w *world) identities(dir string, rows ...string) {
	w.write(filepath.Join(dir, "_identities.tsv"), "doc_id\tpfad\tcontent_hash\trevision\n"+strings.Join(rows, ""))
}

func row(docID, relative, hash string) string {
	return docID + "\t" + relative + "\t" + hash + "\t1\n"
}

// lines writes the registry as registered so far and asks.
func (w *world) lines(ch privacy.Channel, port search.SearchPort) ([]string, error) {
	w.t.Helper()
	w.write(filepath.Join(w.registry, "registry.toml"), w.entries.String())
	return Lines(ch, port, w.registry, w.legacy, asked())
}

func listing(paths ...string) search.ScriptedIndexed {
	return search.ScriptedIndexed{Paths: paths}
}

func expect(t *testing.T, got []string, err error, want ...string) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("lines\n got %q\nwant %q", got, want)
	}
}

// refused asserts an abort: an error, no lines, no backlog asked, and the
// engine asked for exactly the listings named. It hands the error back for a
// check of its wording.
func refused(t *testing.T, got []string, err error, port *search.FakePort, listed ...string) error {
	t.Helper()
	if err == nil || got != nil {
		t.Fatalf("got %q, %v; want an error and no lines", got, err)
	}
	if !reflect.DeepEqual(port.Listed, listed) || port.SearchableCounts != 0 {
		t.Fatalf("engine asked for %q and the backlog %d times; want %q and never",
			port.Listed, port.SearchableCounts, listed)
	}
	return err
}

func TestLinesNameAMissingStampNever(t *testing.T) {
	w := newWorld(t)
	port := search.NewFakePort()
	got, err := w.lines(privacy.ChannelLocal, port)
	expect(t, got, err, neverReconciled)
	if port.SearchableCounts != 1 {
		t.Fatalf("backlog asked %d times, want once", port.SearchableCounts)
	}
}

func TestLinesNameTheStampAndWhetherItIsADayOld(t *testing.T) {
	// Measured at the reference: datetime.fromisoformat(s).isoformat() and
	// now - stamp >= timedelta(hours=24) with now = 2026-09-15T12:00:00+00:00.
	for _, c := range []struct{ stamp, line string }{
		{"2000-01-01T00:00:00\n", neverReconciled},
		{"2000-01-01T00:00:00+00:00\n", "last reconcile: 2000-01-01T00:00:00+00:00; older than 24 h, run `brain reconcile`"},
		{"2000-01-01T00:00:00Z\n", "last reconcile: 2000-01-01T00:00:00+00:00; older than 24 h, run `brain reconcile`"},
		{"2026-09-14T12:00:00+00:00\n", "last reconcile: 2026-09-14T12:00:00+00:00; older than 24 h, run `brain reconcile`"},
		{"2026-09-14T12:00:00.000001+00:00\n", "last reconcile: 2026-09-14T12:00:00.000001+00:00"},
		{"2999-01-01T00:00:00.255+02:00\n", "last reconcile: 2999-01-01T00:00:00.255000+02:00"},
	} {
		w := newWorld(t)
		w.stamp(c.stamp)
		got, err := w.lines(privacy.ChannelLocal, search.NewFakePort())
		expect(t, got, err, c.line)
	}
}

func TestLinesAbortWhenTheStampCannotBeRead(t *testing.T) {
	w := newWorld(t)
	w.stamp("2000-01-01T00:00:00+00:00\n")
	testlock.Lock(t, filepath.Join(w.legacy, "maintenance", "last-run.txt"))
	port := search.NewFakePort()
	got, err := w.lines(privacy.ChannelLocal, port)
	refused(t, got, err, port)
}

func TestLinesAbortWithoutARegistry(t *testing.T) {
	port := search.NewFakePort()
	got, err := Lines(privacy.ChannelLocal, port, t.TempDir(), t.TempDir(), asked())
	refused(t, got, err, port)
}

func TestLinesAbortWhenAWritableAreaIsGone(t *testing.T) {
	// A writable area keeps its manifest under its own path, so a path that
	// is gone is a manifest that is missing, and the reference aborts on it.
	w := newWorld(t)
	w.register("alpha", filepath.Join(w.repos, "gone"), false)
	port := search.NewFakePort()
	got, err := w.lines(privacy.ChannelLocal, port)
	refused(t, got, err, port)
}

func TestLinesCountTheBacklogOnce(t *testing.T) {
	for _, c := range []struct {
		pending search.ScriptedPending
		want    []string
	}{
		{search.ScriptedPending{Count: 7}, []string{"7 documents are indexed but not yet searchable; run `brain embed`"}},
		{search.ScriptedPending{Err: errors.New("qmd exited with 1: boom")},
			[]string{"the search engine did not answer (qmd exited with 1: boom); the backlog was not counted"}},
		{search.ScriptedPending{Count: 0}, nil},
	} {
		w := newWorld(t)
		port := search.NewFakePort()
		port.Pending = []search.ScriptedPending{c.pending}
		got, err := w.lines(privacy.ChannelLocal, port)
		expect(t, got, err, append([]string{neverReconciled}, c.want...)...)
		if port.SearchableCounts != 1 {
			t.Fatalf("backlog asked %d times, want once", port.SearchableCounts)
		}
	}
}

func TestLinesNameIncludeGlobsTheEngineDoesNotSee(t *testing.T) {
	w := newWorld(t)
	for _, a := range []struct{ scope, include string }{
		{"one", `["docs/**/*.md"]`},
		{"two", `["*.md", "extra/*.md"]`},
		{"three", `["docs/**/*.md", "README*.md", "notes/*.md"]`},
	} {
		w.graph(w.writable(a.scope, a.scope, "[index]\ninclude = "+a.include+"\n"), 0, 0, "")
	}
	port := search.NewFakePort()
	port.Listings = map[string]search.ScriptedIndexed{"one": listing(), "two": listing(), "three": listing()}
	got, err := w.lines(privacy.ChannelLocal, port)
	expect(t, got, err, neverReconciled,
		"two: the search engine sees only *.md; also declared: extra/*.md",
		"three: the search engine sees only docs/**/*.md; also declared: README*.md, notes/*.md")
}

func TestLinesSkipAReadOnlyAreaWhosePathIsGone(t *testing.T) {
	// The registry spells the path with a `.` segment and a trailing slash;
	// str(Path) drops both and turns the separators.
	w := newWorld(t)
	w.readOnly("project/beta", "project-beta", filepath.ToSlash(w.repos)+"/gone/./beta/",
		"[index]\ninclude = [\"*.md\", \"extra/*.md\"]\n")
	port := search.NewFakePort()
	got, err := w.lines(privacy.ChannelLocal, port)
	expect(t, got, err, neverReconciled,
		"project/beta: the search engine sees only *.md; also declared: extra/*.md",
		"project/beta: "+filepath.Join(w.repos, "gone", "beta")+" does not exist; skipped")
	if len(port.Listed) != 0 {
		t.Fatalf("engine asked for %q; a skipped area is not compared", port.Listed)
	}
}

func TestLinesSkipAnAreaThatWasNeverIndexed(t *testing.T) {
	// A read-only area's graph lies in the legacy directory: `kept` has one
	// there and none in the area, `bare` one in the area and none there. Only
	// `kept` has a listing scripted; an unscripted listing would add a line.
	w := newWorld(t)
	w.writable("gamma", "repo-gamma", "")
	w.graph(w.readOnly("kept", "kept", w.repos, ""), 0, 0, "")
	w.readOnly("bare", "bare", w.repos, "")
	w.graph(w.repos, 0, 0, "")
	port := search.NewFakePort()
	port.Listings["kept"] = listing()
	got, err := w.lines(privacy.ChannelLocal, port)
	expect(t, got, err, neverReconciled,
		"gamma: never indexed; run `brain reindex`",
		"bare: never indexed; run `brain reindex`")
}

func TestLinesReportLinksResolvedForLessThanHalf(t *testing.T) {
	for _, c := range []struct {
		total, resolved int
		dropped         string
		want            []string
	}{
		{5, 2, `"unknown_target": 2, "external": 1`, []string{"alpha: only 2 of 5 links resolved (external=1, unknown_target=2)"}},
		{4, 1, ``, []string{"alpha: only 1 of 4 links resolved ()"}},
		{4, 2, `"external": 2`, nil},
		{0, 0, ``, nil},
	} {
		w := newWorld(t)
		w.graph(w.writable("alpha", "repo-alpha", ""), c.total, c.resolved, c.dropped)
		port := search.NewFakePort()
		port.Listings["alpha"] = listing()
		got, err := w.lines(privacy.ChannelLocal, port)
		expect(t, got, err, append([]string{neverReconciled}, c.want...)...)
	}
}

func TestLinesAbortOnABrokenGraphBeforeAnyRegister(t *testing.T) {
	// The skipped area `one` carries a broken register as well; the graph of
	// `two` is met first, as in the reference.
	w := newWorld(t)
	one := w.readOnly("one", "one", filepath.ToSlash(w.repos)+"/gone", "")
	w.write(filepath.Join(one, "_identities.tsv"), "doc_id\tpfad\tcontent_hash\trevision\nbroken\n")
	two := w.writable("two", "repo-two", "")
	w.write(filepath.Join(two, "graph.json"), "{\"edges\": []}\n")
	port := search.NewFakePort()
	port.Listings["two"] = listing()
	got, err := w.lines(privacy.ChannelLocal, port)
	err = refused(t, got, err, port)
	if want := filepath.Join(two, "graph.json") + ": graph is missing links; delete it and run `brain reindex`"; err.Error() != want {
		t.Fatalf("error\n got %q\nwant %q", err, want)
	}
}

// collectionNotFound is what the CLI port hands on for `qmd ls` of a
// collection qmd 2.8.3 does not know (measured 2026-09-15): stderr stripped
// at both ends, one newline left inside.
const collectionNotFound = "qmd exited with 1: Collection not found: delta\nRun 'qmd ls' to see available collections."

// One file name in both normal forms, and the ellipsis of L6a, spelt by code
// point so that no editor or copy can normalise them into each other.
const (
	nfdCafe  = "Cafe" + string(rune(0x0301)) + ".md"
	nfcCafe  = "Caf" + string(rune(0x00e9)) + ".md"
	ellipsis = string(rune(0x2026))
)

func TestLinesNameDocumentsTheEngineCannotReturn(t *testing.T) {
	// alpha: `never` and `unsearched` stay out of both counts, the register's
	// NFD name is the engine's NFC one, four missing end in ", " + ellipsis.
	// project/three: exactly three missing, no tail; its collection is the
	// flat scope. whole: nothing missing. empty: no register, asked anyway.
	w := newWorld(t)
	alpha := w.writable("alpha", "repo-alpha", "[index]\nunsearched = [\"drafts/**\"]\n\n[privacy]\nnever = [\"secret/**\"]\n")
	w.graph(alpha, 0, 0, "")
	w.identities(alpha, row("01", "a.md", "sha256:1"), row("02", "b.md", "sha256:2"), row("03", "c.md", "sha256:3"),
		row("04", "d.md", "sha256:4"), row("05", "e.md", "sha256:5"), row("06", "secret/x.md", "sha256:6"),
		row("07", "drafts/y.md", "sha256:7"), row("08", nfdCafe, "sha256:8"))
	three := w.writable("project/three", "repo-three", "")
	w.graph(three, 0, 0, "")
	w.identities(three, row("11", "c.md", "sha256:a"), row("12", "a.md", "sha256:b"), row("13", "b.md", "sha256:c"))
	whole := w.writable("whole", "repo-whole", "")
	w.graph(whole, 0, 0, "")
	w.identities(whole, row("21", "x.md", "sha256:d"))
	w.graph(w.writable("empty", "repo-empty", ""), 0, 0, "")
	port := search.NewFakePort()
	port.Listings = map[string]search.ScriptedIndexed{
		"alpha":         listing("b.md", nfcCafe),
		"project-three": listing(),
		"whole":         listing("x.md"),
		"empty":         listing(),
	}
	got, err := w.lines(privacy.ChannelLocal, port)
	expect(t, got, err, neverReconciled,
		"alpha: 4 of 6 indexed documents are unknown to the search engine, so `brain search` can never return them; e.g. a.md, c.md, d.md, "+ellipsis,
		"project/three: 3 of 3 indexed documents are unknown to the search engine, so `brain search` can never return them; e.g. a.md, b.md, c.md")
	if want := []string{"alpha", "project-three", "whole", "empty"}; !reflect.DeepEqual(port.Listed, want) {
		t.Fatalf("engine asked for %q, want %q", port.Listed, want)
	}
}

func TestLinesSayWhenTheEngineDoesNotList(t *testing.T) {
	w := newWorld(t)
	w.graph(w.writable("delta", "repo-delta", ""), 0, 0, "")
	port := search.NewFakePort()
	port.Listings["delta"] = search.ScriptedIndexed{Err: errors.New(collectionNotFound)}
	got, err := w.lines(privacy.ChannelLocal, port)
	expect(t, got, err, neverReconciled,
		"delta: the search engine did not answer (qmd exited with 1: Collection not found: delta\nRun 'qmd ls' to see available collections.); its index was not compared")
}

func TestLinesAbortOnABrokenRegisterBeforeAskingTheEngine(t *testing.T) {
	w := newWorld(t)
	one := w.writable("one", "repo-one", "")
	w.graph(one, 0, 0, "")
	w.write(filepath.Join(one, "_identities.tsv"), "doc_id\tpfad\tcontent_hash\trevision\nbroken\n")
	w.graph(w.writable("two", "repo-two", ""), 0, 0, "")
	port := search.NewFakePort()
	port.Listings = map[string]search.ScriptedIndexed{"one": listing(), "two": listing()}
	got, err := w.lines(privacy.ChannelLocal, port)
	err = refused(t, got, err, port)
	if want := filepath.Join(one, "_identities.tsv") + ": line 2: expected 4 tab-separated fields, found 1"; err.Error() != want {
		t.Fatalf("error\n got %q\nwant %q", err, want)
	}
}

func TestLinesNameContentKeptUnderSeveralPaths(t *testing.T) {
	// Lines follow the hash, not the path. project/beta (path gone) and gamma
	// (never indexed) are skipped by the loop and still counted here. `222`
	// lost its pair to `never`; `999` is held by withheld paths only.
	w := newWorld(t)
	alpha := w.writable("alpha", "repo-alpha", "[privacy]\nnever = [\"secret/**\"]\n")
	w.graph(alpha, 0, 0, "")
	w.identities(alpha, row("01", "a.md", "sha256:222"), row("02", "b.md", "sha256:111"), row("03", "c.md", "sha256:333"),
		row("06", "secret/x.md", "sha256:222"), row("09", "secret/w1.md", "sha256:999"), row("10", "secret/w2.md", "sha256:999"))
	beta := w.readOnly("project/beta", "project-beta", filepath.ToSlash(w.repos)+"/gone", "")
	w.identities(beta, row("11", "z.md", "sha256:111"))
	w.identities(w.writable("gamma", "repo-gamma", ""), row("21", "g.md", "sha256:333"))
	delta := w.writable("delta", "repo-delta", "")
	w.graph(delta, 0, 0, "")
	w.identities(delta, row("31", "a2.md", "sha256:333"))
	port := search.NewFakePort()
	port.Listings = map[string]search.ScriptedIndexed{"alpha": listing("a.md", "b.md", "c.md"), "delta": listing("a2.md")}
	got, err := w.lines(privacy.ChannelLocal, port)
	expect(t, got, err, neverReconciled,
		"project/beta: "+filepath.Join(w.repos, "gone")+" does not exist; skipped",
		"gamma: never indexed; run `brain reindex`",
		"same content hash under 2 paths: alpha/b.md, project/beta/z.md",
		"alpha/a.md: same content hash as a path excluded by [privacy] never",
		"same content hash under 3 paths: alpha/c.md, delta/a2.md, gamma/g.md")
}

func TestLinesAbortOnABrokenRegisterOfASkippedArea(t *testing.T) {
	// The loop skips `one` and compares `two`; the register of `one` is read
	// for the shared hashes afterwards, before the backlog is asked.
	w := newWorld(t)
	one := w.readOnly("one", "one", filepath.ToSlash(w.repos)+"/gone", "")
	w.write(filepath.Join(one, "_identities.tsv"), "doc_id\tpfad\tcontent_hash\trevision\nbroken\n")
	w.graph(w.writable("two", "repo-two", ""), 0, 0, "")
	port := search.NewFakePort()
	port.Listings["two"] = listing()
	got, err := w.lines(privacy.ChannelLocal, port)
	err = refused(t, got, err, port, "two")
	if want := filepath.Join(one, "_identities.tsv") + ": line 2: expected 4 tab-separated fields, found 1"; err.Error() != want {
		t.Fatalf("error\n got %q\nwant %q", err, want)
	}
}

// alphaArea is area alpha of the measured worlds: three include globs, a
// `never` and an `unsearched` glob, two of five links resolved, and a
// register the engine lists two entries of.
func alphaArea(w *world) {
	dir := w.writable("alpha", "repo-alpha", "[index]\n"+
		"include = [\"docs/**/*.md\", \"README*.md\", \"notes/*.md\"]\n"+
		"unsearched = [\"drafts/**\"]\n\n[privacy]\nnever = [\"secret/**\"]\n")
	w.graph(dir, 5, 2, `"unknown_target": 2, "external": 1`)
	w.identities(dir, row("01", "a.md", "sha256:222"), row("02", "b.md", "sha256:111"), row("03", "c.md", "sha256:333"),
		row("04", "d.md", "sha256:444"), row("05", "e.md", "sha256:555"), row("06", "secret/x.md", "sha256:222"),
		row("07", "drafts/y.md", "sha256:666"), row("08", nfdCafe, "sha256:777"),
		row("09", "secret/w1.md", "sha256:999"), row("10", "secret/w2.md", "sha256:999"))
}

// epsilonArea is a `local_only` area with an empty graph and no register.
func epsilonArea(w *world) {
	w.graph(w.writable("epsilon", "repo-epsilon", "\n[privacy]\nmode = \"local_only\"\n"), 0, 0, "")
}

func TestLinesMatchTheMeasuredWorlds(t *testing.T) {
	first := func(t *testing.T, ch privacy.Channel, listed ...string) {
		w := newWorld(t)
		w.stamp("2000-01-01T00:00:00+00:00\n")
		alphaArea(w)
		beta := w.readOnly("project/beta", "project-beta", filepath.ToSlash(w.repos)+"/gone/./beta/",
			"[index]\ninclude = [\"*.md\", \"extra/*.md\"]\n")
		w.identities(beta, row("11", "z.md", "sha256:111"))
		w.identities(w.writable("gamma", "repo-gamma", ""), row("21", "g.md", "sha256:333"))
		delta := w.writable("delta", "repo-delta", "")
		w.graph(delta, 4, 1, "")
		w.identities(delta, row("31", "a2.md", "sha256:333"))
		epsilonArea(w)
		port := search.NewFakePort()
		port.Listings = map[string]search.ScriptedIndexed{
			"alpha":   listing("b.md", nfcCafe),
			"delta":   {Err: errors.New(collectionNotFound)},
			"epsilon": listing(),
		}
		port.Pending = []search.ScriptedPending{{Count: 7}}
		got, err := w.lines(ch, port)
		expect(t, got, err,
			"last reconcile: 2000-01-01T00:00:00+00:00; older than 24 h, run `brain reconcile`",
			"alpha: the search engine sees only docs/**/*.md; also declared: README*.md, notes/*.md",
			"alpha: only 2 of 5 links resolved (external=1, unknown_target=2)",
			"alpha: 4 of 6 indexed documents are unknown to the search engine, so `brain search` can never return them; e.g. a.md, c.md, d.md, "+ellipsis,
			"project/beta: the search engine sees only *.md; also declared: extra/*.md",
			"project/beta: "+filepath.Join(w.repos, "gone", "beta")+" does not exist; skipped",
			"gamma: never indexed; run `brain reindex`",
			"delta: only 1 of 4 links resolved ()",
			"delta: the search engine did not answer ("+collectionNotFound+"); its index was not compared",
			"same content hash under 2 paths: alpha/b.md, project/beta/z.md",
			"alpha/a.md: same content hash as a path excluded by [privacy] never",
			"same content hash under 3 paths: alpha/c.md, delta/a2.md, gamma/g.md",
			"7 documents are indexed but not yet searchable; run `brain embed`")
		if !reflect.DeepEqual(port.Listed, listed) {
			t.Fatalf("engine asked for %q, want %q", port.Listed, listed)
		}
	}
	t.Run("world1 local", func(t *testing.T) { first(t, privacy.ChannelLocal, "alpha", "delta", "epsilon") })
	t.Run("world1 cloud", func(t *testing.T) { first(t, privacy.ChannelCloud, "alpha", "delta") })
	t.Run("world2 cloud", func(t *testing.T) {
		w := newWorld(t)
		w.stamp("2999-01-01T00:00:00.255+02:00\n")
		alphaArea(w)
		epsilonArea(w)
		port := search.NewFakePort()
		port.Listings = map[string]search.ScriptedIndexed{"alpha": listing("b.md", nfcCafe), "epsilon": listing()}
		port.Pending = []search.ScriptedPending{{Err: errors.New("qmd exited with 1: boom")}}
		got, err := w.lines(privacy.ChannelCloud, port)
		expect(t, got, err,
			"last reconcile: 2999-01-01T00:00:00.255000+02:00",
			"alpha: the search engine sees only docs/**/*.md; also declared: README*.md, notes/*.md",
			"alpha: only 2 of 5 links resolved (external=1, unknown_target=2)",
			"alpha: 4 of 6 indexed documents are unknown to the search engine, so `brain search` can never return them; e.g. a.md, c.md, d.md, "+ellipsis,
			"alpha/a.md: same content hash as a path excluded by [privacy] never",
			"the search engine did not answer (qmd exited with 1: boom); the backlog was not counted")
		if want := []string{"alpha"}; !reflect.DeepEqual(port.Listed, want) {
			t.Fatalf("engine asked for %q, want %q", port.Listed, want)
		}
	})
}
