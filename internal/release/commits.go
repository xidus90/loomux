package release

import (
	"fmt"

	"github.com/xidus90/loomux/internal/conventional"
)

var rank = map[string]int{"none": 0, "patch": 1, "minor": 2, "major": 3}

// CheckCommits reports every commit whose header is not Conventional Commits,
// and every commit whose level ranks above the bump the label grants, so a
// feat cannot ship as a patch.
//
// The form is checked here and not only by the commit-msg hook: a malformed
// header asks for no level, so the level check alone passes it under every
// label, and a hook is skipped by anyone who never armed it.
func CheckCommits(bump string, messages []string) []string {
	var problems []string
	for _, msg := range messages {
		lines := conventional.Message(msg)
		if len(lines) > 0 && !conventional.Exempt(lines[0]) {
			if _, err := conventional.Parse(lines[0]); err != nil {
				problems = append(problems, fmt.Sprintf("commit %q: %v", lines[0], err))
				continue
			}
		}
		level := conventional.Level(msg)
		if rank[level] > rank[bump] {
			// A level above none needs a parsed header, so the message has a first line.
			header := conventional.Message(msg)[0]
			problems = append(problems, fmt.Sprintf("commit %q needs at least release:%s, the label is release:%s", header, level, bump))
		}
	}
	return problems
}
