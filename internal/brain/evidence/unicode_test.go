package evidence_test

// Where the moved Go form read characters differently from the Python
// reference. Python's `\s`, `\S`, `\d` and `str.strip()` are Unicode-wide;
// Go's `\s` is `[\t\n\f\r ]` (not even `\v`), its `\d` is ASCII, and
// strings.TrimSpace leaves \x1c-\x1f standing. Every expected value below is
// what src/brain/maintenance/evidence.py answered for the same input on
// 2026-09-22, printed with ascii().

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/brain/evidence"
)

func TestReadProposalReadsCharactersLikePython(t *testing.T) {
	const sound = "## B1 — T\n\n"
	cases := []struct {
		name     string
		proposal string
		want     evidence.Claim
	}{
		{"vertical tab after the segment", sound + "evidence: D1\v\n\n```\nq\n```\n",
			evidence.Claim{Heading: "B1", Segment: "D1", Quote: "q", Body: "evidence: D1"}},
		{"no-break space before the segment", sound + "evidence:\u00a0D1\n\n```\nq\n```\n",
			evidence.Claim{Heading: "B1", Segment: "D1", Quote: "q", Body: "evidence:\u00a0D1"}},
		{"separator control after the segment", sound + "evidence: D1\x1c\n\n```\nq\n```\n",
			evidence.Claim{Heading: "B1", Segment: "D1", Quote: "q", Body: "evidence: D1"}},
		{"no-break space opening the claim text", "## B1 — \u00a0x\n\nevidence: D1\n\n```\nq\n```\n",
			evidence.Claim{Heading: "B1 — \u00a0x", Body: "evidence: D1\n\n```\nq\n```"}},
		{"separator control inside the claim id", "## B1\x1c — x\n\nevidence: D1\n\n```\nq\n```\n",
			evidence.Claim{Heading: "B1\x1c — x", Body: "evidence: D1\n\n```\nq\n```"}},
		{"separator control after a closing fence", sound + "evidence: D1\n\n```\nq\n```\x1c\nmore\n```\n",
			evidence.Claim{Heading: "B1", Segment: "D1", Quote: "q", Body: "evidence: D1\n\n\nmore\n```"}},
		{"ordered list in Arabic-Indic digits", sound + "evidence: D1\n\n```\nq\n```\n\u0661. item\n",
			evidence.Claim{Heading: "B1 (unrecognised line: \u0661. item)", Body: "evidence: D1\n\n```\nq\n```\n\u0661. item"}},
		{"separator control closing the section", sound + "evidence: D1\n\n```\nq\n```\n\x1c",
			evidence.Claim{Heading: "B1", Segment: "D1", Quote: "q", Body: "evidence: D1"}},
		{"separator control closing a heading label", "### Irregular\x1c\n\ntext\n",
			evidence.Claim{Heading: "Irregular", Body: "text"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := evidence.ReadProposal(tc.proposal)
			if want := []evidence.Claim{tc.want}; !reflect.DeepEqual(got, want) {
				t.Fatalf("claims = %+q\nwant     %+q", got, want)
			}
		})
	}
}

func TestCheckEvidenceStripsTheQuoteLikePython(t *testing.T) {
	// "\x1cq\x1c".strip() is "q" in Python, so the quote is found.
	segments := []evidence.Segment{{Number: "D1", Kind: "D", Body: "q"}}
	passed, complaints := evidence.CheckEvidence([]evidence.Claim{{Heading: "B1", Segment: "D1", Quote: "\x1cq\x1c"}}, segments)
	if len(passed) != 1 || len(complaints) != 0 {
		t.Fatalf("passed=%v complaints=%q", passed, complaints)
	}
}

func TestReadPackageReadsTheCountLikePython(t *testing.T) {
	const pkg = "---\nsegments: %s\n---\n\n## D1 — Diff, a\n\n```\nb\n```\n"
	one := []evidence.Segment{{Number: "D1", Kind: "D", Label: "Diff, a", Body: "b"}}
	cases := []struct {
		name, count string
		want        []evidence.Segment
		err         string
	}{
		{"leading zero is quoted as written", "02", nil, "package declares 02 segments but 1 were read"},
		{"Arabic-Indic one", "\u0661", one, ""},
		{"Arabic-Indic two is quoted as written", "\u0662", nil, "package declares \u0662 segments but 1 were read"},
		{"mathematical double-struck one", "\U0001D7D9", one, ""},
		{"a count past every int", "99999999999999999999999999999999999999999", nil,
			"package declares 99999999999999999999999999999999999999999 segments but 1 were read"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := evidence.ReadPackage(fmt.Sprintf(pkg, tc.count))
			message := ""
			if err != nil {
				message = err.Error()
			}
			if message != tc.err || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("segments=%+v err=%q, want %+v %q", got, message, tc.want, tc.err)
			}
		})
	}
}
