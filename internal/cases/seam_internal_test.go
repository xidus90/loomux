package cases

import (
	"errors"
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

// The MCP runner stages into the same temp directory, through the same seam.
func TestRunMCPCaseReportsATempDirItCannotMake(t *testing.T) {
	mkdirTemp = func(string, string) (string, error) { return "", errors.New("no temp") }
	defer func() { mkdirTemp = defaultMkdirTemp }()
	c := &MCPCase{Verb: "v", Name: "n", Path: t.TempDir()}
	if _, err := RunMCPCase(c, nil); err == nil {
		t.Fatal("want error")
	}
}
