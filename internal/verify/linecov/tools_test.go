package linecov

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract/ecmascript"
)

func TestLoadProfilesReadsTheBuiltInTools(t *testing.T) {
	ps, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, p := range ps {
		names = append(names, p.Name)
	}
	want := []string{"c8", "coveragepy", "coverlet", "gcovr", "jacoco", "kover", "lcov", "nanocoverage", "nyc", "phpunit", "rust", "vitest"}
	if !slices.Equal(names, want) {
		t.Fatalf("%q", names)
	}
}

// The typescript coverage lane reads vitest's lcov report: the profile knows
// where it lies, and its v8 markers hold in every file the graph reads as
// JavaScript or TypeScript.
func TestTheVitestProfileReadsTheTypescriptLanesReport(t *testing.T) {
	ps, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(ps, func(p Profile) bool { return p.Name == "vitest" })
	if !slices.Contains(ps[i].Reports, "coverage/lcov.info") {
		t.Errorf("reports %q", ps[i].Reports)
	}
	for _, ext := range append(ecmascript.JavaScript{}.Extensions(), ecmascript.TypeScript{}.Extensions()...) {
		if !slices.Contains(ps[i].Extensions, ext) {
			t.Errorf("%s is not vitest's", ext)
		}
	}
}

func TestParseProfilesRefusesWhatWouldMisleadTheGate(t *testing.T) {
	for _, row := range []struct{ doc, want string }{
		{"[tool.a]\nextension = [\".x\"]\n", "unknown key"},
		{"[tool.a]\n", "no extensions"},
		{"[tool.a]\nextensions=[\".x\"]\nscope=\"some\"\n", "unknown scope"},
		{"[tool.a]\nextensions=[\".x\"]\npatterns=\"fnmatch\"\n", "unknown patterns dialect"},
		{"[tool.a]\nextensions=[\".x\"]\nexclude={globs=[\"omit\"]}\n", "globs need a patterns dialect"},
		{"[tool.a]\nextensions=[\".x\"]\nexclude={always=[\"x/**\"]}\n", "globs need a patterns dialect"},
		{"[tool.a]\nextensions=[\".x\"]\nsettings=[{file=\"f\",syntax=\"yaml\"}]\n", "unknown syntax"},
		{"[tool.a]\nextensions=[\".x\"]\nsettings=[{file=\"f\",syntax=\"xml\"}]\n", "xml needs a path"},
		{"[tool.a]\nextensions=[\".x\"]\nmarkers=[{pattern=\"x\"}]\n", "needs a label and a pattern"},
		{"[tool.a]\nextensions=[\".x\"]\nmarkers=[{label=\"x\"}]\n", "needs a label and a pattern"},
		{"[tool.a]\nextensions=[\".x\"]\nmarkers=[{label=\"x\",pattern=\"x\",key=\"k\"}]\n", "takes no key or template"},
		{"[tool.a]\nextensions=[\".x\"]\nmarkers=[{label=\"x\",pattern=\"x\",template=\"{}\"}]\n", "takes no key or template"},
		{"[tool.a]\nextensions=[\".x\"]\nmarker_keys=[{key=\"k\",pattern=\"x\"}]\n", "takes no pattern"},
		{"[tool.a]\nextensions=[\".x\"]\nmarkers=[{label=\"x\",pattern=\"(\"}]\n", "marker x"},
		{"[tool.a]\nextensions=[\".x\"]\nmarkers=[{label=\"x\",pattern=\"x\",block=\"middle\"}]\n", "unknown block"},
		{"[tool.a]\nextensions=[\".x\"]\nmarker_keys=[{template=\"{}\"}]\n", "needs a key"},
		{"[tool.a]\nextensions=[\".x\"]\nmarker_keys=[{key=\"k\",block=\"middle\"}]\n", "unknown block"},
		{"[tool.a\n", "tools.toml"},
	} {
		if _, err := parseProfiles(row.doc); err == nil || !strings.Contains(err.Error(), row.want) {
			t.Errorf("%q: %v", row.doc, err)
		}
	}
}

func TestMarkerSpecCompilesALiteralAKeyValueAndATemplate(t *testing.T) {
	for _, row := range []struct {
		spec      MarkerSpec
		value     string
		label, re string
	}{
		{MarkerSpec{Label: "pragma: no cover", Pattern: `pragma:\s*no cover`}, "", "pragma: no cover", `pragma:\s*no cover`},
		{MarkerSpec{Key: "lcov_excl_line"}, "MY_EXCL", "MY_EXCL", "MY_EXCL"},
		{MarkerSpec{Key: "exclude-pattern-prefix", Template: `\b{}_EXCL_LINE\b`}, "P.X", "P.X_EXCL_LINE", `\bP\.X_EXCL_LINE\b`},
	} {
		m, err := row.spec.compile(row.value)
		if err != nil || m.Label != row.label || m.Re.String() != row.re {
			t.Errorf("%+v %q: %+v %v", row.spec, row.value, m, err)
		}
	}
	if _, err := (MarkerSpec{Key: "k"}).compile("("); err == nil {
		t.Fatal("a broken configured marker compiled")
	}
}

func TestBuiltInMarkersMatchWhatTheToolHonours(t *testing.T) {
	ps, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	hits := func(tool, line string) []string {
		var labels []string
		for _, p := range ps {
			if p.Name != tool {
				continue
			}
			for _, s := range p.Markers {
				m, err := s.compile("")
				if err != nil {
					t.Fatal(err)
				}
				if m.Re.MatchString(line) {
					labels = append(labels, m.Label)
				}
			}
		}
		return labels
	}
	for _, row := range []struct {
		tool, line string
		want       []string
	}{
		{"phpunit", "// @codeCoverageIgnore", []string{"@codeCoverageIgnore"}},
		{"phpunit", "/* @codeCoverageIgnore */", []string{"@codeCoverageIgnore"}},
		{"phpunit", "// @codeCoverageIgnore because x", nil},
		{"phpunit", "// @codeCoverageIgnoreStart", []string{"@codeCoverageIgnoreStart"}},
		{"phpunit", "// @codeCoverageIgnoreEnd", []string{"@codeCoverageIgnoreEnd"}},
		{"rust", "#[coverage(off)]", []string{"coverage(off)"}},
		{"rust", "#![coverage(off)]", []string{"coverage(off)"}},
		{"rust", "#![cfg_attr(coverage_nightly, coverage(off))]", []string{"coverage(off)"}},
		{"rust", "// LCOV_EXCL_START: platform glue", []string{"LCOV_EXCL_START"}},
		{"rust", "// LCOV_EXCL_STOP", []string{"LCOV_EXCL_STOP"}},
		{"gcovr", "x; // LCOVR_EXCL_LINE", []string{"GCOVR_EXCL_LINE"}},
	} {
		if got := hits(row.tool, row.line); !slices.Equal(got, row.want) {
			t.Errorf("%s %q: %q", row.tool, row.line, got)
		}
	}
}

func TestLoadProfilesHandsOutACopy(t *testing.T) {
	a, _ := LoadProfiles()
	a[0].Name = "changed"
	b, _ := LoadProfiles()
	if b[0].Name != "c8" {
		t.Fatal("the cached profiles were changed through a returned slice")
	}
}

func TestEachSampleLineMatchesExactlyOneBuiltInMarker(t *testing.T) {
	ps, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		"// LCOV_EXCL_LINE", "// GCOVR_EXCL_LINE", "// GCOV_EXCL_START", "// LCOVR_EXCL_STOP",
		"# pragma: no cover", "/* istanbul ignore next */", "/* c8 ignore next */",
		"/* v8 ignore next */", "// @codeCoverageIgnore",
	} {
		var hits []string
		for _, p := range ps {
			for _, s := range p.Markers {
				m, err := s.compile("")
				if err != nil {
					t.Fatal(err)
				}
				if m.Re.MatchString(line) {
					hits = append(hits, p.Name+":"+m.Label)
				}
			}
		}
		if len(hits) != 1 {
			t.Errorf("%q: %q", line, hits)
		}
	}
}

func TestNanoProfileNamesItsMarker(t *testing.T) {
	ps, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range ps {
		want := ""
		if p.Name == "nanocoverage" {
			want = "# __NANO_COVERAGE_INSTRUMENTED__"
		}
		if p.Instrumented != want {
			t.Fatalf("%s: instrumented %q, want %q", p.Name, p.Instrumented, want)
		}
	}
}

// cargo-llvm-cov writes its lcov report to the path the coverage lane gives
// it; the profile knows lcov.info, so the report is found without --report.
func TestTheRustProfileKnowsTheLcovInfoReport(t *testing.T) {
	ps, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(ps, func(p Profile) bool { return p.Name == "rust" })
	if !slices.Equal(ps[i].Reports, []string{"lcov.info"}) {
		t.Errorf("reports %q", ps[i].Reports)
	}
}

// rustFixture copies the Rust fixture project into a temp dir and writes its
// report there as lcov.info, with every SF: path behind prefix(root).
func rustFixture(t *testing.T, prefix func(root string) string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS(filepath.Join("testdata", "tools", "rust"))); err != nil {
		t.Fatal(err)
	}
	tree, err := ScanTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	report, err := os.ReadFile(filepath.Join(dir, "report.info"))
	if err != nil {
		t.Fatal(err)
	}
	out := strings.ReplaceAll(string(report), "SF:src/", "SF:"+prefix(filepath.ToSlash(tree.Root))+"src/")
	if err := os.WriteFile(filepath.Join(dir, "lcov.info"), []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func rustWant(t *testing.T) []string {
	t.Helper()
	w, err := os.ReadFile(filepath.Join("testdata", "tools", "rust", "want.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(w)), "\n")
}

func findingTexts(fs []Finding) []string {
	var got []string
	for _, f := range fs {
		got = append(got, f.String())
	}
	return got
}

// cargo-llvm-cov may spell SF: as an absolute path; one below the root
// reads as the repo path, so the gate sees the same findings as with a
// relative one.
func TestAnAbsoluteSFPathBelowTheRootReadsAsTheRepoPath(t *testing.T) {
	dir := rustFixture(t, func(root string) string { return root + "/" })
	res, err := Check(Options{Root: dir, Reports: []string{filepath.Join(dir, "lcov.info")}})
	if err != nil {
		t.Fatal(err)
	}
	if got := findingTexts(res.Findings); !slices.Equal(got, rustWant(t)) {
		t.Errorf("findings %q, want %q", got, rustWant(t))
	}
	// No record is skipped; the one note says rust has no scope of its own.
	if want := []string{"rust: no scope in its settings; files absent from the report are not checked"}; !slices.Equal(res.Notes, want) {
		t.Errorf("notes %q", res.Notes)
	}
}

// A record outside the root stays unmatched: it is skipped with a note and
// adds no finding of its own.
func TestAnAbsoluteSFPathOutsideTheRootStaysUnmatched(t *testing.T) {
	dir := rustFixture(t, func(string) string { return "/elsewhere/checkout/" })
	res, err := Check(Options{Root: dir, Reports: []string{filepath.Join(dir, "lcov.info")}})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"the reports hold no line records"}
	if got := findingTexts(res.Findings); !slices.Equal(got, want) {
		t.Errorf("findings %q, want %q", got, want)
	}
	if !slices.Contains(res.Notes, "1 records outside the root were skipped") {
		t.Errorf("notes %q", res.Notes)
	}
}

// The coverage lane runs the gate on lcov.info in the root: with no --report
// the profile's report location finds it and activates the rust profile.
func TestTheRustReportAtItsDefaultLocationIsFoundAndChecked(t *testing.T) {
	dir := rustFixture(t, func(string) string { return "" })
	res, err := Check(Options{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := findingTexts(res.Findings); !slices.Equal(got, rustWant(t)) {
		t.Errorf("findings %q, want %q", got, rustWant(t))
	}
}
