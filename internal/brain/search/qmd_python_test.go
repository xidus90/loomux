package search_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/search"
)

func TestQmdPort_StderrIsStrippedLikePython(t *testing.T) {
	port := &search.QmdPort{Runner: func([]string) ([]byte, []byte, int, error) {
		return nil, []byte("\x1c fatal \x1f\n"), 1, nil
	}}
	_, err := port.Indexed("c")
	if err == nil || err.Error() != "qmd exited with 1: fatal" {
		t.Fatalf("got %v", err)
	}
}

func TestQmdPort_IndexedSplitsLinesAndKeepsPathsLikePython(t *testing.T) {
	// "\xc2\x85" is U+0085 NEXT LINE, one of the separators str.splitlines knows.
	output := "1  d  qmd://c/a b.md  \n2  d  qmd://c/x.md\xc2\x853  d  qmd://other/y.md\r\nno uri here\n"
	port := &search.QmdPort{Runner: func([]string) ([]byte, []byte, int, error) {
		return []byte(output), nil, 0, nil
	}}
	got, err := port.Indexed("c")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a b.md  ", "x.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestQmdPort_SearchGoesThroughInvoke(t *testing.T) {
	port := &search.QmdPort{Runner: func([]string) ([]byte, []byte, int, error) {
		return nil, []byte(" broken \n"), 3, nil
	}}
	_, err := port.Search("q", []string{"c"}, search.ProfileFull, 5)
	if err == nil || err.Error() != "qmd exited with 3: broken" {
		t.Fatalf("got %v", err)
	}
}

func TestQmdPort_ALauncherRefusalPassesUnchanged(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	port := &search.QmdPort{Executable: "nonexistent-qmd-xyz"}
	_, err := port.Indexed("c")
	if err == nil || err.Error() != "cannot find 'nonexistent-qmd-xyz' on PATH" {
		t.Fatalf("got %v", err)
	}
}

func TestQmdPort_AStartFailureIsCannotRun(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.exe"), []byte("echo binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	port := &search.QmdPort{Executable: "broken.exe"}
	_, err := port.NotYetSearchable()
	if err == nil || !strings.HasPrefix(err.Error(), "cannot run broken.exe: ") {
		t.Fatalf("got %v", err)
	}
}

func TestDefaultRunner_ALauncherRefusalKeepsItsWords(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, _, code, err := search.DefaultRunner([]string{"nonexistent-qmd-xyz"})
	if code != 1 || err == nil || err.Error() != "cannot find 'nonexistent-qmd-xyz' on PATH" {
		t.Fatalf("code %d, err %v", code, err)
	}
}

func TestDefaultRunner_AShimIsBypassedNotHandedToCmd(t *testing.T) {
	dir := t.TempDir()
	shim := `"%_prog%"  "%dp0%\node_modules\qmd\bin\qmd.js" %*`
	if err := os.WriteFile(filepath.Join(dir, "qmd.cmd"), []byte(shim), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "node.exe"), []byte("not a program"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	_, _, _, err := search.DefaultRunner([]string{"qmd", "ls", "c"})
	if err == nil || !strings.Contains(err.Error(), filepath.Join(dir, "node.exe")) {
		t.Fatalf("expected the start of the node beside the shim to fail, got %v", err)
	}
}
