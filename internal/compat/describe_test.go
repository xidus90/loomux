package compat

import (
	"encoding/json"
	"testing"
)

func TestDescribeNamesRuntimeAndAddonSources(t *testing.T) {
	d, err := Describe("gdscript/godot-4.5/godot4-gdunit4")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(d)
	want := `{"stack":"gdscript","runtime":"godot","version":"4.5","install":"4.5","addons":[` +
		`{"name":"gdUnit4","repo":"https://github.com/godot-gdunit-labs/gdUnit4","tag":"v6.2.2","dir":"addons/gdUnit4","asset":""},` +
		`{"name":"NanoCoverage","repo":"https://github.com/IgorBayerl/nano-coverage-godot","tag":"v0.1.1","dir":"addons/nano_coverage_godot","asset":"nano_coverage_godot_v0.1.1.zip"}]}`
	if string(data) != want {
		t.Fatalf("got  %s\nwant %s", data, want)
	}
	if d, err = Describe("python/python-3.11/python-mypy"); err != nil || d.Addons == nil || len(d.Addons) != 0 {
		t.Fatalf("an entry without addons: %+v, %v", d, err)
	}
	if _, err = Describe("nope"); err == nil {
		t.Error("an unknown entry was described")
	}
}

// The installers' spellings: actions/setup-go takes Go's own "go1.21rc2"
// (README, "Native Go versions"), uv the PEP 440 "3.16.0a1" (uv python list
// --all-versions), godot-builds tags a pre-release "4.8-beta2" bare and a
// stable one "<version>-stable", which compat-setup.sh appends.
func TestDescribeInstallsThePreRelease(t *testing.T) {
	for _, c := range []struct{ runtime, version, pre, want string }{
		{"go", "1.28", "1.28rc1", "go1.28rc1"},
		{"go", "1.27", "", "1.27"},
		{"python", "3.16", "3.16.0a1", "3.16.0a1"},
		{"python", "3.15", "", "3.15"},
		{"godot", "4.8", "4.8-beta2", "4.8-beta2"},
		{"godot", "4.7", "", "4.7"},
	} {
		d, err := DescribeEntry(Entry{Runtime: c.runtime, Version: c.version, Pre: c.pre})
		if err != nil || d.Install != c.want {
			t.Errorf("%s %s %q: install %q, %v; want %q", c.runtime, c.version, c.pre, d.Install, err, c.want)
		}
	}
}

func TestEveryAddonOfTheListHasASource(t *testing.T) {
	entries, _ := Load()
	for _, e := range entries {
		if _, err := Describe(e.Key()); err != nil {
			t.Errorf("%s: %v", e.Key(), err)
		}
	}
	e := entries[0]
	e.Addons = []Addon{{Name: "Nope", Version: "1"}}
	if _, err := DescribeEntry(e); err == nil {
		t.Error("an addon without a source was described")
	}
}
