package runs_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/flow/runs"
)

func marker() runs.Marker {
	return runs.Marker{
		Flow:     "spec-to-board",
		Options:  map[string]string{"zeta": "x\ny", "topic": "a b"},
		Baseline: &runs.Baseline{Commit: "abc", Dirty: []string{"b.txt", "a.txt"}},
		Version:  "0.1.0",
	}
}

func TestWriteMarkerSpellsTheFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs", "0001.flow")
	if err := runs.WriteMarker(path, marker()); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := `spec-to-board
baseline="a.txt\nb.txt"
baseline_commit="abc"
loomux_version="0.1.0"
topic="a b"
zeta="x\ny"
`
	if string(got) != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestAMarkerCarriesOriginOverlaysAndVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "0001.flow")
	in := runs.Marker{
		Flow: "dev-cycle", Origin: "bundled+overlay",
		Overlays: []string{"questions/approve.md", "instructions/review.md"},
		Options:  map[string]string{"max_rounds": "3"},
		Baseline: &runs.Baseline{Commit: "abc", Dirty: []string{"b.go", "a.go"}},
		Version:  "3.3.0",
	}
	if err := runs.WriteMarker(path, in); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	want := "dev-cycle\n" +
		"baseline=\"a.go\\nb.go\"\n" +
		"baseline_commit=\"abc\"\n" +
		"loomux_version=\"3.3.0\"\n" +
		"max_rounds=\"3\"\n" +
		"origin=\"bundled+overlay\"\n" +
		"overlays=\"instructions/review.md\\nquestions/approve.md\"\n"
	if string(raw) != want {
		t.Fatalf("got %q\nwant %q", raw, want)
	}
	out, err := runs.ReadMarker(path)
	if err != nil {
		t.Fatal(err)
	}
	if out.Origin != "bundled+overlay" || out.Version != "3.3.0" ||
		!reflect.DeepEqual(out.Overlays, []string{"instructions/review.md", "questions/approve.md"}) ||
		!reflect.DeepEqual(out.Options, map[string]string{"max_rounds": "3"}) {
		t.Fatalf("read back %+v", out)
	}
}

func TestAnOptionMayNotTakeAMarkerName(t *testing.T) {
	for _, name := range []string{"baseline", "baseline_commit", "origin", "overlays", "loomux_version"} {
		err := runs.WriteMarker(filepath.Join(t.TempDir(), "x.flow"), runs.Marker{Flow: "f", Options: map[string]string{name: "1"}})
		if err == nil || !strings.Contains(err.Error(), "reserved") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestAWrittenMarkerReadsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "0001.flow")
	if err := runs.WriteMarker(path, marker()); err != nil {
		t.Fatal(err)
	}
	got, err := runs.ReadMarker(path)
	if err != nil {
		t.Fatal(err)
	}
	want := marker()
	want.Baseline.Dirty = []string{"a.txt", "b.txt"}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("got  %+v\nwant %+v", *got, want)
	}
}

func TestWriteMarkerRefuses(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	touch(t, blocker)
	reserved := marker()
	reserved.Options = map[string]string{"origin": "project"}
	nameless := marker()
	nameless.Flow = ""
	cases := []struct {
		name   string
		path   string
		marker runs.Marker
		want   string
	}{
		{"reserved option", filepath.Join(dir, "a.flow"), reserved, `option "origin" is reserved for the marker itself`},
		{"no flow", filepath.Join(dir, "b.flow"), nameless, "a marker needs the name of its flow"},
		{"parent is a file", filepath.Join(blocker, "c.flow"), marker(), "creating"},
		{"path is a directory", dir, marker(), "writing"},
	}
	for _, c := range cases {
		err := runs.WriteMarker(c.path, c.marker)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: want an error containing %q, got %v", c.name, c.want, err)
		}
	}
}

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "0001.flow")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadMarkerOfAMissingFileIsNothing(t *testing.T) {
	got, err := runs.ReadMarker(filepath.Join(t.TempDir(), "0001.flow"))
	if err != nil || got != nil {
		t.Fatalf("got %+v, %v", got, err)
	}
}

// A marker saved by a Windows editor carries CRLF line ends, and reads as if
// it had none.
func TestReadMarkerReadsCRLF(t *testing.T) {
	got, err := runs.ReadMarker(write(t, "verify-until-green\r\nchecks=\"ruff pytest\"\r\nmax_rounds=\"123\"\r\n\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := runs.Marker{Flow: "verify-until-green", Options: map[string]string{"checks": "ruff pytest", "max_rounds": "123"}}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("got %+v, want %+v", *got, want)
	}
}

// The commit decides alone: a path set without a commit was written before the
// commit existed and is no baseline; a commit without a path set is one with no
// dirty files.
func TestReadMarkerTakesTheBaselineFromItsCommit(t *testing.T) {
	onlyDirty, err := runs.ReadMarker(write(t, "f\nbaseline=\"a.txt\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if onlyDirty.Baseline != nil || len(onlyDirty.Options) != 0 {
		t.Fatalf("a path set without a commit is no baseline: %+v", onlyDirty)
	}
	onlyCommit, err := runs.ReadMarker(write(t, "f\nbaseline_commit=\"abc\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if onlyCommit.Baseline == nil || onlyCommit.Baseline.Commit != "abc" || len(onlyCommit.Baseline.Dirty) != 0 {
		t.Fatalf("a commit alone is a baseline without dirty files: %+v", onlyCommit.Baseline)
	}
}

func TestReadMarkerRefuses(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"empty", "", "says nothing -- not even which flow it belongs to"},
		{"blank first line", "  \nx=\"1\"\n", "says nothing"},
		{"line without =", "f\nnot an option\n", `option line without '=': "not an option"`},
		{"bare value", "f\nchecks=ruff pytest\n", `option "checks" is not a JSON string`},
		{"value cut mid-JSON", "f\nbaseline_commit=\"ab\n", `option "baseline_commit" is not a JSON string`},
		{"value that is no string", "f\nmax_rounds=123\n", `option "max_rounds" is not a JSON string`},
		{"null value", "f\norigin=null\n", `option "origin" is not a JSON string`},
	}
	for _, c := range cases {
		path := write(t, c.body)
		_, err := runs.ReadMarker(path)
		if err == nil || !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), path) {
			t.Errorf("%s: want an error containing %q, got %v", c.name, c.want, err)
		}
	}
	_, err := runs.ReadMarker(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "reading") {
		t.Errorf("a directory: got %v", err)
	}
}

// A marker is claimed, never overwritten: two runs that computed the same
// number must not share one journal, and the second finds out here.
func TestWriteMarkerRefusesAMarkerThatExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "0001.flow")
	if err := runs.WriteMarker(path, marker()); err != nil {
		t.Fatal(err)
	}
	err := runs.WriteMarker(path, marker())
	if !errors.Is(err, fs.ErrExist) || !strings.Contains(err.Error(), "writing") {
		t.Fatalf("err = %v", err)
	}
}

// A run outside a repository has no baseline, and its marker carries neither
// baseline line.
func TestAMarkerWithoutABaselineHasNoBaselineLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "0001.flow")
	without := marker()
	without.Baseline = nil
	if err := runs.WriteMarker(path, without); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "spec-to-board\nloomux_version=\"0.1.0\"\ntopic=\"a b\"\nzeta=\"x\\ny\"\n"
	if string(got) != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	back, err := runs.ReadMarker(path)
	if err != nil || back.Baseline != nil {
		t.Fatalf("baseline = %+v, err = %v", back.Baseline, err)
	}
}

func TestClaimNumbersRunsInOrder(t *testing.T) {
	root := t.TempDir()
	first, err := runs.Claim(root, marker())
	if err != nil || first != "0001" {
		t.Fatalf("first = %q, err = %v", first, err)
	}
	second, err := runs.Claim(root, marker())
	if err != nil || second != "0002" {
		t.Fatalf("second = %q, err = %v", second, err)
	}
	if _, err := os.Stat(runs.MarkerPath(root, "0002")); err != nil {
		t.Fatal(err)
	}
}

// A refusal that is not a taken number is not tried again.
func TestClaimPassesOnWhatWriteMarkerRefuses(t *testing.T) {
	nameless := marker()
	nameless.Flow = ""
	_, err := runs.Claim(t.TempDir(), nameless)
	if err == nil || !strings.Contains(err.Error(), "a marker needs the name of its flow") {
		t.Fatalf("err = %v", err)
	}
}
