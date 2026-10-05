package evidence_test

// The tests of the reference's tests/maintenance/test_evidence.py that have no
// counterpart in evidence_test.go, one Go test per Python test,
// each named with the Python test and its line. Three Python tests are
// already covered there and are not repeated: test_an_unknown_segment_fails
// (:39, TestCheckEvidence_UnknownSegment), test_an_empty_quote_is_no_evidence
// (:45, TestCheckEvidence_EmptyQuote) and test_crlf_in_the_quote_still_matches
// (:51, TestCheckEvidence_CRLFInQuote).

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/evidence"
	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/brain/pytext"
)

// fixture reads a proposal the way Python's `_fixture` does: read_text with
// universal newlines.
func fixture(t *testing.T, name string) string {
	t.Helper()
	text, err := pytext.ReadText(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return text
}

// altNeu is the one segment most Python tests check against.
func altNeu() []evidence.Segment {
	return []evidence.Segment{{Number: "D1", Kind: "diff", Label: "quelle.md", Body: "- alt\n+ neu"}}
}

func headings(claims []evidence.Claim) []string {
	out := []string{}
	for _, claim := range claims {
		out = append(out, claim.Heading)
	}
	return out
}

func sameHeadings(t *testing.T, claims []evidence.Claim, want ...string) {
	t.Helper()
	if got := headings(claims); !reflect.DeepEqual(got, want) {
		t.Fatalf("headings = %q, want %q", got, want)
	}
}

// test_a_verbatim_quote_passes, test_evidence.py:25
func TestVerbatimQuoteFromFixturePasses(t *testing.T) {
	passed, complaints := evidence.CheckEvidence(evidence.ReadProposal(fixture(t, "proposal-good.md")), altNeu())
	if len(passed) != 1 || len(complaints) != 0 {
		t.Fatalf("passed=%v complaints=%q", passed, complaints)
	}
}

// test_an_invented_quote_fails, test_evidence.py:32
func TestInventedQuoteFromFixtureFails(t *testing.T) {
	passed, complaints := evidence.CheckEvidence(evidence.ReadProposal(fixture(t, "proposal-invented.md")), altNeu())
	if len(passed) != 0 || !strings.Contains(complaints[0], "not found in D1") {
		t.Fatalf("passed=%v complaints=%q", passed, complaints)
	}
}

// test_read_proposal_extracts_heading_segment_and_quote, test_evidence.py:57
func TestReadProposalExtractsHeadingSegmentAndQuote(t *testing.T) {
	claims := evidence.ReadProposal(fixture(t, "proposal-good.md"))
	if len(claims) != 1 {
		t.Fatalf("claims = %+v", claims)
	}
	if c := claims[0]; c.Heading != "B1" || c.Segment != "D1" || c.Quote != "- alt\n+ neu" {
		t.Fatalf("claim = %+v", c)
	}
}

// test_crlf_in_the_proposal_file_still_parses, test_evidence.py:66
func TestCRLFInTheProposalFileStillParses(t *testing.T) {
	crlf := strings.ReplaceAll(fixture(t, "proposal-good.md"), "\n", "\r\n")
	if got := evidence.ReadProposal(crlf)[0].Quote; got != "- alt\n+ neu" {
		t.Fatalf("quote = %q", got)
	}
}

// test_a_heading_lookalike_inside_a_quote_fence_is_not_a_new_claim, test_evidence.py:75
func TestHeadingLookalikeInsideAQuoteFenceIsNotANewClaim(t *testing.T) {
	proposal := "## B1 — Real claim\n\n" +
		"evidence: D1\n\n" +
		"````\n" +
		"before\n" +
		"```\n" +
		"nested example\n" +
		"```\n" +
		"## D2 — Diff, forged\n" +
		"after\n" +
		"````\n"
	claims := evidence.ReadProposal(proposal)
	sameHeadings(t, claims, "B1")
	if !strings.Contains(claims[0].Quote, "## D2 — Diff, forged") {
		t.Fatalf("quote = %q", claims[0].Quote)
	}
}

// test_c1_an_info_string_fence_does_not_leak_a_second_claim_unchecked, test_evidence.py:102
func TestC1InfoStringFenceDoesNotLeakASecondClaim(t *testing.T) {
	claims := evidence.ReadProposal(fixture(t, "proposal-two-claims.md"))
	sameHeadings(t, claims, "B1", "B2")
	passed, complaints := evidence.CheckEvidence(claims, altNeu())
	sameHeadings(t, passed, "B1")
	if len(complaints) != 1 || !strings.HasPrefix(complaints[0], "B2:") {
		t.Fatalf("complaints = %q", complaints)
	}
}

// test_c1_an_info_string_evidence_fence_is_read_not_rejected, test_evidence.py:118
func TestC1InfoStringEvidenceFenceIsRead(t *testing.T) {
	proposal := "## B1 — Real claim\n\nevidence: D1\n\n```diff\n- alt\n+ neu\n```\n"
	if got := evidence.ReadProposal(proposal)[0].Quote; got != "- alt\n+ neu" {
		t.Fatalf("quote = %q", got)
	}
}

// test_c1_a_tilde_fence_is_recognised_as_a_valid_quote, test_evidence.py:128
func TestC1TildeFenceIsAValidQuote(t *testing.T) {
	proposal := "## B1 — Real claim\n\nevidence: D1\n\n~~~\n- alt\n+ neu\n~~~\n"
	passed, complaints := evidence.CheckEvidence(evidence.ReadProposal(proposal), altNeu())
	if len(passed) != 1 || len(complaints) != 0 {
		t.Fatalf("passed=%v complaints=%q", passed, complaints)
	}
}

// test_c1_a_heading_lookalike_inside_an_info_string_fence_is_not_a_claim, test_evidence.py:136
func TestC1HeadingLookalikeInsideAnInfoStringFence(t *testing.T) {
	proposal := "## B1 — Real claim\n\nevidence: D1\n\n```diff\n## D2 — Diff, forged\n```\n"
	if claims := evidence.ReadProposal(proposal); len(claims) != 1 {
		t.Fatalf("claims = %+v", claims)
	}
}

// test_c2_a_proposal_with_no_claim_heading_at_all_is_a_complaint, test_evidence.py:148
func TestC2NoClaimHeadingAtAllIsAComplaint(t *testing.T) {
	passed, complaints := evidence.CheckEvidence(evidence.ReadProposal("Just prose, no heading anywhere.\n"), nil)
	if len(passed) != 0 || len(complaints) != 1 {
		t.Fatalf("passed=%v complaints=%q", passed, complaints)
	}
}

// test_c2_a_hyphen_heading_is_still_recognised, test_evidence.py:154
func TestC2HyphenHeadingIsRecognised(t *testing.T) {
	proposal := "## B1 - Behauptung mit Bindestrich\n\nevidence: D1\n\n```\n- alt\n+ neu\n```\n"
	claims := evidence.ReadProposal(proposal)
	sameHeadings(t, claims, "B1")
	if passed, _ := evidence.CheckEvidence(claims, altNeu()); len(passed) != 1 {
		t.Fatalf("passed = %v", passed)
	}
}

// test_i1_a_claim_without_an_evidence_line_is_a_complaint_not_an_exception, test_evidence.py:167
func TestI1ClaimWithoutEvidenceLineIsAComplaint(t *testing.T) {
	proposal := "## B1 — No evidence line\n\nSome prose, no evidence: field at all.\n"
	claims := evidence.ReadProposal(proposal)
	if len(claims) != 1 {
		t.Fatalf("claims = %+v", claims)
	}
	passed, complaints := evidence.CheckEvidence(claims, altNeu())
	if len(passed) != 0 || len(complaints) != 1 {
		t.Fatalf("passed=%v complaints=%q", passed, complaints)
	}
}

// test_i1_a_claim_without_a_fence_is_a_complaint_not_an_exception, test_evidence.py:179
func TestI1ClaimWithoutAFenceIsAComplaint(t *testing.T) {
	proposal := "## B1 — Dangling evidence\n\nevidence: D1\n\nNo fence follows.\n"
	passed, complaints := evidence.CheckEvidence(evidence.ReadProposal(proposal), altNeu())
	if len(passed) != 0 || !strings.Contains(complaints[0], "empty quote") {
		t.Fatalf("passed=%v complaints=%q", passed, complaints)
	}
}

// test_i1_an_unterminated_evidence_fence_is_a_complaint_not_an_exception, test_evidence.py:187
func TestI1UnterminatedEvidenceFenceIsAComplaint(t *testing.T) {
	proposal := "## B1 — Open fence\n\nevidence: D1\n\n```\n- alt\n+ neu\n"
	passed, complaints := evidence.CheckEvidence(evidence.ReadProposal(proposal), altNeu())
	if len(passed) != 0 || !strings.Contains(complaints[0], "empty quote") {
		t.Fatalf("passed=%v complaints=%q", passed, complaints)
	}
}

// test_i1_one_broken_claim_does_not_take_down_a_sound_one, test_evidence.py:195
func TestI1OneBrokenClaimDoesNotTakeDownASoundOne(t *testing.T) {
	proposal := "## B1 — Sound claim\n\nevidence: D1\n\n```\n- alt\n+ neu\n```\n\n" +
		"## B2 — Broken claim, no evidence line at all\n\nJust prose.\n"
	claims := evidence.ReadProposal(proposal)
	if len(claims) != 2 {
		t.Fatalf("claims = %+v", claims)
	}
	passed, complaints := evidence.CheckEvidence(claims, altNeu())
	sameHeadings(t, passed, "B1")
	if len(complaints) != 1 || !strings.HasPrefix(complaints[0], "B2:") {
		t.Fatalf("complaints = %q", complaints)
	}
}

// test_i2_a_second_evidence_line_is_not_silently_ignored, test_evidence.py:211
func TestI2SecondEvidenceLineIsNotIgnored(t *testing.T) {
	proposal := "## B1 — Two evidence lines\n\n" +
		"evidence: D1\n\n```\n- alt\n+ neu\n```\n\n" +
		"evidence: D9\n\n```\n- erfunden\n```\n"
	claims := evidence.ReadProposal(proposal)
	if len(claims) != 1 {
		t.Fatalf("claims = %+v", claims)
	}
	passed, complaints := evidence.CheckEvidence(claims, altNeu())
	if len(passed) != 0 || len(complaints) != 1 {
		t.Fatalf("passed=%v complaints=%q", passed, complaints)
	}
}

// test_i3_a_fake_evidence_line_inside_an_earlier_fence_is_ignored, test_evidence.py:228
func TestI3FakeEvidenceLineInsideAnEarlierFenceIsIgnored(t *testing.T) {
	proposal := "## B1 — Real claim\n\n" +
		"Beispiel:\n\n```\nevidence: D9\n```\n\n" +
		"evidence: D1\n\n```\n- alt\n+ neu\n```\n"
	claims := evidence.ReadProposal(proposal)
	if len(claims) != 1 || claims[0].Segment != "D1" {
		t.Fatalf("claims = %+v", claims)
	}
	if passed, _ := evidence.CheckEvidence(claims, altNeu()); len(passed) != 1 {
		t.Fatalf("passed = %v", passed)
	}
}

// test_a_backtick_in_a_backtick_fence_info_string_is_not_a_valid_opener, test_evidence.py:244
func TestBacktickInABacktickInfoStringIsNoOpener(t *testing.T) {
	proposal := "## B1 — Real claim\n\n" +
		"```invalid`opener\nignored, not a real fence\n\n" +
		"evidence: D1\n\n```\n- alt\n+ neu\n```\n"
	claims := evidence.ReadProposal(proposal)
	if len(claims) != 1 || claims[0].Segment != "D1" {
		t.Fatalf("claims = %+v", claims)
	}
	if passed, _ := evidence.CheckEvidence(claims, altNeu()); len(passed) != 1 {
		t.Fatalf("passed = %v", passed)
	}
}

// test_i4_an_unterminated_fence_swallows_the_rest_of_the_document, test_evidence.py:262
func TestI4UnterminatedFenceSwallowsTheRest(t *testing.T) {
	proposal := "## B1 — Real claim\n\nevidence: D1\n\n```\n- alt\n+ neu\n```\n\n" +
		"```\nunterminated, swallows everything after it\n\n" +
		"## B2 — Ghost, must never be seen\n\nevidence: D9\n\n```\nbar\n```\n"
	sameHeadings(t, evidence.ReadProposal(proposal), "B1")
}

// soundThenSecond checks the shape the fix-round-2 and -3 tests share: B1
// passes, and the second section surfaces as a claim of its own that fails.
func soundThenSecond(t *testing.T, proposal string) {
	t.Helper()
	claims := evidence.ReadProposal(proposal)
	if len(claims) != 2 {
		t.Fatalf("claims = %+v", claims)
	}
	passed, complaints := evidence.CheckEvidence(claims, altNeu())
	sameHeadings(t, passed, "B1")
	if len(complaints) != 1 {
		t.Fatalf("complaints = %q", complaints)
	}
}

// test_an_en_dash_heading_is_a_complaint_not_swallowed_into_the_previous_claim, test_evidence.py:278
func TestEnDashHeadingIsNotSwallowed(t *testing.T) {
	soundThenSecond(t, "## B1 — Sound claim\n\nevidence: D1\n\n```\n- alt\n+ neu\n```\n\n"+
		"## B2 \u2013 Zweite Behauptung\n\nevidence: D9\n\n```\n- erfunden\n```\n")
}

// test_a_level_three_heading_is_a_complaint_not_silently_absorbed, test_evidence.py:296
func TestLevelThreeHeadingIsNotAbsorbed(t *testing.T) {
	soundThenSecond(t, "## B1 — Sound claim\n\nevidence: D1\n\n```\n- alt\n+ neu\n```\n\n"+
		"### B2 — Falscher Level\n\nevidence: D9\n\n```\n- erfunden\n```\n")
}

// test_a_dash_with_no_surrounding_space_is_a_complaint_not_silently_absorbed, test_evidence.py:313
func TestDashWithoutSpaceIsNotAbsorbed(t *testing.T) {
	soundThenSecond(t, "## B1 — Sound claim\n\nevidence: D1\n\n```\n- alt\n+ neu\n```\n\n"+
		"## B2—KeinLeerzeichen\n\nevidence: D9\n\n```\n- erfunden\n```\n")
}

// test_a_rejected_openers_closer_does_not_become_a_phantom_opener, test_evidence.py:333
func TestRejectedOpenersCloserIsNoPhantomOpener(t *testing.T) {
	soundThenSecond(t, "## B1 — Sound\n\nevidence: D1\n\n```\n- alt\n+ neu\n```\n\n"+
		"~~~ x~y\n~~~\n\n"+
		"## B2 — Fake\n\nevidence: D9\n\n```\n- erfunden\n```\n")
}

// test_an_indented_heading_still_counts_as_a_section_boundary, test_evidence.py:357
func TestIndentedHeadingIsASectionBoundary(t *testing.T) {
	soundThenSecond(t, "## B1 — Sound\n\nevidence: D1\n\n```\n- alt\n+ neu\n```\n\n"+
		"   ## B2 — Fake\n\n```\nevidence: D9\n```\n")
}

// test_a_tab_indented_fence_marker_does_not_open_a_span_for_the_checker, test_evidence.py:380
func TestTabIndentedFenceMarkerOpensNoSpan(t *testing.T) {
	proposal := "## B1 — Sound\n\nevidence: D1\n\n```\n- alt\n+ neu\n```\n\n" +
		"\t```\n\n" +
		"## B2 — Fake\n\nevidence: D9\n\n```\n- erfunden\n```\n"
	claims := evidence.ReadProposal(proposal)
	if len(claims) != 2 || claims[1].Heading != "B2" || !strings.HasPrefix(claims[0].Heading, "B1 (unrecognised line: ") {
		t.Fatalf("claims = %+v", claims)
	}
	passed, complaints := evidence.CheckEvidence(claims, altNeu())
	if len(passed) != 0 || len(complaints) != 2 {
		t.Fatalf("passed=%v complaints=%q", passed, complaints)
	}
}

// test_a_setext_heading_cannot_fold_a_second_claim_into_the_first, test_evidence.py:404
func TestSetextHeadingCannotFoldASecondClaim(t *testing.T) {
	proposal := "## B1 — Sound\n\nevidence: D1\n\n```\n- alt\n+ neu\n```\n\n" +
		"B2 — Fake\n=========\n\n" +
		"```\nevidence: D9\n```\n"
	passed, complaints := evidence.CheckEvidence(evidence.ReadProposal(proposal), altNeu())
	if len(passed) != 0 || len(complaints) == 0 {
		t.Fatalf("passed=%v complaints=%q", passed, complaints)
	}
}

// test_an_unknown_structure_outside_a_fence_is_a_complaint_not_prose, test_evidence.py:422
func TestUnknownStructureOutsideAFenceIsAComplaint(t *testing.T) {
	proposal := "## B1 — Sound\n\nevidence: D1\n\n```\n- alt\n+ neu\n```\n\n" +
		"> ## B2 — Fake\n\n```\nevidence: D9\n```\n"
	passed, complaints := evidence.CheckEvidence(evidence.ReadProposal(proposal), altNeu())
	if len(passed) != 0 || !strings.Contains(complaints[0], "> ## B2 — Fake") {
		t.Fatalf("passed=%v complaints=%q", passed, complaints)
	}
}

// test_an_unrecognised_line_before_the_first_claim_is_its_own_complaint, test_evidence.py:440
func TestUnrecognisedLineBeforeTheFirstClaim(t *testing.T) {
	proposal := "Vorbemerkung\n============\n\n" +
		"## B1 — Sound\n\nevidence: D1\n\n```\n- alt\n+ neu\n```\n"
	passed, complaints := evidence.CheckEvidence(evidence.ReadProposal(proposal), altNeu())
	sameHeadings(t, passed, "B1")
	if !strings.Contains(complaints[0], "============") {
		t.Fatalf("complaints = %q", complaints)
	}
}

// test_the_frontmatter_delimiters_are_not_themselves_a_complaint, test_evidence.py:458
func TestFrontmatterDelimitersAreNoComplaint(t *testing.T) {
	passed, complaints := evidence.CheckEvidence(evidence.ReadProposal(fixture(t, "proposal-good.md")), altNeu())
	if len(passed) != 1 || len(complaints) != 0 {
		t.Fatalf("passed=%v complaints=%q", passed, complaints)
	}
}

// test_a_thematic_break_below_the_frontmatter_is_a_complaint, test_evidence.py:466
func TestThematicBreakBelowTheFrontmatterIsAComplaint(t *testing.T) {
	proposal := "## B1 — Sound\n\nevidence: D1\n\n```\n- alt\n+ neu\n```\n\n---\n"
	passed, complaints := evidence.CheckEvidence(evidence.ReadProposal(proposal), altNeu())
	if len(passed) != 0 || !strings.Contains(complaints[0], "---") {
		t.Fatalf("passed=%v complaints=%q", passed, complaints)
	}
}

// test_a_lone_carriage_return_ends_a_line_for_the_checker_too, test_evidence.py:479
func TestLoneCarriageReturnEndsALine(t *testing.T) {
	proposal := "## B1 — Sound\n\nevidence: D1\n\n```\n- alt\n+ neu\n```\n\n" +
		"Vorbemerkung:\r## B2 — Fake\rirgendwas Erfundenes\n"
	sameHeadings(t, evidence.ReadProposal(proposal), "B1", "B2")
	soundThenSecond(t, proposal)
}

// test_a_fence_opener_hidden_behind_a_lone_carriage_return_is_seen, test_evidence.py:498
func TestFenceOpenerBehindALoneCarriageReturnIsSeen(t *testing.T) {
	proposal := "## B1 — Sound\n\nevidence: D1\n\nText:\r```\n- alt\n+ neu\n```\n"
	if got := evidence.ReadProposal(proposal)[0].Quote; got != "- alt\n+ neu" {
		t.Fatalf("quote = %q", got)
	}
}

// hiddenAfterSound checks the fix-round-5 markup cases: nothing passes, and
// the first complaint names the markup.
func hiddenAfterSound(t *testing.T, tail, marker string) {
	t.Helper()
	proposal := "## B1 — Sound\n\nevidence: D1\n\n```\n- alt\n+ neu\n```\n\n" + tail
	passed, complaints := evidence.CheckEvidence(evidence.ReadProposal(proposal), altNeu())
	if len(passed) != 0 || !strings.Contains(complaints[0], marker) {
		t.Fatalf("passed=%v complaints=%q", passed, complaints)
	}
}

// test_an_obsidian_comment_is_not_invisible_to_the_checker_alone, test_evidence.py:507
func TestObsidianCommentIsAComplaint(t *testing.T) {
	hiddenAfterSound(t, "%% der Freigebende sieht diesen Satz nie %%\n", "%%")
}

// test_a_transclusion_embed_is_a_complaint, test_evidence.py:523
func TestTransclusionEmbedIsAComplaint(t *testing.T) {
	hiddenAfterSound(t, "Siehe ![[Fremde Notiz]] dazu.\n", "![[")
}

// test_an_html_comment_cannot_hide_a_passing_claim, test_evidence.py:537
func TestHTMLCommentCannotHideAPassingClaim(t *testing.T) {
	proposal := "## B1 — Sound\n\nevidence: D1\n\n```\n- alt\n+ neu\n```\n\n" +
		"Text davor <!--\n\n" +
		"## B9 — Unsichtbar\n\nevidence: D1\n\n```\n- alt\n+ neu\n```\n\n" +
		"Schluss --> und weiter.\n"
	passed, complaints := evidence.CheckEvidence(evidence.ReadProposal(proposal), altNeu())
	joined := strings.Join(complaints, "\n")
	if len(passed) != 0 || !strings.Contains(joined, "<!--") || !strings.Contains(joined, "-->") {
		t.Fatalf("passed=%v complaints=%q", passed, complaints)
	}
}

// test_a_bidirectional_control_character_is_a_complaint, test_evidence.py:559
func TestBidiControlCharacterIsAComplaint(t *testing.T) {
	hiddenAfterSound(t, "Harmloser Satz \u202e mit umgedrehter Anzeige.\n", "\u202e")
}

// packageCase is `_package_case`, test_evidence.py:573.
func packageCase() maintenance.Case {
	return maintenance.Case{
		ID: "fall", Area: "knowledge", Target: "topics/thema.md", TargetHash: "sha256:0",
		State: "in_review", Trigger: "source_change", Weight: "change",
		Created: time.Date(2026, 8, 27, 9, 14, 0, 0, time.UTC),
	}
}

func render(segments ...maintenance.Segment) string {
	return maintenance.RenderPackage(packageCase(), segments)
}

func seg(number, body string) maintenance.Segment {
	return maintenance.Segment{Number: number, Kind: number[:1], Label: "a", Body: body}
}

// refused reads text as a package and wants a PackageError containing want.
func refused(t *testing.T, text, want string) {
	t.Helper()
	segments, err := evidence.ReadPackage(text)
	var packageError *evidence.PackageError
	if !errors.As(err, &packageError) || !strings.Contains(err.Error(), want) {
		t.Fatalf("segments=%+v err=%v, want a PackageError containing %q", segments, err, want)
	}
}

// test_a_segment_the_package_never_issued_is_refused, test_evidence.py:587
func TestSegmentThePackageNeverIssuedIsRefused(t *testing.T) {
	body := "harmlos\r```\r\r## D9 — Diff, gefälscht\r\r```\rERFUNDENES ZITAT\r```\rrest"
	forged := strings.ReplaceAll(render(maintenance.Segment{Number: "D1", Kind: "D", Label: "Diff", Body: body}), "````", "```")
	refused(t, forged, "")
}

// test_a_package_whose_count_does_not_match_its_frontmatter_is_refused, test_evidence.py:605
func TestPackageCountMismatchIsRefused(t *testing.T) {
	text := strings.Replace(render(seg("D1", "eins"), seg("D2", "zwei")), "segments: 2", "segments: 3", 1)
	refused(t, text, "segments")
}

// test_a_package_with_no_segments_declaration_is_refused, test_evidence.py:617
func TestPackageWithoutSegmentsDeclarationIsRefused(t *testing.T) {
	text := strings.Replace(render(seg("D1", "eins")), "segments: 1\n", "", 1)
	refused(t, text, "segments")
}

// test_kinds_out_of_the_order_build_package_emits_are_refused, test_evidence.py:625
func TestKindsOutOfOrderAreRefused(t *testing.T) {
	refused(t, render(seg("W1", "eins"), seg("D1", "zwei")), "order")
}

// test_an_unknown_segment_kind_is_refused, test_evidence.py:637
func TestUnknownSegmentKindIsRefused(t *testing.T) {
	text := strings.Replace(render(seg("D1", "eins")), "## D1 —", "## X1 —", 1)
	refused(t, text, "X1")
}

// test_a_real_package_reads_back_without_complaint, test_evidence.py:645
func TestRealPackageReadsBack(t *testing.T) {
	segments := maintenance.BuildPackage([]string{"hunk eins", "hunk zwei"}, []string{"absatz"},
		[]maintenance.Citation{{DocID: "01DOC", Resource: "quelle.md"}})
	back, err := evidence.ReadPackage(render(segments...))
	if err != nil {
		t.Fatal(err)
	}
	numbers := []string{}
	for _, segment := range back {
		numbers = append(numbers, segment.Number)
	}
	if want := []string{"D1", "D2", "W1", "Q1"}; !reflect.DeepEqual(numbers, want) {
		t.Fatalf("numbers = %q, want %q", numbers, want)
	}
}

// test_a_gap_in_the_numbering_is_refused_even_at_the_declared_count, test_evidence.py:655
func TestGapInTheNumberingIsRefused(t *testing.T) {
	refused(t, render(seg("D1", "eins"), seg("D3", "zwei")), "expected segment D2")
}

// The reader has to fit the writer of the package: one segment of each kind,
// one body carrying a fence of its own, rendered by RenderPackage and read
// back with the same count, kinds and bodies.
func TestRenderedPackageRoundTrips(t *testing.T) {
	written := maintenance.BuildPackage(
		[]string{"@@ -1 +1 @@\n-alt\n+neu"},
		[]string{"Ein Absatz mit Beispiel:\n\n```go\nfmt.Println(1)\n```\n\nund danach."},
		[]maintenance.Citation{{DocID: "01DOC", Resource: "quelle.md"}},
	)
	read, err := evidence.ReadPackage(render(written...))
	if err != nil {
		t.Fatal(err)
	}
	if len(read) != len(written) {
		t.Fatalf("read %d segments, wrote %d", len(read), len(written))
	}
	for index, segment := range read {
		want := written[index]
		if segment.Number != want.Number || segment.Kind != want.Kind || segment.Body != want.Body {
			t.Errorf("segment %d = %+v, wrote %+v", index, segment, want)
		}
	}
}
