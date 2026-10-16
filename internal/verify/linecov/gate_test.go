package linecov

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func gateWorld(t *testing.T, files map[string]string, report Report, settingsReports ...string) Input {
	t.Helper()
	act, finds, _, tree := activeIn(t, files, settingsReports...)
	if len(finds) != 0 {
		t.Fatal(finds)
	}
	ps, _ := LoadProfiles()
	now := time.Now()
	return Input{
		Report: report, Tree: tree, Active: act, Natives: Natives(ps, act), Newest: now,
		Read: func(rel string) ([]byte, error) {
			if c, ok := files[rel]; ok {
				return []byte(c), nil
			}
			return nil, errors.New("gone")
		},
		ModTime: func(string) (time.Time, error) { return now, nil },
	}
}

func TestGateHoldsEveryLineToTheRule(t *testing.T) {
	src := "def f():\n" + // 1
		"    return 1\n" + // 2 hit
		"    x = 1  # coverage:exempt windows only\n" + // 3 not hit, exempt
		"    y = 2  # coverage:exempt\n" + // 4 not hit, no reason
		"    z = 3  # coverage:exempt dead\n" + // 5 hit: refuted
		"    w = 4\n" + // 6 not hit
		"    v = 5  # pragma: no cover -- debug\n" // 7 not hit, native
	in := gateWorld(t, map[string]string{".coveragerc": "[run]\nsource = .\n", "a.py": src},
		Report{"a.py": {2: 1, 3: 0, 4: 0, 5: 7, 6: 0, 7: 0}})
	want := []Finding{
		{"a.py", 4, "coverage:exempt without a reason"},
		{"a.py", 5, "coverage:exempt but hit 7 times"},
		{"a.py", 6, "not covered"},
	}
	if got := Gate(in); !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
}

func TestGateTreatsFileMarkersAndAbsence(t *testing.T) {
	files := map[string]string{
		"project.godot": "[nano_coverage]\n",
		"core/a.gd":     "func f():\n\tpass\n",
		"core/b.gd":     "# coverage:exempt-file only _init\n",
		"core/c.gd":     "# coverage:exempt-file never run\nfunc g():\n\tpass\n",
		"core/d.gd":     "func h():\n\tpass\n",
	}
	in := gateWorld(t, files, Report{"core/a.gd": {2: 1}, "core/c.gd": {3: 4}})
	want := []Finding{
		{"core/c.gd", 0, "coverage:exempt-file but 1 lines hit"},
		{"core/d.gd", 0, "is absent from the report"},
	}
	if got := Gate(in); !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
}

func TestGateSkipsExcludedRecordsAndNativeFileMarkers(t *testing.T) {
	files := map[string]string{
		".nycrc":          `{"include":["src/**"],"exclude":["src/vendor/**"]}`,
		"src/a.js":        "x();\n",
		"src/gen.js":      "/* istanbul ignore file: generated */\nx();\n",
		"src/gen2.js":     "/* istanbul ignore file: generated */\nx();\n",
		"src/vendor/v.js": "x();\n",
	}
	in := gateWorld(t, files, Report{"src/a.js": {1: 1}, "src/gen.js": {2: 0}, "src/vendor/v.js": {1: 0}})
	if got := Gate(in); len(got) != 0 {
		t.Fatalf("%v", got)
	}
}

func TestGateReportsASourceNewerThanTheReportAndOneItCannotRead(t *testing.T) {
	in := gateWorld(t, map[string]string{"a.py": "x = 1\n"}, Report{"a.py": {1: 1}, "b.py": {1: 1}})
	in.ModTime = func(rel string) (time.Time, error) {
		if rel == "a.py" {
			return in.Newest.Add(3 * time.Second), nil
		}
		return time.Time{}, errors.New("gone")
	}
	want := []Finding{{"a.py", 0, "is newer than the report"}, {"b.py", 0, "unreadable: gone"}}
	if got := Gate(in); !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
	in.ModTime = func(string) (time.Time, error) { return in.Newest.Add(time.Second), nil }
	if got := Gate(in); len(got) != 1 {
		t.Fatalf("slack not granted: %v", got)
	}
}

func TestGateIgnoresGlueUnderGodotsCacheDirectory(t *testing.T) {
	in := gateWorld(t, map[string]string{
		"project.godot":         "[nano_coverage]\n",
		"core/a.gd":             "func f():\n\tpass\n",
		".godot/loomux/hook.gd": "func g():\n\tpass\n",
	}, Report{"core/a.gd": {2: 1}})
	if got := Gate(in); len(got) != 0 {
		t.Fatalf("%v", got)
	}
}

func TestGateReportsAnExpectedSourceItCannotRead(t *testing.T) {
	in := gateWorld(t, map[string]string{"project.godot": "[nano_coverage]\n", "core/a.gd": "func f():\n\tpass\n"}, Report{})
	in.Read = func(string) ([]byte, error) { return nil, errors.New("locked") }
	want := []Finding{{"core/a.gd", 0, "unreadable: locked"}}
	if got := Gate(in); !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
}

func TestGateAcceptsAnExemptFileWhoseLinesDidNotRun(t *testing.T) {
	in := gateWorld(t, map[string]string{"a.py": "# coverage:exempt-file only on Windows\nx = 1\n"}, Report{"a.py": {2: 0}})
	if got := Gate(in); len(got) != 0 {
		t.Fatalf("%v", got)
	}
}
