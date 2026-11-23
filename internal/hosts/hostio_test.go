package hosts_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/hosts"
)

func TestParseHost(t *testing.T) {
	for _, name := range []string{"claude", "dsh", "antigravity", "codex"} {
		if _, err := hosts.ParseHost(name); err != nil {
			t.Errorf("ParseHost(%q): %v", name, err)
		}
	}
	if got, err := hosts.ParseHost("dsh"); err != nil || got != hosts.HostDsh {
		t.Fatalf(`ParseHost("dsh") = %q, %v; want HostDsh`, got, err)
	}
	// An unknown host is refused here rather than guessed at. The flag is
	// written by loomux init, so a value nobody knows means the two have drifted
	// apart, and answering a hook in the wrong shape is worse than refusing.
	if _, err := hosts.ParseHost("gemini-cli"); err == nil {
		t.Error("an unknown host is refused")
	}
	if _, err := hosts.ParseHost(""); err == nil {
		t.Error("an empty host is refused")
	}
	// The refusal lists the hosts a caller may pass, so a host the seam
	// answers has to appear in it: a reader of the refusal looks for theirs
	// there and nowhere else.
	if _, err := hosts.ParseHost("q"); err == nil || !strings.Contains(err.Error(), "dsh") {
		t.Fatalf(`ParseHost("q") = %v; want a refusal naming dsh`, err)
	}
}

// The DSH harness delivers the Claude Code payload dialect verbatim, so one
// payload reads field for field the same under HostDsh as under HostClaude.
// Measured in a live spike on 2026-11-21 against dsh-llm 0.2.2-rc.1, which
// sent session_id, transcript_path, hook_event_name, tool_name, tool_input,
// tool_use_id, source and stop_hook_active through its hooks.
func TestDshSpeaksTheClaudePayload(t *testing.T) {
	body := `{"hook_event_name":"SubagentStop","session_id":"s-1","agent_id":"a-1","agent_type":"Explore","transcript_path":"/tmp/t.jsonl","tool_name":"Bash","tool_input":{"command":"ls"},"tool_use_id":"tu-1","source":"cli","stop_hook_active":false}`

	fromDsh, err := hosts.Read(hosts.HostDsh, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	fromClaude, err := hosts.Read(hosts.HostClaude, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if fromDsh != fromClaude {
		t.Fatalf("dsh %+v, claude %+v", fromDsh, fromClaude)
	}
	if fromDsh.Event != "SubagentStop" || fromDsh.SessionID != "s-1" || fromDsh.AgentID != "a-1" || fromDsh.AgentType != "Explore" {
		t.Fatalf("the fields the hooks read arrive: %+v", fromDsh)
	}

	// The answer goes back the same way: dsh reads Claude's envelope.
	var out bytes.Buffer
	if err := hosts.WriteContext(hosts.HostDsh, "SessionStart", &out, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"hookEventName":"SessionStart"`) {
		t.Fatalf("dsh gets the claude envelope: %s", out.String())
	}
}

// An Antigravity hook runs with its working directory set to the directory
// holding hooks.json -- `.agents/`, not the project root -- so the root has to
// be found by walking up. Measured on 2026-09-10 against agy 1.1.24 and
// documented in `2026-09-10-antigravity-hook-messung.md`, finding 2, in the
// working papers of the archive release `archive/parity-recordings`.
func TestFindRootWalksUpToTheConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, ".agents")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := hosts.FindRoot(deep)
	if err != nil {
		t.Fatal(err)
	}

	// EvalSymlinks on both sides: a temp directory on macOS is reached through
	// /var, which is a link to /private/var, and the comparison would fail on
	// the spelling rather than on the answer.
	want, _ := filepath.EvalSymlinks(root)
	gotResolved, _ := filepath.EvalSymlinks(got)
	if gotResolved != want {
		t.Fatalf("FindRoot = %q, want %q", gotResolved, want)
	}
}

// The caller passes ".", so a relative start has to be resolved before the
// walk: `filepath.Dir` would end it at "." instead of at the volume root.
func TestFindRootResolvesARelativeStart(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, ".agents")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	// t.Chdir: it restores the directory itself and refuses a parallel test,
	// which is the whole of what the hand-written save and defer here did. The
	// reason for the hand-written form -- that the call arrived in Go 1.24 and
	// this module was on 1.22 -- has not been true since go.mod says 1.25.0.
	t.Chdir(deep)

	got, err := hosts.FindRoot(".")
	if err != nil {
		t.Fatal(err)
	}

	want, _ := filepath.EvalSymlinks(root)
	gotResolved, _ := filepath.EvalSymlinks(got)
	if gotResolved != want {
		t.Fatalf("FindRoot = %q, want %q", gotResolved, want)
	}
}

func TestFindRootWithoutAConfig(t *testing.T) {
	_, err := hosts.FindRoot(t.TempDir())
	if !errors.Is(err, hosts.ErrNoRoot) {
		t.Fatalf("expected ErrNoRoot, got %v", err)
	}
}
