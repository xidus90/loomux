package cases

import (
	"errors"
	"path/filepath"
	"testing"
)

// The temp directory RunCase stages into cannot be made to fail from outside
// the package; the seam is unexported, so this test lives beside it.
func TestRunCaseReportsATempDirItCannotMake(t *testing.T) {
	mkdirTemp = func(string, string) (string, error) { return "", errors.New("no temp") }
	defer func() { mkdirTemp = defaultMkdirTemp }()
	c := &Case{Verb: "v", Name: "n", Path: t.TempDir(), Cmd: "loomux x"}
	if _, err := RunCase(c, nil); err == nil {
		t.Fatal("want error")
	}
}

// A world is staged under the spelling the file system resolves to, the one
// git answers with; a path that does not resolve is kept as given.
func TestLongPathIsTheResolvedSpelling(t *testing.T) {
	dir := t.TempDir()
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := longPath(dir); got != want {
		t.Fatalf("longPath(%q) = %q, want %q", dir, got, want)
	}
	missing := filepath.Join(dir, "gone")
	if got := longPath(missing); got != missing {
		t.Fatalf("longPath(%q) = %q, want it unchanged", missing, got)
	}
}

// The MCP runner stages into the same temp directory, through the same seam.
func TestRunMCPCaseReportsATempDirItCannotMake(t *testing.T) {
	mkdirTemp = func(string, string) (string, error) { return "", errors.New("no temp") }
	defer func() { mkdirTemp = defaultMkdirTemp }()
	c := &MCPCase{Verb: "v", Name: "n", Path: t.TempDir()}
	if _, err := RunMCPCase(c, nil); err == nil {
		t.Fatal("want error")
	}
}
