package maintenance

// The analysis package: every segment a proposal is allowed to cite.
//
// A proposal may only claim what it can quote verbatim from a numbered
// segment, and this is where those segments get their numbers and their text.
// The numbers are handed out once, when the package is built, and hold only
// for the life of that one package -- if the source changes again the case is
// discarded and rebuilt, so no caller may read `D3` as the same hunk across
// two packages.
//
// Segment boundaries are given by the caller, not found here: one diff hunk,
// one blank-line-separated wiki paragraph, one `sources` entry. This file only
// numbers and renders what it is handed.
//
// A `D` or `W` body is caller-supplied text and may itself carry a line that
// looks like a segment heading. FenceFor still keeps that line unambiguous: it
// sits inside the escalated fence, never at top level. But that holds only for
// a reader that tracks open and closed fence spans -- one that greps for `^## `
// line by line would take it for a real heading, and a proposal could then
// "quote" a segment number nobody issued. Whatever parses a rendered package
// back into segments must read fence-aware.
//
// The original is `src/brain/maintenance/package.py`.

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// fenceRun is `_FENCE_RUN`: a run of at least three backticks at the start of
// a line, after any indentation. Go's `^` under `(?m)` breaks after `\n` and
// nowhere else, exactly as Python's does -- which is why folded below has to
// run first.
var fenceRun = regexp.MustCompile("(?m)^[ \t]*(`{3,})")

// kindLabels is `_KIND_LABELS`, and its words stay German: they are written
// into `package.md`, and the evidence check of stage 3b reads that file back
// through a heading pattern of its own. A translation here would be a
// different file format.
var kindLabels = map[string]string{"D": "Diff", "W": "Wiki", "Q": "Quelle"}

// labelMax is `_LABEL_MAX`, counted in characters the way Python counts them.
const labelMax = 80

// Segment is one numbered, quotable unit of the package.
//
// Number is the checker's lookup key (`D1`, `W2`, `Q1`, ...); Body is the
// exact text a proposal's evidence quote must be a substring of. Label is for
// a human skimming the file and is never compared.
type Segment struct {
	Number string
	Kind   string
	Label  string
	Body   string
}

// Citation is one `sources` entry of the page under review, the pair Python
// hands `build_package` as a tuple. It is not SourceState: that one carries
// the revision and the content hash a case is compared by, and none of those
// belong in a quotable body.
type Citation struct {
	DocID    string
	Resource string
}

// FenceFor is a backtick fence longer than any backtick run already inside
// body (`fence_for`).
//
// Wiki paragraphs carry code blocks of their own; a three-backtick fence
// around one of them would close at the example's own closing backticks, and
// the checker would then look for a quote in half a segment. CommonMark lets a
// fence be any length from three up, so one backtick more than the longest run
// inside always wins.
//
// Only backtick runs are counted, not tilde runs: this file always wraps in
// backticks, and CommonMark's closing rule wants the same fence character at
// equal or greater length. A tilde block inside body, however long, cannot end
// a backtick wrapper early.
//
// Python writes `max(3, longest + 1)` where longest falls back to 2. The floor
// is already in that fallback, so it is not repeated here: a second branch
// no input can take is a branch no test can justify.
func FenceFor(body string) string {
	longest := 2
	for _, match := range fenceRun.FindAllStringSubmatch(folded(body), -1) {
		if run := len(match[1]); run > longest {
			longest = run
		}
	}
	return strings.Repeat("`", longest+1)
}

// folded turns every line ending CommonMark knows into `\n` before the runs
// are counted (`_folded`).
//
// This is a measured forgery rather than a tidiness fix. The pattern above
// breaks after `\n` but not after a lone `\r`, while every reader of a
// rendered package folds a lone `\r` to `\n` -- and reconcile carries such a
// byte into a `D` segment deliberately, because the content hash does.
// Counted unfolded, a backtick run behind a lone `\r` is invisible here, the
// wrapper stays at three, and the reader then closes the segment at a fence
// the writer never saw: everything after it reads as top level, so a line
// shaped like `## D9 - ...` becomes a segment number nobody issued, and a
// proposal may cite it. Escalation and reader have to fold by the same rule.
func folded(body string) string {
	return strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\r", "\n")
}

// label is a short, single-line caption for a segment's heading (`_label`).
//
// Purely for a human skimming the file -- the checker never reads it, only the
// number and the body.
//
// pytext.Strip and pytext.SplitLines and not the strings package: Python
// strips 29 characters where Go's unicode.IsSpace knows 25, and splits lines
// at ten boundaries where a `\n` split knows one. A diff body reaches both
// differences, because reconcile leaves a lone `\r` standing in one.
//
// The cut counts runes, because Python's `len` counts characters. A byte cut
// would shorten a line of umlauts to half its text and could split a rune.
func label(text string) string {
	stripped := pytext.Strip(text)
	if stripped == "" {
		return ""
	}
	firstLine := pytext.SplitLines(stripped)[0]
	if runes := []rune(firstLine); len(runes) > labelMax {
		return string(runes[:labelMax-1]) + "…"
	}
	return firstLine
}

// BuildPackage numbers every hunk, paragraph and citation, typed by kind
// (`build_package`).
//
// Numbering runs per kind and starts at 1 (`D1`, `D2`, ..., `W1`, ..., `Q1`,
// ...), so "does this segment exist" stays a heading lookup instead of
// arithmetic over one shared counter. A citation's label is its resource
// whole, and not put through label above: a path cut at 80 characters would
// name a file that is not there.
func BuildPackage(diffHunks, pageParagraphs []string, sources []Citation) []Segment {
	var segments []Segment
	for index, hunk := range diffHunks {
		segments = append(segments, Segment{
			Number: fmt.Sprintf("D%d", index+1), Kind: "D", Label: label(hunk), Body: hunk,
		})
	}
	for index, paragraph := range pageParagraphs {
		segments = append(segments, Segment{
			Number: fmt.Sprintf("W%d", index+1), Kind: "W", Label: label(paragraph), Body: paragraph,
		})
	}
	for index, source := range sources {
		segments = append(segments, Segment{
			Number: fmt.Sprintf("Q%d", index+1), Kind: "Q", Label: source.Resource,
			Body: "doc_id: " + source.DocID + "\nresource: " + source.Resource,
		})
	}
	return segments
}

// RenderPackage renders the frontmatter plus one heading and one fenced body
// per segment (`render_package`).
//
// The stamp is the case's `created` and not the clock: a package built from
// the same case and the same segments has to render byte-identical every time,
// which a wall-clock read would break. pytext.IsoFormat and not time.RFC3339,
// because Python writes `+00:00` where Go writes `Z`.
//
// Python's `_generated_at` refuses a `created` without a zone; there is no
// such value here. Every time.Time carries a Location and IsoFormat always
// writes an offset, so the refusal has nothing left to refuse and no branch
// stands in for it -- the same reading WriteCase arrived at.
func RenderPackage(c Case, segments []Segment) string {
	lines := []string{
		"---",
		"case: " + c.ID,
		"generated.at: " + pytext.IsoFormat(c.Created),
		fmt.Sprintf("segments: %d", len(segments)),
		"---",
		"",
	}
	for _, segment := range segments {
		fence := FenceFor(segment.Body)
		art, known := kindLabels[segment.Kind]
		if !known {
			// Python subscripts the mapping and ends in a KeyError. A Go
			// lookup answers "" and would write `## S1 — , label`, which the
			// reader of stage 3b still parses as a heading: a segment kind
			// nobody agreed to, gone quiet. The loud end is the honest one.
			panic(fmt.Sprintf("maintenance: segment %s has no known kind, found %q", segment.Number, segment.Kind))
		}
		lines = append(lines,
			fmt.Sprintf("## %s — %s, %s", segment.Number, art, segment.Label),
			"", fence, segment.Body, fence, "")
	}
	// Only `\n` is cut, and exactly one is put back. A body ending in a
	// newline therefore keeps the blank line it opens inside its own fence --
	// that is what the reference writes, and the reader counts on it.
	return strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
}
