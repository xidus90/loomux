package query

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/testlock"
)

func TestOnlyDropsRefusedIDsAndCountsThem(t *testing.T) {
	d := Drift{
		Added:   []string{"lib/a.go#A", "secrets/k.go#K"},
		Removed: []string{"secrets/old.go"},
		Changed: []string{"lib/b.go#B"},
		paths: map[string]string{
			"lib/a.go#A": "lib/a.go", "secrets/k.go#K": "secrets/k.go",
			"secrets/old.go": "secrets/old.go", "lib/b.go#B": "lib/b.go",
		},
	}
	got := d.Only(func(path string) bool { return strings.HasPrefix(path, "lib/") })

	if len(got.Added) != 1 || got.Added[0] != "lib/a.go#A" {
		t.Errorf("Added = %v", got.Added)
	}
	if len(got.Removed) != 0 {
		t.Errorf("Removed = %v, want none", got.Removed)
	}
	if len(got.Changed) != 1 {
		t.Errorf("Changed = %v", got.Changed)
	}
	if got.Hidden != 2 {
		t.Errorf("Hidden = %d, want 2", got.Hidden)
	}
	if got.OK {
		t.Error("hiding drift must not turn it into OK")
	}
}

func TestOnlyWithNilKeepsEverything(t *testing.T) {
	d := Drift{Added: []string{"x.go#X"}}
	if got := d.Only(nil); len(got.Added) != 1 || got.Hidden != 0 {
		t.Errorf("got %+v", got)
	}
}

func TestCheckReportNamesHiddenDrift(t *testing.T) {
	out := CheckReport(Drift{Changed: []string{"lib/a.go#A"}, Hidden: 3})
	if !strings.Contains(out, "3 more under the area's never globs") {
		t.Errorf("report %q must say that drift was hidden", out)
	}
}

func TestCheckReportWithoutHiddenDriftSaysNothingOfIt(t *testing.T) {
	// The cloud channel zeroes Hidden; this keeps both the count and the line
	// out of what a remote model reads.
	out := CheckReport(Drift{Changed: []string{"lib/a.go#A"}})
	want := "loomux graph check: DRIFT\n\n  changed  lib/a.go#A\n\nRun `loomux graph build`.\n"
	if out != want {
		t.Errorf("report = %q, want %q", out, want)
	}
}

// built is sample() with a graph written for it.
func built(t *testing.T) string {
	t.Helper()
	root := repo(t, sample())
	if _, _, err := Build(root, ignore); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCheckIsOKRightAfterABuild(t *testing.T) {
	d, err := Check(built(t))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d, Drift{OK: true}) {
		t.Errorf("Check = %+v, want only OK", d)
	}
}

func TestCheckWithoutAGraphIsMissingAndNoError(t *testing.T) {
	d, err := Check(repo(t, sample()))
	if err != nil {
		t.Fatalf("a missing graph is a finding, not an error: %v", err)
	}
	if !reflect.DeepEqual(d, Drift{Missing: true}) {
		t.Errorf("Check = %+v, want only Missing", d)
	}
}

// writeWiring puts body where the graph belongs.
func writeWiring(t *testing.T, root, body string) {
	t.Helper()
	if err := os.MkdirAll(store.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.WiringPath(root), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckOnASchemaOneGraphIsOutdated(t *testing.T) {
	root := repo(t, sample())
	writeWiring(t, root, `{"meta":{"version":1},"nodes":[],"edges":[]}`+"\n")
	d, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d, Drift{Outdated: true}) {
		t.Errorf("Check = %+v, want only Outdated", d)
	}
}

func TestCheckReportsACorruptGraphAsAnError(t *testing.T) {
	root := repo(t, sample())
	writeWiring(t, root, "{corrupt")
	d, err := Check(root)
	if err == nil {
		t.Fatalf("Check = %+v, want the decode error", d)
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Errorf("err = %v must not read as a missing graph", err)
	}
}

func TestCheckCallsAGraphFromAnotherExtractorForeignBeforeDiffing(t *testing.T) {
	root := built(t)
	g, err := store.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	g.Meta.Extractor = "go/0"
	// A node that disagrees too: a diff run before the stamp check would
	// report it, so the result below tells the two orders apart.
	g.Nodes[0].BodyHash = "not-a-real-hash"
	if err := store.Write(root, g); err != nil {
		t.Fatal(err)
	}
	d, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d, Drift{Foreign: "go/0"}) {
		t.Errorf("Check = %+v, want only Foreign go/0", d)
	}
}

func TestCheckReportsAFailedReExtraction(t *testing.T) {
	root := built(t)
	testlock.Lock(t, filepath.Join(root, "main.go"))
	if d, err := Check(root); err == nil {
		t.Fatalf("Check = %+v, want the extraction error", d)
	}
}

func TestCheckNamesAddedRemovedAndChangedSymbols(t *testing.T) {
	root := built(t)
	// Run is gone, Walk is new, and main's body changes.
	if err := os.WriteFile(filepath.Join(root, "lib", "lib.go"), []byte("package lib\n\nfunc Walk() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	main := "package main\n\nimport \"example.com/repo/lib\"\n\nfunc main() { lib.Walk() }\n"
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	if d.OK {
		t.Error("drift must not be OK")
	}
	if !reflect.DeepEqual(d.Added, []string{"lib/lib.go#Walk"}) {
		t.Errorf("Added = %v, want [lib/lib.go#Walk]", d.Added)
	}
	if !reflect.DeepEqual(d.Removed, []string{"lib/lib.go#Run"}) {
		t.Errorf("Removed = %v, want [lib/lib.go#Run]", d.Removed)
	}
	if !contains(d.Changed, "main.go#main") {
		t.Errorf("Changed = %v, want main.go#main among them", d.Changed)
	}
	if contains(d.Changed, "lib/lib_test.go#TestRun") {
		t.Errorf("Changed = %v lists a symbol whose body did not change", d.Changed)
	}
}

func contains(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func TestOnlyAfterCheckJudgesTheNodesPathAndNeverAPathCutFromTheID(t *testing.T) {
	// "#gen.go" is a file node whose id has no '#' added, and "dir#1" a
	// directory; cutting either id at its first '#' reads "" or "dir", which
	// the never globs below do not cover.
	files := sample()
	files["#gen.go"] = "package main\n\nfunc Gen() {}\n"
	files["dir#1/a.go"] = "package dir\n\nfunc A() {}\n"
	root := repo(t, files)
	if _, _, err := Build(root, ignore); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "#gen.go"), []byte("package main\n\nfunc Gen() { _ = 1 }\n\nfunc More() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "dir#1", "a.go"), []byte("package dir\n\nfunc B() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	keep := func(p string) bool { return !strings.HasPrefix(p, "#") && !strings.HasPrefix(p, "dir#1/") }
	got := d.Only(keep)

	for _, ids := range [][]string{got.Added, got.Removed, got.Changed} {
		for _, id := range ids {
			t.Errorf("%s lies under a never glob and must be hidden", id)
		}
	}
	if got.Hidden == 0 || got.OK {
		t.Errorf("Hidden = %d, OK = %v; want the drift counted and still not OK", got.Hidden, got.OK)
	}
}

func TestAnIDWithNoRecordedPathIsHiddenAndNeverParsed(t *testing.T) {
	// keep admits every path it is asked about, so the ids can only be dropped
	// if keep is never asked -- no path is cut from an id and judged.
	var asked []string
	got := Drift{Changed: []string{"#gen.go#Gen", "lib/a.go#A"}}.Only(func(p string) bool {
		asked = append(asked, p)
		return true
	})
	if len(got.Changed) != 0 || got.Hidden != 2 {
		t.Errorf("got %+v, want both ids hidden: a path nobody recorded fails closed", got)
	}
	if len(asked) != 0 {
		t.Errorf("keep was asked about %q; an id without a recorded path must not be parsed", asked)
	}
}

func TestTheRecordedPathsStayOutOfTheJSON(t *testing.T) {
	d, err := Check(built(t))
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(b), `{"ok":true,"missing":false,"added":null,"removed":null,"changed":null}`; got != want {
		t.Errorf("json = %s, want %s", got, want)
	}
}

func TestCheckRecordsThePathOfEveryIDItReports(t *testing.T) {
	// Only drops an id whose path is not recorded, so a path lost for one id
	// hides it even from a keep that admits everything.
	root := built(t)
	if err := os.WriteFile(filepath.Join(root, "lib", "lib.go"), []byte("package lib\n\nfunc Walk() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	reported := len(d.Added) + len(d.Removed) + len(d.Changed)
	if reported < 2 {
		t.Fatalf("drift %+v, want at least two ids to tell one recorded path from all", d)
	}
	got := d.Only(func(string) bool { return true })
	if got.Hidden != 0 || len(got.Added)+len(got.Removed)+len(got.Changed) != reported {
		t.Errorf("got %+v, want every reported id kept: each one has its path recorded", got)
	}
}
