package apply_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/apply"
	"github.com/xidus90/loomux/internal/brain/identity"
)

// goldenNow, goldenReviewer and goldenUpdates are the values the generator
// handed `_advance` when it wrote testdata/frontmatter: every update carries
// revision 7, which no page holds, so a golden shows whose revision advanced.
var goldenNow = time.Date(2026, 9, 22, 8, 16, 27, 936837000, time.UTC)

const goldenReviewer = "human:tester"

func goldenUpdates() []apply.SourceUpdate {
	ids := []string{
		"01M0QGS594F2KCWTWK9XV07M05",
		"01M0QGS58WEG5TSWKWEJM89E5T",
		"01M0QGS5931BKD9QS6B8C2TNCT",
		"01M0QGS593D5A8B9E3P0SV6GD9",
		"01M0QGS593NNV4TQ4XFBRQWD2K",
		"01SYNTH",
		"123",
		"True",
		"1.5",
		"2026-01-01",
		"None",
		"01NOMATCH",
	}
	updates := make([]apply.SourceUpdate, len(ids))
	for i, id := range ids {
		updates[i] = apply.SourceUpdate{DocID: id, ContentHash: "sha256:" + strings.Repeat("0", 64), Revision: 7}
	}
	return updates
}

// knownRefusals are inputs the reference renders and this port refuses on
// purpose; each is a row of the parity list.
var knownRefusals = map[string]string{
	"x01-alias":        "an alias to a collection: PyYAML writes an &id001 anchor",
	"x02-date-alias":   "an alias to a date: PyYAML writes an &id001 anchor",
	"x03-explicit-tag": "an explicit tag",
	"x04-merge":        "a merge key",
	// yaml.v3's scanner is stricter than PyYAML's here, and the parse fails
	// before the port sees a node.
	"x05-tab-first-in-nested-block":          "a tab first on a block scalar's first line",
	"x06-tab-first-in-nested-sequence-block": "a tab first on a block scalar's first line",
	"x07-tab-first-in-block":                 "a tab first on a block scalar's first line",
	"x08-slash-escape":                       `the escape \/ in a double-quoted scalar`,
}

func TestAdvanceFrontmatterMatchesTheReference(t *testing.T) {
	inputs, err := filepath.Glob(filepath.Join("testdata", "frontmatter", "*.in.md"))
	if err != nil || len(inputs) == 0 {
		t.Fatalf("no golden inputs: %v", err)
	}
	for _, input := range inputs {
		name := strings.TrimSuffix(filepath.Base(input), ".in.md")
		t.Run(name, func(t *testing.T) {
			page, err := os.ReadFile(input)
			if err != nil {
				t.Fatal(err)
			}
			got, err := apply.AdvanceFrontmatter(string(page), goldenUpdates(), goldenReviewer, goldenNow)
			base := strings.TrimSuffix(input, ".in.md")
			if _, refused := os.Stat(base + ".err"); refused == nil {
				if err == nil {
					t.Fatalf("the reference refuses this page, the port wrote:\n%s", got)
				}
				return
			}
			if why, ok := knownRefusals[name]; ok {
				if err == nil {
					t.Fatalf("expected a refusal (%s), got:\n%s", why, got)
				}
				return
			}
			want, rerr := os.ReadFile(base + ".out.md")
			if rerr != nil {
				t.Fatal(rerr)
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != string(want) {
				t.Fatalf("output differs from the reference\n--- got\n%s\n--- want\n%s", got, want)
			}
		})
	}
}

func TestAdvanceFrontmatterNamesTheMissingFrontmatter(t *testing.T) {
	_, err := apply.AdvanceFrontmatter("no frontmatter\n", nil, goldenReviewer, goldenNow)
	if err == nil || err.Error() != "the target page has no frontmatter to advance" {
		t.Fatalf("got %v", err)
	}
	_, err = apply.AdvanceFrontmatter("---\n- a\n---\n", nil, goldenReviewer, goldenNow)
	if err == nil || err.Error() != "frontmatter is not a mapping" {
		t.Fatalf("got %v", err)
	}
}

func TestIsoFormatIsPythonsIsoformat(t *testing.T) {
	for _, tc := range []struct {
		in   time.Time
		want string
	}{
		{goldenNow, "2026-09-22T08:16:27.936837+00:00"},
		{time.Date(2026, 9, 22, 8, 16, 27, 0, time.UTC), "2026-09-22T08:16:27+00:00"},
		{time.Date(2026, 9, 22, 10, 16, 27, 5000, time.FixedZone("", 2*3600)), "2026-09-22T08:16:27.000005+00:00"},
	} {
		if got := apply.IsoFormat(tc.in); got != tc.want {
			t.Errorf("IsoFormat(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestAdvanceRegisterMovesItsRowsAndKeepsTheRest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "_identities.tsv")
	before := identity.IdentitiesHeader + "\n" +
		"01A\ta.md\tsha256:a\t1\n" +
		"01B\tb.md\tsha256:b\t3\n" +
		"01C\tc.md\tsha256:c\t5\n"
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	// The case knows revisions 3 and 5 of its sources; the register gets 4
	// and 6.
	caseStates := map[string]identity.Identity{
		"b.md": {DocID: "01B", Relative: "b.md", ContentHash: "sha256:new", Revision: 3},
		"c.md": {DocID: "01C", Relative: "c.md", ContentHash: "sha256:newer", Revision: 5},
	}
	got, err := apply.AdvanceRegister(path, caseStates)
	if err != nil {
		t.Fatal(err)
	}
	want := identity.IdentitiesHeader + "\n" +
		"01A\ta.md\tsha256:a\t1\n" +
		"01B\tb.md\tsha256:new\t4\n" +
		"01C\tc.md\tsha256:newer\t6\n"
	if got != want {
		t.Fatalf("register:\n%q\nwant\n%q", got, want)
	}
	standing, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(standing) != before {
		t.Fatalf("AdvanceRegister wrote the register; the barrier's write is the caller's")
	}
	if caseStates["b.md"].Revision != 3 {
		t.Fatalf("the caller's case state was advanced in place")
	}
}

func TestAdvanceRegisterStartsAMissingRegister(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "_identities.tsv")
	caseStates := map[string]identity.Identity{
		"a.md": {DocID: "01A", Relative: "a.md", ContentHash: "sha256:a", Revision: 1},
	}
	got, err := apply.AdvanceRegister(path, caseStates)
	if err != nil {
		t.Fatal(err)
	}
	if want := identity.IdentitiesHeader + "\n01A\ta.md\tsha256:a\t2\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatalf("AdvanceRegister created the register's directory: %v", err)
	}
}

func TestAdvanceRegisterRefusesABrokenRegister(t *testing.T) {
	path := filepath.Join(t.TempDir(), "_identities.tsv")
	if err := os.WriteFile(path, []byte(identity.IdentitiesHeader+"\n01A\ta.md\th\tx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := apply.AdvanceRegister(path, nil); err == nil {
		t.Fatal("a register with an unreadable revision was read")
	}
}
