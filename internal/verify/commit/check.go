package commit

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xidus90/loomux/internal/conventional"
)

// prefix names the command in every line it prints, so a refusal read in a
// git hook says which gate wrote it.
const prefix = "loomux check commit-msg: "

const wayOut = "Rewrite it, or use `git commit --no-verify` if this cannot wait. The next\ncommit runs this check again."

// Check validates that a commit message is non-empty, written in the policy's
// language, and (if policy.Conventional is true) starts with a valid
// Conventional Commits header.
func Check(msg string, policy Policy) error {
	lines := conventional.Message(msg)
	if lines == nil {
		return errors.New(prefix + "commit message cannot be empty")
	}

	findings := Scan(msg, policy.Language, policy.Threshold, policy.Allow)

	var headerErr error
	if policy.Conventional {
		firstLine := lines[0]
		if !conventional.Exempt(firstLine) {
			_, headerErr = conventional.Parse(firstLine)
		}
	}

	if len(findings) == 0 && headerErr == nil {
		return nil
	}

	var parts []string
	if len(findings) > 0 {
		parts = append(parts, formatRefusal(findings, policy.Language))
	}
	if headerErr != nil {
		parts = append(parts, prefix+headerErr.Error())
	}
	return errors.New(strings.Join(parts, "\n"))
}

func formatRefusal(findings []Finding, lang string) string {
	target := "English"
	other := "German"
	if lang == "de" {
		target = "German"
		other = "English"
	}

	var b strings.Builder
	fmt.Fprintf(&b, prefix+"this message reads as %s, and commits here are %s.\n", other, target)
	for _, f := range findings {
		label := fmt.Sprintf("  line %d: ", f.LineNumber)
		fmt.Fprintf(&b, "%s%s\n", label, f.Line)
		fmt.Fprintf(&b, "%shits: %s\n", strings.Repeat(" ", len(label)), strings.Join(f.Hits, ", "))
	}
	b.WriteString(wayOut)
	return b.String()
}
