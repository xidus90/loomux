package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/query"
	"github.com/xidus90/loomux/internal/gitenv"
)

// hookBinary builds loomux once for the tests that run it from a real git
// hook, where the lane's commands start the binary as child processes.
func hookBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "loomux.exe")
	build := exec.Command("go", "build", "-o", binary, "github.com/xidus90/loomux/cmd/loomux")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return binary
}

// hookRepo is a committed repository whose lib.Run has six callers and no
// test, with a graph built, Run's body changed in the working tree, and a
// pre-commit hook that runs the graph lane with binary.
func hookRepo(t *testing.T, binary string) func(args ...string) (string, error) {
	t.Helper()
	files := map[string]string{
		".gitignore": ".loomux/\n",
		"go.mod":     "module example.com/repo\n",
		"lib/lib.go": "package lib\n\n// Run does the thing.\nfunc Run() {}\n",
	}
	var app strings.Builder
	app.WriteString("package app\n\nimport \"example.com/repo/lib\"\n")
	for _, name := range []string{"A", "B", "C", "D", "E", "F"} {
		app.WriteString("\n// " + name + " calls Run.\nfunc " + name + "() { lib.Run() }\n")
	}
	files["app/app.go"] = app.String()
	root := repo(t, files)
	hooksDir := t.TempDir()
	hook := "#!/bin/sh\nexec \"" + filepath.ToSlash(binary) + "\" check graph --root \"" + filepath.ToSlash(root) + "\"\n"
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	// gitenv.Environ: this test runs inside loomux's own pre-commit gate,
	// whose git pointers would otherwise aim these commands at loomux.
	gitIn := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = gitenv.Environ()
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	for _, args := range [][]string{
		{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "t"},
		{"config", "core.autocrlf", "false"}, {"config", "commit.gpgsign", "false"},
		{"add", "."}, {"commit", "-qm", "init"},
		{"config", "core.hooksPath", filepath.ToSlash(hooksDir)},
	} {
		if out, err := gitIn(args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if _, _, err := query.Build(root, func(string) {}); err != nil {
		t.Fatal(err)
	}
	body := "package lib\n\n// Run does the thing.\nfunc Run() { _ = 1 }\n"
	if err := os.WriteFile(filepath.Join(root, "lib", "lib.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return gitIn
}

// The graph lane reads the index git hands the pre-commit hook: under
// `git commit -a` or `git commit <path>` that is a temporary one, and
// .git/index holds nothing staged. Every form of the same commit is refused
// for the same finding.
func TestTheGraphLaneRefusesACommitWhateverIndexGitHandsTheHook(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	binary := hookBinary(t)
	for name, commit := range map[string][][]string{
		"commit -a":      {{"commit", "-a", "-m", "change Run"}},
		"commit a path":  {{"commit", "-m", "change Run", "lib/lib.go"}},
		"add and commit": {{"add", "lib/lib.go"}, {"commit", "-m", "change Run"}},
	} {
		t.Run(name, func(t *testing.T) {
			gitIn := hookRepo(t, binary)
			var out string
			var err error
			for _, args := range commit {
				if out, err = gitIn(args...); err != nil {
					break
				}
			}
			if err == nil {
				t.Fatalf("the commit went through:\n%s", out)
			}
			if !strings.Contains(out, "graph/go: failed") || !strings.Contains(out, "lib/lib.go [none]: Run in-degree 6") {
				t.Fatalf("refused, but not for the finding: %v\n%s", err, out)
			}
			if head, _ := gitIn("rev-list", "--count", "HEAD"); strings.TrimSpace(head) != "1" {
				t.Fatalf("HEAD has %s commits, want 1", head)
			}
		})
	}
}
