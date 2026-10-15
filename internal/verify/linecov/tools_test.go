package linecov

import (
	"slices"
	"strings"
	"testing"
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
