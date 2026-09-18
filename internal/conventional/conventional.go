// Package conventional reads Conventional Commits headers: whether a header
// has the required form, and which release level a whole message asks for.
// Both the commit-msg hook and the release rules use it.
package conventional

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// Form is the header shape shown to whoever wrote a wrong one.
const Form = "<type>[(<scope>)][!]: <description>, type one of feat, fix, build, chore, ci, docs, style, refactor, perf, test, revert"

var types = map[string]bool{
	"feat": true, "fix": true, "build": true, "chore": true, "ci": true, "docs": true,
	"style": true, "refactor": true, "perf": true, "test": true, "revert": true,
}

// exempt are the headers git writes itself; they carry no type.
var exempt = []string{"Merge ", "fixup! ", "squash! ", "amend! ", `Revert "`}

// headerPattern splits a header into type, optional scope with parentheses,
// the breaking mark and the description. It is compiled on first use, since
// this package loads on every loomux start.
var headerPattern = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^([A-Za-z]+)(\(([^()]*)\))?(!)?: (.*)$`)
})

// Header is a parsed Conventional Commits header.
type Header struct {
	Type     string
	Breaking bool
}

// Exempt reports whether git itself wrote the header.
func Exempt(header string) bool {
	for _, p := range exempt {
		if strings.HasPrefix(header, p) {
			return true
		}
	}
	return false
}

// Parse checks a header and names what is wrong with it.
func Parse(header string) (Header, error) {
	m := headerPattern().FindStringSubmatch(header)
	if m == nil {
		return Header{}, fmt.Errorf("header %q is not of the form %s", header, Form)
	}
	if !types[m[1]] {
		return Header{}, fmt.Errorf("type %q is unknown; expected %s", m[1], Form)
	}
	if m[2] != "" && (m[3] == "" || strings.ContainsAny(m[3], " \t")) {
		return Header{}, fmt.Errorf("scope %q must be non-empty and without spaces; expected %s", m[3], Form)
	}
	if strings.TrimSpace(m[5]) == "" {
		return Header{}, fmt.Errorf("description is empty; expected %s", Form)
	}
	return Header{Type: m[1], Breaking: m[4] == "!"}, nil
}

// Message drops git's comment lines and surrounding blank space and returns
// what remains, split into lines.
func Message(msg string) []string {
	var lines []string
	for _, l := range strings.Split(strings.ReplaceAll(msg, "\r\n", "\n"), "\n") {
		if !strings.HasPrefix(l, "#") {
			lines = append(lines, l)
		}
	}
	trimmed := strings.TrimSpace(strings.Join(lines, "\n"))
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

// Level returns the release level a commit message asks for: major for a
// breaking change, minor for feat, patch for fix and none otherwise,
// including exempt and malformed headers.
func Level(message string) string {
	lines := Message(message)
	if len(lines) == 0 || Exempt(lines[0]) {
		return "none"
	}
	h, err := Parse(lines[0])
	if err != nil {
		return "none"
	}
	breaking := h.Breaking
	for _, l := range lines[1:] {
		if strings.HasPrefix(l, "BREAKING CHANGE:") || strings.HasPrefix(l, "BREAKING-CHANGE:") {
			breaking = true
		}
	}
	switch {
	case breaking:
		return "major"
	case h.Type == "feat":
		return "minor"
	case h.Type == "fix":
		return "patch"
	}
	return "none"
}
