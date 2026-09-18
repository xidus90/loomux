package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/code/store"
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
	// This is the one place the port goes past Graft, which drops a package
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
	// Graft has no separate code for this, and the pillar-3 spec asks only that
	// it be reported cleanly as not initialised. A third code would be an
	// invention.
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(out.String(), "graph build") {
		t.Errorf("report %q must point at the command that fixes it", out.String())
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
	if !strings.Contains(out.String(), "extractor") {
		t.Errorf("report %q must say the graph is from another extractor", out.String())
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

func TestGraphBuildWarnsButSucceedsWhenTheFingerprintCannotBeWritten(t *testing.T) {
	root := repo(t, sample())
	if err := os.MkdirAll(store.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	// A regular file named "cache" blocks freshness.Write's own MkdirAll,
	// while leaving store.Write's directory untouched.
	if err := os.WriteFile(filepath.Join(store.Dir(root), "cache"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"build", "--root", root}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d, want 0, stderr %q", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "freshness record not written") {
		t.Errorf("stderr %q must warn about the freshness record", errOut.String())
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

func TestBuildGraphFailsWhenTheRootDoesNotExist(t *testing.T) {
	if _, _, err := buildGraph(filepath.Join(t.TempDir(), "gone")); err == nil {
		t.Fatal("want an error for a missing root")
	}
}

func TestBuildGraphFailsWhenASourceFileCannotBeRead(t *testing.T) {
	root := repo(t, sample())
	testlock.Lock(t, filepath.Join(root, "main.go"))
	if _, _, err := buildGraph(root); err == nil {
		t.Fatal("want an error for an unreadable source file")
	}
}

func TestBuildGraphFailsWhenGoModCannotBeRead(t *testing.T) {
	root := repo(t, sample())
	testlock.Lock(t, filepath.Join(root, "go.mod"))
	if _, _, err := buildGraph(root); err == nil {
		t.Fatal("want an error for an unreadable go.mod")
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

func TestGoModPathsSkipsADirectoryItCannotRead(t *testing.T) {
	root := repo(t, sample())
	blocked := filepath.Join(root, "blocked")
	if err := os.MkdirAll(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(blocked, "go.mod"), []byte("module blocked\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testlock.LockDir(t, blocked)
	// The walk must not fail the whole build over one unreadable directory.
	got := goModPaths(root)
	for _, p := range got {
		if strings.HasPrefix(p, "blocked/") {
			t.Errorf("goModPaths(%q) must not see into the locked directory, got %v", root, got)
		}
	}
}
