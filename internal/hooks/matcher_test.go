package hooks

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/setup/hostfile"
)

// The hook entries init writes are kept by hand, apart from the table the
// guard judges by. A command tool missing from every PreToolUse matcher is
// never shown to the guard, and its lines run unjudged.
func TestEveryCommandToolIsInAPreToolUseMatcher(t *testing.T) {
	var matched []string
	for _, host := range []hosts.Host{hosts.HostClaude, hosts.HostAntigravity} {
		for _, entry := range hostfile.Entries(host, hostfile.Checkout) {
			if entry.Event == "PreToolUse" {
				matched = append(matched, strings.Split(entry.Matcher, "|")...)
			}
		}
	}
	for name := range commandTools {
		if !slices.Contains(matched, name) {
			t.Errorf("%s is in commandTools but in no PreToolUse matcher (%v)", name, matched)
		}
	}
}
