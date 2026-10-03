package cli

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/answer"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
)

// brainWorldDirs are the three directories of a brain test world.
type brainWorldDirs struct {
	state, legacy, area string
}

// brainWorld registers one writable area, project/a, whose manifest carries
// the given body after its [area] table; it points both state variables at the
// world and writes the files into the area.
func brainWorld(t *testing.T, manifest string, files map[string]string) brainWorldDirs {
	t.Helper()
	w := brainWorldDirs{state: t.TempDir(), legacy: t.TempDir(), area: t.TempDir()}
	t.Setenv("LOOMUX_STATE_DIR", w.state)
	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", w.legacy)
	writeFile(t, filepath.Join(w.state, "registry.toml"),
		"[[area]]\nscope = \"project/a\"\npath = "+strconv.Quote(filepath.ToSlash(w.area))+"\n")
	writeFile(t, filepath.Join(w.area, ".loomux", "config.toml"), "[area]\nscope = \"project/a\"\n"+manifest)
	for name, body := range files {
		writeFile(t, filepath.Join(w.area, filepath.FromSlash(name)), body)
	}
	return w
}

// brainReadOnlyWorld registers one read-only area, project/r, whose manifest
// and artefacts lie in the legacy directory under areas/project-r.
func brainReadOnlyWorld(t *testing.T, files map[string]string) brainWorldDirs {
	t.Helper()
	w := brainWorldDirs{state: t.TempDir(), legacy: t.TempDir(), area: t.TempDir()}
	t.Setenv("LOOMUX_STATE_DIR", w.state)
	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", w.legacy)
	writeFile(t, filepath.Join(w.state, "registry.toml"),
		"[[area]]\nscope = \"project/r\"\npath = "+strconv.Quote(filepath.ToSlash(w.area))+"\nreadonly = true\n")
	artefacts := filepath.Join(w.legacy, "areas", "project-r")
	writeFile(t, filepath.Join(artefacts, ".loomux", "config.toml"), "[area]\nscope = \"project/r\"\n")
	for name, body := range files {
		writeFile(t, filepath.Join(artefacts, filepath.FromSlash(name)), body)
	}
	return w
}

// stubBrainPorts replaces one of the command line's engines for one test and
// leaves the others as they were, so that two stubs in one test add up.
func stubBrainPorts(t *testing.T, change func(*answer.Ports)) {
	t.Helper()
	saved := brainPorts
	ports := saved()
	change(&ports)
	brainPorts = func() answer.Ports { return ports }
	t.Cleanup(func() { brainPorts = saved })
}

func stubBrainSearchPort(t *testing.T, port search.SearchPort, notice string) {
	t.Helper()
	stubBrainPorts(t, func(p *answer.Ports) {
		p.Search = func(announce func(string)) search.SearchPort {
			if notice != "" {
				announce(notice)
			}
			return port
		}
	})
}

func stubBrainStatusPort(t *testing.T, port search.SearchPort) {
	t.Helper()
	stubBrainPorts(t, func(p *answer.Ports) {
		p.Status = func() search.SearchPort { return port }
	})
}

func stubBrainNow(t *testing.T, now time.Time) {
	t.Helper()
	stubBrainPorts(t, func(p *answer.Ports) {
		p.Now = func() time.Time { return now }
	})
}

// closingPort is a search port that can be closed, like the MCP port.
type closingPort struct {
	*search.FakePort
	closed int
}

func (p *closingPort) Close() error {
	p.closed++
	return nil
}

const brainGraph = `{"scope":"project/a","nodes":[],"edges":[{"from":"notes/a.md","to":"notes/b.md"},{"from":"notes/c.md","to":"notes/a.md"}],"links":{"total":2,"resolved":2,"dropped":{}}}`

func TestBrainUsageErrorsNameTheParserThatRefused(t *testing.T) {
	const top = "usage: loomux brain {search,catalog,read,neighbors,status} ...\n"
	for _, c := range []struct {
		args   []string
		stderr string
	}{
		{[]string{"brain"}, top + "loomux brain: error: the following arguments are required: command\n"},
		{[]string{"brain", "frob"}, top + "loomux brain: error: argument command: invalid choice: 'frob' (choose from 'search', 'catalog', 'read', 'neighbors', 'status')\n"},
		{[]string{"brain", "search"}, "usage: loomux brain search [--scope SCOPE] [--profile {fast,full,keyword}] [-n N] [--channel {local,cloud}] query\n" +
			"loomux brain search: error: the following arguments are required: query\n"},
		{[]string{"brain", "read", "rel"}, "usage: loomux brain read --scope SCOPE [--section SECTION] [--channel {local,cloud}] relative\n" +
			"loomux brain read: error: the following arguments are required: --scope\n"},
		{[]string{"brain", "status", "--channel", "x"}, "usage: loomux brain status [--channel {local,cloud}]\n" +
			"loomux brain status: error: argument --channel: invalid choice: 'x' (choose from 'local', 'cloud')\n"},
		{[]string{"brain", "catalog", "extra"}, top + "loomux brain: error: unrecognized arguments: extra\n"},
	} {
		code, out, errOut := run(c.args...)
		if code != 2 || out != "" || errOut != c.stderr {
			t.Fatalf("%q: code %d, out %q\n got %q\nwant %q", c.args, code, out, errOut, c.stderr)
		}
	}
}

func TestBrainDefaultPortsAreTheQmdPorts(t *testing.T) {
	// Both ports read the machine-wide file; never the real one.
	t.Setenv(config.StateDirEnv, t.TempDir())
	if _, ok := brainPorts().Search(func(string) {}).(*search.QmdMcpPort); !ok {
		t.Fatal("brain search must ask the qmd daemon")
	}
	// No runner of its own: the port runs qmd on its backbone.
	port, ok := brainPorts().Status().(*search.QmdPort)
	if !ok || port.Executable != "qmd" || port.Runner != nil || port.Backbone != search.DefaultBackbone {
		t.Fatalf("brain status must ask the qmd command line, got %#v", port)
	}
}

func TestBrainSearchPrintsHitsAndThenItsNotes(t *testing.T) {
	brainWorld(t, "", nil)
	fake := search.NewFakePort()
	fake.Results = []search.ScriptedSearch{{Hits: []search.SearchHit{
		{Collection: "project-a", Relative: "notes/a.md", Line: 3, Title: "A", Snippet: "one\ntwo", Score: 0.5},
	}}}
	stubBrainSearchPort(t, fake, "starting the search engine")

	code, out, errOut := run("brain", "search", "what", "--profile", "full", "-n", "2")
	wantOut := "brain://project/a/notes/a.md:3  50%  A\n    one\n    two\n\n"
	wantErr := "note: starting the search engine\n" +
		"note: project/a/notes/a.md: hit is not in the register; reindex to catch up\n"
	if code != 0 || out != wantOut || errOut != wantErr {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errOut)
	}
	want := []search.SearchCall{{Query: "what", Collections: []string{"project-a"}, Profile: search.ProfileFull, N: 2}}
	if !reflect.DeepEqual(fake.Calls, want) {
		t.Fatalf("calls %+v", fake.Calls)
	}
}

func TestBrainSearchWithoutMatchesSaysSoAndWhy(t *testing.T) {
	brainWorld(t, "", nil)
	fake := search.NewFakePort()
	fake.Results = []search.ScriptedSearch{{}, {}}
	stubBrainSearchPort(t, fake, "")

	code, out, errOut := run("brain", "search", "nothing")
	wantErr := "note: the search engine answered empty twice in a row on profile fast; an empty answer to a meaning " +
		"search is practically unreachable, so this is more likely a silent failure of the engine than an absence " +
		"of matches (spec 16.14)\n"
	if code != 0 || out != "no matches\n" || errOut != wantErr {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errOut)
	}
	if len(fake.Calls) != 2 || fake.Calls[1].N != 5 || fake.Calls[1].Profile != search.ProfileFast {
		t.Fatalf("calls %+v", fake.Calls)
	}
}

func TestBrainSearchJudgesTheLegacyStampAtBrainNow(t *testing.T) {
	w := brainWorld(t, "", map[string]string{
		"_identities.tsv": "doc_id\tpfad\tcontent_hash\trevision\nd1\tnotes/a.md\tsha256:00\t1\n",
	})
	writeFile(t, filepath.Join(w.legacy, "maintenance", "last-run.txt"), "2999-01-01T00:00:00+00:00\n")
	stubBrainNow(t, time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC))
	fake := search.NewFakePort()
	fake.Results = []search.ScriptedSearch{{Hits: []search.SearchHit{
		{Collection: "project-a", Relative: "notes/a.md", Line: 1, Title: "A", Score: 1},
	}}}
	stubBrainSearchPort(t, fake, "")

	code, out, errOut := run("brain", "search", "a")
	wantErr := "note: the last full reconciliation was 2999-01-01T00:00:00+00:00, more than 24 hours ago: " +
		"a source may have changed without this answer knowing (run `brain reconcile`)\n"
	if code != 0 || out != "brain://project/a/notes/a.md:1  100%  A\n\n" || errOut != wantErr {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errOut)
	}
}

func TestBrainSearchClosesThePortAndReportsTheEngine(t *testing.T) {
	brainWorld(t, "", nil)
	port := &closingPort{FakePort: search.NewFakePort()}
	port.Results = []search.ScriptedSearch{{Err: errors.New("the search engine did not answer in 3 attempts: refused")}}
	stubBrainSearchPort(t, port, "")

	code, out, errOut := run("brain", "search", "q")
	if code != 1 || out != "" || errOut != "error: the search engine did not answer in 3 attempts: refused\n" {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errOut)
	}
	if port.closed != 1 {
		t.Fatalf("closed %d times", port.closed)
	}
}

func TestBrainSearchRefusesAnUnknownScopeBeforeAsking(t *testing.T) {
	brainWorld(t, "", nil)
	fake := search.NewFakePort()
	stubBrainSearchPort(t, fake, "")

	code, out, errOut := run("brain", "search", "q", "--scope", "nope")
	if code != 1 || out != "" || errOut != "error: unknown scope 'nope'; known scopes are: project/a\n" {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errOut)
	}
	if len(fake.Calls) != 0 {
		t.Fatalf("the engine was asked: %+v", fake.Calls)
	}
}

func TestBrainCatalogAnswersTheRootOrOneArea(t *testing.T) {
	index := "# project/a\n\n* [a](brain://project/a/notes/a.md)\n"
	brainWorld(t, "", map[string]string{"index.md": index})

	if code, out, errOut := run("brain", "catalog"); code != 0 || out != "# brain\n\n* [project/a](brain://project/a/)\n" {
		t.Fatalf("root: code %d\nout %q\nerr %q", code, out, errOut)
	}
	if code, out, errOut := run("brain", "catalog", "--scope", "project/a"); code != 0 || out != index {
		t.Fatalf("area: code %d\nout %q\nerr %q", code, out, errOut)
	}
}

func TestBrainCatalogHidesALocalOnlyAreaFromTheCloud(t *testing.T) {
	brainWorld(t, "[privacy]\nmode = \"local_only\"\n", map[string]string{"index.md": "# project/a\n"})

	if code, out, errOut := run("brain", "catalog", "--channel", "cloud"); code != 0 || out != "# brain\n\n" {
		t.Fatalf("root: code %d\nout %q\nerr %q", code, out, errOut)
	}
	code, out, errOut := run("brain", "catalog", "--channel", "cloud", "--scope", "project/a")
	if code != 1 || out != "" || errOut != "error: unknown scope 'project/a'; known scopes are: \n" {
		t.Fatalf("area: code %d\nout %q\nerr %q", code, out, errOut)
	}
}

func TestBrainCatalogReportsAMissingIndex(t *testing.T) {
	w := brainWorld(t, "", nil)
	_, readErr := os.ReadFile(filepath.Join(w.area, "index.md"))

	code, out, errOut := run("brain", "catalog", "--scope", "project/a")
	if code != 1 || out != "" || errOut != "error: "+readErr.Error()+"\n" {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errOut)
	}
}

func TestBrainReadsAReadOnlyAreaFromTheLegacyDirectory(t *testing.T) {
	brainReadOnlyWorld(t, map[string]string{"index.md": "# project/r\n", "graph.json": brainGraph})

	if code, out, errOut := run("brain", "catalog", "--scope", "project/r"); code != 0 || out != "# project/r\n" {
		t.Fatalf("catalog: code %d\nout %q\nerr %q", code, out, errOut)
	}
	code, out, errOut := run("brain", "neighbors", "notes/a.md", "--scope", "project/r")
	if code != 0 || out != "incoming: notes/c.md\noutgoing: notes/b.md\n" {
		t.Fatalf("neighbors: code %d\nout %q\nerr %q", code, out, errOut)
	}
}

// brainNewGraph is brainGraph with other edges, so that an answer names which
// of the two directories its graph came from.
const brainNewGraph = `{"scope":"project/r","nodes":[],"edges":[{"from":"notes/a.md","to":"notes/x.md"},{"from":"notes/y.md","to":"notes/a.md"}],"links":{"total":2,"resolved":2,"dropped":{}}}`

// Lies an area under both places, the new one wins. Without this test the move
// would be unprovable: with an empty state directory the fallback answers
// exactly what the old fixed grip into the legacy directory answered, and the
// case suite 1b-1 points both variables at one and the same directory.
func TestBrainPrefersTheNewStateDirOverTheLegacyOne(t *testing.T) {
	w := brainReadOnlyWorld(t, map[string]string{"index.md": "# old\n", "graph.json": brainGraph})
	moved := filepath.Join(w.state, "areas", "project-r")
	writeFile(t, filepath.Join(moved, ".loomux", "config.toml"), "[area]\nscope = \"project/r\"\n")
	writeFile(t, filepath.Join(moved, "index.md"), "# new\n")
	writeFile(t, filepath.Join(moved, "graph.json"), brainNewGraph)

	if code, out, errOut := run("brain", "catalog", "--scope", "project/r"); code != 0 || out != "# new\n" {
		t.Fatalf("catalog answered from the legacy directory: code %d\nout %q\nerr %q", code, out, errOut)
	}
	code, out, errOut := run("brain", "neighbors", "notes/a.md", "--scope", "project/r")
	if code != 0 || out != "incoming: notes/y.md\noutgoing: notes/x.md\n" {
		t.Fatalf("neighbors answered from the legacy directory: code %d\nout %q\nerr %q", code, out, errOut)
	}
}

func TestBrainReadAnswersTheFileOrOneSection(t *testing.T) {
	doc := "# Top\nintro\n## Part\nbody\n# Next\nrest\n"
	brainWorld(t, "", map[string]string{"notes/a.md": doc})

	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"brain", "read", "notes/a.md", "--scope", "project/a"}, doc},
		{[]string{"brain", "read", "--scope", "project/a", "notes/a.md", "--channel", "cloud"}, doc},
		{[]string{"brain", "read", "notes/a.md", "--scope", "project/a", "--section", "Part"}, "## Part\nbody\n"},
	} {
		if code, out, errOut := run(c.args...); code != 0 || out != c.want || errOut != "" {
			t.Fatalf("%q: code %d\nout %q\nerr %q", c.args, code, out, errOut)
		}
	}
}

func TestBrainReadRefusesWithTheReasonAndNoOutput(t *testing.T) {
	w := brainWorld(t, "[privacy]\nnever = [\"secret/**\"]\n[layout]\nreview = \"review\"\n", map[string]string{
		"notes/a.md":     "# A\n",
		"secret/k.md":    "key\n",
		"review/case.md": "case\n",
	})
	_, missing := os.ReadFile(filepath.Join(w.area, "notes", "gone.md"))

	for _, c := range []struct {
		args   []string
		stderr string
	}{
		{[]string{"../x.md", "--scope", "project/a"}, "error: project/a/../x.md leaves the area\n"},
		{[]string{"secret/k.md", "--scope", "project/a"}, "error: project/a/secret/k.md is excluded by [privacy] never\n"},
		{[]string{"review/case.md", "--scope", "project/a", "--channel", "cloud"}, "error: project/a/review/case.md is the review centre; refused on the cloud channel\n"},
		{[]string{"notes/a.md", "--scope", "project/a", "--section", "Nope"}, "error: no section titled 'Nope'\n"},
		{[]string{"notes/a.md", "--scope", "nope"}, "error: unknown scope 'nope'; known scopes are: project/a\n"},
		{[]string{"notes/gone.md", "--scope", "project/a"}, "error: " + missing.Error() + "\n"},
	} {
		code, out, errOut := run(append([]string{"brain", "read"}, c.args...)...)
		if code != 1 || out != "" || errOut != c.stderr {
			t.Fatalf("%q: code %d\nout %q\n got %q\nwant %q", c.args, code, out, errOut, c.stderr)
		}
	}
	if code, out, _ := run("brain", "read", "review/case.md", "--scope", "project/a"); code != 0 || out != "case\n" {
		t.Fatalf("the review centre stays open locally: code %d, out %q", code, out)
	}
}

func TestBrainNeighborsListsBothDirections(t *testing.T) {
	brainWorld(t, "", map[string]string{"graph.json": brainGraph})

	for _, c := range []struct {
		relative string
		want     string
	}{
		{"notes/a.md", "incoming: notes/c.md\noutgoing: notes/b.md\n"},
		{`notes\a.md`, "incoming: notes/c.md\noutgoing: notes/b.md\n"},
		{"notes/z.md", "incoming: -\noutgoing: -\n"},
	} {
		code, out, errOut := run("brain", "neighbors", c.relative, "--scope", "project/a")
		if code != 0 || out != c.want || errOut != "" {
			t.Fatalf("%s: code %d\nout %q\nerr %q", c.relative, code, out, errOut)
		}
	}
}

func TestBrainNeighborsRefusesWithTheReasonAndNoOutput(t *testing.T) {
	brainWorld(t, "", nil)

	for _, c := range []struct {
		args   []string
		stderr string
	}{
		{[]string{"notes/a.md", "--scope", "project/a"}, "error: project/a: never indexed; run `brain reindex`\n"},
		{[]string{"../x.md", "--scope", "project/a"}, "error: project/a/../x.md leaves the area\n"},
		{[]string{"notes/a.md", "--scope", "nope"}, "error: unknown scope 'nope'; known scopes are: project/a\n"},
	} {
		code, out, errOut := run(append([]string{"brain", "neighbors"}, c.args...)...)
		if code != 1 || out != "" || errOut != c.stderr {
			t.Fatalf("%q: code %d\nout %q\n got %q\nwant %q", c.args, code, out, errOut, c.stderr)
		}
	}
}

func TestBrainStatusPrintsEveryLineAtBrainNow(t *testing.T) {
	w := brainWorld(t, "", nil)
	writeFile(t, filepath.Join(w.legacy, "maintenance", "last-run.txt"), "2999-01-01T00:00:00+00:00\n")
	stubBrainNow(t, time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC))
	fake := search.NewFakePort()
	stubBrainStatusPort(t, fake)

	code, out, errOut := run("brain", "status")
	want := "last reconcile: 2999-01-01T00:00:00+00:00; older than 24 h, run `brain reconcile`\n" +
		"project/a: never indexed; run `brain reindex`\n"
	if code != 0 || out != want || errOut != "" {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errOut)
	}
	if fake.SearchableCounts != 1 {
		t.Fatalf("the backlog was asked %d times", fake.SearchableCounts)
	}
}

func TestBrainStatusOnTheCloudLeavesALocalOnlyAreaOut(t *testing.T) {
	brainWorld(t, "[privacy]\nmode = \"local_only\"\n", nil)
	stubBrainStatusPort(t, search.NewFakePort())

	code, out, errOut := run("brain", "status", "--channel", "cloud")
	if code != 0 || out != "last reconcile: never; run `brain reconcile`\n" || errOut != "" {
		t.Fatalf("code %d\nout %q\nerr %q", code, out, errOut)
	}
}

// addBrainWorkspace appends project/ws to the world's registry: an entry
// without [area] at its path, as `init --brain=none` registers it. workspace
// false makes it an area that forgot its declaration.
func addBrainWorkspace(t *testing.T, w brainWorldDirs, workspace bool) {
	t.Helper()
	ws := t.TempDir()
	writeFile(t, filepath.Join(ws, ".loomux", "config.toml"), "[verify]\n")
	registry := filepath.Join(w.state, "registry.toml")
	body, err := os.ReadFile(registry)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, registry, string(body)+"\n[[area]]\nscope = \"project/ws\"\npath = "+
		strconv.Quote(filepath.ToSlash(ws))+"\nworkspace = "+strconv.FormatBool(workspace)+"\n")
}

func TestBrainReadersSkipAWorkspaceWithoutAnArea(t *testing.T) {
	w := brainWorld(t, "", map[string]string{"index.md": "# project/a\n"})
	addBrainWorkspace(t, w, true)
	stubBrainNow(t, time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC))
	stubBrainStatusPort(t, search.NewFakePort())
	fake := search.NewFakePort()
	fake.Results = []search.ScriptedSearch{{Hits: []search.SearchHit{
		{Collection: "project-a", Relative: "index.md", Line: 1, Title: "A", Snippet: "a", Score: 0.5},
	}}}
	stubBrainSearchPort(t, fake, "")

	code, out, errOut := run("brain", "status")
	if code != 0 || out != "last reconcile: never; run `brain reconcile`\nproject/a: never indexed; run `brain reindex`\n" || errOut != "" {
		t.Fatalf("status: code %d\nout %q\nerr %q", code, out, errOut)
	}
	code, out, errOut = run("brain", "catalog", "--scope", "all")
	if code != 0 || out != "# brain\n\n* [project/a](brain://project/a/)\n" || errOut != "" {
		t.Fatalf("catalog: code %d\nout %q\nerr %q", code, out, errOut)
	}
	if code, out, errOut := run("brain", "search", "what"); code != 0 {
		t.Fatalf("search: code %d\nout %q\nerr %q", code, out, errOut)
	}
	if len(fake.Calls) != 1 || !reflect.DeepEqual(fake.Calls[0].Collections, []string{"project-a"}) {
		t.Fatalf("search calls %+v", fake.Calls)
	}
}

func TestBrainReadersStillRefuseAnAreaWithoutDeclaration(t *testing.T) {
	w := brainWorld(t, "", map[string]string{"index.md": "# project/a\n"})
	addBrainWorkspace(t, w, false)
	stubBrainStatusPort(t, search.NewFakePort())
	for _, args := range [][]string{{"status"}, {"catalog", "--scope", "all"}} {
		code, out, errOut := run(append([]string{"brain"}, args...)...)
		if code != 1 || out != "" || !strings.Contains(errOut, "no manifest found") {
			t.Fatalf("%v: code %d\nout %q\nerr %q", args, code, out, errOut)
		}
	}
}

func TestBrainRuntimeErrorsWriteOnlyTheErrorLine(t *testing.T) {
	w := brainWorld(t, "", nil)
	writeFile(t, filepath.Join(w.state, "registry.toml"), "[[area]\n")
	stubBrainSearchPort(t, search.NewFakePort(), "")
	stubBrainStatusPort(t, search.NewFakePort())

	for _, args := range [][]string{
		{"search", "q"},
		{"catalog"},
		{"read", "notes/a.md", "--scope", "project/a"},
		{"neighbors", "notes/a.md", "--scope", "project/a"},
		{"status"},
	} {
		code, out, errOut := run(append([]string{"brain"}, args...)...)
		// The wording belongs to config.ReadRegistry; this test pins only
		// that it arrives as one error line and nothing else.
		if code != 1 || out != "" || !strings.HasPrefix(errOut, "error: ") ||
			!strings.Contains(errOut, "registry.toml: not valid TOML") || !strings.HasSuffix(errOut, "\n") {
			t.Fatalf("%q: code %d\nout %q\nerr %q", args, code, out, errOut)
		}
	}
}
