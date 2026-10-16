package godotsuite

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPlanBalancesByWeightLargestFirst(t *testing.T) {
	got := Plan([]string{"a", "b", "c", "d"}, map[string]float64{"a": 5, "b": 4, "c": 3, "d": 2}, 2)
	if want := [][]string{{"a", "d"}, {"b", "c"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestPlanWithoutTimesWeighsEverythingTheSameAndGoesByPath(t *testing.T) {
	got := Plan([]string{"d", "b", "c", "a", "e"}, nil, 2)
	if want := [][]string{{"a", "c", "e"}, {"b", "d"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestPlanGivesAMissingSuiteTheMeanOfTheKnownOnes(t *testing.T) {
	// mean of 6 and 2 is 4: x lands between a (6) and b (2).
	got := Plan([]string{"a", "b", "x"}, map[string]float64{"a": 6, "b": 2}, 2)
	if want := [][]string{{"a"}, {"x", "b"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestPlanBreaksTiesByPathAndThenByLowestShard(t *testing.T) {
	got := Plan([]string{"b", "a"}, map[string]float64{"a": 1, "b": 1}, 2)
	if want := [][]string{{"a"}, {"b"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestPlanDropsEmptyShards(t *testing.T) {
	got := Plan([]string{"a"}, nil, 4)
	if want := [][]string{{"a"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if got := Plan(nil, nil, 3); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestPlanTreatsANonPositiveCountAsOne(t *testing.T) {
	got := Plan([]string{"b", "a"}, nil, 0)
	if want := [][]string{{"a", "b"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestShardCount(t *testing.T) {
	for _, c := range []struct{ suites, cpus, want int }{
		{3, 16, 3}, {100, 4, 4}, {100, 32, 8}, {0, 4, 1}, {5, 0, 1},
	} {
		if got := ShardCount(c.suites, c.cpus); got != c.want {
			t.Errorf("ShardCount(%d, %d) = %d, want %d", c.suites, c.cpus, got, c.want)
		}
	}
}

func TestLoadTimesMissingFileIsEmptyWithoutANote(t *testing.T) {
	m, note := LoadTimes(filepath.Join(t.TempDir(), "root.json"))
	if len(m) != 0 || note != "" {
		t.Fatalf("got %v %q", m, note)
	}
}

func TestLoadTimesBrokenJSONIsEmptyWithANote(t *testing.T) {
	p := filepath.Join(t.TempDir(), "root.json")
	if err := os.WriteFile(p, []byte("{nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, note := LoadTimes(p)
	if len(m) != 0 || note != ".loomux/state/godot-suite/root.json is unreadable; planning without timings" {
		t.Fatalf("got %v %q", m, note)
	}
}

func TestLoadTimesUnreadablePathIsEmptyWithANote(t *testing.T) {
	m, note := LoadTimes(t.TempDir()) // a directory cannot be read as a file
	if len(m) != 0 || !strings.HasSuffix(note, " is unreadable; planning without timings") {
		t.Fatalf("got %v %q", m, note)
	}
}

func TestSaveTimesKeepsForeignEntriesAndOverwritesTheRunOnes(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "godot.json")
	if err := SaveTimes(p, map[string]float64{"a": 1, "b": 2}); err != nil {
		t.Fatal(err)
	}
	if err := SaveTimes(p, map[string]float64{"b": 9, "c": 3}); err != nil {
		t.Fatal(err)
	}
	m, note := LoadTimes(p)
	if want := map[string]float64{"a": 1, "b": 9, "c": 3}; !reflect.DeepEqual(m, want) || note != "" {
		t.Fatalf("got %v %q", m, note)
	}
}

func TestSaveTimesReplacesAnUnreadableFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "godot.json")
	os.WriteFile(p, []byte("garbage"), 0o644)
	if err := SaveTimes(p, map[string]float64{"a": 1}); err != nil {
		t.Fatal(err)
	}
	if m, _ := LoadTimes(p); !reflect.DeepEqual(m, map[string]float64{"a": 1}) {
		t.Fatalf("got %v", m)
	}
}

func TestSaveTimesLeavesNoTempFileBehind(t *testing.T) {
	dir := t.TempDir()
	if err := SaveTimes(filepath.Join(dir, "godot.json"), map[string]float64{"a": 1}); err != nil {
		t.Fatal(err)
	}
	if tmp, _ := filepath.Glob(filepath.Join(dir, "*.tmp")); len(tmp) != 0 {
		t.Fatalf("left %v", tmp)
	}
}

func TestALeftoverTempFileNeverChangesWhatLoadTimesReads(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "godot.json")
	if err := SaveTimes(p, map[string]float64{"a": 1}); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "123456.tmp"), []byte(`{"a": 99, "z": 1}`), 0o644)
	if m, note := LoadTimes(p); !reflect.DeepEqual(m, map[string]float64{"a": 1}) || note != "" {
		t.Fatalf("got %v %q", m, note)
	}
	if err := SaveTimes(p, map[string]float64{"b": 2}); err != nil {
		t.Fatal(err)
	}
	if m, _ := LoadTimes(p); !reflect.DeepEqual(m, map[string]float64{"a": 1, "b": 2}) {
		t.Fatalf("got %v", m)
	}
}

func TestSaveTimesFailsWhenTheDirectoryCannotBeMade(t *testing.T) {
	file := filepath.Join(t.TempDir(), "afile")
	os.WriteFile(file, nil, 0o644)
	if err := SaveTimes(filepath.Join(file, "godot.json"), map[string]float64{"a": 1}); err == nil {
		t.Fatal("want an error")
	}
}

func TestSaveTimesFailsOnAValueJSONCannotCarry(t *testing.T) {
	if err := SaveTimes(filepath.Join(t.TempDir(), "godot.json"), map[string]float64{"a": math.Inf(1)}); err == nil {
		t.Fatal("want an error")
	}
}

func TestSaveTimesFailedRenameRemovesItsTempFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "godot.json")
	if err := os.Mkdir(target, 0o755); err != nil { // rename onto a directory fails
		t.Fatal(err)
	}
	if err := SaveTimes(target, map[string]float64{"a": 1}); err == nil {
		t.Fatal("want an error")
	}
	if tmp, _ := filepath.Glob(filepath.Join(dir, "*.tmp")); len(tmp) != 0 {
		t.Fatalf("left %v", tmp)
	}
}

func TestTimesPath(t *testing.T) {
	r := filepath.Join("r", "root")
	for area, tail := range map[string]string{
		"godot": "godot.json", "a/b": "a_b.json", ".": "root.json",
	} {
		got := filepath.ToSlash(TimesPath(r, area))
		if want := "r/root/.loomux/state/godot-suite/" + tail; got != want {
			t.Errorf("TimesPath(%q) = %s, want %s", area, got, want)
		}
	}
}

func TestLoadTimesNullJSONIsAnEmptyUsableMap(t *testing.T) {
	p := filepath.Join(t.TempDir(), "root.json")
	os.WriteFile(p, []byte("null"), 0o644)
	m, note := LoadTimes(p)
	if m == nil || note == "" {
		t.Fatalf("got %v %q", m, note)
	}
	m["x"] = 1 // must not panic
}
