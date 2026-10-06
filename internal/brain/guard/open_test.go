package guard

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// openAt writes an open.toml into state that lists files, each in slash form.
func openAt(t *testing.T, state string, files ...string) {
	t.Helper()
	quoted := make([]string, len(files))
	for i, file := range files {
		quoted[i] = "'" + posix(file) + "'"
	}
	write(t, filepath.Join(state, openName), "files = ["+strings.Join(quoted, ", ")+"]\n")
}

func TestAFileOpenTomlListsPassesBesideAWritableWiki(t *testing.T) {
	tmp := t.TempDir()
	homeAt(t, filepath.Join(tmp, "home"), "")
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	learnings := filepath.Join(tmp, "home", ".claude", "AGENT_LEARNINGS.md")
	openAt(t, state, learnings)
	allow(t, writeCall(learnings), state)
}

// Open on the terms memory is: a registry that opens nothing, or that cannot
// be read at all, closes nothing open.toml opens.
func TestAFileOpenTomlListsPassesWhateverTheRegistrySays(t *testing.T) {
	tmp := t.TempDir()
	homeAt(t, filepath.Join(tmp, "home"), "")
	learnings := filepath.Join(tmp, "home", ".claude", "AGENT_LEARNINGS.md")
	state := registryOf(t, tmp, "")
	openAt(t, state, learnings)
	allow(t, writeCall(learnings), state)
	write(t, filepath.Join(state, "registry.toml"), "[[area]\n")
	allow(t, writeCall(learnings), state)
}

// Only the file opens, not its directory and not a neighbour.
func TestOpenTomlOpensNothingBesideTheFile(t *testing.T) {
	tmp := t.TempDir()
	homeAt(t, filepath.Join(tmp, "home"), "")
	claude := filepath.Join(tmp, "home", ".claude")
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	openAt(t, state, filepath.Join(claude, "AGENT_LEARNINGS.md"))
	for _, name := range []string{"CLAUDE.md", "settings.json", "AGENT_LEARNINGS.md.bak", filepath.Join("archive", "AGENT_LEARNINGS.md")} {
		deny(t, writeCall(filepath.Join(claude, name)), state, "lies outside every writable tree")
	}
}

// In a call that also writes elsewhere, the open file is not what is refused.
func TestAMixedCallIsJudgedForItsTargetOutsideTheOpenFiles(t *testing.T) {
	tmp := t.TempDir()
	homeAt(t, filepath.Join(tmp, "home"), "")
	learnings := filepath.Join(tmp, "home", ".claude", "AGENT_LEARNINGS.md")
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	openAt(t, state, learnings)
	outside := filepath.Join(tmp, "elsewhere", "a.md")
	reason := deny(t, map[string]any{
		"tool_name":  "MultiEdit",
		"tool_input": map[string]any{"file_path": outside, "notebook_path": learnings},
	}, state, "lies outside every writable tree")
	if strings.HasPrefix(reason, learnings) || strings.Contains(reason, learnings+" lies") {
		t.Fatalf("the refusal names the open file as refused: %q", reason)
	}
}

// open.toml is read whole or not at all, and a refusal says why it was not.
func TestABrokenOpenTomlOpensNothingAndSaysWhy(t *testing.T) {
	tmp := t.TempDir()
	homeAt(t, filepath.Join(tmp, "home"), "")
	learnings := filepath.Join(tmp, "home", ".claude", "AGENT_LEARNINGS.md")
	directory := filepath.Join(tmp, "home", ".claude")
	mkdir(t, directory)
	for name, tc := range map[string]struct{ body, want string }{
		"not TOML":         {"files = [\n", "not valid TOML"},
		"unknown key":      {"files = ['" + posix(learnings) + "']\ndirs = ['x']\n", `unknown key "dirs"`},
		"not a string":     {"files = [3]\n", "not valid TOML"},
		"relative":         {"files = ['AGENT_LEARNINGS.md']\n", `files #1 "AGENT_LEARNINGS.md" is not an absolute path`},
		"a directory":      {"files = ['" + posix(directory) + "']\n", "names a directory; only single files open"},
		"in the state dir": {"files = ['" + posix(filepath.Join(tmp, "state", "registry.toml")) + "']\n", "lies in the state directory"},
	} {
		t.Run(name, func(t *testing.T) {
			state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
			write(t, filepath.Join(state, openName), tc.body)
			reason := deny(t, writeCall(learnings), state, "lies outside every writable tree")
			if !strings.Contains(reason, filepath.Join(state, openName)+" is ignored: ") || !strings.Contains(reason, tc.want) {
				t.Fatalf("the refusal does not say why open.toml is ignored (%s): %q", tc.want, reason)
			}
		})
	}
}

// An open.toml that cannot be read is ignored like one that cannot be used.
func TestAnUnreadableOpenTomlOpensNothingAndSaysWhy(t *testing.T) {
	tmp := t.TempDir()
	homeAt(t, filepath.Join(tmp, "home"), "")
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	mkdir(t, filepath.Join(state, openName))
	deny(t, writeCall(filepath.Join(tmp, "home", ".claude", "AGENT_LEARNINGS.md")), state,
		filepath.Join(state, openName)+" is ignored: ")
}

// An entry whose kind cannot be read is not an entry the barrier can vouch
// for: the file is ignored whole, not told apart from one not yet written.
func TestAnOpenTomlEntryWhoseKindCannotBeReadOpensNothing(t *testing.T) {
	tmp := t.TempDir()
	homeAt(t, filepath.Join(tmp, "home"), "")
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	learnings := filepath.Join(tmp, "home", ".claude", "AGENT_LEARNINGS.md")
	kind := filepath.Join(tmp, "elsewhere", "kind.md")
	openAt(t, state, learnings, kind)
	old := statEntry
	t.Cleanup(func() { statEntry = old })
	statEntry = func(path string) (fs.FileInfo, error) {
		if filepath.Base(path) == "kind.md" {
			return nil, fmt.Errorf("stat %s: %w", path, fs.ErrPermission)
		}
		return os.Stat(path)
	}
	reason := deny(t, writeCall(learnings), state, filepath.Join(state, openName)+" is ignored: ")
	for _, want := range []string{`files #2 "`, "permission denied"} {
		if !strings.Contains(reason, want) {
			t.Fatalf("the refusal does not carry %q: %q", want, reason)
		}
	}
}

// A link cycle is the unelevated way to a path that does not resolve: once as
// the listed file, once as the state directory itself.
func TestAnOpenTomlWhosePathsDoNotResolveOpensNothing(t *testing.T) {
	if filepath.Separator != '\\' {
		t.Skip("mklink is the unelevated way to build a link cycle")
	}
	tmp := t.TempDir()
	homeAt(t, filepath.Join(tmp, "home"), "")
	loop := filepath.Join(tmp, "loop")
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", loop, loop).CombinedOutput(); err != nil {
		t.Skipf("no junction on this machine: %v (%s)", err, out)
	}
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	openAt(t, state, filepath.Join(loop, "AGENT_LEARNINGS.md"))
	deny(t, writeCall(filepath.Join(tmp, "repo", "a.py")), state, `files #1 "`)
	if files, err := openFiles(loop); files != nil || err == nil {
		t.Fatalf("openFiles over a state directory that loops = %v, %v; want an error", files, err)
	}
}

func TestARefusalNamesTheOpenFiles(t *testing.T) {
	tmp := t.TempDir()
	homeAt(t, filepath.Join(tmp, "home"), "")
	learnings := filepath.Join(tmp, "home", ".claude", "AGENT_LEARNINGS.md")
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	openAt(t, state, learnings)
	resolved, err := ResolvePath(learnings)
	if err != nil {
		t.Fatal(err)
	}
	deny(t, writeCall(filepath.Join(tmp, "repo", "a.py")), state,
		", plus the files open.toml opens: "+resolved)
}

func TestARefusalWithoutWritableTreesNamesTheOpenFiles(t *testing.T) {
	tmp := t.TempDir()
	homeAt(t, filepath.Join(tmp, "home"), "")
	scratchpadAt(t)
	learnings := filepath.Join(tmp, "home", ".claude", "AGENT_LEARNINGS.md")
	state := registryOf(t, tmp, "")
	openAt(t, state, learnings)
	deny(t, writeCall(filepath.Join(tmp, "out.md")), state,
		"so nothing outside the agents' memory, the files open.toml opens and the session scratchpad below: ")
	write(t, filepath.Join(state, openName), "files = [\n")
	deny(t, writeCall(filepath.Join(tmp, "out.md")), state, openName+" is ignored: ")
}
