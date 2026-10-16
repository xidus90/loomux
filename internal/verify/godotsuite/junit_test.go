package godotsuite

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// shardWith lays reports under <dir>/reports/report_<n>/results.xml.
func shardWith(t *testing.T, reports map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for n, body := range reports {
		d := filepath.Join(dir, "reports", "report_"+n)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "results.xml"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "junit", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestVerdictNoReportCarriesExitCodeAndOutput(t *testing.T) {
	for _, code := range []int{0, 139} {
		f, times := Verdict(Shard{Index: 2, Suites: []string{"res://a.gd", "res://b.gd", "res://c.gd"},
			Dir: t.TempDir(), Code: code, Output: "boom"})
		want := fmt.Sprintf("shard 2 wrote no test report (exit %d); it ran 3 suites:\nboom", code)
		if len(f) != 1 || f[0] != want || times != nil {
			t.Fatalf("got %q %v", f, times)
		}
	}
}

func TestVerdictUnreadableXML(t *testing.T) {
	dir := shardWith(t, map[string]string{"1": "<testsuites><oops"})
	f, _ := Verdict(Shard{Index: 0, Dir: dir})
	prefix := "shard 0 wrote an unreadable test report " + filepath.Join(dir, "reports", "report_1", "results.xml") + ": "
	if len(f) != 1 || !strings.HasPrefix(f[0], prefix) {
		t.Fatalf("got %q", f)
	}
}

func TestVerdictUnreadableWhenResultsIsNotAFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "reports", "report_1", "results.xml"), 0o755); err != nil {
		t.Fatal(err)
	}
	f, _ := Verdict(Shard{Index: 4, Dir: dir})
	if len(f) != 1 || !strings.HasPrefix(f[0], "shard 4 wrote an unreadable test report ") {
		t.Fatalf("got %q", f)
	}
}

func TestVerdictSyntheticReportWithoutCases(t *testing.T) {
	// Synthetic: no probe produced one (gdUnit4 writes no report for an
	// empty run), so green.xml is stripped of its testcase elements.
	var kept []string
	for _, l := range strings.Split(fixture(t, "green.xml"), "\n") {
		if !strings.Contains(l, "testcase") {
			kept = append(kept, l)
		}
	}
	dir := shardWith(t, map[string]string{"1": strings.Join(kept, "\n")})
	f, _ := Verdict(Shard{Index: 0, Suites: []string{"res://x.gd"}, Dir: dir})
	if len(f) != 1 || f[0] != "shard 0 ran no test cases; none of its 1 suites loaded" {
		t.Fatalf("got %q", f)
	}
}

func TestVerdictFailureTextComesFromTheBlockNotTheLocation(t *testing.T) {
	dir := shardWith(t, map[string]string{"1": fixture(t, "failing.xml")})
	f, _ := Verdict(Shard{Index: 0, Suites: []string{"res://test/lx/failing_test.gd"}, Dir: dir, Code: 100})
	if len(f) != 1 || f[0] != "res://test/lx/failing_test.gd test_fails: Expecting:" {
		t.Fatalf("got %q", f)
	}
}

func TestVerdictErrorCountsLikeFailure(t *testing.T) {
	dir := shardWith(t, map[string]string{"1": fixture(t, "errors.xml")})
	f, _ := Verdict(Shard{Index: 0, Suites: []string{"res://test/lxrt/runtime_error_test.gd"}, Dir: dir, Code: 100})
	if len(f) != 1 || !strings.HasSuffix(f[0], ": Godot Runtime Error !") ||
		!strings.HasPrefix(f[0], "res://test/lxrt/runtime_error_test.gd ") {
		t.Fatalf("got %q", f)
	}
}

func TestVerdictFallsBackToTheMessageAttribute(t *testing.T) {
	x := `<testsuites><testsuite name="a_test"><testcase name="t"><failure message="FAILED: here"></failure></testcase></testsuite></testsuites>`
	f, _ := Verdict(Shard{Suites: []string{"res://a_test.gd"}, Dir: shardWith(t, map[string]string{"1": x})})
	if len(f) != 1 || f[0] != "res://a_test.gd t: FAILED: here" {
		t.Fatalf("got %q", f)
	}
}

func TestVerdictPlannedSuiteMissing(t *testing.T) {
	dir := shardWith(t, map[string]string{"1": fixture(t, "green.xml")})
	f, _ := Verdict(Shard{Index: 1, Dir: dir, Suites: []string{
		"res://test/core/animation_throttle_test.gd", "res://test/smoke_test.gd",
		"res://test/z_test.gd", "res://test/y_test.gd"}})
	if len(f) != 1 || f[0] != "shard 1 did not run: res://test/z_test.gd, res://test/y_test.gd" {
		t.Fatalf("got %q", f)
	}
}

func TestVerdictExitCodeWithCleanReport(t *testing.T) {
	dir := shardWith(t, map[string]string{"1": fixture(t, "green.xml")})
	f, _ := Verdict(Shard{Index: 0, Dir: dir, Code: 1, Output: "engine says",
		Suites: []string{"res://test/core/animation_throttle_test.gd", "res://test/smoke_test.gd"}})
	if len(f) != 1 || f[0] != "shard 0 exited with 1 while its report shows nothing wrong:\nengine says" {
		t.Fatalf("got %q", f)
	}
}

func TestVerdictGreenReturnsTimes(t *testing.T) {
	dir := shardWith(t, map[string]string{"1": fixture(t, "green.xml")})
	f, times := Verdict(Shard{Index: 0, Dir: dir,
		Suites: []string{"res://test/core/animation_throttle_test.gd", "res://test/smoke_test.gd"}})
	if len(f) != 0 || times["res://test/core/animation_throttle_test.gd"] != 0.155 || len(times) != 2 {
		t.Fatalf("got %q %v", f, times)
	}
}

func setMtime(t *testing.T, dir, n string, at time.Time) {
	t.Helper()
	if err := os.Chtimes(filepath.Join(dir, "reports", "report_"+n, "results.xml"), at, at); err != nil {
		t.Fatal(err)
	}
}

func TestVerdictTheHigherReportNumberWinsOnEqualMtime(t *testing.T) {
	dir := shardWith(t, map[string]string{"9": fixture(t, "green.xml"), "10": fixture(t, "failing.xml")})
	same := time.Now()
	setMtime(t, dir, "9", same)
	setMtime(t, dir, "10", same)
	f, _ := Verdict(Shard{Dir: dir, Suites: []string{"res://test/lx/failing_test.gd"}})
	if len(f) != 1 || !strings.Contains(f[0], "test_fails") {
		t.Fatalf("report_10 should win: %q", f)
	}
}

func TestVerdictTheNewerMtimeWinsOverTheHigherNumber(t *testing.T) {
	dir := shardWith(t, map[string]string{"9": fixture(t, "failing.xml"), "10": fixture(t, "green.xml")})
	now := time.Now()
	setMtime(t, dir, "9", now)
	setMtime(t, dir, "10", now.Add(-time.Hour))
	f, _ := Verdict(Shard{Dir: dir, Suites: []string{"res://test/lx/failing_test.gd"}})
	if len(f) != 1 || !strings.Contains(f[0], "test_fails") {
		t.Fatalf("report_9 should win: %q", f)
	}
}

func TestVerdictReadsAReportWithBOMAndCRLF(t *testing.T) {
	x := "\xef\xbb\xbf" + strings.ReplaceAll(fixture(t, "green.xml"), "\n", "\r\n")
	f, times := Verdict(Shard{Dir: shardWith(t, map[string]string{"1": x}),
		Suites: []string{"res://test/core/animation_throttle_test.gd", "res://test/smoke_test.gd"}})
	if len(f) != 0 || len(times) != 2 {
		t.Fatalf("got %q %v", f, times)
	}
}

func TestVerdictMatchesASuiteNamedByPathToItsPlannedPath(t *testing.T) {
	x := "<testsuites><testsuite name=\"test/lx/failing_test.gd\" time=\"1.5\"><testcase name=\"t\"><failure message=\"m\">\n why\n</failure></testcase></testsuite></testsuites>"
	f, times := Verdict(Shard{Dir: shardWith(t, map[string]string{"1": x}),
		Suites: []string{"res://test/lx/failing_test.gd"}})
	if len(f) != 1 || f[0] != "res://test/lx/failing_test.gd t: why" || times["res://test/lx/failing_test.gd"] != 1.5 {
		t.Fatalf("got %q %v", f, times)
	}
}

func TestVerdictReportsAnUnplannedSuiteByItsName(t *testing.T) {
	x := `<testsuites><testsuite name="other_test"><testcase name="t"><failure message="m"/></testcase></testsuite></testsuites>`
	f, times := Verdict(Shard{Dir: shardWith(t, map[string]string{"1": x}), Suites: []string{"res://a_test.gd"}})
	if len(f) != 2 || f[0] != "other_test t: m" || len(times) != 0 {
		t.Fatalf("got %q %v", f, times)
	}
}

func TestVerdictTellsSameNamedSuitesApartByPackage(t *testing.T) {
	x := `<testsuites>
<testsuite name="x_test" package="test/a" time="1"><testcase name="t1"></testcase></testsuite>
<testsuite name="x_test" package="test/b" time="2"><testcase name="t2"><failure message="m">bad</failure></testcase></testsuite>
</testsuites>`
	f, times := Verdict(Shard{Dir: shardWith(t, map[string]string{"1": x}),
		Suites: []string{"res://test/a/x_test.gd", "res://test/b/x_test.gd"}})
	if len(f) != 1 || f[0] != "res://test/b/x_test.gd t2: bad" ||
		times["res://test/a/x_test.gd"] != 1 || times["res://test/b/x_test.gd"] != 2 {
		t.Fatalf("got %q %v", f, times)
	}
}

func TestVerdictMatchesASuiteAtTheProjectRootWithoutPackage(t *testing.T) {
	for _, pkg := range []string{"", ` package="."`} {
		x := `<testsuites><testsuite name="root_test"` + pkg + ` time="3"><testcase name="t"><failure message="m"/></testcase></testsuite></testsuites>`
		f, times := Verdict(Shard{Dir: shardWith(t, map[string]string{"1": x}), Suites: []string{"res://root_test.gd"}})
		if len(f) != 1 || f[0] != "res://root_test.gd t: m" || times["res://root_test.gd"] != 3 {
			t.Fatalf("pkg %q: got %q %v", pkg, f, times)
		}
	}
}
