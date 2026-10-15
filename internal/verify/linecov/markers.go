package linecov

import (
	"path"
	"regexp"
	"slices"
	"strings"
)

// exemption is what makes a line exempt: loomux's own marker (own), or a
// tool's native one; reason says whether the own marker carries one.
type exemption struct {
	own    bool
	reason bool
}

// Scan is what one source file says about its exemptions.
type Scan struct {
	File       bool // the file carries the own file marker
	FileReason bool
	NativeFile bool // a tool's whole-file marker such as "istanbul ignore file"
	Lines      map[int]exemption
	Findings   []Finding
}

var ownMarker = regexp.MustCompile(`coverage:exempt(-file)?([^\w-].*)?$`)

// commentTokens start a comment in the languages the profiles cover;
// longer tokens come before their prefixes.
var commentTokens = []string{"<!--", "//", "/**", "/*", "#", "--", ";", "'", "%", "*"}

// operatorTokens are the comment tokens that code uses as operators too.
var operatorTokens = []string{"--", ";", "'", "%", "*"}

// ScanSource reads the exemptions of one file. A marker on a line that holds
// nothing but a comment covers the line below it as well. Native markers
// apply only to files of their profile's languages.
func ScanSource(rel string, src []byte, natives []Marker) Scan {
	ext := path.Ext(rel)
	natives = slices.DeleteFunc(slices.Clone(natives), func(m Marker) bool { return !slices.Contains(m.Extensions, ext) })
	lines := strings.Split(text(src), "\n")
	s := Scan{Lines: map[int]exemption{}}
	openAt, openLabel := 0, ""
	for i, line := range lines {
		line = strings.TrimRight(line, "\r")
		n := i + 1
		if loc := ownMarker.FindStringSubmatchIndex(line); loc != nil && opensComment(line, loc[0]) {
			reason := loc[4] >= 0 && cleanReason(line[loc[4]:loc[5]]) != ""
			if loc[2] >= 0 {
				s.File, s.FileReason = true, s.FileReason || reason
				if !reason {
					s.Findings = append(s.Findings, Finding{rel, 0, "coverage:exempt-file without a reason"})
				}
			} else {
				t := target(line, loc[0], false, "", n)
				s.Lines[t] = exemption{own: true, reason: reason || s.Lines[t].reason}
			}
		}
		for _, mk := range natives {
			loc := mk.Re.FindStringSubmatchIndex(line)
			if loc == nil || !isCode(line, loc) && !opensComment(line, loc[0]) {
				continue
			}
			if !mk.NoReason && reasonOf(mk, line, loc) == "" && !commentAbove(lines, i, natives) {
				s.Findings = append(s.Findings, Finding{rel, n, mk.Label + " without a reason"})
			}
			switch mk.Block {
			case "":
				s.native(n)
				s.native(target(line, loc[0], isCode(line, loc), line[loc[1]:], n))
			case "file":
				s.NativeFile = true
			case "start":
				if openAt == 0 {
					openAt, openLabel = n, mk.Label
				}
			case "stop":
				for l := openAt; openAt > 0 && l <= n; l++ {
					s.native(l)
				}
				openAt = 0
			}
		}
	}
	if openAt > 0 {
		s.Findings = append(s.Findings, Finding{rel, openAt, openLabel + " without a stop"})
		for l := openAt; l <= len(lines); l++ {
			s.native(l)
		}
	}
	return s
}

// native marks a line exempt by a tool's marker unless loomux's own marker
// already rules it: only the own marker is refuted by a hit.
func (s Scan) native(n int) {
	if _, ok := s.Lines[n]; !ok {
		s.Lines[n] = exemption{}
	}
}

// target is the line a marker at byte offset at applies to: the line below
// when only one comment opener precedes it, or when it is a code marker
// standing alone on its line; otherwise its own line.
func target(line string, at int, code bool, rest string, n int) int {
	before := strings.TrimSpace(line[:at])
	if slices.Contains(commentTokens, before) || (code && before == "" && (strings.TrimSpace(rest) == "" || commentOnly(rest))) {
		return n + 1
	}
	return n
}

// opensComment says whether the comment marker at offset at begins a
// comment: it opens one itself, or only blanks stand between it and the
// nearest opener before it. A token that code uses as an operator counts
// as that opener only at the start of the line, so a quote before a marker
// reads as a string.
func opensComment(line string, at int) bool {
	before := strings.TrimRight(line[:at], " \t")
	for _, tok := range commentTokens {
		if strings.HasPrefix(line[at:], tok) {
			return true
		}
		if strings.HasSuffix(before, tok) && (!slices.Contains(operatorTokens, tok) || strings.TrimSpace(before) == tok) {
			return true
		}
	}
	return false
}

// isCode says whether a marker is code itself, as an attribute such as
// [ExcludeFromCodeCoverage] or #[coverage(off)] is, rather than comment text:
// attribute markers are bracketed and end in a closing bracket.
func isCode(line string, loc []int) bool {
	return strings.HasSuffix(line[loc[0]:loc[1]], "]")
}

// reasonOf is the reason a marker carries: its own reason group, or the
// comment text after it up to the comment's closer. After a code marker only
// a trailing comment counts, never the code that follows.
func reasonOf(mk Marker, line string, loc []int) string {
	if i := mk.Re.SubexpIndex("reason"); i > 0 && loc[2*i] >= 0 {
		if r := cleanReason(line[loc[2*i]:loc[2*i+1]]); r != "" {
			return r
		}
	}
	rest := line[loc[1]:]
	if isCode(line, loc) && !commentOnly(rest) {
		return ""
	}
	return cleanReason(rest)
}

// commentAbove says whether the line above holds a comment with text, which
// counts as the reason of a native marker that cannot carry one itself.
// A comment that is itself a native marker is no reason.
func commentAbove(lines []string, i int, natives []Marker) bool {
	if i == 0 || !commentOnly(lines[i-1]) || ownMarker.MatchString(lines[i-1]) || cleanReason(strings.TrimRight(lines[i-1], "\r")) == "" {
		return false
	}
	for _, mk := range natives {
		if mk.Re.MatchString(lines[i-1]) {
			return false
		}
	}
	return true
}

func commentOnly(line string) bool {
	t := strings.TrimSpace(line)
	for _, tok := range commentTokens {
		if !strings.HasPrefix(t, tok) {
			continue
		}
		// The short tokens are also operators, so they open a comment only
		// when a blank or the line end follows.
		rest := t[len(tok):]
		if !slices.Contains(operatorTokens, tok) || rest == "" || rest[0] == ' ' || rest[0] == '\t' {
			return true
		}
	}
	return false
}

// cleanReason strips comment syntax and separators around a reason.
func cleanReason(s string) string {
	s = strings.TrimSpace(s)
	for _, end := range []string{"*/", "-->"} {
		s, _, _ = strings.Cut(s, end)
		s = strings.TrimSpace(s)
	}
	for _, tok := range commentTokens {
		if strings.HasPrefix(s, tok) {
			s = s[len(tok):]
			break
		}
	}
	return strings.TrimSpace(strings.TrimLeft(s, ":-– \t"))
}
