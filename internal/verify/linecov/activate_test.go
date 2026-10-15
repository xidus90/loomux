package linecov

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func activeIn(t *testing.T, files map[string]string, reports ...string) ([]Active, []Finding, []string, *Tree) {
	t.Helper()
	tree, err := ScanTree(writeTree(t, files))
	if err != nil {
		t.Fatal(err)
	}
	ps, err := LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	act, finds, notes := Activate(tree, reports, ps)
	return act, finds, notes, tree
}

func names(act []Active) []string {
	var out []string
	for _, a := range act {
		out = append(out, a.Profile.Name)
	}
	return out
}

func TestActivateReadsCoveragePyScopeFromTheFirstFileThatConfiguresIt(t *testing.T) {
	act, finds, _, _ := activeIn(t, map[string]string{
		// setup.cfg comes first in the lookup order but holds no coverage
		// section, so the later pyproject.toml is read.
		"setup.cfg":      "[metadata]\nname = x\n",
		"pyproject.toml": "[tool.coverage.run]\nsource = [\"src\"]\nomit = [\"src/gen/*\"]\n",
		"src/a.py":       "", "src/gen/b.py": "", "tools/c.py": "",
	})
	if len(finds) != 0 || !slices.Equal(names(act), []string{"coveragepy"}) || act[0].Settings != "pyproject.toml" {
		t.Fatalf("%v %v", names(act), finds)
	}
	a := act[0]
	for rel, want := range map[string]bool{"src/a.py": true, "src/gen/b.py": false, "tools/c.py": false, "src/a.txt": false} {
		if a.Expects(rel) != want {
			t.Errorf("%s: %v", rel, !want)
		}
	}
}

func TestActivateAppliesTheDefaultExcludesOfTheJavaScriptTools(t *testing.T) {
	for _, settings := range []string{".nycrc", ".c8rc"} {
		act, _, _, _ := activeIn(t, map[string]string{settings: `{"include": ["src/**"]}`})
		for rel, want := range map[string]bool{
			"src/a.js": true, "src/a.test.js": false, "src/a-test.ts": false, "src/types.d.ts": false,
			"src/__tests__/b.js": false, "src/node_modules/x/i.js": false, "src/jest.config.ts": false,
		} {
			if act[0].Expects(rel) != want {
				t.Errorf("%s %s: %v", settings, rel, !want)
			}
		}
	}
	// A configured exclude replaces the defaults; node_modules has its own switch.
	for settings, key := range map[string]string{".nycrc": "excludeNodeModules", ".c8rc": "exclude-node-modules"} {
		act, _, _, _ := activeIn(t, map[string]string{settings: `{"include": ["src/**"], "exclude": ["src/gen/**"], "` + key + `": false}`})
		for rel, want := range map[string]bool{"src/a.test.js": true, "src/node_modules/x/i.js": true, "src/gen/a.js": false} {
			if act[0].Expects(rel) != want {
				t.Errorf("configured %s %s: %v", settings, rel, !want)
			}
		}
	}
}

func TestActivateLeavesOutWhatCoveragePyNeverMeasures(t *testing.T) {
	act, _, _, _ := activeIn(t, map[string]string{".coveragerc": "[run]\nsource = .\n"})
	for rel, want := range map[string]bool{"a.py": true, ".venv/lib/site-packages/x.py": false, ".venv/Lib/site-packages/y/z.py": false, ".venv/tool.py": true} {
		if act[0].Expects(rel) != want {
			t.Errorf("%s: %v", rel, !want)
		}
	}
}

func TestActivateTakesNanoCoverageDefaultsAndSwitches(t *testing.T) {
	act, _, _, _ := activeIn(t, map[string]string{
		"project.godot": "[nano_coverage]\ngeneral/ignore_paths=PackedStringArray(\"**/*_test.gd\", \"res://harness/**\")\n",
		"core/a.gd":     "", "harness/p.gd": "", "addons/x/y.gd": "", "test/a_test.gd": "",
	})
	a := act[0]
	if !a.Scoped() || a.Settings != "project.godot" {
		t.Fatalf("%+v", a)
	}
	for rel, want := range map[string]bool{"core/a.gd": true, "harness/p.gd": false, "addons/x/y.gd": false, "test/a_test.gd": false} {
		if a.Expects(rel) != want {
			t.Errorf("%s: %v", rel, !want)
		}
	}
	off, _, _, _ := activeIn(t, map[string]string{"project.godot": "[nano_coverage]\ngeneral/ignore_addons=false\n", "addons/x/y.gd": "", "a_test.gd": ""})
	if !off[0].Expects("addons/x/y.gd") || off[0].Expects("a_test.gd") {
		t.Fatal("switch off or defaults not applied")
	}
}

func TestActivateFindsAToolByItsReportAloneAndSaysWhenItHasNoScope(t *testing.T) {
	act, _, _, _ := activeIn(t, map[string]string{"coverage/lcov.info": "", "src/a.ts": ""}, "coverage/lcov.info")
	if !slices.Equal(names(act), []string{"c8", "nyc", "vitest"}) {
		t.Fatalf("%v", names(act))
	}
	for _, a := range act {
		if a.Scoped() || a.Expects("src/a.ts") || !a.Owns("src/a.ts") {
			t.Errorf("%s: scoped without settings", a.Profile.Name)
		}
	}
}

func TestActivateReportsUnreadableSettingsPatternsAndAmbiguousFiles(t *testing.T) {
	_, finds, _, _ := activeIn(t, map[string]string{".coveragerc": "[run\n"})
	if len(finds) != 1 || finds[0].Path != ".coveragerc" || !strings.HasPrefix(finds[0].Text, "unreadable settings:") {
		t.Fatalf("%v", finds)
	}
	_, finds, _, _ = activeIn(t, map[string]string{"gcovr.cfg": "filter = (\n"})
	if len(finds) != 1 || !strings.Contains(finds[0].Text, `unreadable pattern "("`) {
		t.Fatalf("%v", finds)
	}
	_, finds, _, _ = activeIn(t, map[string]string{".lcovrc": "lcov_excl_line = (\n"})
	if len(finds) != 1 || !strings.Contains(finds[0].Text, "unreadable marker") {
		t.Fatalf("%v", finds)
	}
	act, _, notes, _ := activeIn(t, map[string]string{"a.runsettings": "", "b.runsettings": ""})
	if len(act) != 0 || len(notes) != 1 || !strings.Contains(notes[0], "2 files match *.runsettings") {
		t.Fatalf("%v %q", names(act), notes)
	}
}

func TestActivateReadsDirectoriesAndConfiguredMarkers(t *testing.T) {
	act, finds, _, _ := activeIn(t, map[string]string{
		"phpunit.xml": `<phpunit><source><include><directory suffix=".php">src</directory></include><exclude><directory>src/Legacy</directory></exclude></source></phpunit>`,
		"gcovr.cfg":   "exclude-pattern-prefix = MYPROJ\n",
	})
	if len(finds) != 0 {
		t.Fatal(finds)
	}
	var php, gcovr Active
	for _, a := range act {
		switch a.Profile.Name {
		case "phpunit":
			php = a
		case "gcovr":
			gcovr = a
		}
	}
	if !php.Expects("src/A.php") || php.Expects("src/Legacy/B.php") || php.Expects("lib/C.php") {
		t.Fatal("phpunit scope")
	}
	ps, _ := LoadProfiles()
	var labels []string
	for _, m := range Natives(ps, []Active{gcovr, gcovr}) {
		labels = append(labels, m.Label)
	}
	for _, want := range []string{"MYPROJ_EXCL_LINE", "MYPROJ_EXCL_START", "MYPROJ_EXCL_STOP", "LCOV_EXCL_LINE", "pragma: no cover"} {
		if !slices.Contains(labels, want) {
			t.Errorf("missing %s in %q", want, labels)
		}
	}
	if len(labels) != len(slices.Compact(slices.Sorted(slices.Values(labels)))) {
		t.Errorf("duplicate markers: %q", labels)
	}
}

// Glob characters in the root's own name are no pattern.
func TestActivateFindsSettingsUnderARootNamedLikeAPattern(t *testing.T) {
	root := filepath.Join(writeTree(t, map[string]string{"[client]/.coveragerc": "[run]\nsource = src\n"}), "[client]")
	tree, err := ScanTree(root)
	if err != nil {
		t.Fatal(err)
	}
	ps, _ := LoadProfiles()
	act, _, _ := Activate(tree, nil, ps)
	if !slices.Equal(names(act), []string{"coveragepy"}) || act[0].Settings != ".coveragerc" {
		t.Fatalf("%v", names(act))
	}
}

// Each marker carries the extensions of the profile that brings it, so one
// pattern from two profiles stays twice.
func TestNativesCarryTheirProfilesExtensions(t *testing.T) {
	act, _, _, _ := activeIn(t, map[string]string{"gcovr.cfg": "exclude-pattern-prefix = MYPROJ\n"})
	ps, _ := LoadProfiles()
	seen := 0
	for _, m := range Natives(ps, act) {
		want := map[string][]string{"MYPROJ_EXCL_LINE": act[0].Profile.Extensions, "pragma: no cover": {".py"}, "coverage(off)": {".rs"}}[m.Label]
		if want == nil {
			continue
		}
		seen++
		if !slices.Equal(m.Extensions, want) {
			t.Errorf("%s: %q", m.Label, m.Extensions)
		}
	}
	if seen != 3 {
		t.Fatalf("saw %d of 3 markers", seen)
	}
	two := []Profile{
		{Name: "a", Extensions: []string{".a"}, Markers: []MarkerSpec{{Label: "M", Pattern: "M"}}},
		{Name: "b", Extensions: []string{".b"}, Markers: []MarkerSpec{{Label: "M", Pattern: "M"}}},
	}
	if got := Natives(two, nil); len(got) != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestActivateReportsASettingsFileItCannotRead(t *testing.T) {
	// project.godot is a directory here: it matches the name, it cannot be read.
	_, finds, _, _ := activeIn(t, map[string]string{"project.godot/x": ""})
	if len(finds) != 1 || finds[0].Path != "project.godot" || !strings.HasPrefix(finds[0].Text, "unreadable settings:") {
		t.Fatalf("%v", finds)
	}
}
