package release

import (
	"fmt"
	"strings"
)

// Parsed is what a pull request says about its release.
type Parsed struct {
	Bump      string `json:"bump"`
	Changelog string `json:"changelog"`
}

var bumps = map[string]string{
	"release:major": "major",
	"release:minor": "minor",
	"release:patch": "patch",
	"release:none":  "none",
}

var headings = map[string]bool{
	"Added": true, "Changed": true, "Deprecated": true,
	"Removed": true, "Fixed": true, "Security": true,
}

// ParseBody checks labels and body of a pull request and returns the bump
// and the changelog block. A label problem or a missing changelog section
// stops the check early; inside the changelog block every problem is
// reported, so an author fixes the entries in one round.
func ParseBody(labels []string, body string) (Parsed, []string) {
	var found []string
	for _, l := range labels {
		if strings.HasPrefix(l, "release:") {
			found = append(found, l)
		}
	}
	if len(found) != 1 {
		return Parsed{}, []string{fmt.Sprintf("exactly one release:* label is required, found %d", len(found))}
	}
	bump, ok := bumps[found[0]]
	if !ok {
		return Parsed{}, []string{fmt.Sprintf("unknown release label %q", found[0])}
	}
	if bump == "none" {
		return Parsed{Bump: bump}, nil
	}
	block, ok := changelogBlock(body)
	if !ok {
		return Parsed{}, []string{`the body has no "## Changelog" section`}
	}
	var problems []string
	entries, inSection := 0, false
	for _, line := range block {
		switch {
		case strings.HasPrefix(line, "### "):
			if !headings[strings.TrimSpace(line[4:])] {
				problems = append(problems, fmt.Sprintf("changelog heading %q is not one of Added, Changed, Deprecated, Removed, Fixed, Security", line))
			}
			inSection = true
		case strings.HasPrefix(line, "- "):
			if !inSection {
				problems = append(problems, fmt.Sprintf("changelog entry %q stands before any ### heading", line))
				continue
			}
			entries++
		case strings.TrimSpace(line) != "":
			problems = append(problems, fmt.Sprintf("changelog line %q is neither a ### heading nor a \"- \" entry", line))
		}
	}
	if entries == 0 {
		problems = append(problems, "the changelog has no entry")
	}
	if len(problems) > 0 {
		return Parsed{}, problems
	}
	return Parsed{Bump: bump, Changelog: strings.TrimSpace(strings.Join(block, "\n")) + "\n"}, nil
}

// changelogBlock returns the lines after "## Changelog" up to the next
// second-level heading. GitHub stores bodies with CRLF when they were typed
// in the browser, so line ends are normalised first.
func changelogBlock(body string) ([]string, bool) {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "## Changelog" {
			continue
		}
		rest := lines[i+1:]
		for j, l := range rest {
			if strings.HasPrefix(l, "## ") {
				return rest[:j], true
			}
		}
		return rest, true
	}
	return nil, false
}
