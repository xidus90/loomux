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
	want := `{"stack":"gdscript","runtime":"godot","version":"4.5","addons":[` +
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
