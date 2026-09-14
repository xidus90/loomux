package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runWith is `run` with something on stdin: the hook commands read a payload.
func runWith(stdin string, args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Run(args, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

// project is a directory loomux recognises as a root.
func project(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestHookNeedsAnEvent(t *testing.T) {
	code, _, errOut := run("hook")
	if code != 2 || !strings.Contains(errOut, "usage: loomux hook <event>") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// A mistyped event is refused for what it is rather than falling through to
// the write barrier: that barrier reads stdin and decides about a file, and
// answering a hook call that way is a verdict about the wrong question.
func TestHookRefusesAnUnknownEvent(t *testing.T) {
	code, _, errOut := run("hook", "stop", "--host", "claude")
	if code != 2 || !strings.Contains(errOut, `unknown event "stop"`) {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// `--host` has no default, so a call without it is refused rather than handed
// a Claude envelope. The root is given, so what this pins is that the missing
// flag is what stops the call.
func TestHookSessionStartNeedsAHost(t *testing.T) {
	code, _, errOut := run("hook", "session-start", "--root", t.TempDir())
	if code != 1 || !strings.Contains(errOut, "--host is required") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestHookPreToolUseRefusesWithoutAHost(t *testing.T) {
	code, _, _ := run("hook", "pre-tool-use", "--root", t.TempDir())
	if code != 2 {
		t.Fatalf("a write barrier refuses a malformed call, got %d", code)
	}
}

// An unknown host is refused by hosts.ParseHost, not guessed at, and an
// announcement never blocks over it.
func TestHookSessionStartRefusesAnUnknownHost(t *testing.T) {
	code, _, errOut := run("hook", "session-start", "--host", "nobody", "--root", t.TempDir())
	if code != 1 || !strings.Contains(errOut, "nobody") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestHookRefusesAnUnknownFlag(t *testing.T) {
	code, _, _ := run("hook", "session-start", "--host", "claude", "--invalid-flag")
	if code != 1 {
		t.Fatalf("code %d", code)
	}
}

// Without `--root` the root is the first directory at or above the working one
// that holds `.loomux/config.toml`, and the hook runs against that. The state
// file is the evidence that the walk found this project and not the checkout
// the test binary happens to run in.
func TestHookWalksUpToTheRootWhenNoneIsGiven(t *testing.T) {
	root := project(t)
	inside := filepath.Join(root, "deep", "deeper")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(inside)

	code, _, errOut := runWith(`{"session_id":"s1"}`, "hook", "session-start", "--host", "claude")
	if code != 0 {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// A directory with no `.loomux/config.toml` above it has no root to find, and
// the refusal says what was looked for.
func TestHookOutsideAProjectSaysWhatItLookedFor(t *testing.T) {
	t.Chdir(t.TempDir())

	code, _, errOut := run("hook", "session-start", "--host", "claude")
	if code != 1 || !strings.Contains(errOut, "no .loomux/config.toml") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestHookPostToolUseRunsAndPasses(t *testing.T) {
	payload := `{"tool_name":"Edit","tool_input":{"file_path":"x.json"}}`
	code, _, errOut := runWith(payload, "hook", "post-tool-use", "--host", "claude", "--root", t.TempDir())
	if code != 0 {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestStatusPrintsTheBanner(t *testing.T) {
	for _, name := range []string{"status", "explain", "doctor"} {
		code, out, errOut := run(name, "--root", t.TempDir())
		if code != 0 || !strings.Contains(out, "loomux Hook Inspection") {
			t.Fatalf("%s: code %d, out %q, err %q", name, code, out, errOut)
		}
		if code, _, _ := run(name, "--invalid-flag"); code != 1 {
			t.Fatalf("%s: an unknown flag ends with %d", name, code)
		}
	}
}

func TestWorktreeNeedsASubcommand(t *testing.T) {
	code, _, errOut := run("worktree")
	if code != 2 || !strings.Contains(errOut, "usage: loomux worktree") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestWorktreeRefusesAnUnknownSubcommand(t *testing.T) {
	code, _, errOut := run("worktree", "frobnicate")
	if code != 2 || !strings.Contains(errOut, `unknown subcommand "frobnicate"`) {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestWorktreeRefusesAnUnknownFlag(t *testing.T) {
	if code, _, _ := run("worktree", "link", "--bad"); code != 1 {
		t.Fatalf("code %d", code)
	}
	if code, _, _ := run("worktree", "unlink", "--bad"); code != 1 {
		t.Fatalf("code %d", code)
	}
}

// The path is an argument and not a flag, so the arity is the only thing
// standing between a mistyped call and a directory that disappears. The empty
// string is turned away here as well: it is a path nothing can stat, so every
// identity check downstream answers "no" about it.
func TestWorktreeRemoveNeedsExactlyOnePath(t *testing.T) {
	for _, args := range [][]string{{"worktree", "remove"}, {"worktree", "remove", ""}, {"worktree", "remove", "a", "b"}} {
		if code, _, _ := run(args...); code != 1 {
			t.Fatalf("%v: code %d", args, code)
		}
	}
}

// Outside a repository there is nothing to remove, and a path named by hand
// that is wrong is a fault rather than a silent "nothing to do".
func TestWorktreeRemoveDispatches(t *testing.T) {
	code, _, errOut := run("worktree", "remove", t.TempDir())
	if code != 1 || errOut == "" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// link and unlink are silent when there is nothing to mirror, which is what
// every session start and end in a directory that is not a worktree looks
// like.
func TestWorktreeLinkAndUnlinkDispatch(t *testing.T) {
	if code, _, errOut := run("worktree", "link", "--root", t.TempDir()); code != 0 || errOut != "" {
		t.Fatalf("link: code %d, err %q", code, errOut)
	}
	code, _, errOut := runWith(`{"session_id":"mine"}`, "worktree", "unlink", "--root", t.TempDir())
	if code != 0 || errOut != "" {
		t.Fatalf("unlink: code %d, err %q", code, errOut)
	}
}

// The barrier is global and needs no project: with a host and no root it is
// still reached, and until Task 10 puts hooks.PreToolUse behind it, what it
// answers is the refusal a malformed barrier call gets.
func TestHookPreToolUseIsReachedWithoutARoot(t *testing.T) {
	t.Chdir(t.TempDir())

	code, _, errOut := runWith(`{"tool_name":"Write","tool_input":{"file_path":"x.txt"}}`,
		"hook", "pre-tool-use", "--host", "claude")
	if code != 2 || errOut != "" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}
