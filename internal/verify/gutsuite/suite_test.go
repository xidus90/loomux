package gutsuite

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/child"
)

// project writes a GUT project of line 3 or 4 with the given files besides
// project.godot and the addon's command-line script.
func project(t *testing.T, line int, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	godot := "config_version=5\nconfig/features=PackedStringArray(\"4.7\")\n"
	if line == 3 {
		godot = "config_version=4\n_global_script_classes=[ {\n\"class\": \"GutTest\"\n} ]\n"
	}
	all := map[string]string{"project.godot": godot, "addons/gut/gut_cmdln.gd": "extends SceneTree\n", "addons/gut/test.gd": "class_name GutTest\n"}
	for k, v := range files {
		all[k] = v
	}
	for path, body := range all {
		p := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const twoScripts = "extends GutTest\nfunc test_x():\n\tpass\n"

// engine is a fake GUT run: it writes report (unless empty) where the
// command line asks and returns code with output.
type engine struct {
	report string
	code   int
	output string
	argv   []string
}

func (e *engine) start(s child.Spec) child.Result {
	e.argv = s.Argv
	for _, a := range s.Argv {
		if p, ok := strings.CutPrefix(a, "-gjunit_xml_file="); ok && e.report != "" {
			if err := os.WriteFile(p, []byte(e.report), 0o644); err != nil {
				panic(err)
			}
		}
	}
	return child.Result{Code: e.code, Stdout: e.output}
}

func run(t *testing.T, root string, e *engine) Result {
	t.Helper()
	r, err := Run(Options{Root: root, Godot: "godot", RunID: "R", Start: e.start})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// The report as GUT 9 writes it (names without res://) and as GUT 7 writes
// it (with res://), with one case per planned script.
const green9 = `<testsuites name="GutTests" failures="0" tests="2">
<testsuite name="test/test_a.gd" tests="1"><testcase name="test_x" status="pass" classname="test/test_a.gd"></testcase></testsuite>
<testsuite name="test/sub/test_b.gd" tests="1"><testcase name="test_x" status="pass" classname="test/sub/test_b.gd"></testcase></testsuite>
</testsuites>`

func TestRunPassesOnAGreenReport(t *testing.T) {
	root := project(t, 4, map[string]string{"test/test_a.gd": twoScripts, "test/sub/test_b.gd": twoScripts})
	if r := run(t, root, &engine{report: green9}); len(r.Findings) != 0 {
		t.Fatalf("%v", r.Findings)
	}
	green7 := strings.ReplaceAll(green9, `name="test/`, `name="res://test/`)
	if r := run(t, root, &engine{report: green7}); len(r.Findings) != 0 {
		t.Fatalf("GUT 7 names: %v", r.Findings)
	}
}

func TestRunFailsOnAFailingCase(t *testing.T) {
	root := project(t, 4, map[string]string{"test/test_a.gd": twoScripts})
	report := `<testsuites><testsuite name="test/test_a.gd"><testcase name="test_x" status="fail"><failure message="failed"><![CDATA[[1] expected to equal [2]
  at line 7]]></failure></testcase></testsuite></testsuites>`
	r := run(t, root, &engine{report: report, code: 1})
	if len(r.Findings) != 1 || r.Findings[0] != "test/test_a.gd test_x: [1] expected to equal [2]" {
		t.Fatalf("%v", r.Findings)
	}
}

// Godot 4 without its import exits 0 and writes no report at all.
func TestRunFailsWithoutAReport(t *testing.T) {
	root := project(t, 4, map[string]string{"test/test_a.gd": twoScripts})
	r := run(t, root, &engine{output: "ERROR: Some GUT class_names have not been imported.\n"})
	if len(r.Findings) != 1 || !strings.HasPrefix(r.Findings[0], "GUT wrote no report") || !strings.Contains(r.Findings[0], "Some GUT class_names") {
		t.Fatalf("%v", r.Findings)
	}
}

// Godot 3 without its class list says "No tests ran", exits 0 and writes a
// report with no case.
func TestRunFailsOnAReportWithoutCases(t *testing.T) {
	root := project(t, 3, map[string]string{"test/test_a.gd": twoScripts})
	r := run(t, root, &engine{report: `<testsuites name="GutTests" failures="0" tests="0"></testsuites>`, output: "No tests ran\n"})
	if len(r.Findings) != 1 || !strings.HasPrefix(r.Findings[0], "GUT ran no test") {
		t.Fatalf("%v", r.Findings)
	}
}

func TestRunFailsWhenAPlannedScriptDidNotRun(t *testing.T) {
	root := project(t, 4, map[string]string{"test/test_a.gd": twoScripts, "test/sub/test_b.gd": twoScripts, "test/sub/test_c.gd": twoScripts})
	r := run(t, root, &engine{report: green9})
	if !slices.Equal(r.Findings, []string{"test/sub/test_c.gd: planned, but not in GUT's report"}) {
		t.Fatalf("%v", r.Findings)
	}
}

func TestRunFailsOnANonZeroExitWithACleanReport(t *testing.T) {
	root := project(t, 4, map[string]string{"test/test_a.gd": twoScripts, "test/sub/test_b.gd": twoScripts})
	r := run(t, root, &engine{report: green9, code: 3, output: "crash\n"})
	if len(r.Findings) != 1 || !strings.HasPrefix(r.Findings[0], "the engine exited with 3") {
		t.Fatalf("%v", r.Findings)
	}
}

func TestRunFailsWithoutTestScripts(t *testing.T) {
	root := project(t, 4, map[string]string{"test/helper.gd": "extends Node\n"})
	e := &engine{}
	r := run(t, root, e)
	if !slices.Equal(r.Findings, []string{"no GUT test script (test_*.gd) under res://test, res://tests"}) || e.argv != nil {
		t.Fatalf("%v %v", r.Findings, e.argv)
	}
}

func TestRunNeedsTheAddon(t *testing.T) {
	root := project(t, 4, map[string]string{"test/test_a.gd": twoScripts})
	if err := os.Remove(filepath.Join(root, "addons", "gut", "gut_cmdln.gd")); err != nil {
		t.Fatal(err)
	}
	r := run(t, root, &engine{})
	if !slices.Equal(r.Findings, []string{"GUT is not installed: addons/gut/gut_cmdln.gd is missing"}) {
		t.Fatalf("%v", r.Findings)
	}
}

// Godot 3 has no --headless and takes --no-window; the report goes to the
// state directory, as an absolute path GUT can write to.
func TestTheCommandLineFollowsTheLine(t *testing.T) {
	for line, flag := range map[int]string{3: "--no-window", 4: "--headless"} {
		root := project(t, line, map[string]string{"test/test_a.gd": twoScripts})
		e := &engine{}
		run(t, root, e)
		report := filepath.Join(root, ".loomux", "state", "gut-suite", "R", "junit.xml")
		want := []string{"godot", flag, "--path", root, "-s", "res://addons/gut/gut_cmdln.gd",
			"-gdir=res://test", "-ginclude_subdirs", "-gprefix=test_", "-gexit", "-gjunit_xml_file=" + report}
		if !slices.Equal(e.argv, want) {
			t.Errorf("line %d: %q", line, e.argv)
		}
	}
}

// A .gutconfig.json says where the tests are; GUT reads it itself, and loomux
// reads the same keys to know what to expect in the report.
func TestScriptsFollowGutsRule(t *testing.T) {
	root := project(t, 4, map[string]string{
		"test/test_a.gd":       twoScripts,
		"test/sub/test_b.gd":   twoScripts,
		"tests/test_c.gd":      twoScripts,
		"test/not_a_test.gd":   twoScripts,
		"spec/check_d.gd":      twoScripts,
		"spec/deep/check_e.gd": twoScripts,
	})
	got, dirs, err := Scripts(root)
	if err != nil || !slices.Equal(got, []string{"test/sub/test_b.gd", "test/test_a.gd", "tests/test_c.gd"}) || !slices.Equal(dirs, []string{"res://test", "res://tests"}) {
		t.Fatalf("%v %v %v", got, dirs, err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gutconfig.json"), []byte(`{"dirs": ["res://spec"], "prefix": "check_", "include_subdirs": false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _, err = Scripts(root)
	if err != nil || !slices.Equal(got, []string{"spec/check_d.gd"}) {
		t.Fatalf("%v %v", got, err)
	}
	e := &engine{}
	run(t, root, e)
	if !slices.Contains(e.argv, "-gconfig=res://.gutconfig.json") || slices.ContainsFunc(e.argv, func(a string) bool { return strings.HasPrefix(a, "-gdir") }) {
		t.Fatalf("%q", e.argv)
	}
	if err := os.WriteFile(filepath.Join(root, ".gutconfig.json"), []byte(`{`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Scripts(root); err == nil || !strings.Contains(err.Error(), ".gutconfig.json") {
		t.Fatalf("%v", err)
	}
}
