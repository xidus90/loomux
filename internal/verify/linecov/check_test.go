package linecov

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func check(t *testing.T, root string, reports ...string) Result {
	t.Helper()
	res, err := Check(Options{Root: root, Reports: reports})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestCheckRunsWithoutFlagsFromTheDefaultLocations(t *testing.T) {
	root := writeTree(t, map[string]string{
		".coveragerc":  "[run]\nsource = src\n",
		"src/app.py":   "def f():\n    return 1\n    x = 2\n",
		"coverage.xml": `<coverage><sources><source>src</source></sources><packages><package><classes><class filename="app.py"><lines><line number="2" hits="1"/><line number="3" hits="0"/></lines></class></classes></package></packages></coverage>`,
	})
	res := check(t, root)
	if want := []Finding{{"src/app.py", 3, "not covered"}}; !reflect.DeepEqual(res.Findings, want) {
		t.Fatalf("%v %q", res.Findings, res.Notes)
	}
	if len(res.Notes) != 0 {
		t.Fatalf("a clean scoped run carries no notes: %q", res.Notes)
	}
}

// Freshness is judged against the newest report, whichever is read last.
func TestCheckJudgesSourcesAgainstTheNewestReport(t *testing.T) {
	root := writeTree(t, map[string]string{
		"a.c":    "x();\n",
		"a.info": "SF:a.c\nDA:1,1\nend_of_record\n",
		"b.info": "SF:a.c\nDA:1,1\nend_of_record\n",
		"c.info": "SF:a.c\nDA:1,1\nend_of_record\n",
	})
	base := time.Now().Add(-time.Hour)
	for name, at := range map[string]time.Duration{"a.c": 10 * time.Second, "a.info": 20 * time.Second, "b.info": 0, "c.info": 5 * time.Second} {
		if err := os.Chtimes(filepath.Join(root, name), base.Add(at), base.Add(at)); err != nil {
			t.Fatal(err)
		}
	}
	if res := check(t, root, filepath.Join(root, "*.info")); len(res.Findings) != 0 {
		t.Fatalf("%v", res.Findings)
	}
	if res := check(t, root, filepath.Join(root, "b.info")); !reflect.DeepEqual(res.Findings, []Finding{{"a.c", 0, "is newer than the report"}}) {
		t.Fatalf("%v", res.Findings)
	}
}

func TestCheckSaysWhereItLookedWhenNoReportExists(t *testing.T) {
	// Profiles are visited by name, so c8's locations come first; each
	// location is named once although three profiles share it.
	res := check(t, writeTree(t, map[string]string{"a.py": ""}))
	if len(res.Findings) != 1 ||
		!strings.HasPrefix(res.Findings[0].Text, "no coverage report found; looked in coverage/lcov.info, coverage/cobertura-coverage.xml, coverage.xml, coverage.lcov, coverage.cobertura.xml") ||
		strings.Count(res.Findings[0].Text, "coverage/lcov.info") != 1 {
		t.Fatalf("%v", res.Findings)
	}
	root := writeTree(t, nil)
	res = check(t, root, filepath.Join(root, "nothing", "*.info"))
	if len(res.Findings) != 1 || !strings.Contains(res.Findings[0].Text, "nothing") {
		t.Fatalf("%v", res.Findings)
	}
}

// An empty or binary report is a finding, never a panic.
func TestCheckReportsUnreadableUnknownAndEmptyReports(t *testing.T) {
	root := writeTree(t, map[string]string{"r/empty.info": "", "r/bin.dat": "\x00\x01\x02", "r/only.info": "SF:a.py\nend_of_record\n", "a.py": "x\n"})
	res := check(t, root, filepath.Join(root, "r"))
	want := []Finding{
		{"", 0, "the reports hold no line records"},
		{"r/bin.dat", 0, "unknown coverage format"},
		{"r/empty.info", 0, "unknown coverage format"},
	}
	if !reflect.DeepEqual(res.Findings, want) {
		t.Fatalf("%v", res.Findings)
	}
}

// One file listed twice is one file with summed hits.
func TestCheckSumsAFileListedTwiceAndAcrossReports(t *testing.T) {
	root := writeTree(t, map[string]string{
		"a.c":      "x();\ny();\n",
		"one.info": "SF:a.c\nDA:1,0\nDA:2,1\nend_of_record\nSF:a.c\nDA:1,0\nend_of_record\n",
		"two.info": "SF:a.c\nDA:1,3\nend_of_record\n",
	})
	if res := check(t, root, filepath.Join(root, "*.info")); len(res.Findings) != 0 {
		t.Fatalf("%v", res.Findings)
	}
	if res := check(t, root, filepath.Join(root, "one.info")); !reflect.DeepEqual(res.Findings, []Finding{{"a.c", 1, "not covered"}}) {
		t.Fatalf("%v", res.Findings)
	}
}

func TestCheckResolvesEveryOutcomeAndNotesWhatItSkipped(t *testing.T) {
	root := writeTree(t, map[string]string{
		"go.mod":         "module example.com/m\n",
		"a.go":           "package m\n",
		"w/.git":         "gitdir: x",
		"w/b.c":          "",
		"x/com/y/D.java": "", "z/com/y/D.java": "",
		"r.info": "SF:/usr/include/stdio.h\nDA:1,0\nSF:w/b.c\nDA:1,0\nSF:gone.c\nDA:1,0\nend_of_record\n",
		"g.out":  "mode: set\nexample.com/m/a.go:1.1,1.9 1 1\n",
		"j.xml":  `<report><package name="com/y"><sourcefile name="D.java"><line nr="1" ci="1"/></sourcefile></package></report>`,
	})
	res := check(t, root, filepath.Join(root, "r.info"), filepath.Join(root, "g.out"), filepath.Join(root, "j.xml"))
	want := []Finding{
		{"com/y/D.java", 0, "ambiguous source: x/com/y/D.java, z/com/y/D.java"},
		{"gone.c", 0, "is in the report but not on disk"},
	}
	if !reflect.DeepEqual(res.Findings, want) {
		t.Fatalf("%v", res.Findings)
	}
	for _, note := range []string{"1 records outside the root were skipped", "1 records in other checkouts were skipped"} {
		if !slices.Contains(res.Notes, note) {
			t.Errorf("missing note %q in %q", note, res.Notes)
		}
	}
}

func TestCheckNotesAProfileWithoutScopeAndReportsReadProblems(t *testing.T) {
	root := writeTree(t, map[string]string{
		"coverage/lcov.info": "SF:src/a.js\nDA:1,1\nDA:x,1\nend_of_record\n",
		"src/a.js":           "x();\n",
	})
	res := check(t, root)
	if !reflect.DeepEqual(res.Findings, []Finding{{"coverage/lcov.info", 0, `src/a.js: unreadable record "DA:x,1"`}}) {
		t.Fatalf("%v", res.Findings)
	}
	if !slices.Contains(res.Notes, "vitest: no scope in its settings; files absent from the report are not checked") {
		t.Fatalf("%q", res.Notes)
	}
}

// A tool another scoped profile stands for is one the project does not use.
func TestCheckNotesNoMissingScopeForToolsAScopedOneCovers(t *testing.T) {
	root := writeTree(t, map[string]string{
		".nycrc":             `{"include": ["src/**"]}`,
		"coverage/lcov.info": "SF:src/a.js\nDA:1,1\nend_of_record\n",
		"src/a.js":           "x();\n",
	})
	if res := check(t, root); len(res.Findings) != 0 || len(res.Notes) != 0 {
		t.Fatalf("%v %q", res.Findings, res.Notes)
	}
}

func TestCheckReportsAReportDirectoryItCannotList(t *testing.T) {
	root := writeTree(t, map[string]string{"r/a.info": "SF:a.c\nDA:1,1\nend_of_record\n"})
	old := readDir
	t.Cleanup(func() { readDir = old })
	readDir = func(string) ([]os.DirEntry, error) { return nil, errors.New("denied") }
	res := check(t, root, filepath.Join(root, "r"))
	if want := []Finding{{"r", 0, "unreadable: denied"}}; !reflect.DeepEqual(res.Findings, want) {
		t.Fatalf("%v", res.Findings)
	}
}

func TestCheckSaysWhyGoMODGivesNoModule(t *testing.T) {
	root := writeTree(t, map[string]string{"go.mod": "garbage\n", "a.go": "package m\n", "g.out": "mode: set\nexample.com/m/a.go:1.1,1.9 1 1\n"})
	res := check(t, root, filepath.Join(root, "g.out"))
	if !slices.Contains(res.Notes, "go.mod has no module line; Go profile paths cannot be resolved") {
		t.Fatalf("%q", res.Notes)
	}
}

func TestCheckFailsOnAMissingRootAndABadGlob(t *testing.T) {
	if _, err := Check(Options{Root: filepath.Join(t.TempDir(), "absent")}); err == nil {
		t.Fatal("no error for a missing root")
	}
	if _, err := Check(Options{Root: t.TempDir(), Reports: []string{"["}}); err == nil {
		t.Fatal("no error for a bad glob")
	}
}

func TestCheckReadsEveryFileOfAReportDirectoryButNotItsSubdirectories(t *testing.T) {
	root := writeTree(t, map[string]string{
		"a.c": "x();\n", "r/one.info": "SF:a.c\nDA:1,1\nend_of_record\n", "r/deeper/two.info": "SF:a.c\nDA:1,0\nend_of_record\n",
	})
	if res := check(t, root, filepath.Join(root, "r")); len(res.Findings) != 0 {
		t.Fatalf("%v", res.Findings)
	}
}

func TestCheckReportsAReportItCannotRead(t *testing.T) {
	root := writeTree(t, map[string]string{"a.c": "x();\n", "r.info": "SF:a.c\nDA:1,1\nend_of_record\n"})
	old := readFile
	t.Cleanup(func() { readFile = old })
	readFile = func(string) ([]byte, error) { return nil, errors.New("locked") }
	res := check(t, root, filepath.Join(root, "r.info"))
	want := []Finding{{"", 0, "the reports hold no line records"}, {"r.info", 0, "unreadable: locked"}}
	if !reflect.DeepEqual(res.Findings, want) {
		t.Fatalf("%v", res.Findings)
	}
}

func TestCheckFailsWhenTheProfilesCannotBeLoaded(t *testing.T) {
	old := loadProfiles
	t.Cleanup(func() { loadProfiles = old })
	loadProfiles = func() ([]Profile, error) { return nil, errors.New("broken") }
	if _, err := Check(Options{Root: t.TempDir()}); err == nil {
		t.Fatal("no error for unloadable profiles")
	}
}

// A source whose mtime cannot be read is never called stale.
func TestCheckIgnoresASourceItCannotStat(t *testing.T) {
	root := writeTree(t, map[string]string{"a.c": "x();\n", "r.info": "SF:a.c\nDA:1,1\nend_of_record\n"})
	old := statFile
	t.Cleanup(func() { statFile = old })
	statFile = func(name string) (os.FileInfo, error) {
		if strings.HasSuffix(name, "a.c") {
			return nil, errors.New("vanished")
		}
		return os.Stat(name)
	}
	if res := check(t, root, filepath.Join(root, "r.info")); len(res.Findings) != 0 {
		t.Fatalf("%v", res.Findings)
	}
}

func TestCheckNamesAReportOutsideTheRootAsGiven(t *testing.T) {
	outside := writeTree(t, map[string]string{"r.info": ""})
	res := check(t, writeTree(t, map[string]string{"a.c": ""}), filepath.Join(outside, "r.info"))
	if len(res.Findings) == 0 || res.Findings[len(res.Findings)-1].Path != filepath.ToSlash(filepath.Join(outside, "r.info")) {
		t.Fatalf("%v", res.Findings)
	}
}
