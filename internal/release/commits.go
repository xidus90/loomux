package release

import (
	"fmt"

	"github.com/xidus90/loomux/internal/conventional"
)

var rank = map[string]int{"none": 0, "patch": 1, "minor": 2, "major": 3}

// CheckCommits reports every commit whose Conventional Commits level ranks
// above the bump the label grants, so a feat cannot ship as a patch.
func CheckCommits(bump string, messages []string) []string {
	var problems []string
	for _, msg := range messages {
		level := conventional.Level(msg)
		if rank[level] > rank[bump] {
			// A level above none needs a parsed header, so the message has a first line.
			header := conventional.Message(msg)[0]
			problems = append(problems, fmt.Sprintf("commit %q needs at least release:%s, the label is release:%s", header, level, bump))
		}
	}
	return problems
}
