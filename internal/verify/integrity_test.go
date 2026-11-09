package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// unwritableHome is a home whose .loomux cannot be made: it is a file.
func unwritableHome(t *testing.T) string {
	t.Helper()
	home := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(home, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return home
}

func stubLowIntegrity(t *testing.T, low bool) {
	t.Helper()
	old := lowIntegrity
	t.Cleanup(func() { lowIntegrity = old })
	lowIntegrity = func() bool { return low }
}

func TestGoWorkNamesTheLowLabel(t *testing.T) {
	stubLowIntegrity(t, true)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	_, pre, note := goWork(unwritableHome(t), unset)
	if pre != StateUnready || !strings.Contains(note, "runs at Low integrity") || !strings.Contains(note, exe) {
		t.Fatalf("pre %q note %q", pre, note)
	}
}

func TestGoWorkKeepsTheSandboxHintAtMedium(t *testing.T) {
	stubLowIntegrity(t, false)
	_, _, note := goWork(unwritableHome(t), unset)
	if !strings.Contains(note, "grant it in the sandbox or set GOTMPDIR") || strings.Contains(note, "Low integrity") {
		t.Fatalf("note %q", note)
	}
}

// The test runs as an ordinary process, at Medium on Windows and with no
// integrity levels elsewhere.
func TestIsLowIntegrityIsFalseForThisProcess(t *testing.T) {
	if isLowIntegrity() {
		t.Fatal("a test process is not Low")
	}
}
