package maintenance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// goldenHunks are the inputs of testdata/hunks.golden.json, by the same names
// and in the same shapes the oracle builds them from (its script is in the
// archive release archive/parity-recordings). Every one of them is there for a
// rule of `_hunks`, and the comment on each says which.
func goldenHunks() map[string]Changed {
	ten := repeatLines(1, 10)
	twenty := repeatLines(1, 20)
	long := repeatLines(1, 250)
	baseline := func(text string) *string { return &text }
	return map[string]Changed{
		// One changed line in the middle: one hunk with three lines of context.
		"middle": {Relative: "src/a.go",
			Text: strings.Replace(ten, "line 5\n", "line five\n", 1), Baseline: baseline(ten)},
		// Two changes far apart: the grouping splits them into two hunks.
		"two hunks": {Relative: "src/a.go",
			Text: strings.Replace(strings.Replace(twenty, "line 2\n", "line two\n", 1),
				"line 18\n", "line eighteen\n", 1),
			Baseline: baseline(twenty)},
		// The new text has no trailing newline, so the marker applies to its
		// last line -- and to nothing else.
		"no newline": {Relative: "src/a.go", Text: "a\nb\nc", Baseline: baseline("a\nb\nc\n")},
		// The old text has none either: the marker lands on a removed line too.
		"no newline before": {Relative: "src/a.go", Text: "a\nb\nc\n", Baseline: baseline("a\nb\nc")},
		// No verified baseline: `/dev/null`, the note above the header, and
		// the whole new file as one hunk.
		"no baseline": {Relative: "src/a.go", Text: "a\nb\n", Baseline: nil},
		// A committed empty file is a baseline that happens to be empty: the
		// header names the file and not `/dev/null`. This is the pair the
		// pointer in Changed.Baseline exists for.
		"empty baseline": {Relative: "src/a.go", Text: "a\nb\n", Baseline: baseline("")},
		// The new text is empty, the old one was not.
		"emptied": {Relative: "src/a.go", Text: "", Baseline: baseline("a\nb\n")},
		// An added empty line is a `+` on its own, and it keeps the newline
		// that belongs to it -- TrimSuffix, not a trim of all whitespace.
		"added empty line": {Relative: "src/a.go", Text: "a\n\nb\n", Baseline: baseline("a\nb\n")},
		// A lone carriage return stays inside a line: pyLines splits at `\n`
		// and nowhere else.
		"lone carriage return": {Relative: "src/a.go", Text: "a\rb\nc\n", Baseline: baseline("a\nc\n")},
		// Past the 200-line mark where autojunk starts, with a blank line
		// repeated often enough to be struck from the index as popular.
		"autojunk": {Relative: "src/a.go",
			Text: strings.Replace(strings.Replace(long, "line 7\n", "\n", 1),
				"line 200\n", "\n", 1) + strings.Repeat("\n", 20) + "tail\n",
			Baseline: baseline(long + strings.Repeat("\n", 20))},
		// Nothing moved: `unified_diff` yields nothing at all, so there is no
		// header to take the first two lines from.
		"identical": {Relative: "src/a.go", Text: ten, Baseline: baseline(ten)},
		// A path with a space and a non-ASCII name, both of which go into the
		// header verbatim.
		"umlaut path": {Relative: "src/ä b.md", Text: "neu\n", Baseline: baseline("alt\n")},
		// Both sides empty: there is no opcode at all, and difflib invents one
		// so its two fixups have something to index.
		"both empty": {Relative: "src/a.go", Text: "", Baseline: baseline("")},
		// The case autojunk decides on its own, and the only one in this set
		// that dies when the rule is left out: every second line of b is the
		// same one, 100 times over 200 lines, so it counts as popular and is
		// struck from the index. The matcher then finds no anchor and the
		// whole file becomes one replacement -- without the rule the same
		// input yields 200 opcodes and a hunk per repeated line.
		"autojunk strikes a popular line": {Relative: "src/a.go",
			Text: alternating("v"), Baseline: baseline(alternating("u"))},
		// A struck line in front of the anchor: the matcher finds a one-line
		// block on a unique line and then grows it outwards over the popular
		// ones, which it may because a struck line is not junk.
		"a struck line grows the block": {Relative: "src/a.go",
			Text:     strings.Replace(leadingX(), "u50\n", "CHANGED\n", 1),
			Baseline: baseline(leadingX())},
		// A line repeated before and after the longest block, so the recursion
		// into the box on its left meets an occurrence beyond that box's end
		// and the one into the box on its right meets occurrences before its
		// start -- the two arms that skip an index out of range.
		"a repeated line on both sides": {Relative: "src/a.go",
			Text:     "m\nA\nm\np\nq\nr\ns\nt\nm\nZ\nm\n",
			Baseline: baseline("m\na\nm\np\nq\nr\ns\nt\nm\nz\nm\n")},
		// Below the 200-line mark autojunk does not start at all, however
		// popular a line is: 150 lines, every second one the same, and the
		// matcher still anchors on it.
		"autojunk stays off below the mark": {Relative: "src/a.go",
			Text: alternatingN("v", 75), Baseline: baseline(alternatingN("u", 75))},
		// Exactly `len(b)/100 + 1` occurrences, which is one too few to be
		// popular: over 200 lines a line appearing three times stays in the
		// index, and it is the only line the two sides still share.
		"three occurrences are not popular": {Relative: "src/a.go",
			Text: threeShared("v"), Baseline: baseline(threeShared("u"))},
		// Three lines against one, and the one is repeated: the matcher has to
		// take the **first** of the two equally long blocks, so the kept line
		// is the leading one and the two added ones follow. `>=` in place of
		// `>` takes the later block and puts the kept line at the end. The
		// corpus below finds this on its own; the vector is here because it is
		// the smallest input that says what the comparison decides.
		"the first of two equal blocks": {Relative: "src/a.go",
			Text: "L1\nL0\nL1\n", Baseline: baseline("L1\n")},
		// Five unchanged lines between two changes: fewer than the six that
		// would end the group, so the two stay in one hunk.
		"one hunk over a short stretch": {Relative: "src/a.go",
			Text: strings.Replace(strings.Replace(twenty, "line 2\n", "line two\n", 1),
				"line 8\n", "line eight\n", 1),
			Baseline: baseline(twenty)},
	}
}

// alternating is 200 lines of which every second one is the same: the shape
// that makes autojunk strike, with the prefix naming the side.
func alternating(prefix string) string {
	return alternatingN(prefix, 100)
}

// alternatingN is the same shape at a length the caller picks, so a run below
// the 200-line mark can ask the same question.
func alternatingN(prefix string, pairs int) string {
	var out strings.Builder
	for i := 0; i < pairs; i++ {
		fmt.Fprintf(&out, "%s%d\nx\n", prefix, i)
	}
	return out.String()
}

// threeShared is 200 lines of which exactly three are the same on both sides:
// one under the popularity mark, and the only anchor the matcher has.
func threeShared(prefix string) string {
	var out strings.Builder
	for i := 0; i < 200; i++ {
		if i == 0 || i == 100 || i == 199 {
			out.WriteString("x\n")
			continue
		}
		fmt.Fprintf(&out, "%s%d\n", prefix, i)
	}
	return out.String()
}

// leadingX is alternating's mirror image: the repeated line comes first, so a
// block anchored on a unique line has a struck one in front of it to grow over.
func leadingX() string {
	var out strings.Builder
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&out, "x\nu%d\n", i)
	}
	return out.String()
}

func repeatLines(from, to int) string {
	var out strings.Builder
	for i := from; i <= to; i++ {
		fmt.Fprintf(&out, "line %d\n", i)
	}
	return out.String()
}

// The hunks are an interface, not a rendering preference: they become the `D`
// segments of `package.md`, and the evidence check of stage 3b looks a
// proposal's quote up inside one of them. So they are pinned against a file
// the reference's own `_hunks` wrote, case by case.
func TestHunksMatchThePythonDiff(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "hunks.golden.json"))
	if err != nil {
		t.Fatalf("ReadFile golden: %v", err)
	}
	var want map[string][]string
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatalf("Unmarshal golden: %v", err)
	}
	inputs := goldenHunks()
	if len(inputs) != len(want) {
		t.Fatalf("the golden holds %d cases, the inputs %d", len(want), len(inputs))
	}
	for name, item := range inputs {
		t.Run(name, func(t *testing.T) {
			expected, known := want[name]
			if !known {
				t.Fatalf("the golden knows no case %q", name)
			}
			got := hunksOf(item)
			if len(got) != len(expected) {
				t.Fatalf("hunksOf gave %d hunks, want %d:\ngot  %q\nwant %q",
					len(got), len(expected), got, expected)
			}
			for i := range got {
				if got[i] != expected[i] {
					t.Fatalf("hunk %d differs:\ngot  %q\nwant %q", i, got[i], expected[i])
				}
			}
		})
	}
}

// The twenty vectors above are readable, one rule each -- and they are not
// enough. Three mutants in the dynamic-programming core of findLongestMatch
// survive them: dropping `j2len = newj2len`, reading `j2len[j]` instead of
// `j2len[j-1]`, and taking `>=` for `>` when a longer block is chosen. Each of
// them needs a repeated line in a particular place, and choosing those places
// by hand is guessing at the shape of a bug.
//
// So a corpus stands beside the vectors: 400 pairs from a fixed seed, over
// small alphabets and over lengths on both sides of the 200-line autojunk
// mark, every answer taken from the reference's own `_hunks`. The inputs live
// in the golden with the answers, so no second copy of the corpus can drift
// away from it.
func TestHunksMatchThePythonDiffOverTheCorpus(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "hunks-fuzz.golden.json"))
	if err != nil {
		t.Fatalf("ReadFile golden: %v", err)
	}
	var corpus struct {
		Inputs map[string][2]string `json:"inputs"`
		Hunks  map[string][]string  `json:"hunks"`
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatalf("Unmarshal golden: %v", err)
	}
	if len(corpus.Inputs) != 400 || len(corpus.Hunks) != len(corpus.Inputs) {
		t.Fatalf("the corpus holds %d inputs and %d answers, want 400 of each",
			len(corpus.Inputs), len(corpus.Hunks))
	}
	for name, pair := range corpus.Inputs {
		baseline := pair[0]
		got := hunksOf(Changed{
			DocID: "d", Relative: "src/a.go", Revision: 1, ContentHash: "sha256:00",
			Text: pair[1], Baseline: &baseline,
		})
		want := corpus.Hunks[name]
		if len(got) != len(want) {
			t.Fatalf("%s: %d hunks, want %d\nbaseline %q\ntext %q\ngot %q\nwant %q",
				name, len(got), len(want), pair[0], pair[1], got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("%s: hunk %d differs\nbaseline %q\ntext %q\ngot  %q\nwant %q",
					name, i, pair[0], pair[1], got[i], want[i])
			}
		}
	}
}
