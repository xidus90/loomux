package hooks

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/sessions"
)

// Nothing to announce writes no envelope at all, rather than one carrying an
// empty additionalContext -- hosts.WriteContext makes that decision in its
// Claude arm, and this only shows it reaching the hook's own output. Named
// rather than cited by line: the decision has already moved between files once.
func TestHookSessionStartIsSilentWithNothingWaiting(t *testing.T) {
	root := project(t)

	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "claude")

	if code != ExitOK || stdout.Len() != 0 {
		t.Fatalf("expected exit 0 and no output, got %d and %q", code, stdout.String())
	}
}

// The base commit is written here and nowhere else: by the time the first Stop
// fires, the turn has already run, and anything it committed would sit inside
// the baseline that is supposed to expose it.
//
// Read back through the package's own door rather than by parsing the file
// again: a second reader here would pass while ReadState was broken.
func TestHookSessionStartRecordsTheBaseCommit(t *testing.T) {
	root := project(t)
	gitInit(t, root)

	var stdout, stderr bytes.Buffer
	SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "claude")

	if state := sessions.ReadState(root, "s1"); len(state.Base) != 40 {
		t.Fatalf("expected a full sha as the base, got %q", state.Base)
	}
}

// Silent in both failure cases, as session_start.py (fa3dd38):43-59's
// _record_base was:
// without a session id there is nowhere to file it, and outside a repository
// there is nothing to file. Neither is a defect of the project, and neither is
// worth a line in every session of every checkout that is not a repository.
// The stop gate is where the absence matters and where it is said out loud.
func TestHookSessionStartRecordsNoBaseWithoutARepository(t *testing.T) {
	root := project(t)

	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "claude")

	if code != ExitOK {
		t.Fatalf("a checkout that is not a repository is not a failure, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("and it is not worth a word, got %q", stderr.String())
	}
}

func TestHookSessionStartRecordsNoBaseWithoutASessionID(t *testing.T) {
	root := project(t)
	gitInit(t, root)

	var stdout, stderr bytes.Buffer
	SessionStart(strings.NewReader(`{"hook_event_name":"SessionStart"}`), &stdout, &stderr, root, "claude")

	dir := filepath.Join(root, filepath.FromSlash(sessions.StateDir))
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		t.Fatalf("no session id means nowhere to file it, got %d files", len(entries))
	}
}

// A base that cannot be written is said out loud and ends the call with exit
// 1, unlike the two absences above. state.py:56-65's `write` catches nothing,
// so an OSError there leaves the Python `run` by itself; keeping quiet here
// would be a departure from the original rather than the parity the two
// silences are.
func TestHookSessionStartReportsABaseItCannotWrite(t *testing.T) {
	root := project(t)
	gitInit(t, root)
	// A file where the state directory belongs, so creating it must fail.
	stateDir := filepath.Join(root, filepath.FromSlash(sessions.StateDir))
	if err := os.MkdirAll(filepath.Dir(stateDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stateDir, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "claude")

	if code != ExitInternal {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "hooks") {
		t.Fatalf("the refusal names the path it could not write, got %q", stderr.String())
	}
}

// payload.py's exit protocol: 1 for an internal failure, and never 2 -- this
// hook is an announcement and has nothing to block.
func TestHookSessionStartOnABadPayload(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader("{not json"), &stdout, &stderr, project(t), "claude")

	if code != ExitInternal {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if strings.TrimSpace(stderr.String()) == "" {
		t.Fatal("a refusal says why")
	}
}

func TestHookSessionStartOnAnUnknownHost(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, project(t), "gemini-cli")

	if code != ExitInternal {
		t.Fatalf("an unknown host is refused, got %d", code)
	}
}

// A host whose adapter is still a promise is refused where it is first asked a
// question -- hosts.Read -- and the refusal carries that host's own words.
func TestHookSessionStartOnAHostWithoutAnAdapter(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, project(t), "antigravity")

	if code != ExitInternal {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "antigravity") {
		t.Fatalf("the refusal names the host, got %q", stderr.String())
	}
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

func gitInit(t *testing.T, root string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "t@example.invalid"},
		{"config", "user.name", "Test"},
	} {
		command := exec.Command("git", args...)
		command.Dir = root
		command.Env = gitenv.Environ()
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "a.txt"}, {"commit", "-m", "first"}} {
		command := exec.Command("git", args...)
		command.Dir = root
		command.Env = gitenv.Environ()
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}
