package gutsuite

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestRunReportsWhatKeepsItFromStarting(t *testing.T) {
	root := project(t, 4, map[string]string{"test/test_a.gd": twoScripts, ".gutconfig.json": "{"})
	if _, err := Run(Options{Root: root, Godot: "godot", RunID: "R", Start: (&engine{}).start}); err == nil || !strings.Contains(err.Error(), ".gutconfig.json") {
		t.Fatalf("broken config: %v", err)
	}
	root = project(t, 4, map[string]string{"test/test_a.gd": twoScripts, ".loomux": "a file where the state directory goes"})
	if _, err := Run(Options{Root: root, Godot: "godot", RunID: "R", Start: (&engine{}).start}); err == nil {
		t.Fatal("no state directory, no error")
	}
}

// Without a Start the real runner starts the engine; one that does not exist
// writes no report, which is a finding and no crash.
func TestRunUsesTheRealRunnerByDefault(t *testing.T) {
	root := project(t, 4, map[string]string{"test/test_a.gd": twoScripts})
	r, err := Run(Options{Root: root, Godot: filepath.Join(root, "no-such-godot"), RunID: "R"})
	if err != nil || len(r.Findings) != 1 || !strings.HasPrefix(r.Findings[0], "GUT wrote no report") {
		t.Fatalf("%v %v", r.Findings, err)
	}
}

func TestRunNamesAReportItCannotRead(t *testing.T) {
	root := project(t, 4, map[string]string{"test/test_a.gd": twoScripts})
	r := run(t, root, &engine{report: "<testsuites><testsuite"})
	if len(r.Findings) != 1 || !strings.HasPrefix(r.Findings[0], "GUT's report cannot be read") {
		t.Fatalf("%v", r.Findings)
	}
}

func TestASuffixFromTheConfig(t *testing.T) {
	root := project(t, 4, map[string]string{"spec/a_spec.gd": twoScripts, "spec/test_b.gd": twoScripts,
		".gutconfig.json": `{"dirs": ["res://spec"], "prefix": "", "suffix": "_spec.gd"}`})
	got, _, err := Scripts(root)
	if err != nil || !slices.Equal(got, []string{"spec/a_spec.gd"}) {
		t.Fatalf("%v %v", got, err)
	}
}

func TestTailAndFirstLine(t *testing.T) {
	if got := tail("1\n2\n3\n4\n5\n6\n7\n"); got != "3 | 4 | 5 | 6 | 7" {
		t.Errorf("tail %q", got)
	}
	if got := firstLine("\n  \n"); got != "" {
		t.Errorf("firstLine %q", got)
	}
}
