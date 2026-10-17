package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// A hard link is the file itself under a second name, so no rule on the
// name sees the manifest behind notes.txt; the link count does.
func TestAHardLinkToTheManifestIsRefused(t *testing.T) {
	root := project(t)
	link := filepath.Join(root, "notes.txt")
	if err := os.Link(filepath.Join(root, ".loomux", "config.toml"), link); err != nil {
		t.Skipf("no hard link here: %v", err)
	}
	for tool, input := range map[string]map[string]any{
		"Write": {"file_path": "notes.txt"},
		"Edit":  {"file_path": link},
		"Bash":  command("echo x > notes.txt"),
	} {
		for _, policy := range []config.Policy{{}, strictPolicy} {
			got := checkTool(root, tool, input, policy)
			if len(got) != 1 || !strings.Contains(got[0], "has 2 links") {
				t.Fatalf("%s, strict %v: reasons %q", tool, policy.Strict, got)
			}
		}
	}
	// Removing a name leaves the other alone.
	if got := checkTool(root, "Bash", command("rm notes.txt"), config.Policy{}); len(got) != 0 {
		t.Fatalf("rm: reasons %q", got)
	}
}
