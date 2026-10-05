package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/query"
	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/testlock"
)

// repo writes a small Go repository and returns its root.
func repo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func sample() map[string]string {
	return map[string]string{
		"go.mod":          "module example.com/repo\n",
		"main.go":         "package main\n\nimport \"example.com/repo/lib\"\n\nfunc main() { lib.Run() }\n",
		"lib/lib.go":      "package lib\n\n// Run does the thing.\nfunc Run() {}\n",
		"lib/lib_test.go": "package lib\n\nimport \"testing\"\n\nfunc TestRun(t *testing.T) { Run() }\n",
	}
}

func TestGraphBuildWritesAGraphAndReportsItsShape(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer

	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
	if _, err := os.Stat(store.WiringPath(root)); err != nil {
		t.Fatalf("no graph written: %v", err)
	}
	// The count line is the acceptance view on the extractor, and the number a
	// parity file needs.
	report := out.String()
	for _, want := range []string{"nodes", "edges", "calls", "imports", "contains"} {
		if !strings.Contains(report, want) {
			t.Errorf("report %q is missing %q", report, want)
		}
	}
}

func TestGraphBuildIsByteIdenticalOnASecondRun(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer

	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	first, err := os.ReadFile(store.WiringPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	second, err := os.ReadFile(store.WiringPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Error("two builds of one tree produced different bytes")
	}
}

func TestGraphBuildWiresACallAcrossPackages(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}

	g, err := store.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	// This is the one place the port goes past the original, which drops a package
	// selector for Go and would leave the call structure to import edges alone.
	found := false
	for _, e := range g.Edges {
		if e.Source == "main.go#main" && e.Target == "lib/lib.go#Run" && e.Relation == "calls" {
			found = true
		}
	}
	if !found {
		t.Errorf("the cross-package call is missing; edges %+v", g.Edges)
	}
}

func TestGraphBuildRecordsTheFingerprintSoCheckIsCleanAfterwards(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if _, err := os.Stat(filepath.Join(store.Dir(root), "cache", "fingerprint.json")); err != nil {
		t.Fatalf("no fingerprint written: %v", err)
	}
}

func TestGraphBuildRefusesAFileItCannotParse(t *testing.T) {
	root := repo(t, map[string]string{"go.mod": "module x\n", "broken.go": "package ???\n"})
	var out, errOut bytes.Buffer

	code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut)
	// A build is explicit and must not quietly write half a graph: a file that
	// does not parse is the user's next move, not something to skip.
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "broken.go") {
		t.Errorf("stderr %q must name the file", errOut.String())
	}
}

func TestGraphWithoutASubcommandIsAUsageError(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := graphCommand(nil, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "build") || !strings.Contains(errOut.String(), "check") {
		t.Errorf("usage %q must name the subcommands", errOut.String())
	}
}

func TestGraphRejectsAnUnknownSubcommand(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"frobnicate"}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
}

func TestGraphBuildRejectsAnUnknownFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--nope"}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
}

func TestGraphIsInTheCommandTable(t *testing.T) {
	if _, ok := commands["graph"]; !ok {
		t.Fatal(`commands["graph"] missing`)
	}
}

func TestGraphCheckIsCleanRightAfterABuild(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()

	if code := graphCommand([]string{"check", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, want 0; stdout %q", code, out.String())
	}
}

func TestGraphCheckIsCleanAfterATouch(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	later := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(filepath.Join(root, "lib", "lib.go"), later, later); err != nil {
		t.Fatal(err)
	}

	// check re-extracts, so it sees content and not a timestamp.
	if code := graphCommand([]string{"check", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, want 0 after a touch", code)
	}
}

func TestGraphCheckReportsAChangedBody(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()
	body := "package lib\n\n// Run does the thing.\nfunc Run() { println(\"now it does something\") }\n"
	if err := os.WriteFile(filepath.Join(root, "lib", "lib.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	code := graphCommand([]string{"check", "--root", root}, nil, &out, &errOut)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(out.String(), "lib/lib.go#Run") {
		t.Errorf("report %q must name the changed node", out.String())
	}
}

func TestGraphCheckIgnoresARewordedDocComment(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	body := "package lib\n\n// Run is documented differently now.\nfunc Run() {}\n"
	if err := os.WriteFile(filepath.Join(root, "lib", "lib.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	// The symbol's hash runs over Pos()..End(), which excludes the comment.
	// The FILE node hashes the whole file, so the file is reported changed --
	// but no symbol is, and that is the distinction worth having.
	code := graphCommand([]string{"check", "--root", root}, nil, &out, &errOut)
	if code != 1 {
		t.Fatalf("exit %d, want 1: the file node's hash covers the comment", code)
	}
	if strings.Contains(out.String(), "lib/lib.go#Run") {
		t.Errorf("no symbol changed; report %q must not name Run", out.String())
	}
}

func TestGraphCheckReportsAnAddedAndARemovedSymbol(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()
	if err := os.WriteFile(filepath.Join(root, "lib", "lib.go"), []byte("package lib\n\nfunc Other() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if code := graphCommand([]string{"check", "--root", root}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	report := out.String()
	if !strings.Contains(report, "lib/lib.go#Other") || !strings.Contains(report, "lib/lib.go#Run") {
		t.Errorf("report %q must name both the added and the removed symbol", report)
	}
}

func TestGraphCheckReportsAMissingGraphAndPointsAtBuild(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer

	code := graphCommand([]string{"check", "--root", root}, nil, &out, &errOut)
	// The original has no separate code for this, and the pillar-3 spec asks only
	// that it be reported cleanly as not initialised. A third code would be an
	// invention.
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(out.String(), "graph build") {
		t.Errorf("report %q must point at the command that fixes it", out.String())
	}
}

func TestGraphCheckReportsAnOutdatedSchemaAndPointsAtBuild(t *testing.T) {
	root := repo(t, sample())
	if err := os.MkdirAll(store.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	// A schema-1 graph, written by hand: it has no extractor stamp at all, so
	// the version check must catch it before that comparison is even reached.
	const schema1 = `{"meta":{"version":1},"nodes":[],"edges":[]}` + "\n"
	if err := os.WriteFile(store.WiringPath(root), []byte(schema1), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer

	code := graphCommand([]string{"check", "--root", root}, nil, &out, &errOut)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	report := out.String()
	if !strings.Contains(report, "graph build") {
		t.Errorf("report %q must point at the command that fixes it", report)
	}
	if strings.Contains(report, "want 2") {
		t.Errorf("report %q must not leak the raw decode error; use the same guidance as every other unusable-graph case", report)
	}
}

func TestGraphCheckRefusesAGraphFromAnotherExtractor(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	g, err := store.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	g.Meta.Extractor = "go/0"
	// Also disagree with the tree on a node, not just on the stamp. If the
	// stamp check ran after the diff instead of before it, the diff alone
	// would report drift here and this test would catch that; with the
	// disagreement removed, a "diff first" regression would produce the same
	// clean-looking foreign report as the correct order, and the test
	// couldn't tell the two designs apart.
	g.Nodes[0].BodyHash = "not-a-real-hash"
	if err := store.Write(root, g); err != nil {
		t.Fatal(err)
	}
	out.Reset()

	code := graphCommand([]string{"check", "--root", root}, nil, &out, &errOut)
	// Foreign, not stale: its nodes say nothing about this code, so comparing
	// them one by one would be noise.
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	report := out.String()
	if !strings.Contains(report, "extractor") {
		t.Errorf("report %q must say the graph is from another extractor", report)
	}
	if strings.Contains(report, "added") || strings.Contains(report, "removed") || strings.Contains(report, "changed") {
		t.Errorf("report %q must not diff a foreign graph node by node", report)
	}
}

func TestGraphCheckJSONCarriesTheThreeCategories(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()

	if code := graphCommand([]string{"check", "--root", root, "--json"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	var got struct {
		OK      bool     `json:"ok"`
		Missing bool     `json:"missing"`
		Added   []string `json:"added"`
		Removed []string `json:"removed"`
		Changed []string `json:"changed"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out.String())
	}
	if !got.OK || got.Missing {
		t.Errorf("got %+v, want ok and not missing", got)
	}
}

func TestGraphBuildFailsWhenProjectRootCannotBeResolved(t *testing.T) {
	saved := getwd
	getwd = func() (string, error) { return "", os.ErrPermission }
	t.Cleanup(func() { getwd = saved })

	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build"}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
}

func TestGraphBuildFailsWhenTheGraphCannotBeWritten(t *testing.T) {
	root := repo(t, sample())
	// A regular file where store.Write needs a directory component makes
	// os.MkdirAll fail without touching any permission.
	if err := os.WriteFile(filepath.Join(root, ".loomux"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1, stderr %q", code, errOut.String())
	}
}

func TestGraphBuildFailsWhenTheCacheDirectoryCannotBeCreated(t *testing.T) {
	root := repo(t, sample())
	if err := os.MkdirAll(store.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	// A regular file named "cache" leaves store.Write's directory untouched
	// and blocks the MkdirAll of everything under it. Both sidecars live
	// there -- store.CachePath builds their paths -- so the build fails at
	// lexicon.Write, the first of query.Build's two sidecar writes, and
	// never reaches freshness.Write.
	if err := os.WriteFile(filepath.Join(store.Dir(root), "cache"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1, stderr %q", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "loomux graph build:") {
		t.Errorf("stderr %q must report the build error", errOut.String())
	}
}

func TestGraphBuildWarnsButSucceedsWhenTheFingerprintCannotBeWritten(t *testing.T) {
	root := repo(t, sample())
	// A directory in the fingerprint's place leaves every earlier write alone:
	// the cache directory exists, and the ask sidecar is a different name
	// under it, so the build gets past store.Write and lexicon.Write and only
	// then finds that the fingerprint's path is not a file it may write.
	if err := os.MkdirAll(freshness.Path(root), 0o755); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	// Everything a question reads is on disk, so this build succeeded. Only
	// the next probe pays, and it pays with its fast path, not with a worse
	// answer.
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, want 0, stderr %q", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "freshness record not written") {
		t.Errorf("stderr %q must warn about the freshness record", errOut.String())
	}
	// The named path is what separates this arm from the two before it: their
	// errors name "cache" or the ask sidecar, never the fingerprint.
	if !strings.Contains(errOut.String(), "fingerprint.json") {
		t.Errorf("stderr %q must name the fingerprint it could not write", errOut.String())
	}
	if !strings.Contains(out.String(), "nodes") {
		t.Errorf("stdout %q must still carry the report of what was built", out.String())
	}
	if _, err := os.Stat(lexicon.Path(root)); err != nil {
		t.Errorf("the sidecar must be on disk: %v", err)
	}
}

func TestGraphBuildCountsFilesWithoutASymbol(t *testing.T) {
	files := sample()
	files["empty.go"] = "package main\n"
	root := repo(t, files)
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "1 files without a symbol") {
		t.Errorf("report %q must count the symbol-less file", out.String())
	}
}

func TestGraphBuildNoReuseParsesEveryFile(t *testing.T) {
	root := repo(t, sample())
	for _, step := range []struct {
		args []string
		want string
	}{
		{[]string{"build", "--root", root}, "  go: 3 files, 3 parsed, 0 reused, 0 parse errors\n"},
		// The first build left a cache, so the second parses nothing ...
		{[]string{"build", "--root", root}, "  go: 3 files, 0 parsed, 3 reused, 0 parse errors\n"},
		// ... and --no-reuse parses every file all the same.
		{[]string{"build", "--no-reuse", "--root", root}, "  go: 3 files, 3 parsed, 0 reused, 0 parse errors\n"},
	} {
		var out, errOut bytes.Buffer
		if code := graphCommand(step.args, nil, &out, &errOut); code != 0 {
			t.Fatalf("%v: exit %d: %s", step.args, code, errOut.String())
		}
		if !strings.Contains(out.String(), step.want) {
			t.Errorf("%v: report %q lacks %q", step.args, out.String(), step.want)
		}
	}
}

func TestReportListsEveryLanguageAndItsParseErrors(t *testing.T) {
	stats := query.Stats{PerLanguage: map[string]query.LangStats{
		"python": {Files: 9, Parsed: 2, Reused: 7, ParseErrors: 11, ErrorFiles: []string{
			"a.py", "b.py", "c.py", "d.py", "e.py", "f.py", "g.py",
		}},
		"go":   {Files: 4, Parsed: 1, Reused: 3},
		"rust": {Files: 2, Parsed: 2, ParseErrors: 1, ErrorFiles: []string{"x.rs"}},
	}}
	got := report(&model.Graph{}, stats, time.Second)
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	want := []string{
		"  go: 4 files, 1 parsed, 3 reused, 0 parse errors",
		"  python: 9 files, 2 parsed, 7 reused, 11 parse errors",
		"  python parse errors in: a.py, b.py, c.py, d.py, e.py (+2 more)",
		"  rust: 2 files, 2 parsed, 0 reused, 1 parse errors",
		"  rust parse errors in: x.rs",
	}
	// After the two count lines, sorted by language.
	if len(lines) != 2+len(want) || strings.Join(lines[2:], "\n") != strings.Join(want, "\n") {
		t.Errorf("report =\n%s\nwant the two count lines, then\n%s", got, strings.Join(want, "\n"))
	}
}

// The first line counts extends only where there are some: a Go graph has
// none, and its report reads as it did before Python brought base classes.
func TestReportCountsExtendsOnlyWhenThereAreSome(t *testing.T) {
	edge := func(rel model.Relation) model.Edge {
		return model.Edge{Source: "a", Target: "b", Relation: rel, Confidence: model.ConfidenceExtracted}
	}
	stats := query.Stats{}
	for _, c := range []struct {
		edges []model.Edge
		want  string
	}{
		{[]model.Edge{edge(model.RelationContains), edge(model.RelationCalls), edge(model.RelationCalls), edge(model.RelationImports)},
			"0 files, 0 nodes, 4 edges (1 contains, 2 calls, 1 imports)"},
		{[]model.Edge{edge(model.RelationContains), edge(model.RelationExtends), edge(model.RelationExtends)},
			"0 files, 0 nodes, 3 edges (1 contains, 0 calls, 0 imports, 2 extends)"},
	} {
		got, _, _ := strings.Cut(report(&model.Graph{Edges: c.edges}, stats, time.Second), "\n")
		if got != c.want {
			t.Errorf("first line %q, want %q", got, c.want)
		}
	}
}

func TestGraphCheckFailsOnAnUnknownFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"check", "--nope"}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
}

func TestGraphCheckFailsWhenProjectRootCannotBeResolved(t *testing.T) {
	saved := getwd
	getwd = func() (string, error) { return "", os.ErrPermission }
	t.Cleanup(func() { getwd = saved })

	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"check"}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
}

func TestGraphCheckFailsWhenTheWrittenGraphIsUnreadable(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	testlock.Lock(t, store.WiringPath(root))

	code := graphCommand([]string{"check", "--root", root}, nil, &out, &errOut)
	if code != 1 {
		t.Fatalf("exit %d, want 1: an unreadable graph is broken, not absent", code)
	}
}

func TestGraphCheckFailsWhenReExtractionFails(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	testlock.Lock(t, filepath.Join(root, "lib", "lib.go"))

	code := graphCommand([]string{"check", "--root", root}, nil, &out, &errOut)
	if code != 1 {
		t.Fatalf("exit %d, want 1: an unreadable source file must fail check, not report it clean", code)
	}
}

func TestGraphBuildWritesTheAskSidecar(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}

	ix, err := lexicon.Read(root)
	if err != nil {
		t.Fatalf("no sidecar written: %v", err)
	}
	if ix.DocCount == 0 || len(ix.DF) == 0 {
		t.Fatalf("sidecar is empty: %+v", ix)
	}
	// It must carry the body, which wiring.json does not: a symbol has to be
	// findable by a word that appears only inside it.
	found := false
	for _, d := range ix.Docs {
		if d.ID == "lib/lib.go#Run" && len(d.Body) > 0 {
			found = true
		}
	}
	if !found {
		t.Error("the sidecar must carry body tokens the graph does not")
	}
}

func TestGraphAskAnswersWithLocationsAndNoCode(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()

	if code := graphCommand([]string{"ask", "run the thing", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	report := out.String()
	if !strings.Contains(report, "lib/lib.go") {
		t.Fatalf("answer %q must name the file", report)
	}
	// The default is a locator. Source arrives only when asked for.
	if strings.Contains(report, "func Run() {}") {
		t.Errorf("answer %q must not inline source without --source", report)
	}
}

func TestGraphAskSourceInlinesTheSpan(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()

	if code := graphCommand([]string{"ask", "run", "--root", root, "--source"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "func Run()") {
		t.Errorf("answer %q must carry the definition", out.String())
	}
}

func TestGraphAskFindsASymbolByAWordOnlyInItsBody(t *testing.T) {
	files := sample()
	files["lib/lib.go"] = "package lib\n\nfunc Run() { retryWithBackoff() }\n\nfunc retryWithBackoff() {}\n"
	root := repo(t, files)
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()

	// The word lives in Run's body and in retryWithBackoff's name. Body
	// indexing is what makes the first findable at all -- wiring.json does not
	// carry the body, the sidecar does.
	if code := graphCommand([]string{"ask", "backoff", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "retryWithBackoff") {
		t.Errorf("answer %q must find the symbol", out.String())
	}
}

func TestGraphAskFindsAFileByAWordInItsImportHeader(t *testing.T) {
	files := sample()
	files["lib/lib.go"] = "package lib\n\nimport \"encoding/json\"\n\nfunc Run() { _ = json.Marshal }\n"
	root := repo(t, files)
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()

	// The file node's residual carries what no symbol span covers.
	if code := graphCommand([]string{"ask", "encoding", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "lib/lib.go") {
		t.Errorf("answer %q must find the file", out.String())
	}
}

func TestGraphAskDoesNotBuildAFirstGraph(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer

	// A fresh clone has no graph. The reference refuses rather than build a
	// whole repository under a question (src/graph/refresh.ts); so does loomux
	// since G3.
	if code := graphCommand([]string{"ask", "run", "--root", root}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "loomux graph build") {
		t.Errorf("stderr %q must point at the command that fixes it", errOut.String())
	}
	if _, err := os.Stat(store.WiringPath(root)); !os.IsNotExist(err) {
		t.Fatalf("ask must not have built the graph: %v", err)
	}
}

func TestGraphAskNoRefreshAnswersFromWhatIsThere(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer

	code := graphCommand([]string{"ask", "run", "--root", root, "--no-refresh"}, nil, &out, &errOut)
	// Nothing built and no rebuild allowed: say so rather than answer from
	// nothing.
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "graph build") {
		t.Errorf("stderr %q must point at the command that fixes it", errOut.String())
	}
}

func TestGraphAskJSONCarriesTheHits(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()

	if code := graphCommand([]string{"ask", "run", "--root", root, "--json"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	var got struct {
		Query string `json:"query"`
		Hits  []struct {
			ID      string  `json:"id"`
			Score   float64 `json:"score"`
			Lexical float64 `json:"lexical"`
			Graph   float64 `json:"graph"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out.String())
	}
	if got.Query != "run" || len(got.Hits) == 0 {
		t.Fatalf("got %+v", got)
	}
	// Both axes are reported, so a reader can see WHY a hit is there.
	if got.Hits[0].Lexical == 0 && got.Hits[0].Graph == 0 {
		t.Error("a hit must say which axis put it there")
	}
}

func TestGraphAskWithoutAQueryIsAUsageError(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"ask"}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
}

func TestGraphAskOnAMissedQuerySaysSoAndExitsZero(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()

	code := graphCommand([]string{"ask", "quantum entanglement", "--root", root}, nil, &out, &errOut)
	// Asking and finding nothing is a successful question with an empty answer,
	// not a failure -- a script must be able to tell the two apart.
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if !strings.Contains(out.String(), "no matching nodes") {
		t.Errorf("answer %q must say it found nothing", out.String())
	}
}

func TestGraphAskRejectsUnknownFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"ask", "run", "--unknown-flag"}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
}

func TestGraphAskReportsBadRoot(t *testing.T) {
	var out, errOut bytes.Buffer
	badRoot := filepath.Join(t.TempDir(), "nonexistent")
	if code := graphCommand([]string{"ask", "run", "--root", badRoot}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
}

func TestGraphAskReportsCorruptGraph(t *testing.T) {
	root := repo(t, sample())
	if err := os.MkdirAll(store.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.WiringPath(root), []byte("{corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"ask", "run", "--root", root, "--no-refresh"}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "loomux graph ask:") {
		t.Errorf("stderr %q must report error", errOut.String())
	}
}

func TestGraphAskFallsBackWhenSidecarIsMissing(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	out.Reset()
	errOut.Reset()
	_ = os.Remove(filepath.Join(root, ".loomux", "state", "graph", "cache", "ask-index.json"))

	if code := graphCommand([]string{"ask", "run", "--root", root, "--no-refresh"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "no ask index") {
		t.Errorf("stderr %q must warn about missing sidecar", errOut.String())
	}
	if !strings.Contains(out.String(), "lib/lib.go") {
		t.Errorf("stdout %q must still answer", out.String())
	}
}

func TestGraphBuildFailsWhenStoreWriteFails(t *testing.T) {
	root := repo(t, sample())
	if err := os.WriteFile(filepath.Join(root, ".loomux"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
}

func TestGraphAskRebuildsADeletedSidecar(t *testing.T) {
	files := sample()
	files["lib/lib.go"] = "package lib\n\nfunc Run() { retryWithBackoff() }\n\nfunc retryWithBackoff() {}\n"
	root := repo(t, files)
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build failed: %s", errOut.String())
	}
	if err := os.Remove(lexicon.Path(root)); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errOut.Reset()

	// Nothing in the tree moved, so the freshness record stays clean and only
	// the sidecar's own absence can trigger the rebuild. Without that, every
	// later question would rank on names and paths for good.
	if code := graphCommand([]string{"ask", "backoff", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if _, err := os.Stat(lexicon.Path(root)); err != nil {
		t.Fatalf("the sidecar was not rebuilt: %v", err)
	}
	if strings.Contains(errOut.String(), "ranking on names and paths only") {
		t.Errorf("stderr %q must not fall back once a rebuild is allowed", errOut.String())
	}
	// The two assertions above are the ones that discriminate: the name token
	// survives the fallback to names and paths, so this last one passes without
	// a rebuild as well. It is here for the answer, not for the regression.
	if !strings.Contains(out.String(), "retryWithBackoff") {
		t.Errorf("answer %q must find the symbol again", out.String())
	}
}

func TestGraphUsage(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := graphCommand(nil, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "Usage: loomux graph") {
		t.Fatalf("usage missing, got %q", errOut.String())
	}
	errOut.Reset()
	if code := graphCommand([]string{"unknown"}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "unknown command") {
		t.Fatalf("unknown command error missing, got %q", errOut.String())
	}
}

func TestGraphCallers(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	// 1. Missing graph -> exit 1
	if code := graphCommand([]string{"callers", "Run", "--root", root}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}

	// 2. Build graph
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build exit %d", code)
	}

	// 3. Normal callers
	out.Reset()
	errOut.Reset()
	if code := graphCommand([]string{"callers", "Run", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "main.go") || !strings.Contains(out.String(), "lib/lib_test.go") {
		t.Fatalf("callers output missing expected callers: %q", out.String())
	}

	// 4. Direction out, depth 2, json
	out.Reset()
	errOut.Reset()
	if code := graphCommand([]string{"callers", "main", "--root", root, "--direction", "out", "-d", "2", "--json"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
	var callersAns query.CallersAnswer
	if err := json.Unmarshal(out.Bytes(), &callersAns); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	// 5. Depth all, in filter
	out.Reset()
	errOut.Reset()
	if code := graphCommand([]string{"callers", "Run", "--root", root, "--direction", "in", "--depth", "all", "--in", "lib/"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
	// "full" is the documented synonym of "all".
	out.Reset()
	errOut.Reset()
	if code := graphCommand([]string{"callers", "Run", "--root", root, "-d", "FULL"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("-d FULL: exit %d, stderr %q", code, errOut.String())
	}

	// 6. Validation errors:
	// a. missing symbol
	errOut.Reset()
	if code := graphCommand([]string{"callers", "--root", root}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	// b. two symbols
	errOut.Reset()
	if code := graphCommand([]string{"callers", "sym1", "sym2", "--root", root}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	// c. invalid direction
	errOut.Reset()
	if code := graphCommand([]string{"callers", "Run", "--direction", "sideways", "--root", root}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	// d. invalid depth
	errOut.Reset()
	if code := graphCommand([]string{"callers", "Run", "-d", "bad", "--root", root}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	// e. flag parse error
	errOut.Reset()
	if code := graphCommand([]string{"callers", "--unknown"}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	// f. projectRoot error
	saved := getwd
	getwd = func() (string, error) { return "", os.ErrPermission }
	defer func() { getwd = saved }()
	errOut.Reset()
	if code := graphCommand([]string{"callers", "Run"}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
}

func TestGraphSkeleton(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	// 1. Missing graph -> exit 1
	if code := graphCommand([]string{"skeleton", "lib/lib.go", "--root", root}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}

	// 2. Build graph
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build exit %d", code)
	}

	// 3. Normal skeleton
	out.Reset()
	errOut.Reset()
	if code := graphCommand([]string{"skeleton", "lib.go", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "func Run()") {
		t.Fatalf("skeleton output missing Run(): %q", out.String())
	}

	// 4. JSON output
	out.Reset()
	errOut.Reset()
	if code := graphCommand([]string{"skeleton", "lib/lib.go", "--root", root, "--json"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
	var skelAns query.SkeletonAnswer
	if err := json.Unmarshal(out.Bytes(), &skelAns); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	// 5. Validation errors:
	// a. missing file
	errOut.Reset()
	if code := graphCommand([]string{"skeleton", "--root", root}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	// b. two files
	errOut.Reset()
	if code := graphCommand([]string{"skeleton", "f1", "f2", "--root", root}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	// c. flag parse error
	errOut.Reset()
	if code := graphCommand([]string{"skeleton", "--unknown"}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	// d. projectRoot error
	saved := getwd
	getwd = func() (string, error) { return "", os.ErrPermission }
	defer func() { getwd = saved }()
	errOut.Reset()
	if code := graphCommand([]string{"skeleton", "lib/lib.go"}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
}

func TestGraphGrep(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	// 1. Missing graph -> exit 1
	if code := graphCommand([]string{"grep", "Run", "--root", root}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}

	// 2. Build graph
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build exit %d", code)
	}

	// 3. Normal grep
	out.Reset()
	errOut.Reset()
	if code := graphCommand([]string{"grep", "Run", "--root", root, "-i", "--fixed", "--in", "lib/", "--max-hits", "10"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "lib/lib.go") {
		t.Fatalf("grep output missing hit: %q", out.String())
	}

	// 4. JSON output
	out.Reset()
	errOut.Reset()
	if code := graphCommand([]string{"grep", "Run", "--root", root, "--json"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
	var grepAns query.GrepAnswer
	if err := json.Unmarshal(out.Bytes(), &grepAns); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	// 5. Validation errors:
	// a. missing pattern
	errOut.Reset()
	if code := graphCommand([]string{"grep", "--root", root}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	// b. two patterns
	errOut.Reset()
	if code := graphCommand([]string{"grep", "p1", "p2", "--root", root}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	// c. flag parse error
	errOut.Reset()
	if code := graphCommand([]string{"grep", "--unknown"}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	// d. projectRoot error
	saved := getwd
	getwd = func() (string, error) { return "", os.ErrPermission }
	defer func() { getwd = saved }()
	errOut.Reset()
	if code := graphCommand([]string{"grep", "Run"}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
}

func TestGraphMap(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	// 1. Missing graph -> exit 1
	if code := graphCommand([]string{"map", "--root", root}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}

	// 2. Build graph
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build exit %d", code)
	}

	// 3. Normal map
	out.Reset()
	errOut.Reset()
	if code := graphCommand([]string{"map", "--root", root, "--max-dirs", "5", "--hubs-per-dir", "2", "--hotspots", "3"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "lib") {
		t.Fatalf("map output missing directory: %q", out.String())
	}

	// 4. JSON output
	out.Reset()
	errOut.Reset()
	if code := graphCommand([]string{"map", "--root", root, "--json"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
	var mapAns query.MapAnswer
	if err := json.Unmarshal(out.Bytes(), &mapAns); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	// 5. Validation errors:
	// a. positional argument
	errOut.Reset()
	if code := graphCommand([]string{"map", "extra", "--root", root}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	// b. flag parse error
	errOut.Reset()
	if code := graphCommand([]string{"map", "--unknown"}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	// c. projectRoot error
	saved := getwd
	getwd = func() (string, error) { return "", os.ErrPermission }
	defer func() { getwd = saved }()
	errOut.Reset()
	if code := graphCommand([]string{"map"}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
}

func TestGraphStats(t *testing.T) {
	root := repo(t, sample())
	var out, errOut bytes.Buffer
	// 1. Missing graph -> exit 1
	if code := graphCommand([]string{"stats", "--root", root}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}

	// 2. Build graph
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build exit %d", code)
	}

	// 3. Normal stats
	out.Reset()
	errOut.Reset()
	if code := graphCommand([]string{"stats", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Code Graph Stats:") {
		t.Fatalf("stats output missing header: %q", out.String())
	}

	// 4. JSON output
	out.Reset()
	errOut.Reset()
	if code := graphCommand([]string{"stats", "--root", root, "--json"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
	var statsAns query.StatsAnswer
	if err := json.Unmarshal(out.Bytes(), &statsAns); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if statsAns.Files == 0 {
		t.Fatalf("expected files > 0, got %d", statsAns.Files)
	}

	// 5. Validation errors:
	// a. positional argument
	errOut.Reset()
	if code := graphCommand([]string{"stats", "extra", "--root", root}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	// b. flag parse error
	errOut.Reset()
	if code := graphCommand([]string{"stats", "--unknown"}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	// c. projectRoot error
	saved := getwd
	getwd = func() (string, error) { return "", os.ErrPermission }
	defer func() { getwd = saved }()
	errOut.Reset()
	if code := graphCommand([]string{"stats"}, nil, &out, &errOut); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
}

func TestGraphBlast(t *testing.T) {
	root := repo(t, sample())
	for _, args := range [][]string{
		{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "t"},
		{"config", "core.autocrlf", "false"}, {"config", "commit.gpgsign", "false"},
		{"add", "."}, {"commit", "-qm", "init"},
	} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = gitenv.Environ()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"blast", "--root", root, "--no-refresh"}, nil, &out, &errOut); code != 1 {
		t.Fatalf("missing graph: exit %d, want 1", code)
	}
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("build exit %d", code)
	}
	lib := filepath.Join(root, "lib", "lib.go")
	if err := os.WriteFile(lib, []byte("package lib\n\n// Run does the thing.\nfunc Run() { _ = 1 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	errOut.Reset()
	if code := graphCommand([]string{"blast", "--root", root, "--no-refresh"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "blast radius: working tree against HEAD") || !strings.Contains(out.String(), "seed Run") {
		t.Fatalf("report = %q", out.String())
	}

	out.Reset()
	if code := graphCommand([]string{"blast", "--root", root, "--no-refresh", "--json", "-d", "all"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("json exit %d, stderr %q", code, errOut.String())
	}
	var ans map[string]any
	if err := json.Unmarshal(out.Bytes(), &ans); err != nil || ans["range"] != "working tree against HEAD" {
		t.Fatalf("json = %v, err %v", ans, err)
	}

	// Without --no-refresh the edit drifts the graph, and the rebuild note
	// goes to stderr under the command's name.
	errOut.Reset()
	if code := graphCommand([]string{"blast", "--root", root, "--depth", "2"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("refresh exit %d, stderr %q", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "loomux graph blast: ") {
		t.Errorf("no note on stderr: %q", errOut.String())
	}

	for _, args := range [][]string{
		{"blast", "--root", root, "--base", "HEAD", "--cached"},
		{"blast", "--root", root, "--bogus"},
		{"blast", "--root", root, "HEAD"},
		{"blast", "--root", root, "--depth", "0"},
	} {
		if code := graphCommand(args, nil, &out, &errOut); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
	if code := graphCommand([]string{"blast", "--root", filepath.Join(root, "missing")}, nil, &out, &errOut); code != 1 {
		t.Errorf("bad root: exit %d, want 1", code)
	}
	if code := graphCommand([]string{"blast", "--root", root, "--no-refresh", "--base=--output=x"}, nil, &out, &errOut); code == 0 {
		t.Errorf("--base=--output=x: exit 0")
	}
}

func TestGraphBlastFailsWhenTheRootCannotBeFound(t *testing.T) {
	saved := getwd
	getwd = func() (string, error) { return "", os.ErrPermission }
	t.Cleanup(func() { getwd = saved })
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"blast"}, nil, &out, &errOut); code != 1 || !strings.HasPrefix(errOut.String(), "loomux graph blast: ") {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
}
