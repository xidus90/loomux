package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestGraphCheckIsAStubAwaitingItsOwnTask(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := graphCommand([]string{"check"}, nil, &out, &errOut); code != 2 {
		t.Fatalf("exit %d, want 2", code)
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
