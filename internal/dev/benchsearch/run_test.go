package benchsearch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/index"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/dev/fakeqmd"
)

// recordingQmd runs qmd command lines against a fake engine and keeps them.
type recordingQmd struct {
	fixture *fakeqmd.Fixture
	calls   [][]string
	fail    string // a subcommand that exits 1
}

func (r *recordingQmd) Runner(t *testing.T) search.RunnerFunc {
	t.Helper()
	return func(argv []string) ([]byte, []byte, int, error) {
		r.calls = append(r.calls, argv)
		if slices.Contains(argv, r.fail) {
			return nil, []byte("refused"), 1, nil
		}
		var out, errOut bytes.Buffer
		code := r.fixture.RunCLI(argv[1:], &out, &errOut)
		return out.Bytes(), errOut.Bytes(), code, nil
	}
}

func (r *recordingQmd) Calls() [][]string { return r.calls }

// corpusFixture is an engine that holds the hundred notes of corpus v1 and
// ranks the expected note first for the first 43 questions, and some other
// note first for the rest.
func corpusFixture(t *testing.T) *recordingQmd {
	t.Helper()
	stand := corpusDir(t)
	qs, err := LoadQuestions(filepath.Join(stand, "questions.yaml"), DefaultShape())
	if err != nil {
		t.Fatal(err)
	}
	notes := noteNames(stand)
	collection := search.CollectionName(CorpusScope)
	fixture := &fakeqmd.Fixture{Collections: map[string][]string{collection: notes}, Queries: map[string][]fakeqmd.Hit{}}
	for i, q := range qs {
		expected := filepath.Base(q.Expect)
		first := expected
		if i >= 43 {
			first = notes[0]
			if first == expected {
				first = notes[1]
			}
		}
		fixture.Queries[q.Query] = []fakeqmd.Hit{{Collection: collection, Relative: first, DocID: "#1"}}
	}
	return &recordingQmd{fixture: fixture}
}

// benchNow is the start of every run in these tests.
var benchNow = time.Date(2026, 9, 26, 12, 34, 0, 0, time.UTC)

// depsWith are dependencies that start nothing: a state of the test's own,
// whose bench folder holds the locks, and fixed answers for the head.
func depsWith(t *testing.T) Deps {
	t.Helper()
	return Deps{
		StateDir:    t.TempDir(),
		FallbackDir: t.TempDir(),
		Daemon:      func() search.SearchPort { t.Fatal("the daemon was asked"); return nil },
		CLI:         func(string) search.SearchPort { t.Fatal("the command line was asked"); return nil },
		QmdVersion:  func() string { return "qmd 2.8.3" },
		Models:      func(string) map[string]string { return map[string]string{"embedding": "E"} },
		Loomux:      "1.2.3",
		Now:         func() time.Time { return benchNow },
		Clock:       tick(time.Millisecond),
		Random:      fixed("a1"),
		Warn:        logTo(t),
		Backbone:    "vulkan",
	}
}

// corpusLeftovers is every throwaway state or lock a corpus run left.
func corpusLeftovers(t *testing.T, d Deps) []string {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(d.StateDir, "bench", indexPrefix+"*"))
	return append(matches, benchLeftovers(t)...)
}

func TestCorpusRefusesScopeQuestionsAndLatency(t *testing.T) {
	for name, c := range map[string]struct {
		o    Options
		want string
	}{
		"scope":     {Options{Corpus: "x", ScopeSet: true, Out: "o"}, "--corpus names its own scope and question set; drop --scope/--questions"},
		"questions": {Options{Corpus: "x", Questions: "q", Out: "o"}, "--corpus names its own scope and question set; drop --scope/--questions"},
		"latency":   {Options{Corpus: "x", Latency: true, Out: "o"}, "--latency measures the everyday chain; drop it with --corpus"},
		"no out":    {Options{Corpus: "x"}, "--corpus has no measurement folder of its own; name one with --out"},
	} {
		if _, err := Bench(c.o, Deps{}); err == nil || err.Error() != c.want {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestAnUnsoundCorpusWritesNothing(t *testing.T) {
	isolate(t)
	stand := copyCorpus(t)
	os.Remove(filepath.Join(stand, "notes", "baustatik-01.md"))
	d := depsWith(t)
	_, err := Bench(Options{Corpus: stand, Out: t.TempDir()}, d)
	var p Problems
	if !errors.As(err, &p) {
		t.Fatalf("err = %v", err)
	}
	if left := corpusLeftovers(t, d); len(left) != 0 {
		t.Fatalf("left behind: %v", left)
	}
	if _, err := os.Stat(filepath.Join(d.StateDir, "bench")); err == nil {
		t.Fatal("the lock folder was made before the check")
	}
}

func TestCorpusRunRanksThroughTheNamedIndex(t *testing.T) {
	isolate(t)
	out := t.TempDir()
	fixture := corpusFixture(t)
	d := depsWith(t)
	var indexes, modelsFrom []string
	d.CLI = func(idx string) search.SearchPort {
		indexes = append(indexes, idx)
		return &search.QmdPort{Index: idx, Runner: fixture.Runner(t)}
	}
	d.Models = func(path string) map[string]string {
		modelsFrom = append(modelsFrom, path)
		return map[string]string{}
	}
	md, err := Bench(Options{Corpus: corpusDir(t), Out: out, Profile: search.ProfileKeyword}, d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "| total | 43/50 |") || !strings.Contains(md, "- corpus: "+corpusDir(t)) ||
		!strings.Contains(md, "- search path: cli") || !strings.Contains(md, "- indexed documents: 100") ||
		!strings.Contains(md, "- qmd backbone: vulkan") {
		t.Fatalf("markdown:\n%s", md)
	}
	if !slices.Equal(indexes, []string{"loomux-bench-a1"}) || !slices.Equal(modelsFrom, []string{index.QmdConfigPathFor("loomux-bench-a1")}) {
		t.Fatalf("indexes %v, models from %v", indexes, modelsFrom)
	}
	files, _ := filepath.Glob(filepath.Join(out, "bench-2026-09-26-1234-keyword.*"))
	if len(files) != 2 {
		t.Fatalf("files = %v", files)
	}
	calls := fixture.Calls()
	if len(calls) < 3 || calls[0][3] != "update" || calls[1][3] != "embed" || calls[2][3] == "status" {
		t.Fatalf("calls = %v", calls)
	}
	for _, argv := range calls {
		if argv[1] != "--index" || argv[2] != "loomux-bench-a1" {
			t.Fatalf("a qmd call without the index: %v", argv)
		}
	}
}

func TestTheReportIsWrittenBeforeCleanupAndPrintedAfter(t *testing.T) {
	isolate(t)
	out := t.TempDir()
	fixture := corpusFixture(t)
	d := depsWith(t)
	d.CLI = func(idx string) search.SearchPort { return &search.QmdPort{Index: idx, Runner: fixture.Runner(t)} }
	md, err := Bench(Options{Corpus: corpusDir(t), Out: out}, d)
	if err != nil {
		t.Fatal(err)
	}
	if left := corpusLeftovers(t, d); len(left) != 0 {
		t.Fatalf("left behind: %v", left)
	}
	written, err := os.ReadFile(filepath.Join(out, "bench-2026-09-26-1234-fast.md"))
	if err != nil || string(written) != md {
		t.Fatalf("written %q, %v", written, err)
	}
	present(t, filepath.Join(out, "bench-2026-09-26-1234-fast.json"))
}

func TestACorpusThatCannotBeIndexedIsCleanedUp(t *testing.T) {
	for _, step := range []string{"update", "embed"} {
		isolate(t)
		fixture := corpusFixture(t)
		fixture.fail = step
		d := depsWith(t)
		d.CLI = func(idx string) search.SearchPort { return &search.QmdPort{Index: idx, Runner: fixture.Runner(t)} }
		_, err := Bench(Options{Corpus: corpusDir(t), Out: t.TempDir()}, d)
		if err == nil || !strings.Contains(err.Error(), "refused") {
			t.Fatalf("%s: err = %v", step, err)
		}
		if left := corpusLeftovers(t, d); len(left) != 0 {
			t.Fatalf("%s: left behind: %v", step, left)
		}
		// An index qmd could not update is not worth the minutes of an embed.
		if step == "update" && slices.ContainsFunc(fixture.Calls(), func(argv []string) bool { return slices.Contains(argv, "embed") }) {
			t.Fatalf("embedded after a failed update: %v", fixture.Calls())
		}
	}
}

// qmd embed can exit 0 and leave most of the corpus without vectors; fast
// and full would then measure an index that is not there.
func TestAPartlyEmbeddedCorpusIsNoMeasurement(t *testing.T) {
	for _, profile := range []search.Profile{search.ProfileFast, search.ProfileFull} {
		isolate(t)
		fixture := corpusFixture(t)
		fixture.fixture.Pending = 94
		d := depsWith(t)
		d.CLI = func(idx string) search.SearchPort { return &search.QmdPort{Index: idx, Runner: fixture.Runner(t)} }
		out := t.TempDir()
		_, err := Bench(Options{Corpus: corpusDir(t), Out: out, Profile: profile}, d)
		want := "94 documents of the corpus are not embedded after qmd embed; a " + string(profile) +
			" report would measure an unembedded index"
		if err == nil || err.Error() != want {
			t.Fatalf("%s: err = %v", profile, err)
		}
		if left := corpusLeftovers(t, d); len(left) != 0 {
			t.Fatalf("%s: left behind: %v", profile, left)
		}
		if written, _ := filepath.Glob(filepath.Join(out, "*")); len(written) != 0 {
			t.Fatalf("%s: wrote %v", profile, written)
		}
	}
}

func TestKeywordNeedsNoVectors(t *testing.T) {
	isolate(t)
	fixture := corpusFixture(t)
	fixture.fixture.Pending = 94
	fixture.fixture.StatusError = "not asked"
	d := depsWith(t)
	d.CLI = func(idx string) search.SearchPort { return &search.QmdPort{Index: idx, Runner: fixture.Runner(t)} }
	if _, err := Bench(Options{Corpus: corpusDir(t), Out: t.TempDir(), Profile: search.ProfileKeyword}, d); err != nil {
		t.Fatal(err)
	}
}

func TestAStatusThatFailsEndsACorpusRun(t *testing.T) {
	isolate(t)
	fixture := corpusFixture(t)
	fixture.fixture.StatusError = "status broke"
	d := depsWith(t)
	d.CLI = func(idx string) search.SearchPort { return &search.QmdPort{Index: idx, Runner: fixture.Runner(t)} }
	_, err := Bench(Options{Corpus: corpusDir(t), Out: t.TempDir()}, d)
	if err == nil || !strings.Contains(err.Error(), "status broke") {
		t.Fatalf("err = %v", err)
	}
	if left := corpusLeftovers(t, d); len(left) != 0 {
		t.Fatalf("left behind: %v", left)
	}
}

func TestACorpusThatCannotBePreparedIsRefused(t *testing.T) {
	isolate(t)
	d := depsWith(t)
	writeFile(t, filepath.Join(d.StateDir, "bench"), "not a folder")
	if _, err := Bench(Options{Corpus: corpusDir(t), Out: t.TempDir()}, d); err == nil {
		t.Fatal("no refusal")
	}
}

// everydayWorld is a registered area with one note, its catalog and a
// question set of the default shape in its measurement folder.
type everydayWorld struct {
	state, area, measure, note string
}

// questionSet renders fifty questions of the default shape, each expecting
// the note beside the measurement folder, the n-th asking "q<n>".
func questionSet() string {
	var b strings.Builder
	i := 0
	for _, k := range kinds {
		for range DefaultShape()[k] {
			i++
			fmt.Fprintf(&b, "- id: q%02d\n  sort: %s\n  query: \"q%d\"\n  expect: \"../a.md\"\n  beleg: \"the evidence\"\n", i, k, i)
		}
	}
	return b.String()
}

func addArea(t *testing.T, state, scope string) string {
	t.Helper()
	area := filepath.Join(t.TempDir(), "area")
	mkdir(t, filepath.Join(area, ".loomux"))
	writeFile(t, filepath.Join(area, ".loomux", "config.toml"), fmt.Sprintf("[area]\nscope = %q\n", scope))
	writeFile(t, filepath.Join(area, "_identities.tsv"), "doc_id\tpfad\tcontent_hash\trevision\nid-a\ta.md\tsha256:0\t1\n")
	writeFile(t, filepath.Join(area, "a.md"), "# A\n\nthe evidence\n")
	writeFile(t, filepath.Join(area, "index.md"), "# Index\n")
	registry := filepath.Join(state, "registry.toml")
	before, _ := os.ReadFile(registry)
	writeFile(t, registry, string(before)+fmt.Sprintf("[[area]]\nscope = %q\npath = %q\n\n", scope, filepath.ToSlash(area)))
	return area
}

func everyday(t *testing.T, d Deps, scope string) everydayWorld {
	t.Helper()
	area := addArea(t, d.StateDir, scope)
	measure := filepath.Join(area, "98 Messung")
	mkdir(t, measure)
	writeYAML(t, measure, questionSet())
	return everydayWorld{state: d.StateDir, area: area, measure: measure, note: filepath.Join(area, "a.md")}
}

// answering is a daemon that lists the one note and finds it every time.
func answering(scope string, searches int) *search.FakePort {
	port := search.NewFakePort()
	collection := search.CollectionName(scope)
	port.Listings[collection] = search.ScriptedIndexed{Paths: []string{"b.md", "a.md"}}
	for range searches {
		port.Results = append(port.Results, search.ScriptedSearch{Hits: []search.SearchHit{{Collection: collection, Relative: "a.md"}}})
	}
	return port
}

func daemon(port search.SearchPort) func() search.SearchPort {
	return func() search.SearchPort { return port }
}

func TestEverydayRunUsesTheDaemonAndTheDefaults(t *testing.T) {
	d := depsWith(t)
	w := everyday(t, d, "knowledge")
	port := answering("knowledge", 50)
	d.Daemon = daemon(port)
	var modelsFrom string
	d.Models = func(path string) map[string]string { modelsFrom = path; return map[string]string{"embedding": "E"} }
	md, err := Bench(Options{}, d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "| total | 50/50 |") || !strings.Contains(md, "profile `fast`") || !strings.Contains(md, "## Findings\n\n"+findingsIntro+"\n\nnone") ||
		!strings.Contains(md, "- qmd backbone: unknown") {
		t.Fatalf("markdown:\n%s", md)
	}
	if modelsFrom != index.QmdConfigPath() {
		t.Fatalf("models from %s", modelsFrom)
	}
	if c := port.Calls[0]; c.N != 10 || c.Profile != search.ProfileFast || !slices.Equal(c.Collections, []string{"knowledge"}) {
		t.Fatalf("call = %+v", c)
	}
	data, err := os.ReadFile(filepath.Join(w.measure, "bench-2026-09-26-1234-fast.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Environment struct {
			Port, Profile, Qmd, Loomux, Backbone string
			Models                               map[string]string
		}
		Payload struct {
			QuestionSet string `json:"question_set"`
			Documents   int
			Corpus      *string
		}
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	env, p := report.Environment, report.Payload
	if env.Port != "daemon" || env.Profile != "fast" || env.Qmd != "qmd 2.8.3" || env.Loomux != "1.2.3" || env.Models["embedding"] != "E" ||
		env.Backbone != "unknown" {
		t.Fatalf("environment = %+v", env)
	}
	if p.QuestionSet != filepath.Join(w.measure, "questions.yaml") || p.Documents != 2 || p.Corpus != nil {
		t.Fatalf("payload = %+v", p)
	}
}

func TestANamedQuestionSetAndOutAreTaken(t *testing.T) {
	d := depsWith(t)
	w := everyday(t, d, "knowledge")
	d.Daemon = daemon(answering("knowledge", 50))
	out := t.TempDir()
	questions := filepath.Join(w.measure, "questions.yaml")
	if _, err := Bench(Options{Questions: questions, Out: out, Channel: "cloud", Profile: search.ProfileFull}, d); err != nil {
		t.Fatal(err)
	}
	present(t, filepath.Join(out, "bench-2026-09-26-1234-full.md"))
}

// The channel reaches the search chain: on the cloud channel a local_only
// area is not there to be asked.
func TestTheCloudChannelDoesNotSeeALocalOnlyArea(t *testing.T) {
	d := depsWith(t)
	w := everyday(t, d, "knowledge")
	writeFile(t, filepath.Join(w.area, ".loomux", "config.toml"),
		"[area]\nscope = \"knowledge\"\n\n[privacy]\nmode = \"local_only\"\n")
	port := answering("knowledge", 50)
	d.Daemon = daemon(port)
	_, err := Bench(Options{Channel: privacy.ChannelCloud}, d)
	if err == nil || !strings.Contains(err.Error(), "unknown scope 'knowledge'") {
		t.Fatalf("err = %v", err)
	}
	if len(port.Calls) != 0 {
		t.Fatalf("searched: %+v", port.Calls)
	}
}

// filepath.Abs refuses a NUL byte on Windows, and that refusal is the answer;
// elsewhere Abs takes it, and the stand check names the missing files.
func TestACorpusPathAbsCannotResolveIsRefusedAsSuch(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("only Windows' filepath.Abs refuses a NUL byte")
	}
	d := depsWith(t)
	_, err := Bench(Options{Corpus: "v1\x00", Out: t.TempDir()}, d)
	if !errors.Is(err, syscall.EINVAL) {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(d.StateDir, "bench")); err == nil {
		t.Fatal("the lock folder was made")
	}
}

func TestAnEmptyIndexIsNoMeasurement(t *testing.T) {
	d := depsWith(t)
	everyday(t, d, "knowledge")
	port := search.NewFakePort()
	port.Listings["knowledge"] = search.ScriptedIndexed{}
	d.Daemon = daemon(port)
	_, err := Bench(Options{}, d)
	want := "the engine holds no document for knowledge, so there is nothing to measure against; index the area before measuring"
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v", err)
	}
	if len(port.Calls) != 0 {
		t.Fatalf("searched: %+v", port.Calls)
	}
}

func TestAScopeWithASlashIsListedByItsCollection(t *testing.T) {
	d := depsWith(t)
	everyday(t, d, "project/x")
	port := answering("project/x", 50)
	d.Daemon = daemon(port)
	md, err := Bench(Options{Scope: "project/x", ScopeSet: true}, d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "| total | 50/50 |") || !slices.Equal(port.Listed, []string{"project-x"}) {
		t.Fatalf("listed %v, markdown:\n%s", port.Listed, md)
	}
}

func TestLatencyProbesTheQueryFirst(t *testing.T) {
	d := depsWith(t)
	everyday(t, d, "knowledge")
	port := answering("knowledge", 50)
	port.Results = append(port.Results, search.ScriptedSearch{}, search.ScriptedSearch{})
	d.Daemon = daemon(port)
	_, err := Bench(Options{Latency: true, LatencyQuery: "nix", Repeat: 1}, d)
	if err == nil || !strings.Contains(err.Error(), `the latency query "nix" finds nothing in "knowledge"`) || !strings.Contains(err.Error(), "--latency-query") {
		t.Fatalf("err = %v", err)
	}
	if len(port.Calls) != 52 {
		t.Fatalf("%d searches", len(port.Calls))
	}
}

func TestLatencyTimesFiveOperationsOnTheFirstDocument(t *testing.T) {
	d := depsWith(t)
	w := everyday(t, d, "knowledge")
	port := answering("knowledge", 50+1+3*3)
	d.Daemon = daemon(port)
	md, err := Bench(Options{Latency: true, LatencyQuery: "latenz", Repeat: 2}, d)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"- document read: `knowledge/a.md`", "- query: `latenz`", "| catalog |", "| read |", "| keyword |", "| fast |", "| full |"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown lacks %q:\n%s", want, md)
		}
	}
	var profiles []search.Profile
	for _, c := range port.Calls[50:] {
		if c.N != 5 || c.Query != "latenz" {
			t.Fatalf("latency call %+v", c)
		}
		profiles = append(profiles, c.Profile)
	}
	want := []search.Profile{"keyword", "keyword", "keyword", "keyword", "fast", "fast", "fast", "full", "full", "full"}
	if !slices.Equal(profiles, want) {
		t.Fatalf("profiles = %v", profiles)
	}
	data, _ := os.ReadFile(filepath.Join(w.measure, "bench-2026-09-26-1234-fast.json"))
	var report struct{ Timings []struct{ Name string } }
	if err := json.Unmarshal(data, &report); err != nil || len(report.Timings) != 5 || report.Timings[0].Name != "catalog" {
		t.Fatalf("timings = %+v, %v", report.Timings, err)
	}
}

func TestLatencyStopsAtAFailingOperation(t *testing.T) {
	d := depsWith(t)
	w := everyday(t, d, "knowledge")
	os.Remove(filepath.Join(w.area, "index.md"))
	d.Daemon = daemon(answering("knowledge", 51))
	if _, err := Bench(Options{Latency: true, LatencyQuery: "latenz", Repeat: 1}, d); err == nil {
		t.Fatal("no refusal")
	}
}

func TestAFailingProbeEndsTheRun(t *testing.T) {
	d := depsWith(t)
	everyday(t, d, "knowledge")
	d.Daemon = daemon(answering("knowledge", 50))
	_, err := Bench(Options{Latency: true, LatencyQuery: "latenz", Repeat: 1}, d)
	if err == nil || !strings.Contains(err.Error(), "ran out of scripted results") {
		t.Fatalf("err = %v", err)
	}
}

func TestFindingsAreNamedByTheirQuery(t *testing.T) {
	d := depsWith(t)
	everyday(t, d, "knowledge")
	port := answering("knowledge", 49)
	port.Results = append([]search.ScriptedSearch{{}, {}}, port.Results...)
	d.Daemon = daemon(port)
	md, err := Bench(Options{}, d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "- q1: the search engine answered empty twice in a row on profile fast;") ||
		!strings.Contains(md, "- q01 (exakt): not found — q1") || !strings.Contains(md, "| total | 49/50 |") {
		t.Fatalf("markdown:\n%s", md)
	}
}

func TestAFailingSearchEndsTheRun(t *testing.T) {
	d := depsWith(t)
	everyday(t, d, "knowledge")
	d.Daemon = daemon(answering("knowledge", 3))
	if _, err := Bench(Options{}, d); err == nil || !strings.Contains(err.Error(), "ran out of scripted results") {
		t.Fatalf("err = %v", err)
	}
}

func TestSeveralAreasNeedOut(t *testing.T) {
	d := depsWith(t)
	addArea(t, d.StateDir, "one")
	addArea(t, d.StateDir, "two")
	_, err := Bench(Options{Scope: "all", ScopeSet: true}, d)
	want := "a run over 2 areas has no measurement folder of its own — name one with --out"
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v", err)
	}
}

func TestEveryAreaIsAskedAndResolvedOnItsOwn(t *testing.T) {
	d := depsWith(t)
	w := everyday(t, d, "one")
	addArea(t, d.StateDir, "two")
	port := answering("two", 50)
	port.Listings["one"] = search.ScriptedIndexed{Paths: []string{"z.md"}}
	d.Daemon = daemon(port)
	out := t.TempDir()
	md, err := Bench(Options{Scope: "all", ScopeSet: true, Out: out, Questions: filepath.Join(w.measure, "questions.yaml")}, d)
	if err != nil {
		t.Fatal(err)
	}
	// The hits name two's note, the questions one's: resolved per area, none
	// of them is the note the set expects.
	if !strings.Contains(md, "| total | 0/50 |") || !strings.Contains(md, "- indexed documents: 3") {
		t.Fatalf("markdown:\n%s", md)
	}
	if c := port.Calls[0]; !slices.Equal(c.Collections, []string{"one", "two"}) {
		t.Fatalf("collections = %v", c.Collections)
	}
}

func TestAnUnknownOrEmptyScopeIsRefused(t *testing.T) {
	d := depsWith(t)
	everyday(t, d, "knowledge")
	for scope, want := range map[string]string{"x": `no area named "x" in the registry`, "": `no area named "" in the registry`} {
		if _, err := Bench(Options{Scope: scope, ScopeSet: true}, d); err == nil || err.Error() != want {
			t.Errorf("%q: err = %v", scope, err)
		}
	}
}

func TestEverydayRefusals(t *testing.T) {
	for name, c := range map[string]struct {
		prepare func(t *testing.T, d *Deps, w everydayWorld) Options
		want    string
	}{
		"no registry": {func(t *testing.T, d *Deps, _ everydayWorld) Options {
			d.StateDir = t.TempDir()
			return Options{}
		}, "registry.toml"},
		"no measurement folder": {func(t *testing.T, _ *Deps, w everydayWorld) Options {
			return Options{Out: filepath.Join(w.area, "missing")}
		}, "no directory at"},
		"a report of that minute": {func(t *testing.T, _ *Deps, w everydayWorld) Options {
			writeFile(t, filepath.Join(w.measure, "bench-2026-09-26-1234-fast.json"), "")
			return Options{}
		}, "already exists"},
		"a broken question set": {func(t *testing.T, _ *Deps, w everydayWorld) Options {
			writeYAML(t, w.measure, "[")
			return Options{}
		}, "not valid YAML"},
		"no listing": {func(t *testing.T, d *Deps, _ everydayWorld) Options {
			d.Daemon = daemon(search.NewFakePort())
			return Options{}
		}, "no listing scripted"},
		"a folder that vanished": {func(t *testing.T, d *Deps, w everydayWorld) Options {
			d.Models = func(string) map[string]string {
				if err := os.RemoveAll(w.measure); err != nil {
					t.Fatal(err)
				}
				return nil
			}
			return Options{}
		}, "bench-2026-09-26-1234-fast.md"},
	} {
		t.Run(name, func(t *testing.T) {
			d := depsWith(t)
			w := everyday(t, d, "knowledge")
			d.Daemon = daemon(answering("knowledge", 50))
			o := c.prepare(t, &d, w)
			if _, err := Bench(o, d); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
		})
	}
}

// starting is a daemon port that says whether it started the daemon.
type starting struct {
	*search.FakePort
	started bool
}

func (s starting) StartedDaemon() bool { return s.started }

// TestEverydayRunNamesTheBackboneOnlyOfADaemonItStarted: a daemon this run
// started runs on the backbone the run resolved; one it found running keeps
// whatever its starter gave it, which nobody here knows.
func TestEverydayRunNamesTheBackboneOnlyOfADaemonItStarted(t *testing.T) {
	for started, want := range map[bool]string{true: "vulkan", false: "unknown"} {
		d := depsWith(t)
		everyday(t, d, "knowledge")
		d.Daemon = daemon(starting{answering("knowledge", 50), started})
		md, err := Bench(Options{}, d)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(md, "- qmd backbone: "+want+"\n") {
			t.Fatalf("started %v:\n%s", started, md)
		}
	}
}
