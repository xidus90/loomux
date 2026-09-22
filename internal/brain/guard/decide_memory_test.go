package guard

import (
	"path/filepath"
	"strings"
	"testing"
)

// claudeMemory is one project's Claude Code memory below a test's home.
func claudeMemory(home string) string {
	return filepath.Join(home, ".claude", "projects", "C--work-demo", "memory")
}

func TestClaudeCodeMemoryPassesBesideAWritableWiki(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	homeAt(t, home, "")
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	allow(t, writeCall(filepath.Join(claudeMemory(home), "a.md")), state)
	allow(t, writeCall(filepath.Join(claudeMemory(home), "MEMORY.md")), state)
}

func TestMemoryPassesWhenTheRegistryOpensNothing(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	homeAt(t, home, "")
	state := registryOf(t, tmp, "")
	allow(t, writeCall(filepath.Join(claudeMemory(home), "a.md")), state)
}

func TestMemoryPassesWhenTheRegistryCannotBeRead(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	homeAt(t, home, "")
	state := filepath.Join(tmp, "state")
	write(t, filepath.Join(state, "registry.toml"), "[[area]\n")
	allow(t, writeCall(filepath.Join(claudeMemory(home), "a.md")), state)
	allow(t, writeCall(filepath.Join(home, ".gemini", "antigravity", "knowledge", "k.md")), state)
}

func TestAntigravityMemoryPasses(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	homeAt(t, home, "")
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	allow(t, writeCall(filepath.Join(home, ".gemini", "antigravity-cli", "knowledge", "k.md")), state)
	allow(t, writeCall(filepath.Join(home, ".gemini", "antigravity-ide", "brain", "0363ae1a", "task.md")), state)
}

func TestTheRestOfTheAgentsHomesStaysShut(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	homeAt(t, home, "")
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	for _, relative := range []string{
		".claude/settings.json",
		".claude/projects/C--work-demo/t.jsonl",
		".claude/projects/memory/a.md",
		".claude/projects/C--work-demo/q/memory/a.md",
		".claude/projects/C--work-demo/memory-alt/a.md",
		".gemini/antigravity-backup/knowledge/k.md",
		".gemini/antigravity/brain/t.md",
		".gemini/antigravity-cli/bin/agy.exe",
		".gemini/antigravity-cli/mcp/x/config.json",
	} {
		t.Run(relative, func(t *testing.T) {
			deny(t, writeCall(filepath.Join(home, filepath.FromSlash(relative))), state,
				"lies outside every writable tree")
		})
	}
}

func TestAManifestInsideMemoryIsStillRefused(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	homeAt(t, home, "")
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	deny(t, writeCall(filepath.Join(claudeMemory(home), ".loomux", "config.toml")), state,
		"the manifest is where the barrier reads its own limits")
}

func TestALinkOutOfMemoryIsRefused(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	homeAt(t, home, "")
	outside := filepath.Join(tmp, "repo", "src")
	mkdir(t, outside)
	mkdir(t, claudeMemory(home))
	linkDir(t, outside, filepath.Join(claudeMemory(home), "j"))
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	deny(t, writeCall(filepath.Join(claudeMemory(home), "j", "a.py")), state,
		"lies outside every writable tree")
}

func TestAMixedCallIsJudgedForItsTargetOutsideMemory(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	homeAt(t, home, "")
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	outside := filepath.Join(tmp, "repo", "a.py")
	payload := map[string]any{
		"tool_name": "NotebookEdit",
		"tool_input": map[string]any{
			"file_path":     filepath.Join(claudeMemory(home), "a.md"),
			"notebook_path": outside,
		},
	}
	reason := deny(t, payload, state, "lies outside every writable tree")
	resolved, err := ResolvePath(outside)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(reason, resolved+" lies outside") {
		t.Fatalf("the refusal does not open with the outside target: %q", reason)
	}
	if strings.Contains(reason, filepath.Join("memory", "a.md")) {
		t.Fatalf("the refusal names the memory target: %q", reason)
	}
}

func TestARefusalNamesTheMemoryTrees(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	homeAt(t, home, "")
	state := registryOf(t, tmp, filepath.Join(tmp, "vault", "demo"))
	reason := deny(t, writeCall(filepath.Join(tmp, "repo", "a.py")), state,
		", plus the agents' memory below: ")
	claude, antigravity := memoryBases()
	if !strings.HasSuffix(reason, ", plus the agents' memory below: "+memoryShown(claude, antigravity)) {
		t.Fatalf("the refusal does not end with the memory trees: %q", reason)
	}
}
