package mutants

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// probeModule writes a module of its own into a fresh directory: one package p
// whose only test asks Sign(1). It skips where no go command is on PATH.
func probeModule(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go binary on PATH")
	}
	root := t.TempDir()
	writePackage(t, root, ".", map[string]string{"go.mod": "module example.com/probe\n\ngo 1.25.0\n"})
	writePackage(t, root, "p", map[string]string{
		"p.go":      signSource,
		"p_test.go": "package p\n\nimport \"testing\"\n\nfunc TestSign(t *testing.T) {\n\tif Sign(1) != 1 {\n\t\tt.Fatal(\"Sign(1) is not 1\")\n\t}\n}\n",
	})
	return root
}

// The one test that starts the real go command: it holds the overlay file, the
// package pattern and classify to what go test does with them.
func TestGoTestRunsARoundThroughOverlays(t *testing.T) {
	root := probeModule(t)
	var out strings.Builder
	sum, err := Round(Options{Packages: []string{"p"}, Root: root, Workers: 2}, GoTest(root), &out)
	if err != nil {
		t.Fatal(err)
	}
	want := "[1/4] SURVIVED  (a1) p.go:4  if a > 0 {  ->  if true {\n" +
		"[2/4] killed    (a1) p.go:4  if a > 0 {  ->  if false {\n" +
		"[3/4] killed    (a4) p.go:4  if a > 0 {  ->  if !(a > 0) {\n" +
		"[4/4] SURVIVED  (a3) p.go:4  if a > 0 {  ->  if a >= 0 {\n" +
		"\n" +
		"4 mutants over p, oracle go\n" +
		"0 do not compile and are no mutants\n" +
		"2 survived:\n" +
		"  (a1) p.go:4  if a > 0 {  ->  if true {\n" +
		"  (a3) p.go:4  if a > 0 {  ->  if a >= 0 {\n"
	if out.String() != want || sum.Killed != 2 || sum.Survived != 2 {
		t.Fatalf("summary %+v, report:\n%s", sum, out.String())
	}
}

func TestGoTestTellsABuildFailureFromAFailure(t *testing.T) {
	root := probeModule(t)
	dir := t.TempDir()
	broken := filepath.Join(dir, "p.go")
	if err := os.WriteFile(broken, []byte("package p\n\nfunc Sign(a int) int {\n\tx := 1\n\treturn 0\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	overlay := filepath.Join(dir, "overlay.json")
	if err := os.WriteFile(overlay, overlayJSON(filepath.Join(root, "p", "p.go"), broken), 0o644); err != nil {
		t.Fatal(err)
	}
	if outcome, err := GoTest(root)("p", overlay); err != nil || outcome != BuildFailed {
		t.Fatalf("outcome %d, err %v", outcome, err)
	}
}

func TestGoTestReportsAGoCommandThatDoesNotStart(t *testing.T) {
	root := probeModule(t)
	t.Setenv("PATH", "")
	if _, err := GoTest(root)("p", ""); err == nil {
		t.Fatal("a go command that cannot be found must be an error")
	}
}
