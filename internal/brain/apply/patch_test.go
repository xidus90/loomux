package apply_test

import (
	"errors"
	"math"
	"math/big"
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/brain/apply"
)

// refused asserts that err is a *RefusedError carrying exactly want: the
// Python text after its `{heading}: `, so the caller can rebuild the message.
func refused(t *testing.T, err error, want string) {
	t.Helper()
	var r *apply.RefusedError
	if !errors.As(err, &r) {
		t.Fatalf("want *RefusedError %q, got %T %v", want, err, err)
	}
	if r.Error() != want {
		t.Fatalf("want message %q, got %q", want, r.Error())
	}
}

func parse(t *testing.T, diff string) []apply.Hunk {
	t.Helper()
	hunks, err := apply.ParseUnifiedDiff(diff)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff(%q): %v", diff, err)
	}
	return hunks
}

func wantHunks(t *testing.T, got []apply.Hunk, want ...apply.Hunk) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %#v, got %#v", want, got)
	}
}

// --- CollectDiff: `_collect` ------------------------------------------------

func TestCollectDiffReturnsEveryDiffFenceInOrder(t *testing.T) {
	body := "Text.\n\n```diff\n@@ -1 +1 @@\n-a\n+b\n```\n\n```python\nx\n```\n\n" +
		"```diff-extra\n@@ -3 +3 @@\n-c\n+d\n```\n"
	got, err := apply.CollectDiff(body)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"@@ -1 +1 @@\n-a\n+b", "@@ -3 +3 @@\n-c\n+d"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %q, got %q", want, got)
	}
}

// test_a_claim_without_a_diff_block_is_refused (test_apply.py:436)
func TestCollectDiffRefusesABodyWithoutADiffFence(t *testing.T) {
	_, err := apply.CollectDiff("Text.\n\n```python\nx\n```\n")
	refused(t, err, "no proposed diff in the claim's section")
}

// An unterminated fence is no fence: FencedBlocks skips it.
func TestCollectDiffRefusesAnUnterminatedDiffFence(t *testing.T) {
	_, err := apply.CollectDiff("```diff\n@@ -1 +1 @@\n-a\n+b\n")
	refused(t, err, "no proposed diff in the claim's section")
}

// --- ParseUnifiedDiff: `_hunks` -----------------------------------------------

// test_context_lines_and_the_no_newline_marker_are_read (test_apply.py:773)
func TestParseUnifiedDiffReadsContextAndSkipsTheNoNewlineMarker(t *testing.T) {
	got := parse(t, "@@ -10,2 +10,2 @@\n-old line 1\n-old line 2\n+new line 1\n+new line 2\n"+
		" context line\n\\ No newline at end of file")
	wantHunks(t, got, apply.Hunk{
		At:  9,
		Old: []string{"old line 1", "old line 2", "context line"},
		New: []string{"new line 1", "new line 2", "context line"},
	})
}

// `_hunks` splits without trimming: a trailing newline is one more empty
// context line. The real path never sees it, FencedBlocks strips the body.
func TestParseUnifiedDiffKeepsATrailingNewlineAsAnEmptyContextLine(t *testing.T) {
	got := parse(t, "@@ -10,2 +10,2 @@\n-a\n+b\n")
	wantHunks(t, got, apply.Hunk{At: 9, Old: []string{"a", ""}, New: []string{"b", ""}})
}

// Nor does `_hunks` fold line ends; a carriage return stays in the payload.
func TestParseUnifiedDiffDoesNotFoldCarriageReturns(t *testing.T) {
	got := parse(t, "@@ -1 +1 @@\r\n-a\r\n+b")
	wantHunks(t, got, apply.Hunk{At: 0, Old: []string{"a\r"}, New: []string{"b"}})
}

// test_a_pure_insertion_hunk_lands_where_it_says (test_apply.py:456)
func TestParseUnifiedDiffAnchorsAPureInsertionAtItsStart(t *testing.T) {
	got := parse(t, "@@ -5,0 +6,2 @@\n+inserted 1\n+inserted 2")
	wantHunks(t, got, apply.Hunk{At: 5, New: []string{"inserted 1", "inserted 2"}})
}

// test_a_bare_empty_line_counts_as_context (test_apply.py:785)
func TestParseUnifiedDiffReadsABareEmptyLineAsContext(t *testing.T) {
	got := parse(t, "@@ -1,3 +1,3 @@\nline 1\n\nline 3")
	wantHunks(t, got, apply.Hunk{
		At:  0,
		Old: []string{"line 1", "", "line 3"},
		New: []string{"line 1", "", "line 3"},
	})
}

// Python's `\d` and `int()` read every Unicode decimal digit, mixed scripts too.
func TestParseUnifiedDiffReadsUnicodeDigitsInTheHeader(t *testing.T) {
	got := parse(t, "@@ -١٢,٠ +1 @@\n+x\n@@ -2٢ +1 @@\n-y")
	wantHunks(t, got,
		apply.Hunk{At: 12, New: []string{"x"}},
		apply.Hunk{At: 21, Old: []string{"y"}},
	)
}

// Before the first header only blank lines may stand, blank as `str.strip`
// sees it: U+001C is whitespace to Python and not to Go's unicode.IsSpace.
func TestParseUnifiedDiffSkipsLinesPythonCallsBlankBeforeTheHeader(t *testing.T) {
	got := parse(t, "\x1c\n\n@@ -1 +1 @@\n-x")
	wantHunks(t, got, apply.Hunk{At: 0, Old: []string{"x"}})
}

func TestParseUnifiedDiffRefusals(t *testing.T) {
	for name, diff := range map[string]string{
		"garbage before the header": "garbage before header\n@@ -1,1 +1,1 @@\n-a\n+b",
		// test_a_diff_without_a_hunk_header_is_refused (test_apply.py:430)
		"no header at all": "- alt\n+ neu",
		// test_an_empty_diff_block_is_refused (test_apply.py:794)
		"empty block":                       "",
		"only blank lines":                  "\n\n",
		"a header whose number is no digit": "@@ -a +1 @@\n-x",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := apply.ParseUnifiedDiff(diff)
			refused(t, err, "diff does not start with a hunk header")
		})
	}
}

// --- ApplyHunks: `_patch` ------------------------------------------------------

func TestApplyHunksAppliesEveryHunk(t *testing.T) {
	got, err := apply.ApplyHunks("line 1\nline 2\nline 3\nline 4\nline 5", []apply.Hunk{
		{At: 1, Old: []string{"line 2"}, New: []string{"line 2 modified"}},
		{At: 3, Old: []string{"line 4"}, New: []string{"line 4 modified"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := "line 1\nline 2 modified\nline 3\nline 4 modified\nline 5"; got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestApplyHunksSortsHunksByTheirAnchor(t *testing.T) {
	got, err := apply.ApplyHunks("first\nsecond\nthird", []apply.Hunk{
		{At: 2, Old: []string{"third"}, New: []string{"third modified"}},
		{At: 0, Old: []string{"first"}, New: []string{"first modified"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := "first modified\nsecond\nthird modified"; got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

// Python's `sorted` is stable: insertions at one anchor keep the diff's order.
func TestApplyHunksKeepsTheOrderOfInsertionsAtOneAnchor(t *testing.T) {
	var hunks []apply.Hunk
	want := "a"
	for _, word := range []string{"p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z", "o", "n", "m"} {
		hunks = append(hunks, apply.Hunk{At: 1, New: []string{word}})
		want += "\n" + word
	}
	got, err := apply.ApplyHunks("a", hunks)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

// test_a_pure_insertion_hunk_lands_where_it_says (test_apply.py:456)
func TestApplyHunksLandsAPureInsertionAfterTheLineItNames(t *testing.T) {
	got, err := apply.ApplyHunks("a\nb", parse(t, "@@ -1,000 +1 @@\n+x"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "a\nx\nb"; got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestApplyHunksRefusals(t *testing.T) {
	for _, c := range []struct {
		name, page string
		hunks      []apply.Hunk
		want       string
	}{
		// test_two_hunks_touching_the_same_line_are_refused (test_apply.py:444)
		{"overlap", "line 1\nline 2\nline 3", []apply.Hunk{
			{At: 0, Old: []string{"line 1", "line 2"}, New: []string{"new 1", "new 2"}},
			{At: 1, Old: []string{"line 2"}, New: []string{"conflict"}},
		}, "hunks overlap at line 2"},
		// test_an_insertion_past_the_end_of_the_page_is_refused (test_apply.py:926)
		{"past the end", "line 1\nline 2", []apply.Hunk{
			{At: 5, New: []string{"too far"}},
		}, "hunk at line 6 reaches past the end of the page"},
		// test_a_diff_whose_context_does_not_match_is_a_malformed_proposal (test_apply.py:424)
		{"context mismatch", "line 1\nline 2\nline 3", []apply.Hunk{
			{At: 1, Old: []string{"wrong line content"}, New: []string{"replacement"}},
		}, "hunk at line 2 does not match the page"},
		{"old lines past the end", "line 1\nline 2", []apply.Hunk{
			{At: 1, Old: []string{"line 2", "line 3 (missing)"}, New: []string{"replacement"}},
		}, "hunk at line 2 does not match the page"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := apply.ApplyHunks(c.page, c.hunks)
			refused(t, err, c.want)
		})
	}
}

// A hunk built by hand at the largest int still names the next line.
func TestApplyHunksNamesTheLineAfterTheLargestAnchor(t *testing.T) {
	_, err := apply.ApplyHunks("a", []apply.Hunk{{At: math.MaxInt, New: []string{"x"}}})
	refused(t, err, "hunk at line "+new(big.Int).Add(big.NewInt(math.MaxInt), big.NewInt(1)).String()+
		" reaches past the end of the page")
}

// The numbers in these messages are Python's, read with `_hunks` and
// `_patch` over the page "a\nb": a line zero anchors before the page, and a
// number past what an int holds is still printed in full.
func TestApplyHunksReportsTheLinePythonReports(t *testing.T) {
	for _, c := range []struct{ diff, want string }{
		{"@@ -0 +1 @@\n-x", "hunks overlap at line 0"},
		{"@@ -99999999999999999999999 +1 @@\n-x",
			"hunk at line 99999999999999999999999 reaches past the end of the page"},
		{"@@ -99999999999999999999999,0 +1 @@\n+x",
			"hunk at line 100000000000000000000000 reaches past the end of the page"},
		{"@@ -9223372036854775808 +1 @@\n-x",
			"hunk at line 9223372036854775808 reaches past the end of the page"},
		{"@@ -100000000000000000000 +1 @@\n-x\n@@ -99999999999999999999 +1 @@\n-y",
			"hunk at line 99999999999999999999 reaches past the end of the page"},
		{"@@ -99999999999999999999 +1 @@\n-x\n@@ -99999999999999999998 +1 @@\n-y",
			"hunk at line 99999999999999999998 reaches past the end of the page"},
	} {
		t.Run(c.diff, func(t *testing.T) {
			_, err := apply.ApplyHunks("a\nb", parse(t, c.diff))
			refused(t, err, c.want)
		})
	}
}
