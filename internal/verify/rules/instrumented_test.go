package rules

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/verify/linecov"
)

const instrumentedLine = "# __NANO_COVERAGE_INSTRUMENTED__"

func nanoProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "project.godot"), []byte("[nano_coverage]\n\ngeneral/x=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestInstrumented(t *testing.T) {
	const tail = "godot/core/a.gd still holds nanocoverage instrumentation; restore the sources before committing"
	for name, row := range map[string]struct {
		content string
		hit     bool
	}{
		"line one":      {instrumentedLine + "\nextends Node\n", true},
		"carriage ret":  {instrumentedLine + "\r\nextends Node\r\n", true},
		"bom":           {"\xef\xbb\xbf" + instrumentedLine + "\n", true},
		"leading space": {"  \t" + instrumentedLine + "\n", true},
		"line two":      {"extends Node\n" + instrumentedLine + "\n", false},
		"mentioned":     {"# see " + instrumentedLine + "\n", false},
		"longer":        {instrumentedLine + "x\n", false},
	} {
		t.Run(name, func(t *testing.T) {
			files := []File{{"godot/core/a.gd", []byte(row.content)}, {"godot/core/a.py", []byte(instrumentedLine + "\n")}}
			got, err := Instrumented(files, nanoProject(t))
			if err != nil {
				t.Fatal(err)
			}
			if !row.hit {
				if len(got) != 0 {
					t.Fatalf("got %v", got)
				}
				return
			}
			if len(got) != 1 || got[0].String() != tail {
				t.Fatalf("got %v", got)
			}
		})
	}
}

func TestInstrumentedNeedsADetectedProfile(t *testing.T) {
	got, err := Instrumented([]File{{"a.gd", []byte(instrumentedLine + "\n")}}, t.TempDir())
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestInstrumentedReportsEveryFileInOrder(t *testing.T) {
	files := []File{{"b.gd", []byte(instrumentedLine)}, {"a.gd", []byte(instrumentedLine)}, {"c.gd", []byte("extends Node")}}
	got, err := Instrumented(files, nanoProject(t))
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, f := range got {
		paths = append(paths, f.Path)
	}
	if !slices.Equal(paths, []string{"a.gd", "b.gd"}) {
		t.Fatalf("got %v", paths)
	}
}

// A detected tool without a marker of its own never matches a blank line.
func TestInstrumentedSkipsAToolWithoutAMarker(t *testing.T) {
	root := nanoProject(t)
	if err := os.WriteFile(filepath.Join(root, ".coveragerc"), []byte("[run]\nbranch = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Instrumented([]File{{"a.py", []byte("\nx = 1\n")}, {"a.gd", []byte(instrumentedLine)}}, root)
	if err != nil || len(got) != 1 || got[0].Path != "a.gd" {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestInstrumentedReportsAProfileLoadError(t *testing.T) {
	old := instrumentationProfiles
	instrumentationProfiles = func() ([]linecov.Profile, error) { return nil, errors.New("boom") }
	t.Cleanup(func() { instrumentationProfiles = old })
	if _, err := Instrumented(nil, nanoProject(t)); err == nil {
		t.Fatal("want an error")
	}
}

func TestInstrumentedFailsOnUnreadableSettings(t *testing.T) {
	root := nanoProject(t)
	if err := os.WriteFile(filepath.Join(root, "project.godot"), []byte("[nano_coverage\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Instrumented([]File{{"a.gd", []byte(instrumentedLine)}}, root)
	if err == nil || !strings.Contains(err.Error(), "project.godot") || !strings.Contains(err.Error(), "unclosed section header") {
		t.Fatalf("got %v", err)
	}
}

func TestInstrumentedReportsAScanError(t *testing.T) {
	if _, err := Instrumented(nil, filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("want an error")
	}
}
