package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/setup/gitfiles"
)

const (
	formatted   = "package x\n\nfunc F() {\n}\n"
	unformatted = "package x\n\nfunc F() {\nreturn\n}\n"
)

// armWorld is a committed repository with two project lanes that the loomux
// binary itself judges -- lint is red when a/ holds an unformatted file,
// types when b/ does -- the file holding armed, and as its pre-commit hook
// hook, or the one init writes when hook is "". The project ignores only its
// state, as init sets it up. The state directory and LOCALAPPDATA point at
// throwaway places, so the binary the hook starts reads nothing of this
// machine; gitenv.Environ keeps the pointers of loomux's own pre-commit gate,
// which runs this suite, out of every git call here, and core.hooksPath is
// set in the repository, above any global one.
func armWorld(t *testing.T, binary, armed, hook string) (root string, gitIn func(args ...string) (string, error)) {
	t.Helper()
	t.Setenv("LOOMUX_STATE_DIR", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	root = repo(t, map[string]string{
		".gitignore":          "/.loomux/state/\n",
		".loomux/config.toml": "[verify.project]\nlint = \"{loomux} check gofmt a\"\ntypes = \"{loomux} check gofmt b\"\n",
		".loomux/armed.toml":  armed,
		"a/x.go":              formatted,
		"b/y.go":              formatted,
		"c.txt":               "c\n",
	})
	if hook == "" {
		hook = gitfiles.Hooks(filepath.ToSlash(binary))["pre-commit"]
	}
	if err := os.MkdirAll(filepath.Join(root, ".githooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".githooks", "pre-commit"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn = func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = gitenv.Environ()
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	for _, args := range [][]string{
		{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "t"},
		{"config", "core.autocrlf", "false"}, {"config", "commit.gpgsign", "false"},
		{"add", "."}, {"commit", "-qm", "init", "--no-verify"},
		{"config", "core.hooksPath", ".githooks"},
	} {
		if out, err := gitIn(args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root, gitIn
}

func put(t *testing.T, root, rel, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func must(t *testing.T, gitIn func(...string) (string, error), args ...string) string {
	t.Helper()
	out, err := gitIn(args...)
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(out)
}

func armedAtHead(t *testing.T, gitIn func(...string) (string, error)) string {
	t.Helper()
	return must(t, gitIn, "show", "HEAD:.loomux/armed.toml")
}

func armedOnDisk(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".loomux", "armed.toml"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(data))
}

// The pre-commit hook init writes, run by a real git commit: every write and
// every staging is the real binary's, under the index git hands the hook.
func TestTheHookArmsTheGreenLanesIntoTheSameCommit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	binary := hookBinary(t)
	none := strings.TrimSpace(armedText())
	both := strings.TrimSpace(armedText("lint/project@.", "types/project@."))

	// (a) and (b)
	t.Run("a commit arms the green lanes, and the file is in it", func(t *testing.T) {
		root, gitIn := armWorld(t, binary, armedText(), "")
		put(t, root, "c.txt", "c2\n")
		must(t, gitIn, "add", "c.txt")
		out := must(t, gitIn, "commit", "-m", "change c")
		if !strings.Contains(out, "armed: lint/project@., types/project@.") {
			t.Fatalf("the hook said %q", out)
		}
		if names := must(t, gitIn, "show", "--name-only", "--format=", "HEAD"); !strings.Contains(names, ".loomux/armed.toml") || !strings.Contains(names, "c.txt") {
			t.Fatalf("the commit holds %q", names)
		}
		if got := armedAtHead(t, gitIn); got != both {
			t.Fatalf("armed at HEAD: %q", got)
		}
		if status := must(t, gitIn, "status", "--porcelain"); status != "" {
			t.Fatalf("left dirty: %q", status)
		}
		// A second commit has nothing to arm and does not touch the file.
		put(t, root, "c.txt", "c3\n")
		must(t, gitIn, "add", "c.txt")
		must(t, gitIn, "commit", "-qm", "change c again")
		if names := must(t, gitIn, "show", "--name-only", "--format=", "HEAD"); names != "c.txt" {
			t.Fatalf("the second commit holds %q", names)
		}
	})

	// (c)
	t.Run("a red lane in probation lets the commit through, an armed one holds it", func(t *testing.T) {
		armed := armedText("lint/project@.")
		root, gitIn := armWorld(t, binary, armed, "")
		put(t, root, "b/y.go", unformatted)
		out := must(t, gitIn, "commit", "-am", "break the lane in probation")
		if !strings.Contains(out, "types/project: failed (probation)") || armedAtHead(t, gitIn) != strings.TrimSpace(armed) {
			t.Fatalf("out %q, armed %q", out, armedAtHead(t, gitIn))
		}
		// Now the armed lane is red and the other green: refused, and the
		// green lane is not armed by a run that did not go through.
		put(t, root, "b/y.go", formatted)
		put(t, root, "a/x.go", unformatted)
		commits := must(t, gitIn, "rev-list", "--count", "HEAD")
		out, err := gitIn("commit", "-am", "break the armed lane")
		if err == nil || !strings.Contains(out, "lint/project: failed [config]") {
			t.Fatalf("the commit went through: %v\n%s", err, out)
		}
		if armedOnDisk(t, root) != strings.TrimSpace(armed) {
			t.Fatalf("a refused commit wrote the file: %q", armedOnDisk(t, root))
		}
		if staged := must(t, gitIn, "diff", "--cached", "--name-only"); strings.Contains(staged, "armed.toml") {
			t.Fatalf("a refused commit left the file in the index: %q", staged)
		}
		if status := must(t, gitIn, "status", "--porcelain", "--", ".loomux/armed.toml"); status != "" || must(t, gitIn, "rev-list", "--count", "HEAD") != commits {
			t.Fatalf("status %q", status)
		}
	})

	// (d)
	for name, commit := range map[string][]string{
		"git commit <path>":        {"commit", "-m", "only a", "a/x.go"},
		"git commit --only <path>": {"commit", "--only", "-m", "only a", "a/x.go"},
	} {
		t.Run("a partial commit arms nothing: "+name, func(t *testing.T) {
			root, gitIn := armWorld(t, binary, armedText(), "")
			put(t, root, "a/x.go", formatted+"\n// more\n")
			put(t, root, "c.txt", "c2\n")
			out := must(t, gitIn, commit...)
			if !strings.Contains(out, "not armed: this commit takes only some paths") {
				t.Fatalf("the hook said %q", out)
			}
			if names := must(t, gitIn, "show", "--name-only", "--format=", "HEAD"); names != "a/x.go" {
				t.Fatalf("the commit holds %q", names)
			}
			// File and index as they were: no staged revert, nothing to see.
			if armedOnDisk(t, root) != none || armedAtHead(t, gitIn) != none {
				t.Fatalf("armed on disk %q, at HEAD %q", armedOnDisk(t, root), armedAtHead(t, gitIn))
			}
			if status := must(t, gitIn, "status", "--porcelain", "--", ".loomux/armed.toml"); status != "" {
				t.Fatalf("after the partial commit: %q", status)
			}
			// The next ordinary commit arms.
			must(t, gitIn, "add", "c.txt")
			must(t, gitIn, "commit", "-qm", "c")
			if status := must(t, gitIn, "status", "--porcelain"); status != "" || armedAtHead(t, gitIn) != both {
				t.Fatalf("after the next commit: status %q, armed %q", status, armedAtHead(t, gitIn))
			}
		})
	}

	// (d) with a human's edit of the file not yet staged: a partial commit
	// takes neither the edit nor a staged copy of it, and the edit stays
	// where it was.
	t.Run("a partial commit leaves an unstaged edit of the file alone", func(t *testing.T) {
		root, gitIn := armWorld(t, binary, armedText(), "")
		edited := armedText("lint/project@.")
		put(t, root, ".loomux/armed.toml", edited)
		put(t, root, "a/x.go", formatted+"\n// more\n")
		out := must(t, gitIn, "commit", "-m", "only a", "a/x.go")
		if !strings.Contains(out, "not armed: this commit takes only some paths") {
			t.Fatalf("the hook said %q", out)
		}
		if names := must(t, gitIn, "show", "--name-only", "--format=", "HEAD"); names != "a/x.go" {
			t.Fatalf("the commit holds %q", names)
		}
		if armedOnDisk(t, root) != strings.TrimSpace(edited) || armedAtHead(t, gitIn) != none {
			t.Fatalf("armed on disk %q, at HEAD %q", armedOnDisk(t, root), armedAtHead(t, gitIn))
		}
		if status, _ := gitIn("status", "--porcelain", "--", ".loomux/armed.toml"); status != " M .loomux/armed.toml\n" {
			t.Fatalf("the edit is no longer only in the working tree: %q", status)
		}
	})

	// (e)
	t.Run("git commit -a arms like a plain commit", func(t *testing.T) {
		root, gitIn := armWorld(t, binary, armedText(), "")
		put(t, root, "c.txt", "c2\n")
		must(t, gitIn, "commit", "-qam", "change c")
		if names := must(t, gitIn, "show", "--name-only", "--format=", "HEAD"); !strings.Contains(names, ".loomux/armed.toml") || !strings.Contains(names, "c.txt") {
			t.Fatalf("the commit holds %q", names)
		}
		if status := must(t, gitIn, "status", "--porcelain"); status != "" || armedAtHead(t, gitIn) != both {
			t.Fatalf("status %q, armed %q", status, armedAtHead(t, gitIn))
		}
	})

	// (f)
	t.Run("--no-verify arms nothing", func(t *testing.T) {
		root, gitIn := armWorld(t, binary, armedText(), "")
		put(t, root, "c.txt", "c2\n")
		must(t, gitIn, "commit", "-qam", "unchecked", "--no-verify")
		if armedAtHead(t, gitIn) != none || armedOnDisk(t, root) != none {
			t.Fatalf("armed at HEAD %q, on disk %q", armedAtHead(t, gitIn), armedOnDisk(t, root))
		}
	})

	// (g)
	t.Run("a hook without --arm never arms", func(t *testing.T) {
		old := "#!/bin/sh\n# loomux pre-commit hook: the check chain of .loomux/config.toml.\nexec \"" + filepath.ToSlash(binary) + "\" check precommit\n"
		root, gitIn := armWorld(t, binary, armedText(), old)
		put(t, root, "c.txt", "c2\n")
		out := must(t, gitIn, "commit", "-am", "through the old hook")
		if !strings.Contains(out, "probation: lint/project@., types/project@.") || armedAtHead(t, gitIn) != none || armedOnDisk(t, root) != none {
			t.Fatalf("out %q, armed %q", out, armedAtHead(t, gitIn))
		}
	})
}
