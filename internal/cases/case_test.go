package cases_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/cases"
)

func writeCaseFile(t *testing.T, dir, name string, content []byte) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func TestLoadCase_Valid(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "guard", "allow-write")
	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "world_after"), 0o755); err != nil {
		t.Fatal(err)
	}

	writeCaseFile(t, dir, "cmd", []byte("brain guard write.txt\n"))
	writeCaseFile(t, dir, "exit", []byte("0\n"))
	writeCaseFile(t, dir, "stdout", []byte("ok\n"))
	writeCaseFile(t, dir, "stdin", []byte("data\n"))
	writeCaseFile(t, dir, "notes.md", []byte("test notes\n"))

	c, err := cases.LoadCase(dir)
	if err != nil {
		t.Fatalf("unexpected error loading valid case: %v", err)
	}

	if c.Verb != "guard" {
		t.Errorf("expected verb guard, got %s", c.Verb)
	}
	if c.Name != "allow-write" {
		t.Errorf("expected name allow-write, got %s", c.Name)
	}
	if c.Cmd != "brain guard write.txt" {
		t.Errorf("expected cmd 'brain guard write.txt', got %q", c.Cmd)
	}
	if c.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", c.ExitCode)
	}
	if string(c.Stdout) != "ok\n" {
		t.Errorf("expected stdout 'ok\\n', got %q", string(c.Stdout))
	}
	if string(c.Stdin) != "data\n" {
		t.Errorf("expected stdin 'data\\n', got %q", string(c.Stdin))
	}
	if c.Notes != "test notes\n" {
		t.Errorf("expected notes 'test notes\\n', got %q", c.Notes)
	}
	if !c.HasWorldAfter {
		t.Errorf("expected HasWorldAfter to be true")
	}
}

func TestLoadCase_MissingWorld(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "verb", "name")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeCaseFile(t, dir, "cmd", []byte("echo 1\n"))
	writeCaseFile(t, dir, "exit", []byte("0\n"))
	writeCaseFile(t, dir, "stdout", []byte("1\n"))

	_, err := cases.LoadCase(dir)
	if err == nil {
		t.Fatal("expected error for missing world directory, got nil")
	}
}

func TestLoadCase_MissingCmd(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "verb", "name")
	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeCaseFile(t, dir, "exit", []byte("0\n"))
	writeCaseFile(t, dir, "stdout", []byte("1\n"))

	_, err := cases.LoadCase(dir)
	if err == nil {
		t.Fatal("expected error for missing cmd file, got nil")
	}
}

func TestLoadCase_EmptyCmd(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "verb", "name")
	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeCaseFile(t, dir, "cmd", []byte("   \n"))
	writeCaseFile(t, dir, "exit", []byte("0\n"))
	writeCaseFile(t, dir, "stdout", []byte("1\n"))

	_, err := cases.LoadCase(dir)
	if err == nil {
		t.Fatal("expected error for empty cmd file, got nil")
	}
}

func TestLoadCase_MissingExit(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "verb", "name")
	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeCaseFile(t, dir, "cmd", []byte("echo 1\n"))
	writeCaseFile(t, dir, "stdout", []byte("1\n"))

	_, err := cases.LoadCase(dir)
	if err == nil {
		t.Fatal("expected error for missing exit file, got nil")
	}
}

func TestLoadCase_InvalidExit(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "verb", "name")
	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeCaseFile(t, dir, "cmd", []byte("echo 1\n"))
	writeCaseFile(t, dir, "exit", []byte("not-an-int\n"))
	writeCaseFile(t, dir, "stdout", []byte("1\n"))

	_, err := cases.LoadCase(dir)
	if err == nil {
		t.Fatal("expected error for invalid exit code, got nil")
	}
}

func TestLoadCase_MissingStdout(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "verb", "name")
	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeCaseFile(t, dir, "cmd", []byte("echo 1\n"))
	writeCaseFile(t, dir, "exit", []byte("0\n"))

	_, err := cases.LoadCase(dir)
	if err == nil {
		t.Fatal("expected error for missing stdout file, got nil")
	}
}

func TestDiscoverCases(t *testing.T) {
	root := t.TempDir()

	setupCase := func(verb, name string) {
		caseDir := filepath.Join(root, verb, name)
		if err := os.MkdirAll(filepath.Join(caseDir, "world"), 0o755); err != nil {
			t.Fatal(err)
		}
		writeCaseFile(t, caseDir, "cmd", []byte("echo 1\n"))
		writeCaseFile(t, caseDir, "exit", []byte("0\n"))
		writeCaseFile(t, caseDir, "stdout", []byte("1\n"))
	}

	setupCase("guard", "deny-read")
	setupCase("guard", "allow-write")
	setupCase("wiki", "lint-broken")
	writeCaseFile(t, root, "README.md", []byte("not a case\n"))

	allCases, err := cases.DiscoverCases(root, "")
	if err != nil {
		t.Fatalf("unexpected error discovering cases: %v", err)
	}
	if len(allCases) != 3 {
		t.Fatalf("expected 3 cases, got %d", len(allCases))
	}
	if allCases[0].Verb != "guard" || allCases[0].Name != "allow-write" {
		t.Errorf("expected guard/allow-write first, got %s/%s", allCases[0].Verb, allCases[0].Name)
	}
	if allCases[1].Verb != "guard" || allCases[1].Name != "deny-read" {
		t.Errorf("expected guard/deny-read second, got %s/%s", allCases[1].Verb, allCases[1].Name)
	}
	if allCases[2].Verb != "wiki" || allCases[2].Name != "lint-broken" {
		t.Errorf("expected wiki/lint-broken third, got %s/%s", allCases[2].Verb, allCases[2].Name)
	}

	guardCases, err := cases.DiscoverCases(root, "guard")
	if err != nil {
		t.Fatalf("unexpected error discovering guard cases: %v", err)
	}
	if len(guardCases) != 2 {
		t.Fatalf("expected 2 guard cases, got %d", len(guardCases))
	}
}

func TestLoadCaseDefaultsCompareToData(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "verb", "name")
	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeCaseFile(t, dir, "cmd", []byte("loomux x\n"))
	writeCaseFile(t, dir, "exit", []byte("0\n"))
	writeCaseFile(t, dir, "stdout", []byte(""))

	c, err := cases.LoadCase(dir)
	if err != nil || c.Compare != "data" {
		t.Fatalf("%v %q", err, c.Compare)
	}

	writeCaseFile(t, dir, "compare", []byte("message\n"))
	c, err = cases.LoadCase(dir)
	if err != nil || c.Compare != "message" {
		t.Fatalf("%v %q", err, c.Compare)
	}
}

func TestLoadCaseRefusesAnUnknownCompare(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "verb", "name")
	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeCaseFile(t, dir, "cmd", []byte("loomux x\n"))
	writeCaseFile(t, dir, "exit", []byte("0\n"))
	writeCaseFile(t, dir, "stdout", []byte(""))
	writeCaseFile(t, dir, "compare", []byte("sideways\n"))

	if _, err := cases.LoadCase(dir); err == nil {
		t.Fatal("want error")
	}
}

func TestDiscoverCasesReportsWhatItCannotRead(t *testing.T) {
	if _, err := cases.DiscoverCases(filepath.Join(t.TempDir(), "gone"), ""); err == nil {
		t.Fatal("want error for a missing root")
	}

	root := t.TempDir()
	broken := filepath.Join(root, "verb", "name")
	if err := os.MkdirAll(broken, 0o755); err != nil {
		t.Fatal(err)
	}
	writeCaseFile(t, broken, "cmd", []byte("loomux x\n"))
	if _, err := cases.DiscoverCases(root, ""); err == nil {
		t.Fatal("want error for a case without a world")
	}
}
