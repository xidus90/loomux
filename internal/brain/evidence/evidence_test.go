package evidence_test

import (
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/evidence"
)

func TestCheckEvidence_VerbatimQuote(t *testing.T) {
	segments := []evidence.Segment{
		{Number: "D1", Kind: "D", Label: "quelle.md", Body: "- alt\n+ neu"},
	}
	claims := []evidence.Claim{
		{Heading: "B1", Segment: "D1", Quote: "- alt\n+ neu", Body: "diff body"},
	}
	passed, complaints := evidence.CheckEvidence(claims, segments)
	if len(passed) != 1 || len(complaints) != 0 {
		t.Fatalf("expected 1 passed and 0 complaints, got %d passed, %v complaints", len(passed), complaints)
	}
}

func TestCheckEvidence_InventedQuote(t *testing.T) {
	segments := []evidence.Segment{
		{Number: "D1", Kind: "D", Label: "quelle.md", Body: "- alt\n+ neu"},
	}
	claims := []evidence.Claim{
		{Heading: "B1", Segment: "D1", Quote: "+ erfunden", Body: "diff body"},
	}
	passed, complaints := evidence.CheckEvidence(claims, segments)
	if len(passed) != 0 || len(complaints) != 1 {
		t.Fatalf("expected 0 passed and 1 complaint, got %d passed, %v complaints", len(passed), complaints)
	}
	if !strings.Contains(complaints[0], "quote not found in D1") {
		t.Fatalf("unexpected complaint: %s", complaints[0])
	}
}

func TestCheckEvidence_UnknownSegment(t *testing.T) {
	claims := []evidence.Claim{
		{Heading: "B1", Segment: "D9", Quote: "x", Body: ""},
	}
	passed, complaints := evidence.CheckEvidence(claims, nil)
	if len(passed) != 0 || len(complaints) != 1 {
		t.Fatalf("expected 0 passed and 1 complaint, got %d passed, %v complaints", len(passed), complaints)
	}
	if !strings.Contains(complaints[0], "no segment D9 in the package") {
		t.Fatalf("unexpected complaint: %s", complaints[0])
	}
}

func TestCheckEvidence_EmptyQuote(t *testing.T) {
	segments := []evidence.Segment{
		{Number: "D1", Kind: "D", Label: "quelle.md", Body: "- alt"},
	}
	claims := []evidence.Claim{
		{Heading: "B1", Segment: "D1", Quote: "   \t\n  ", Body: ""},
	}
	passed, complaints := evidence.CheckEvidence(claims, segments)
	if len(passed) != 0 || len(complaints) != 1 {
		t.Fatalf("expected 0 passed and 1 complaint, got %d passed, %v complaints", len(passed), complaints)
	}
	if !strings.Contains(complaints[0], "empty quote is no evidence") {
		t.Fatalf("unexpected complaint: %s", complaints[0])
	}
}

func TestCheckEvidence_CRLFInQuote(t *testing.T) {
	segments := []evidence.Segment{
		{Number: "D1", Kind: "D", Label: "quelle.md", Body: "- alt\n+ neu"},
	}
	claims := []evidence.Claim{
		{Heading: "B1", Segment: "D1", Quote: "- alt\r\n+ neu", Body: ""},
	}
	passed, complaints := evidence.CheckEvidence(claims, segments)
	if len(passed) != 1 || len(complaints) != 0 {
		t.Fatalf("expected CRLF quote to match after normalization, got passed=%d, complaints=%v", len(passed), complaints)
	}
}

func TestCheckEvidence_EmptyClaims(t *testing.T) {
	passed, complaints := evidence.CheckEvidence(nil, nil)
	if len(passed) != 0 || len(complaints) != 1 {
		t.Fatalf("expected 1 complaint for empty claims, got %v", complaints)
	}
	if complaints[0] != "proposal contains no checkable claims" {
		t.Fatalf("unexpected complaint: %s", complaints[0])
	}
}

func TestCheckEvidence_NoSegmentCited(t *testing.T) {
	claims := []evidence.Claim{
		{Heading: "B1", Segment: "", Quote: "something", Body: ""},
	}
	passed, complaints := evidence.CheckEvidence(claims, nil)
	if len(passed) != 0 || len(complaints) != 1 {
		t.Fatalf("expected 1 complaint for uncited segment, got %v", complaints)
	}
	if !strings.Contains(complaints[0], "no segment cited") {
		t.Fatalf("unexpected complaint: %s", complaints[0])
	}
}

func TestReadProposal_BasicAndCRLF(t *testing.T) {
	proposal := "---\ncase: test-1\n---\n\n" +
		"## B1 — Erstes Thema\n\n" +
		"evidence: D1\n\n" +
		"```\n" +
		"- alt\r\n+ neu\r" +
		"```\n\n" +
		"```diff\n" +
		"@@ -1,1 +1,1 @@\n-alt\n+neu\n" +
		"```\n"

	claims := evidence.ReadProposal(proposal)
	if len(claims) != 1 {
		t.Fatalf("expected 1 claim, got %d", len(claims))
	}
	c := claims[0]
	if c.Heading != "B1" {
		t.Errorf("expected heading B1, got %s", c.Heading)
	}
	if c.Segment != "D1" {
		t.Errorf("expected segment D1, got %s", c.Segment)
	}
	if c.Quote != "- alt\n+ neu" {
		t.Errorf("expected quote '- alt\\n+ neu', got %q", c.Quote)
	}
	if !strings.Contains(c.Body, "```diff") {
		t.Errorf("expected diff fence in body, got %q", c.Body)
	}
}

func TestReadProposal_TwoClaims(t *testing.T) {
	proposal := "## B1 – Claim One\n\n" +
		"evidence: D1\n\n" +
		"```diff\n" +
		"quote 1\n" +
		"```\n\n" +
		"```diff\n" +
		"@@ -1 +1 @@\n" +
		"```\n\n" +
		"## B2 — Claim Two\n\n" +
		"evidence: W1\n\n" +
		"~~~text\n" +
		"quote 2\n" +
		"~~~\n"

	claims := evidence.ReadProposal(proposal)
	if len(claims) != 2 {
		t.Fatalf("expected 2 claims, got %d", len(claims))
	}
	if claims[0].Heading != "B1" || claims[0].Segment != "D1" || claims[0].Quote != "quote 1" {
		t.Errorf("unexpected claim 1: %+v", claims[0])
	}
	if claims[1].Heading != "B2" || claims[1].Segment != "W1" || claims[1].Quote != "quote 2" {
		t.Errorf("unexpected claim 2: %+v", claims[1])
	}
}

func TestReadProposal_HeadingInsideQuoteFence(t *testing.T) {
	proposal := "## B1 - Main claim\n\n" +
		"evidence: D1\n\n" +
		"````\n" +
		"nested block\n" +
		"## D2 — Fake heading\n" +
		"evidence: W2\n" +
		"````\n"

	claims := evidence.ReadProposal(proposal)
	if len(claims) != 1 {
		t.Fatalf("expected 1 claim, got %d", len(claims))
	}
	if claims[0].Heading != "B1" {
		t.Errorf("expected B1, got %s", claims[0].Heading)
	}
	if !strings.Contains(claims[0].Quote, "## D2 — Fake heading") {
		t.Errorf("expected quoted fake heading in quote, got %q", claims[0].Quote)
	}
}

func TestReadProposal_UnrecognisedLines(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"bullet list", "## B1 - Test\n\nevidence: D1\n\n```\nq\n```\n- stray bullet list"},
		{"ordered list", "## B1 - Test\n\nevidence: D1\n\n```\nq\n```\n1. ordered list item"},
		{"table row", "## B1 - Test\n\nevidence: D1\n\n```\nq\n```\n| col1 | col2 |"},
		{"indented code spaces", "## B1 - Test\n\nevidence: D1\n\n```\nq\n```\n    indented code"},
		{"indented code tab", "## B1 - Test\n\nevidence: D1\n\n```\nq\n```\n\tindented tab code"},
		{"blockquote", "## B1 - Test\n\nevidence: D1\n\n```\nq\n```\n> quote line"},
		{"hidden html comment", "## B1 - Test\n\nevidence: D1\n\n```\nq\n```\n<!-- secret -->"},
		{"hidden obsidian comment", "## B1 - Test\n\nevidence: D1\n\n```\nq\n```\n%% secret %%"},
		{"hidden transclusion", "## B1 - Test\n\nevidence: D1\n\n```\nq\n```\n![[Note]]"},
		{"preamble junk", "Vorbemerkung\n============\n\n## B1 - Test\n\nevidence: D1\n\n```\nq\n```\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			claims := evidence.ReadProposal(tc.text)
			if len(claims) == 0 {
				t.Fatal("expected at least one claim, got 0")
			}
			foundUnrecognised := false
			for _, c := range claims {
				if strings.Contains(c.Heading, "unrecognised line") {
					foundUnrecognised = true
					if c.Segment != "" {
						t.Errorf("expected empty segment for unrecognised section, got %s", c.Segment)
					}
				}
			}
			if !foundUnrecognised {
				t.Errorf("expected claim with 'unrecognised line' complaint label, got %+v", claims)
			}
		})
	}
}

func TestReadProposal_MalformedClaim(t *testing.T) {
	t.Run("no evidence line", func(t *testing.T) {
		p := "## B1 - Test\n\n```\nquote\n```\n"
		claims := evidence.ReadProposal(p)
		if len(claims) != 1 || claims[0].Segment != "" {
			t.Fatalf("expected claim with empty segment, got %+v", claims)
		}
	})

	t.Run("duplicate evidence line", func(t *testing.T) {
		p := "## B1 - Test\n\nevidence: D1\nevidence: D2\n\n```\nquote\n```\n"
		claims := evidence.ReadProposal(p)
		if len(claims) != 1 || claims[0].Segment != "" {
			t.Fatalf("expected claim with empty segment on duplicate evidence, got %+v", claims)
		}
	})

	t.Run("missing fence pair", func(t *testing.T) {
		p := "## B1 - Test\n\nevidence: D1\n\nNo fence here\n"
		claims := evidence.ReadProposal(p)
		if len(claims) != 1 || claims[0].Quote != "" {
			t.Fatalf("expected claim with empty quote on missing fence, got %+v", claims)
		}
	})

	t.Run("unclosed fence", func(t *testing.T) {
		p := "## B1 - Test\n\nevidence: D1\n\n```\nunterminated quote\n"
		claims := evidence.ReadProposal(p)
		if len(claims) != 1 || claims[0].Quote != "" {
			t.Fatalf("expected claim with empty quote on unclosed fence, got %+v", claims)
		}
	})

	t.Run("non-claim heading", func(t *testing.T) {
		p := "### Irregular heading\n\nsome text\n"
		claims := evidence.ReadProposal(p)
		if len(claims) != 1 || claims[0].Heading != "Irregular heading" || claims[0].Segment != "" {
			t.Fatalf("expected non-claim heading parsed as empty segment claim, got %+v", claims)
		}
	})
}

func TestReadPackage_Valid(t *testing.T) {
	text := "---\n" +
		"case: topic-1\n" +
		"generated.at: 2026-08-27T09:14:00Z\n" +
		"segments: 3\n" +
		"---\n\n" +
		"## D1 — Diff, quelle.md\n\n" +
		"```\n" +
		"+new diff\n" +
		"```\n\n" +
		"## W1 — Wiki, thema.md\n\n" +
		"````\n" +
		"wiki paragraph\n" +
		"````\n\n" +
		"## Q1 — Quelle, quelle.md\n\n" +
		"```\n" +
		"doc_id: 01DOC0\nresource: quelle.md\n" +
		"```\n"

	segments, err := evidence.ReadPackage(text)
	if err != nil {
		t.Fatalf("unexpected error reading valid package: %v", err)
	}
	if len(segments) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(segments))
	}
	if segments[0].Number != "D1" || segments[0].Kind != "D" || segments[0].Body != "+new diff" {
		t.Errorf("unexpected segment 0: %+v", segments[0])
	}
	if segments[1].Number != "W1" || segments[1].Kind != "W" || segments[1].Body != "wiki paragraph" {
		t.Errorf("unexpected segment 1: %+v", segments[1])
	}
	if segments[2].Number != "Q1" || segments[2].Kind != "Q" || !strings.Contains(segments[2].Body, "01DOC0") {
		t.Errorf("unexpected segment 2: %+v", segments[2])
	}
}

func TestReadPackage_ValidationErrors(t *testing.T) {
	t.Run("missing segments count", func(t *testing.T) {
		text := "---\ncase: 1\n---\n## D1 — Diff\n```\nbody\n```\n"
		_, err := evidence.ReadPackage(text)
		if err == nil || !strings.Contains(err.Error(), "declares no segments count") {
			t.Fatalf("expected 'declares no segments count' error, got %v", err)
		}
	})

	t.Run("mismatched count", func(t *testing.T) {
		text := "---\nsegments: 2\n---\n## D1 — Diff\n```\nbody\n```\n"
		_, err := evidence.ReadPackage(text)
		if err == nil || !strings.Contains(err.Error(), "segments but 1 were read") {
			t.Fatalf("expected count mismatch error, got %v", err)
		}
	})

	t.Run("unknown kind", func(t *testing.T) {
		text := "---\nsegments: 1\n---\n## X1 — Unknown\n```\nbody\n```\n"
		_, err := evidence.ReadPackage(text)
		if err == nil || !strings.Contains(err.Error(), "is of no kind a package can carry") {
			t.Fatalf("expected unknown kind error, got %v", err)
		}
	})

	t.Run("broken kind order", func(t *testing.T) {
		text := "---\nsegments: 2\n---\n" +
			"## W1 — Wiki\n```\nw\n```\n" +
			"## D1 — Diff\n```\nd\n```\n"
		_, err := evidence.ReadPackage(text)
		if err == nil || !strings.Contains(err.Error(), "breaks the kind order") {
			t.Fatalf("expected kind order error, got %v", err)
		}
	})

	t.Run("gap in numbering", func(t *testing.T) {
		text := "---\nsegments: 2\n---\n" +
			"## D1 — Diff\n```\nd1\n```\n" +
			"## D3 — Diff\n```\nd3\n```\n"
		_, err := evidence.ReadPackage(text)
		if err == nil || !strings.Contains(err.Error(), "expected segment D2, found D3") {
			t.Fatalf("expected gap error, got %v", err)
		}
	})
}

func TestFencedBlocks(t *testing.T) {
	text := "Leading text\n\n" +
		"```diff\n" +
		"@@ -1 +1 @@\n" +
		"```\n\n" +
		"Middle text\n\n" +
		"~~~text\n" +
		"plain body\n" +
		"~~~\n\n" +
		"```\nunterminated\n"

	blocks := evidence.FencedBlocks(text)
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks (unterminated skipped), got %d", len(blocks))
	}
	if blocks[0][0] != "diff" || blocks[0][1] != "@@ -1 +1 @@" {
		t.Errorf("unexpected block 0: %+v", blocks[0])
	}
	if blocks[1][0] != "text" || blocks[1][1] != "plain body" {
		t.Errorf("unexpected block 1: %+v", blocks[1])
	}
}

func TestFencePairs_InvalidOpeners(t *testing.T) {
	t.Run("backtick in backtick info string", func(t *testing.T) {
		text := "```invalid`opener\nline\n\n```valid\nbody\n```\n"
		blocks := evidence.FencedBlocks(text)
		if len(blocks) != 1 || blocks[0][0] != "valid" {
			t.Fatalf("expected 1 valid block after invalid backtick opener, got %+v", blocks)
		}
	})

	t.Run("tilde in tilde info string closed", func(t *testing.T) {
		text := "~~~invalid~opener\nline\n~~~\n\n~~~valid\nbody\n~~~\n"
		blocks := evidence.FencedBlocks(text)
		if len(blocks) != 1 || blocks[0][0] != "valid" {
			t.Fatalf("expected 1 valid block after invalid tilde opener, got %+v", blocks)
		}
	})

	t.Run("tilde in tilde info string unclosed", func(t *testing.T) {
		text := "~~~invalid~opener\nline\n\n```valid\nbody\n```\n"
		blocks := evidence.FencedBlocks(text)
		if len(blocks) != 1 || blocks[0][0] != "valid" {
			t.Fatalf("expected 1 valid block after unclosed invalid tilde opener, got %+v", blocks)
		}
	})
}

func TestReadPackage_IgnoredHeadingsAndMissingFences(t *testing.T) {
	t.Run("non-segment heading ignored", func(t *testing.T) {
		text := "---\nsegments: 1\n---\n\n# Overview\n\n## D1 — Diff, quelle.md\n\n```\nbody\n```\n"
		segs, err := evidence.ReadPackage(text)
		if err != nil || len(segs) != 1 {
			t.Fatalf("expected 1 segment when non-segment heading is present, got %v, err=%v", segs, err)
		}
	})

	t.Run("segment heading with no fence ignored", func(t *testing.T) {
		text := "---\nsegments: 1\n---\n\n## D1 — Diff, quelle.md\n\nNo fence here\n\n## D2 — Diff, quelle2.md\n\n```\nbody\n```\n"
		_, err := evidence.ReadPackage(text)
		if err == nil {
			t.Fatal("expected error due to missing D1, got nil")
		}
	})

	t.Run("segment heading with unclosed fence ignored", func(t *testing.T) {
		text := "---\nsegments: 1\n---\n\n## D1 — Diff, quelle.md\n\n```\nunclosed\n"
		_, err := evidence.ReadPackage(text)
		if err == nil {
			t.Fatal("expected error due to missing closed segment, got nil")
		}
	})
}
