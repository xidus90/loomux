// Package evidence binds every claim of a proposal to a verbatim quote from a
// segment of its package, and reads a rendered package back into segments.
//
// Pure text work, no model call: a claim passes only if it names a segment
// the package has and quotes it, character for character, in a fenced block.
// Outside a fence, the lines a proposal may carry are a whitelist; anything
// else refuses the section it sits in.
//
// Moved from ultra-brain's pkg/maintenance/evidence.go. The original is
// `src/brain/maintenance/evidence.py`, and where the two differed the Python
// form holds.
package evidence

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// Segment is one numbered, quotable unit of the package.
type Segment struct {
	Number string
	Kind   string
	Label  string
	Body   string
}

// Claim is one claim from a proposal: what it asserts, and where it says the proof is.
type Claim struct {
	Heading string
	Segment string
	Quote   string
	Body    string
}

// PackageError is raised when a package file does not match what build_package produces.
type PackageError struct {
	Message string
}

func (e *PackageError) Error() string {
	return e.Message
}

// pySpace is the body of a character class holding what Python's `\s` holds
// in a str pattern: the 29 characters pytext.IsSpace lists. Go's `\s` is
// `[\t\n\f\r ]` -- not even `\v` -- so a pattern copied with `\s` and `\S`
// reads `evidence: D1<NBSP>` or a claim id ending in \x1c differently from
// the reference. `\n` belongs in the class: `evidence:\s*` crosses a line
// break in both languages.
const pySpace = `\t\n\v\f\r\x1c-\x1f \x{85}\x{a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}`

// space and nonSpace stand in for Python's `\s` and `\S`. Python's `\d` is
// every decimal digit Unicode knows, which is `\p{Nd}` here; Go's tables may
// lag Python's by a Unicode version, a difference no package or proposal
// written by this system can reach.
const (
	space    = `[` + pySpace + `]`
	nonSpace = `[^` + pySpace + `]`
)

var (
	anyHeading   = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`(?m)^[ ]{0,3}#{1,6}[ \t].*$`) })
	claimHeading = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`^##[ \t]+(` + nonSpace + `+)[ \t]+[-\x{2013}\x{2014}][ \t]+` + nonSpace + `.*$`)
	})
	evidenceLine = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`(?m)^evidence:` + space + `*(` + nonSpace + `+)` + space + `*$`)
	})
	fenceLineRegex   = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`(?m)^[ ]{0,3}(` + "`" + `{3,}|~{3,})(.*)$`) })
	fenceShape       = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^[ ]{0,3}(?:` + "`" + `{3,}|~{3,})`) })
	frontmatterRegex = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`(?s)\A---\n.*?\n---[ \t]*\n`) })
	blockStart       = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`^(?:[ ]*\t|[ ]{4,}|[ ]{0,3}[-+*_=<>|\[:~` + "`" + `#]|[ ]{0,3}\p{Nd}{1,9}[.)][ \t])`)
	})
	hiddenMarkup = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`%%|!\[\[|<!--|-->|[\x{202a}-\x{202e}\x{2066}-\x{2069}]`)
	})
	segmentCountRegex = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`(?m)^segments:[ \t]*(\p{Nd}+)[ \t]*$`) })
	segmentHeading    = sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`^##[ \t]+(` + nonSpace + `+)[ \t]+[-\x{2013}\x{2014}][ \t]+(` + nonSpace + `.*)$`)
	})
)

// Normalised folds CRLF and lone CR down to LF.
func Normalised(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n")
}

type fenceLine struct {
	start  int
	end    int
	char   byte
	length int
	info   string
}

func parseFenceLines(text string) []fenceLine {
	var lines []fenceLine
	matches := fenceLineRegex().FindAllStringSubmatchIndex(text, -1)
	for _, m := range matches {
		fullStart := m[0]
		fullEnd := m[1]
		charRunStart := m[2]
		charRunEnd := m[3]
		infoStart := m[4]
		infoEnd := m[5]

		c := text[charRunStart]
		l := charRunEnd - charRunStart
		info := pytext.Strip(text[infoStart:infoEnd])
		lines = append(lines, fenceLine{
			start:  fullStart,
			end:    fullEnd,
			char:   c,
			length: l,
			info:   info,
		})
	}
	return lines
}

func isValidOpener(line fenceLine) bool {
	return !strings.ContainsRune(line.info, rune(line.char))
}

func isBareCloser(line fenceLine, openChar byte, openLength int) bool {
	return line.char == openChar && line.length >= openLength && line.info == ""
}

type fencePair struct {
	opener fenceLine
	closer *fenceLine
}

func fencePairs(text string) []fencePair {
	lines := parseFenceLines(text)
	var pairs []fencePair
	index := 0
	for index < len(lines) {
		opener := lines[index]
		closeIndex := -1
		for candidate := index + 1; candidate < len(lines); candidate++ {
			if isBareCloser(lines[candidate], opener.char, opener.length) {
				closeIndex = candidate
				break
			}
		}
		if !isValidOpener(opener) {
			if opener.char == '~' {
				if closeIndex != -1 {
					index = closeIndex + 1
				} else {
					index = index + 1
				}
			} else {
				index++
			}
			continue
		}
		if closeIndex == -1 {
			pairs = append(pairs, fencePair{opener: opener, closer: nil})
			break
		}
		closer := lines[closeIndex]
		pairs = append(pairs, fencePair{opener: opener, closer: &closer})
		index = closeIndex + 1
	}
	return pairs
}

func fencedSpans(text string) [][2]int {
	pairs := fencePairs(text)
	spans := make([][2]int, len(pairs))
	for i, p := range pairs {
		end := len(text)
		if p.closer != nil {
			end = p.closer.end
		}
		spans[i] = [2]int{p.opener.start, end}
	}
	return spans
}

func inside(pos int, spans [][2]int) bool {
	for _, s := range spans {
		if pos >= s[0] && pos < s[1] {
			return true
		}
	}
	return false
}

func isAllowedLine(line string) bool {
	if hiddenMarkup().MatchString(line) {
		return false
	}
	if pytext.Strip(line) == "" {
		return true
	}
	if anyHeading().MatchString(line) || evidenceLine().MatchString(line) || fenceShape().MatchString(line) {
		return true
	}
	return !blockStart().MatchString(line)
}

type unrecognisedLine struct {
	offset int
	line   string
}

func unrecognisedLines(text string) []unrecognisedLine {
	spans := fencedSpans(text)
	frontEnd := 0
	if loc := frontmatterRegex().FindStringIndex(text); loc != nil {
		frontEnd = loc[1]
	}
	var offending []unrecognisedLine
	offset := 0
	for _, line := range strings.Split(text, "\n") {
		if offset >= frontEnd && !inside(offset, spans) && !isAllowedLine(line) {
			offending = append(offending, unrecognisedLine{offset: offset, line: line})
		}
		offset += len(line) + 1
	}
	return offending
}

func unrecognisedLabel(owner, line string) string {
	return fmt.Sprintf("%s (unrecognised line: %s)", owner, pytext.Strip(line))
}

type headingMatch struct {
	line  string
	start int
	end   int
}

func topLevelHeadings(text string) []headingMatch {
	spans := fencedSpans(text)
	indices := anyHeading().FindAllStringIndex(text, -1)
	var matches []headingMatch
	for _, loc := range indices {
		if !inside(loc[0], spans) {
			matches = append(matches, headingMatch{
				line:  text[loc[0]:loc[1]],
				start: loc[0],
				end:   loc[1],
			})
		}
	}
	return matches
}

func parseClaim(heading, section string) Claim {
	spans := fencedSpans(section)
	evidenceIndices := evidenceLine().FindAllStringSubmatchIndex(section, -1)
	var validEvidence [][]int
	for _, m := range evidenceIndices {
		if !inside(m[0], spans) {
			validEvidence = append(validEvidence, m)
		}
	}
	if len(validEvidence) != 1 {
		return Claim{Heading: heading, Segment: "", Quote: "", Body: pytext.Strip(section)}
	}
	ev := validEvidence[0]
	segment := section[ev[2]:ev[3]]

	pairs := fencePairs(section)
	var selected *fencePair
	for i := range pairs {
		if pairs[i].opener.start >= ev[1] {
			selected = &pairs[i]
			break
		}
	}
	if selected == nil || selected.closer == nil {
		return Claim{Heading: heading, Segment: segment, Quote: "", Body: pytext.Strip(section)}
	}

	quote := strings.Trim(section[selected.opener.end:selected.closer.start], "\n")
	body := pytext.Strip(section[:selected.opener.start] + section[selected.closer.end:])
	return Claim{Heading: heading, Segment: segment, Quote: quote, Body: body}
}

// ReadProposal parses a proposal into individual claims.
func ReadProposal(text string) []Claim {
	normalised := Normalised(text)
	headings := topLevelHeadings(normalised)
	offending := unrecognisedLines(normalised)
	var claims []Claim

	firstSection := len(normalised)
	if len(headings) > 0 {
		firstSection = headings[0].start
	}

	var preamble *unrecognisedLine
	for i := range offending {
		if offending[i].offset < firstSection {
			preamble = &offending[i]
			break
		}
	}
	if preamble != nil {
		claims = append(claims, Claim{
			Heading: unrecognisedLabel("proposal", preamble.line),
			Segment: "",
			Quote:   "",
			Body:    "",
		})
	}

	for i, h := range headings {
		line := h.line
		start := h.end
		end := len(normalised)
		if i+1 < len(headings) {
			end = headings[i+1].start
		}
		section := normalised[start:end]

		claimSub := claimHeading().FindStringSubmatch(line)
		var smuggled *unrecognisedLine
		for j := range offending {
			if offending[j].offset >= start && offending[j].offset < end {
				smuggled = &offending[j]
				break
			}
		}

		if smuggled != nil {
			label := strings.TrimLeft(pytext.Strip(line), "#")
			if len(claimSub) > 1 {
				label = claimSub[1]
			}
			heading := unrecognisedLabel(pytext.Strip(label), smuggled.line)
			claims = append(claims, Claim{
				Heading: heading,
				Segment: "",
				Quote:   "",
				Body:    pytext.Strip(section),
			})
		} else if len(claimSub) == 0 {
			label := pytext.Strip(strings.TrimLeft(pytext.Strip(line), "#"))
			claims = append(claims, Claim{
				Heading: label,
				Segment: "",
				Quote:   "",
				Body:    pytext.Strip(section),
			})
		} else {
			claims = append(claims, parseClaim(claimSub[1], section))
		}
	}

	return claims
}

// CheckEvidence evaluates claims against package segments.
func CheckEvidence(claims []Claim, segments []Segment) ([]Claim, []string) {
	if len(claims) == 0 {
		return nil, []string{"proposal contains no checkable claims"}
	}
	bodies := make(map[string]string, len(segments))
	for _, seg := range segments {
		bodies[seg.Number] = Normalised(seg.Body)
	}

	var passed []Claim
	var complaints []string

	for _, claim := range claims {
		body, exists := bodies[claim.Segment]
		quote := pytext.Strip(Normalised(claim.Quote))

		if claim.Segment == "" {
			complaints = append(complaints, fmt.Sprintf("%s: no segment cited", claim.Heading))
		} else if !exists {
			complaints = append(complaints, fmt.Sprintf("%s: no segment %s in the package", claim.Heading, claim.Segment))
		} else if quote == "" {
			complaints = append(complaints, fmt.Sprintf("%s: empty quote is no evidence", claim.Heading))
		} else if !strings.Contains(body, quote) {
			complaints = append(complaints, fmt.Sprintf("%s: quote not found in %s", claim.Heading, claim.Segment))
		} else {
			passed = append(passed, claim)
		}
	}

	return passed, complaints
}

// ReadPackage reads a rendered package.md back into its segments.
func ReadPackage(text string) ([]Segment, error) {
	normalised := Normalised(text)
	headings := topLevelHeadings(normalised)
	var segments []Segment

	for i, h := range headings {
		m := segmentHeading().FindStringSubmatch(h.line)
		if len(m) == 0 {
			continue
		}
		start := h.end
		end := len(normalised)
		if i+1 < len(headings) {
			end = headings[i+1].start
		}
		section := normalised[start:end]

		pairs := fencePairs(section)
		if len(pairs) == 0 || pairs[0].closer == nil {
			continue
		}
		number := m[1]
		segments = append(segments, Segment{
			Number: number,
			Kind:   string(number[0]),
			Label:  m[2],
			Body:   strings.Trim(section[pairs[0].opener.end:pairs[0].closer.start], "\n"),
		})
	}

	if err := verifyPackage(normalised, segments); err != nil {
		return nil, err
	}
	return segments, nil
}

func verifyPackage(text string, segments []Segment) error {
	m := segmentCountRegex().FindStringSubmatch(text)
	if len(m) == 0 {
		return &PackageError{Message: "package declares no segments count in its frontmatter"}
	}
	// The count is quoted as written, the way Python formats the matched
	// group: `02` stays `02` in the message.
	if declared, fits := decimal(m[1]); !fits || declared != len(segments) {
		return &PackageError{
			Message: fmt.Sprintf("package declares %s segments but %d were read", m[1], len(segments)),
		}
	}

	seen := make(map[string]int)
	rank := -1
	kindPositions := map[string]int{"D": 0, "W": 1, "Q": 2}

	for _, seg := range segments {
		pos, ok := kindPositions[seg.Kind]
		if !ok {
			return &PackageError{
				Message: fmt.Sprintf("segment %s is of no kind a package can carry", seg.Number),
			}
		}
		if pos < rank {
			return &PackageError{
				Message: fmt.Sprintf("segment %s breaks the kind order of a package", seg.Number),
			}
		}
		rank = pos
		expected := seen[seg.Kind] + 1
		if seg.Number != fmt.Sprintf("%s%d", seg.Kind, expected) {
			return &PackageError{
				Message: fmt.Sprintf("expected segment %s%d, found %s", seg.Kind, expected, seg.Number),
			}
		}
		seen[seg.Kind] = expected
	}

	return nil
}

// decimal is Python's `int()` over a run of Unicode decimal digits, which is
// all segmentCountRegex lets through. fits is false once the value passes
// what an int holds; Python's int has no ceiling, but a count that large
// cannot equal a number of segments read, so the comparison needs no more.
//
// A digit's value is its distance from the start of its run of Nd characters,
// modulo ten: Unicode encodes every decimal digit set as a contiguous 0..9, so
// a maximal run is whole sets laid end to end and always opens on a zero.
func decimal(digits string) (value int, fits bool) {
	for _, digit := range digits {
		zero := digit
		for unicode.Is(unicode.Nd, zero-1) {
			zero--
		}
		if value > (math.MaxInt-9)/10 {
			return 0, false
		}
		value = value*10 + int(digit-zero)%10
	}
	return value, true
}

// FencedBlocks extracts all top-level fences as (info, body) pairs.
func FencedBlocks(text string) [][2]string {
	normalised := Normalised(text)
	pairs := fencePairs(normalised)
	var blocks [][2]string
	for _, p := range pairs {
		if p.closer != nil {
			info := p.opener.info
			body := strings.Trim(normalised[p.opener.end:p.closer.start], "\n")
			blocks = append(blocks, [2]string{info, body})
		}
	}
	return blocks
}
