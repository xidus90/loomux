package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const pageWithoutType = "---\ntitle: T\n---\n"

func writeFile(t *testing.T, path, body string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// stubGetwd answers the working directory for one test.
func stubGetwd(t *testing.T, dir string, err error) {
	t.Helper()
	saved := getwd
	getwd = func() (string, error) { return dir, err }
	t.Cleanup(func() { getwd = saved })
}

func TestLintRejectsAnUnknownFlag(t *testing.T) {
	if code, _, _ := run("lint", "--nope", "x.md"); code != 2 {
		t.Fatalf("code %d", code)
	}
}

// The relative path in the finding shows which wiki root the page was
// linted against: the project's wiki makes it sub/page.md.
func TestLintUsesTheWikiOfTheGivenRoot(t *testing.T) {
	project := t.TempDir()
	page := writeFile(t, filepath.Join(project, "wiki", "sub", "page.md"), pageWithoutType)
	code, out, errOut := run("lint", "--root", project, page)
	if code != 1 || out != "" || !strings.Contains(errOut, "[error:missing-type] sub/page.md:") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestLintTakesTheWorkingDirectoryWithoutRoot(t *testing.T) {
	project := t.TempDir()
	page := writeFile(t, filepath.Join(project, "wiki", "sub", "page.md"), pageWithoutType)
	stubGetwd(t, project, nil)
	code, _, errOut := run("lint", page)
	if code != 1 || !strings.Contains(errOut, "[error:missing-type] sub/page.md:") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestLintFallsBackToTheDirectoryOfThePage(t *testing.T) {
	page := writeFile(t, filepath.Join(t.TempDir(), "sub", "page.md"), pageWithoutType)
	code, _, errOut := run("lint", "--root", t.TempDir(), page)
	if code != 1 || !strings.Contains(errOut, "[error:missing-type] page.md:") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestLintFailsWhenTheWorkingDirectoryIsUnknown(t *testing.T) {
	stubGetwd(t, "", errors.New("gone"))
	code, _, errOut := run("lint", "page.md")
	if code != 1 || errOut != "loomux lint: gone\n" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestWikiGateReportsForTheGivenRoot(t *testing.T) {
	project := t.TempDir()
	writeFile(t, filepath.Join(project, "wiki", "page.md"), pageWithoutType)
	code, _, errOut := run("wiki-gate", "--root", project)
	if code != 1 || !strings.Contains(errOut, "Wiki-Gate Violation(s) Detected") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestWikiGateTakesTheWorkingDirectoryWithoutRoot(t *testing.T) {
	stubGetwd(t, t.TempDir(), nil)
	code, out, _ := run("wiki-gate")
	if code != 0 || !strings.HasPrefix(out, "OK: Wiki Gate passed.") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestWikiGateRejectsAnUnknownFlag(t *testing.T) {
	if code, _, _ := run("wiki-gate", "--nope"); code != 2 {
		t.Fatalf("code %d", code)
	}
}

func TestWikiGateFailsWhenTheWorkingDirectoryIsUnknown(t *testing.T) {
	stubGetwd(t, "", errors.New("gone"))
	code, _, errOut := run("wiki-gate")
	if code != 1 || errOut != "loomux wiki-gate: gone\n" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}
