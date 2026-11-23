package guard

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/setup/hostfile"
)

// The hook entries init writes are kept by hand, apart from writingTools. A
// writing tool missing from the PreToolUse matchers is never shown to the
// barrier, and one missing from the PostToolUse matchers is never checked
// after it wrote.
func TestEveryWritingToolIsInThePreAndPostToolUseMatchers(t *testing.T) {
	matched := map[string][]string{}
	for _, host := range []hosts.Host{hosts.HostClaude, hosts.HostAntigravity, hosts.HostDsh} {
		for _, entry := range hostfile.Entries(host, hostfile.Checkout) {
			matched[entry.Event] = append(matched[entry.Event], strings.Split(entry.Matcher, "|")...)
		}
	}
	// DSH gates every tool with the regular expression .*, which therefore
	// carries every name; the other hosts list their tools by hand.
	covers := func(event, name string) bool {
		return slices.Contains(matched[event], ".*") || slices.Contains(matched[event], name)
	}
	for name := range writingTools {
		for _, event := range []string{"PreToolUse", "PostToolUse"} {
			if !covers(event, name) {
				t.Errorf("%s is in writingTools but in no %s matcher (%v)", name, event, matched[event])
			}
		}
	}
}
