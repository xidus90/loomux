package gutsuite

import (
	"slices"
	"testing"
)

// GUT includes subdirectories only when told: without the key in a
// .gutconfig.json it keeps to the top of each directory (gut.gd in GUT 7 and
// 9: _include_subdirectories = false), so loomux expects no deeper script.
func TestAConfigWithoutIncludeSubdirsKeepsToTheTop(t *testing.T) {
	root := project(t, 4, map[string]string{
		"test/test_a.gd":      twoScripts,
		"test/unit/test_b.gd": twoScripts,
		".gutconfig.json":     `{"dirs": ["res://test"]}`,
	})
	got, _, err := Scripts(root)
	if err != nil || !slices.Equal(got, []string{"test/test_a.gd"}) {
		t.Fatalf("%v %v", got, err)
	}
}

// GUT names the suite of an inner test class after its script and class
// (collected_script.gd get_full_name: path + "." + class), so a script whose
// tests all sit in inner classes ran when such a suite is in the report.
func TestInnerClassSuitesCountForTheirScript(t *testing.T) {
	root := project(t, 4, map[string]string{"test/test_a.gd": "extends GutTest\nclass TestFoo extends GutTest:\n\tfunc test_x():\n\t\tpass\n"})
	report := `<testsuites><testsuite name="test/test_a.gd.TestFoo"><testcase name="test_x" status="pass"></testcase></testsuite></testsuites>`
	if r := run(t, root, &engine{report: report}); len(r.Findings) != 0 {
		t.Fatalf("%v", r.Findings)
	}
	// A longer name of another script is no inner class of this one.
	report = `<testsuites><testsuite name="test/test_a.gdx"><testcase name="test_x" status="pass"></testcase></testsuite></testsuites>`
	if r := run(t, root, &engine{report: report}); !slices.Equal(r.Findings, []string{"test/test_a.gd: planned, but not in GUT's report"}) {
		t.Fatalf("%v", r.Findings)
	}
}
