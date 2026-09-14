package guard

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestScratchpadOfAClaudeSessionIsOpen(t *testing.T) {
	temp := t.TempDir()
	tempDir = func() string { return temp }
	defer func() { tempDir = os.TempDir }()
	file := filepath.Join(temp, "claude", "C--project", "0b1c-session", "scratchpad", "notes.txt")
	os.MkdirAll(filepath.Dir(file), 0o755)
	base := scratchpadBase()
	resolved, err := resolvePath(file)
	if err != nil {
		t.Fatal(err)
	}
	if !isScratchpad(resolved, base) {
		t.Fatalf("%s not recognised below %s", resolved, base)
	}
}

func TestOnlyTheScratchpadDirectoryIsOpen(t *testing.T) {
	temp := t.TempDir()
	tempDir = func() string { return temp }
	defer func() { tempDir = os.TempDir }()
	base := scratchpadBase()
	for _, rel := range []string{
		filepath.Join("claude", "C--project", "0b1c-session", "tasks", "x.output"),
		filepath.Join("claude", "C--project", "scratchpad", "x"),
		filepath.Join("claude", "C--project", "0b1c-session", "scratchpad"),
		filepath.Join("other", "C--project", "0b1c-session", "scratchpad", "x"),
	} {
		path := filepath.Join(temp, rel)
		os.MkdirAll(filepath.Dir(path), 0o755)
		resolved, _ := resolvePath(path)
		if isScratchpad(resolved, base) {
			t.Fatalf("%s must stay closed", rel)
		}
	}
}

func TestScratchpadBaseIsEmptyWhenTempIsNotAbsolute(t *testing.T) {
	tempDir = func() string { return "relative" }
	defer func() { tempDir = os.TempDir }()
	if scratchpadBase() != "" {
		t.Fatal("a relative temp directory must open nothing")
	}
}

func TestAnEmptyScratchpadBaseOpensNothing(t *testing.T) {
	// Where temp names no base, a path that looks like a scratchpad is still
	// no scratchpad: the empty base would otherwise be a prefix of every path.
	target := filepath.Join(strangeVolume(), "claude", "p", "s", "scratchpad", "x")
	if isScratchpad(target, "") {
		t.Fatal("an empty base opened a scratchpad")
	}
}

func TestAScratchpadBaseThatCannotBeResolvedIsLeftOut(t *testing.T) {
	if filepath.Separator != '\\' {
		t.Skip("mklink is the unelevated way to build a link cycle")
	}
	temp := t.TempDir()
	tempDir = func() string { return temp }
	defer func() { tempDir = os.TempDir }()
	loop := filepath.Join(temp, "claude")
	made := exec.Command("cmd", "/c", "mklink", "/J", loop, loop)
	if out, err := made.CombinedOutput(); err != nil {
		t.Skipf("no junction on this machine: %v (%s)", err, out)
	}
	if base := scratchpadBase(); base != "" {
		t.Fatalf("an unresolvable base was kept: %q", base)
	}
}

// scratchpadAt points the temp directory at a fresh tree for one test and
// answers the scratchpad of one session inside it.
func scratchpadAt(t *testing.T) (temp, scratchpad string) {
	t.Helper()
	temp = t.TempDir()
	old := tempDir
	tempDir = func() string { return temp }
	t.Cleanup(func() { tempDir = old })
	session := filepath.Join(temp, "claude", "C--work-demo", "0b1c-session")
	return temp, filepath.Join(session, "scratchpad")
}

func TestDecideLetsTheSessionScratchpadThrough(t *testing.T) {
	tmp := t.TempDir()
	_, scratchpad := scratchpadAt(t)
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	allow(t, writeCall(filepath.Join(scratchpad, "notes.txt")), state)
	allow(t, writeCall(filepath.Join(scratchpad, "deeper", "x.md")), state)
	deny(t, writeCall(filepath.Join(filepath.Dir(scratchpad), "tasks", "x.output")),
		state, "lies outside every writable tree")
}

func TestTheScratchpadPassesWhenTheRegistryCannotBeRead(t *testing.T) {
	tmp := t.TempDir()
	_, scratchpad := scratchpadAt(t)
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"), "[[area]\n")
	allow(t, writeCall(filepath.Join(scratchpad, "notes.txt")), state)
}

func TestAMixedCallIsJudgedForItsTargetOutsideTheScratchpad(t *testing.T) {
	tmp := t.TempDir()
	_, scratchpad := scratchpadAt(t)
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	payload := map[string]any{
		"tool_name": "NotebookEdit",
		"tool_input": map[string]any{
			"file_path":     filepath.Join(scratchpad, "a.md"),
			"notebook_path": filepath.Join(tmp, "repo", "a.py"),
		},
	}
	reason := deny(t, payload, state, "lies outside every writable tree")
	if strings.Contains(reason, filepath.Join("scratchpad", "a.md")) {
		t.Fatalf("the refusal names the scratchpad target: %q", reason)
	}
}

func TestAManifestInsideTheScratchpadIsStillRefused(t *testing.T) {
	tmp := t.TempDir()
	_, scratchpad := scratchpadAt(t)
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	deny(t, writeCall(filepath.Join(scratchpad, ".loomux", "config.toml")), state,
		"the manifest is where the barrier reads its own limits")
}

func TestARefusalNamesTheScratchpadTree(t *testing.T) {
	tmp := t.TempDir()
	scratchpadAt(t)
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	reason := deny(t, writeCall(filepath.Join(tmp, "repo", "a.py")), state,
		"lies outside every writable tree")
	want := ", plus the session scratchpad below: " +
		filepath.Join(scratchpadBase(), "*", "*", "scratchpad")
	if !strings.Contains(reason, want) {
		t.Fatalf("the refusal does not name the scratchpad: %q", reason)
	}
}
