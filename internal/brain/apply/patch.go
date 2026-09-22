// Package apply carries out a reviewed case: it applies the diff a proposal
// carries to its target page, exactly, or refuses it.
//
// patch.go is moved from ultra-brain's pkg/maintenance/patch.go. The original
// is `_collect`, `_hunks` and `_patch` in `src/brain/maintenance/apply.py`,
// and where the two differed the Python form holds.
package apply

import (
	"math"
	"math/big"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/xidus90/loomux/internal/brain/evidence"
	"github.com/xidus90/loomux/internal/brain/pytext"
)

// Hunk is one unified diff hunk. At is the 0-based line of the page it
// applies at; Old is what must stand there, New what replaces it.
type Hunk struct {
	At       int
	Old, New []string

	// line is the 1-based line a message names, in full, for a hunk whose
	// anchor does not fit an int; At then holds math.MaxInt. Such a hunk
	// always lies past the end of any page, but Python prints its number.
	line string
}

// RefusedError is a proposal that cannot be applied. Python raises
// `ProposalRefused` for it and writes nothing, and Msg is Python's text
// after the `{heading}: ` that `_collect` and `_hunks` put in front.
type RefusedError struct{ Msg string }

func (e *RefusedError) Error() string { return e.Msg }

// diffInfo opens the info string of a fence that holds a proposed diff.
const diffInfo = "diff"

// hunkHeader is `_HUNK_HEADER` (apply.py:225). Python's `\d` in a str pattern
// is every Unicode decimal digit, which Go spells \p{Nd}.
var hunkHeader = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^@@ -(\p{Nd}+)(?:,(\p{Nd}+))? \+(\p{Nd}+)(?:,(\p{Nd}+))? @@`)
})

// CollectDiff returns the body of every fence in a claim's body whose info
// string starts with "diff", in order: each one is parsed on its own, since
// each must open with a hunk header of its own.
//
// FencedBlocks departs from Python on CRLF text (Python pairs fences on the
// folded text but cuts from the raw one), so the two agree only on LF text.
// A claim body from evidence.ReadProposal is already folded, so it goes in
// as it is.
func CollectDiff(body string) ([]string, error) {
	var diffs []string
	for _, block := range evidence.FencedBlocks(body) {
		if strings.HasPrefix(block[0], diffInfo) {
			diffs = append(diffs, block[1])
		}
	}
	if len(diffs) == 0 {
		return nil, &RefusedError{Msg: "no proposed diff in the claim's section"}
	}
	return diffs, nil
}

// ParseUnifiedDiff reads one diff fence body into hunks, refusing anything
// that is not one. Like `_hunks` it splits on "\n" and nothing else: no
// trimming, no folding of line ends.
func ParseUnifiedDiff(diff string) ([]Hunk, error) {
	var hunks []Hunk
	current := -1
	for _, line := range strings.Split(diff, "\n") {
		header := hunkHeader().FindStringSubmatch(line)
		switch {
		case header != nil:
			hunks = append(hunks, anchored(header[1], header[2]))
			current = len(hunks) - 1
		case current < 0:
			if pytext.Strip(line) != "" {
				return nil, &RefusedError{Msg: "diff does not start with a hunk header"}
			}
		case strings.HasPrefix(line, "\\"):
			// git's "\ No newline at end of file" marker: information about
			// the neighbouring line, never a line of its own.
		case strings.HasPrefix(line, "-"):
			hunks[current].Old = append(hunks[current].Old, line[1:])
		case strings.HasPrefix(line, "+"):
			hunks[current].New = append(hunks[current].New, line[1:])
		default:
			// A context line, including the empty one a blank context line is
			// written as: difflib and git both drop the trailing space.
			payload := strings.TrimPrefix(line, " ")
			hunks[current].Old = append(hunks[current].Old, payload)
			hunks[current].New = append(hunks[current].New, payload)
		}
	}
	if len(hunks) == 0 {
		return nil, &RefusedError{Msg: "diff does not start with a hunk header"}
	}
	return hunks, nil
}

// anchored places a hunk from its header's old start and count. An old
// count of zero is a pure insertion, which unified diff spells as the line
// after which the new lines go, so the anchor is the start itself.
func anchored(start, count string) Hunk {
	at := decimal(start)
	if count == "" || decimal(count).Sign() != 0 {
		at.Sub(at, big.NewInt(1))
	}
	if at.Cmp(big.NewInt(math.MaxInt)) < 0 {
		return Hunk{At: int(at.Int64())}
	}
	return Hunk{At: math.MaxInt, line: at.Add(at, big.NewInt(1)).String()}
}

// decimal is what Python's `int()` makes of a run of Unicode decimal digits,
// without a ceiling. A digit's value is its distance from the start of its
// run of Nd characters, modulo ten: Unicode encodes every decimal digit set
// as a contiguous 0..9, so a maximal run is whole sets laid end to end.
func decimal(digits string) *big.Int {
	value, ten := new(big.Int), big.NewInt(10)
	for _, digit := range digits {
		zero := digit
		for unicode.Is(unicode.Nd, zero-1) {
			zero--
		}
		value.Mul(value, ten)
		value.Add(value, big.NewInt(int64((digit-zero)%10)))
	}
	return value
}

// lineOf is the 1-based line a message names for h.
func lineOf(h Hunk) string {
	if h.line != "" {
		return h.line
	}
	// Through big.Int, so a hand-built hunk at math.MaxInt cannot wrap.
	return new(big.Int).Add(big.NewInt(int64(h.At)), big.NewInt(1)).String()
}

// before orders hunks as Python's `sorted(key=at)`: by anchor, anchors past
// an int by their full number, and equal anchors in the order given.
func before(a, b Hunk) bool {
	if a.At != b.At {
		return a.At < b.At
	}
	if len(a.line) != len(b.line) {
		return len(a.line) < len(b.line)
	}
	return a.line < b.line
}

// ApplyHunks applies every hunk to text, exactly, with no fuzz and no
// reordering of the page. Exactness is affordable because the hash guard
// already ran: the page is the one the proposal was written against, so a
// hunk whose context does not match is a malformed proposal.
func ApplyHunks(text string, hunks []Hunk) (string, error) {
	lines := strings.Split(text, "\n")
	sorted := append([]Hunk(nil), hunks...)
	sort.SliceStable(sorted, func(i, j int) bool { return before(sorted[i], sorted[j]) })

	var out []string
	cursor := 0
	for _, hunk := range sorted {
		if hunk.At < cursor {
			return "", &RefusedError{Msg: "hunks overlap at line " + lineOf(hunk)}
		}
		if hunk.At > len(lines) {
			// A pure insertion has no context to fail on, so without this it
			// would land quietly at the end of the page whatever line it named.
			return "", &RefusedError{Msg: "hunk at line " + lineOf(hunk) + " reaches past the end of the page"}
		}
		end := hunk.At + len(hunk.Old)
		if end > len(lines) || !slices.Equal(lines[hunk.At:end], hunk.Old) {
			return "", &RefusedError{Msg: "hunk at line " + lineOf(hunk) + " does not match the page"}
		}
		out = append(out, lines[cursor:hunk.At]...)
		out = append(out, hunk.New...)
		cursor = end
	}
	out = append(out, lines[cursor:]...)
	return strings.Join(out, "\n"), nil
}
