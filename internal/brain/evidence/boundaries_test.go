package evidence_test

// Edges of the reader that no test of test_evidence.py reaches. Every
// expectation below is what the Python reference (`read_proposal`,
// `read_package` in src/brain/maintenance/evidence.py) answers for the same text, measured on 2026-09-23.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/evidence"
)

// quoted is the tail of a claim that cites D1 and quotes altNeu verbatim.
const quoted = "\n\nevidence: D1\n\n```\n- alt\n+ neu\n```\n"

type seen struct{ heading, segment, quote string }

func readAs(t *testing.T, text string, want ...seen) {
	t.Helper()
	got := []seen{}
	for _, claim := range evidence.ReadProposal(text) {
		got = append(got, seen{claim.Heading, claim.Segment, claim.Quote})
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadProposal(%q)\n = %q\nwant %q", text, got, want)
	}
}

var passing = seen{"B1", "D1", "- alt\n+ neu"}

// The opener line belongs to its fence: markup in its info string is quoted
// material, not a line the whitelist judges.
func TestAnOpenersInfoStringIsInsideItsFence(t *testing.T) {
	readAs(t, "## B1 — Satz\n\nevidence: D1\n\n```text %%\n- alt\n+ neu\n```\n", passing)
}

// A line of nothing but spaces or a tab is a blank line, although the block
// grammar would read four spaces or a tab as the start of indented code.
func TestAWhitespaceOnlyLineIsBlank(t *testing.T) {
	readAs(t, "## B1 — Satz\n\nevidence: D1\n    \n\t\n```\n- alt\n+ neu\n```\n", passing)
}

// The first line after the frontmatter -- or the first line of a proposal
// without one -- is judged like every other.
func TestTheLineRightAfterTheFrontmatterIsJudged(t *testing.T) {
	junk := seen{"proposal (unrecognised line: - Liste)", "", ""}
	readAs(t, "---\ncase: x\n---\n- Liste\n## B1 — Satz"+quoted, junk, passing)
	readAs(t, "- Liste\n## B1 — Satz"+quoted, junk, passing)
}

// A line is charged to the section it stands in and to no earlier one.
func TestAnUnrecognisedLineRefusesOnlyItsOwnSection(t *testing.T) {
	readAs(t, "## B1 — Eins"+quoted+"\n## B2 — Zwei\n\n- Liste"+quoted,
		passing, seen{"B2 (unrecognised line: - Liste)", "", ""})
}

// A heading that is no claim heading still names the section it refuses.
func TestAPlainHeadingNamesTheSectionItRefuses(t *testing.T) {
	readAs(t, "## Notizen\n\n- Liste\n", seen{"Notizen (unrecognised line: - Liste)", "", ""})
}

// Hidden markup on a heading line is charged to no section: the section of a
// heading begins after it, and the one before ends where it starts. This is
// the reference's behaviour, recorded rather than endorsed; the parity
// record names it as a gap.
func TestHiddenMarkupOnAHeadingLineIsChargedToNoSection(t *testing.T) {
	readAs(t, "## B1 — Satz %%x%%"+quoted+"## B2 — Zwei <!-- x -->"+quoted,
		passing, seen{"B2", "D1", "- alt\n+ neu"})
}

// A heading that is no segment heading is passed over, fence and all.
func TestAPackageHeadingThatIsNoSegmentIsPassedOver(t *testing.T) {
	segments, err := evidence.ReadPackage("---\nsegments: 1\n---\n## Anmerkung\n```\nx\n```\n## D1 — quelle.md\n```\nbody\n```\n")
	if err != nil || len(segments) != 1 || segments[0].Number != "D1" || segments[0].Body != "body" {
		t.Fatalf("ReadPackage = %+v, %v", segments, err)
	}
}

// A segment ends at the next heading of any kind, so a segment without a
// fence of its own does not borrow the fence under the heading after it.
func TestASegmentWithoutAFenceBorrowsNone(t *testing.T) {
	_, err := evidence.ReadPackage("---\nsegments: 1\n---\n## D1 — quelle.md\n\nkein Zaun\n# Anhang\n```\nbody\n```\n")
	if err == nil || !strings.Contains(err.Error(), "package declares 1 segments but 0 were read") {
		t.Fatalf("err = %v", err)
	}
}
