package maintenance_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/maintenance"
)

// packageCase is the case the golden file below was rendered from. Its zone is
// +02:00 and not UTC, so the offset in `generated.at` is one a formatter that
// wrote `Z` for UTC would still get wrong.
func packageCase() maintenance.Case {
	return maintenance.Case{
		ID: "loomux-2026-09-19-abcd", Area: "project/loomux",
		Target: "docs/wiki/a.md", TargetHash: "sha256:aa",
		State: "source_changed", Trigger: "source_change", Weight: "change",
		Created: time.Date(2026, 9, 19, 10, 0, 0, 0, time.FixedZone("", 2*3600)),
	}
}

// goldenSegments are the inputs of testdata/package.golden.md, each one chosen
// for a rendering rule: a body ending in a newline, a body carrying a fence of
// its own, a body carrying a lone carriage return before a longer fence, a
// first line past the label limit in multi-byte runes, a paragraph with
// whitespace on both ends, and a citation whose resource is past that same
// limit.
func goldenSegments() []maintenance.Segment {
	return maintenance.BuildPackage(
		[]string{
			"package a\n",
			"--- a/x\n+++ b/x\n```\nfenced\n```\ntail",
			"a\rb\n```` run\nc",
			strings.Repeat("ä", 85) + "\nsecond line",
		},
		[]string{"  leading and trailing  \n\nrest"},
		[]maintenance.Citation{
			{DocID: "doc-1", Resource: "src/" + strings.Repeat("z", 100) + ".md"},
			{DocID: "doc-2", Resource: "src/b.md"},
		},
	)
}

// The whole format at once, against a file `render_package` itself wrote. The
// package is an interface: the evidence check of stage 3b quotes out of these
// segments and finds them by their headings, so a shifted line is a broken
// contract rather than a matter of taste.
func TestRenderPackageMatchesThePythonPackage(t *testing.T) {
	want, err := os.ReadFile("testdata/package.golden.md")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := maintenance.RenderPackage(packageCase(), goldenSegments())
	if got != string(want) {
		t.Fatalf("package differs from the Python rendering:\ngot:\n%q\nwant:\n%q", got, string(want))
	}
}

func TestRenderPackageCarriesTheCaseHead(t *testing.T) {
	c := packageCase()
	text := maintenance.RenderPackage(c, []maintenance.Segment{
		{Number: "D1", Kind: "D", Label: "src/a.go", Body: "package a\n"},
	})
	if !strings.Contains(text, "case: "+c.ID) {
		t.Fatalf("package does not name the case:\n%s", text)
	}
	if !strings.Contains(text, "## D1 — Diff, src/a.go") {
		t.Fatalf("package does not head the segment:\n%s", text)
	}
	if !strings.Contains(text, "segments: 1") {
		t.Fatalf("package does not count its segments:\n%s", text)
	}
}

// `time.RFC3339` writes `Z` for UTC where Python's `isoformat` writes
// `+00:00`. The golden case above carries an offset of its own and would not
// catch that one character; this one is the case every caller builds.
func TestRenderPackageWritesTheOffsetPythonWrites(t *testing.T) {
	c := packageCase()
	c.Created = time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	text := maintenance.RenderPackage(c, nil)
	if !strings.Contains(text, "generated.at: 2026-09-19T10:00:00+00:00\n") {
		t.Fatalf("package does not stamp the way Python stamps:\n%s", text)
	}
}

// A body that ends in a newline leaves a blank line inside its fence. That is
// what the reference writes, and a reader that counts lines inside a segment
// would be thrown off by a renderer that tidied it away.
func TestRenderPackageKeepsABodysTrailingNewline(t *testing.T) {
	text := maintenance.RenderPackage(packageCase(), []maintenance.Segment{
		{Number: "D1", Kind: "D", Label: "a", Body: "package a\n"},
	})
	if !strings.Contains(text, "```\npackage a\n\n```\n") {
		t.Fatalf("package trimmed the body:\n%q", text)
	}
}

// A package with no segment is the frontmatter and nothing else, and it ends
// in exactly one newline.
func TestRenderPackageOfNoSegmentsIsTheHeadAlone(t *testing.T) {
	want := "---\ncase: loomux-2026-09-19-abcd\ngenerated.at: 2026-09-19T10:00:00+02:00\nsegments: 0\n---\n"
	if got := maintenance.RenderPackage(packageCase(), nil); got != want {
		t.Fatalf("empty package is\n%q\nwant\n%q", got, want)
	}
}

// Python subscripts `_KIND_LABELS` and raises KeyError on a kind it has no
// word for. A Go map lookup would hand back "" and write a heading that names
// no kind at all -- a heading the reader of stage 3b still parses, so the
// quiet reading is the dangerous one.
func TestRenderPackageRefusesAKindItHasNoWordFor(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("RenderPackage rendered a segment of an unknown kind")
		}
	}()
	maintenance.RenderPackage(packageCase(), []maintenance.Segment{
		{Number: "S1", Kind: "S", Label: "a", Body: "x"},
	})
}

// The fence wraps a segment, so it has to be longer than any backtick run the
// body already carries. Each case here is its own reason.
func TestFenceForEscalatesPastTheBodysOwnRuns(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want string
	}{
		// Nothing to escalate past: three is the floor.
		{"plain", "package a\n", "```"},
		// A run at the start of a line closes a wrapper of equal length.
		{"fenced", "```\nx\n```", "````"},
		// Indented by up to the pattern's leading whitespace, still a fence.
		{"indented", "  ````\nx", "`````"},
		// A run in the middle of a line closes nothing, so it must not count.
		{"mid line", "a ``` b", "```"},
		// A tilde block cannot close a backtick wrapper; counting it would
		// only make the wrapper needlessly longer.
		{"tildes", "~~~~~~\nx\n~~~~~~", "```"},
		// The measured forgery: a lone carriage return is a line break to
		// every reader of a package, but not to the regexp's `^`. Counted
		// unfolded, the run behind it is invisible and the wrapper stays at
		// three -- and the reader then ends the segment early.
		{"lone carriage return", "a\r````\nb", "`````"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := maintenance.FenceFor(test.body); got != test.want {
				t.Fatalf("FenceFor(%q) = %q, want %q", test.body, got, test.want)
			}
		})
	}
}

// Numbering is per kind and starts at 1, and the kinds come in the order the
// reader of stage 3b expects them in.
func TestBuildPackageNumbersEveryKindOnItsOwn(t *testing.T) {
	segments := maintenance.BuildPackage(
		[]string{"d one", "d two"},
		[]string{"w one"},
		[]maintenance.Citation{{DocID: "doc-1", Resource: "src/a.md"}},
	)
	var numbers []string
	for _, segment := range segments {
		numbers = append(numbers, segment.Number+segment.Kind)
	}
	if got := strings.Join(numbers, " "); got != "D1D D2D W1W Q1Q" {
		t.Fatalf("numbering is %q", got)
	}
	if segments[3].Body != "doc_id: doc-1\nresource: src/a.md" {
		t.Fatalf("citation body is %q", segments[3].Body)
	}
	if segments[0].Label != "d one" || segments[2].Label != "w one" {
		t.Fatalf("labels are %q and %q", segments[0].Label, segments[2].Label)
	}
}

// A citation's label is its resource, whole. `_label` is not applied there,
// and applying it everywhere is the tidy-looking mistake.
func TestBuildPackageLeavesALongResourceWhole(t *testing.T) {
	resource := strings.Repeat("z", 200) + ".md"
	segments := maintenance.BuildPackage(nil, nil, []maintenance.Citation{
		{DocID: "doc-1", Resource: resource},
	})
	if segments[0].Label != resource {
		t.Fatalf("resource label is %q", segments[0].Label)
	}
}

// The label is cut at 80 code points, not at 80 bytes: `len` counts
// characters in Python, and a byte count would cut a line of umlauts at half
// the text and could split a rune in two.
func TestBuildPackageCutsTheLabelByRunesNotBytes(t *testing.T) {
	for _, test := range []struct {
		name  string
		first string
		want  string
	}{
		{"at the limit", strings.Repeat("ä", 80), strings.Repeat("ä", 80)},
		{"one past it", strings.Repeat("ä", 81), strings.Repeat("ä", 79) + "…"},
	} {
		t.Run(test.name, func(t *testing.T) {
			segments := maintenance.BuildPackage([]string{test.first}, nil, nil)
			if segments[0].Label != test.want {
				t.Fatalf("label is %q (%d runes), want %q", segments[0].Label,
					len([]rune(segments[0].Label)), test.want)
			}
		})
	}
}

// The label is the first line of the stripped text, by Python's two rules:
// `str.strip` takes off the 29 characters Python calls space, and
// `str.splitlines` breaks at ten boundaries rather than at `\n` alone.
func TestBuildPackageTakesTheFirstLinePythonsWay(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want string
	}{
		// \x1c is space to `strip` and a break to `splitlines`. Left standing,
		// the first line would be the empty one before it.
		{"file separator", "\x1cfirst\nsecond", "first"},
		// A lone carriage return breaks a line for `splitlines`, and
		// `reconcile._decode` carries one into a diff on purpose.
		{"lone carriage return", "a\rb", "a"},
		{"blank", "   \n  ", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			segments := maintenance.BuildPackage([]string{test.body}, nil, nil)
			if segments[0].Label != test.want {
				t.Fatalf("label is %q, want %q", segments[0].Label, test.want)
			}
		})
	}
}
