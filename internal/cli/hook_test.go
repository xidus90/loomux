package cli

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/hooks"
	"github.com/xidus90/loomux/internal/sessions"
)

// gitInit makes `root` a repository with one commit: the session start files a
// base commit only where there is one to read.
func gitInit(t *testing.T, root string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "t@example.invalid"},
		{"config", "user.name", "Test"},
		{"add", "a.txt"},
		{"commit", "-m", "first"},
	} {
		command := exec.Command("git", args...)
		command.Dir = root
		command.Env = gitenv.Environ()
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

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

// A flag error ends in the exit code of the event that was called, and the two
// answers differ: a write barrier that cannot read its own call refuses (2),
// while an announcement never blocks (1). One line per event, so the table in
// hook.go is pinned by behaviour and not by itself.
func TestHookRefusesAnUnknownFlag(t *testing.T) {
	for event, want := range map[string]int{
		"session-start": 1,
		"post-tool-use": 1,
		"pre-tool-use":  2,
	} {
		code, _, _ := run("hook", event, "--host", "claude", "--invalid-flag")
		if code != want {
			t.Fatalf("%s: code %d, want %d", event, code, want)
		}
	}
}

// Without `--root` the root is the first directory at or above the working one
// that holds `.loomux/config.toml`, and the hook runs against that: with no
// root to find, hosts.FindRoot refuses and the call ends with 1 instead.
func TestHookWalksUpToTheRootWhenNoneIsGiven(t *testing.T) {
	root := project(t)
	gitInit(t, root)
	inside := filepath.Join(root, "deep", "deeper")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(inside)

	code, _, errOut := runWith(`{"session_id":"s1"}`, "hook", "session-start", "--host", "claude")
	if code != 0 {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	// The walk decided which project the base was filed under, so the state
	// file is the evidence that it found this fixture and not the checkout the
	// test binary happens to run in. Exit 0 alone would say nothing about that.
	if state := sessions.ReadState(root, "s1"); len(state.Base) != 40 {
		t.Fatalf("the base was not filed under the root that was walked to: %q", state.Base)
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

// --budget reaches post-edit as given, and its absence as the default.
func TestHookPostToolUsePassesTheBudgetOn(t *testing.T) {
	saved := postToolUse
	t.Cleanup(func() { postToolUse = saved })
	var got time.Duration
	postToolUse = func(_ io.Reader, _, _ io.Writer, _ string, budget time.Duration) int {
		got = budget
		return 0
	}
	root := t.TempDir()
	if code, _, errOut := runWith(`{}`, "hook", "post-tool-use", "--host", "claude", "--root", root, "--budget", "3s"); code != 0 || got != 3*time.Second {
		t.Fatalf("code %d, budget %v, err %q", code, got, errOut)
	}
	if code, _, errOut := runWith(`{}`, "hook", "post-tool-use", "--host", "claude", "--root", root); code != 0 || got != hooks.DefaultBudget {
		t.Fatalf("code %d, budget %v, err %q", code, got, errOut)
	}
}

// A budget that is no duration is a malformed call, and a malformed
// post-tool-use call announces rather than blocks.
func TestHookPostToolUseRefusesABudgetThatIsNoDuration(t *testing.T) {
	code, _, errOut := runWith(`{}`, "hook", "post-tool-use", "--host", "claude", "--root", t.TempDir(), "--budget", "soon")
	if code != 1 || !strings.Contains(errOut, "flag -budget") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if code, _, _ := run("hook", "pre-tool-use", "--host", "claude", "--budget", "1s"); code != 2 {
		t.Fatalf("only post-tool-use has a budget, pre-tool-use ended with %d", code)
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
// still reached, and a write into a directory no area covers is refused there
// with a reason on both channels. The state directory is a fresh one, so what
// answers is an empty registry and not the machine's own.
func TestHookPreToolUseIsReachedWithoutARoot(t *testing.T) {
	t.Setenv("LOOMUX_STATE_DIR", t.TempDir())
	t.Chdir(t.TempDir())

	code, out, errOut := runWith(`{"tool_name":"Write","tool_input":{"file_path":"x.txt"}}`,
		"hook", "pre-tool-use", "--host", "claude")
	if code != 2 || errOut == "" || !strings.Contains(out, `"permissionDecision": "deny"`) {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}
