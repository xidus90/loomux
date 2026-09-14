package guard

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// homeAt points the memory trees at a home and a Claude config directory the
// test owns, so no test here ever looks at the machine's own ~/.claude.
func homeAt(t *testing.T, home, config string) {
	t.Helper()
	oldHome, oldConfig := userHome, claudeConfigDir
	userHome = func() (string, error) { return home, nil }
	claudeConfigDir = func() string { return config }
	t.Cleanup(func() { userHome, claudeConfigDir = oldHome, oldConfig })
}

// noHome is a machine where the home directory cannot be named.
func noHome(t *testing.T, config string) {
	t.Helper()
	oldHome, oldConfig := userHome, claudeConfigDir
	userHome = func() (string, error) { return "", errors.New("no home") }
	claudeConfigDir = func() string { return config }
	t.Cleanup(func() { userHome, claudeConfigDir = oldHome, oldConfig })
}

// memoryAt resolves home-relative slash paths and asks isMemory about them.
func memoryAt(t *testing.T, home, relative string) bool {
	t.Helper()
	resolved, err := resolvePath(filepath.Join(home, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	claude, antigravity := memoryBases()
	return isMemory(resolved, claude, antigravity)
}

func TestTheAgentsMemoryIsRecognisedAndNothingBesideIt(t *testing.T) {
	home := t.TempDir()
	homeAt(t, home, "")
	cases := []struct {
		relative string
		want     bool
	}{
		{".claude/projects/C--work-demo/memory/a.md", true},
		{".claude/projects/C--work-demo/memory/MEMORY.md", true},
		{".claude/projects/C--work-demo/memory/sub/b.md", true},
		{".claude/projects/C--work-demo/memory", false},
		{".claude/settings.json", false},
		{".claude/CLAUDE.md", false},
		{".claude/plugins/x/hooks.json", false},
		{".claude/projects/C--work-demo/t.jsonl", false},
		{".claude/projects/memory/a.md", false},
		{".claude/projects/C--work-demo/q/memory/a.md", false},
		{".claude/projects/C--work-demo/memory-alt/a.md", false},
		{".gemini/antigravity/knowledge/k.md", true},
		{".gemini/antigravity-cli/knowledge/k.md", true},
		{".gemini/antigravity-ide/brain/0363ae1a/task.md", true},
		{".gemini/antigravity-ide/brain/0363ae1a/scratch/x.txt", true},
		{".gemini/antigravity/knowledge", false},
		{".gemini/antigravity/brain/t.md", false},
		{".gemini/antigravity-backup/knowledge/k.md", false},
		{".gemini/antigravity-cli/bin/agy.exe", false},
		{".gemini/antigravity-cli/mcp/x/config.json", false},
		{".gemini/antigravity-ide/conversations/0363ae1a/x.pb", false},
		{".gemini/antigravity-cli/settings.json", false},
		{".claude/projects-old/C--work-demo/memory/a.md", false},
		{".gemini/settings.json", false},
		{".gemini/config/plugins/x/rules/r.md", false},
	}
	for _, c := range cases {
		t.Run(c.relative, func(t *testing.T) {
			if got := memoryAt(t, home, c.relative); got != c.want {
				t.Fatalf("isMemory(%s) = %v, want %v", c.relative, got, c.want)
			}
		})
	}
}

func TestClaudeConfigDirMovesClaudeCodesMemory(t *testing.T) {
	home := t.TempDir()
	config := t.TempDir()
	homeAt(t, home, config)
	moved, err := resolvePath(filepath.Join(config, "projects", "p", "memory", "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	claude, antigravity := memoryBases()
	if !isMemory(moved, claude, antigravity) {
		t.Fatal("the memory below CLAUDE_CONFIG_DIR is not recognised")
	}
	if memoryAt(t, home, ".claude/projects/p/memory/a.md") {
		t.Fatal("~/.claude still counts while CLAUDE_CONFIG_DIR points elsewhere")
	}
	if !memoryAt(t, home, ".gemini/antigravity/knowledge/k.md") {
		t.Fatal("Antigravity's memory hangs on the home, not on CLAUDE_CONFIG_DIR")
	}
}

func TestWithoutAHomeOnlyAClaudeConfigDirCanNameMemory(t *testing.T) {
	somewhere := t.TempDir()
	noHome(t, "")
	claude, antigravity := memoryBases()
	if claude != "" || len(antigravity) != 0 || memoryShown(claude, antigravity) != "" {
		t.Fatalf("no home and no config named trees: %q %v", claude, antigravity)
	}
	target, err := resolvePath(filepath.Join(somewhere, ".claude", "projects", "p", "memory", "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if isMemory(target, claude, antigravity) {
		t.Fatal("an empty base opened a path")
	}
	bare := filepath.VolumeName(somewhere) + string(filepath.Separator) +
		filepath.Join("memory", "a.md")
	if isMemory(bare, "", nil) {
		t.Fatal("an empty Claude base matched a path whose second component is memory")
	}

	config := t.TempDir()
	noHome(t, config)
	claude, antigravity = memoryBases()
	moved, err := resolvePath(filepath.Join(config, "projects", "p", "memory", "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !isMemory(moved, claude, antigravity) || len(antigravity) != 0 {
		t.Fatal("CLAUDE_CONFIG_DIR alone should name Claude Code's memory and nothing else")
	}
}

func TestARelativeHomeOrConfigNamesNoMemory(t *testing.T) {
	// A relative value would be anchored at the hook's working directory --
	// whatever repository the hook happens to run in -- and open a tree there
	// that is no agent's memory.
	homeAt(t, "relative-home", "relative-config")
	claude, antigravity := memoryBases()
	if claude != "" || len(antigravity) != 0 {
		t.Fatalf("a relative base named trees: %q %v", claude, antigravity)
	}
}

func TestABaseThatCannotBeResolvedIsLeftOut(t *testing.T) {
	if filepath.Separator != '\\' {
		t.Skip("mklink is the unelevated way to build a link cycle")
	}
	home := t.TempDir()
	homeAt(t, home, "")
	for _, loop := range []string{
		filepath.Join(home, ".claude"),
		filepath.Join(home, ".gemini"),
	} {
		made := exec.Command("cmd", "/c", "mklink", "/J", loop, loop)
		if out, err := made.CombinedOutput(); err != nil {
			t.Skipf("no junction on this machine: %v (%s)", err, out)
		}
	}
	claude, antigravity := memoryBases()
	if claude != "" || len(antigravity) != 0 {
		t.Fatalf("an unresolvable base was kept: %q %v", claude, antigravity)
	}
}

func TestMemoryShownNamesEveryTree(t *testing.T) {
	home := t.TempDir()
	homeAt(t, home, "")
	claude, antigravity := memoryBases()
	shown := memoryShown(claude, antigravity)
	want := []string{filepath.Join(claude, "*", "memory")}
	for _, root := range antigravity {
		want = append(want, filepath.Join(root, "knowledge"), filepath.Join(root, "brain", "*"))
	}
	if shown != strings.Join(want, ", ") {
		t.Fatalf("memoryShown = %q, want %q", shown, strings.Join(want, ", "))
	}
	if len(antigravity) != 3 {
		t.Fatalf("expected three Antigravity roots, found %v", antigravity)
	}
}
