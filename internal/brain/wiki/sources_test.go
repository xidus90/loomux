package wiki

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/identity"
)

const testScope = "project/x"

// testPin is one sources[] entry a test page carries.
type testPin struct {
	path, docID, hash string
	revision          int
}

// page renders a wiki page with the given pins, in the form the bundle
// writes: block list, two-space entries, the hash in double quotes.
func page(title string, pins ...testPin) string {
	var b strings.Builder
	b.WriteString("---\ntype: Topic\ntitle: " + title + "\ndescription: d\n")
	if len(pins) > 0 {
		b.WriteString("sources:\n")
		for i, p := range pins {
			b.WriteString("  - id: s" + string(rune('0'+i)) + "\n")
			b.WriteString("    resource: brain://" + testScope + "/" + p.path + "\n")
			b.WriteString("    doc_id: " + p.docID + "\n")
			b.WriteString("    content_hash: \"" + p.hash + "\"\n")
			b.WriteString("    revision: " + itoa(p.revision) + "\n")
		}
	}
	b.WriteString("---\n\n# " + title + "\n")
	return b.String()
}

func itoa(n int) string { return strconv.Itoa(n) }

// project lays files out by project-relative path and writes the register
// from rows; no rows means no register file.
func project(t *testing.T, files map[string]string, rows ...identity.Identity) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		writeAt(t, root, rel, body)
	}
	if len(rows) > 0 {
		register := map[string]identity.Identity{}
		for _, r := range rows {
			register[r.Relative] = r
		}
		writeAt(t, root, "_identities.tsv", identity.RenderIdentities(register))
	}
	return root
}

func writeAt(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hashOf(s string) string { return identity.HashBytes([]byte(s)) }

func drift(t *testing.T, root string) []SourceFinding {
	t.Helper()
	got, err := SourceDrift(root, filepath.Join(root, "docs", "wiki"), testScope)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// A project where every pin matches its row and every row its file has
// nothing to report: the state master is in when this lane arrives.
func TestSourceDriftIsSilentWhenEverythingMatches(t *testing.T) {
	doc := "# A\n"
	root := project(t, map[string]string{
		"docs/de/a.md":          doc,
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf(doc), 5}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(doc), Revision: 5})
	if got := drift(t, root); len(got) != 0 {
		t.Fatalf("findings %+v", got)
	}
}

// register-stale: a cited file changed and nobody ran reindex. Pin and row still
// agree, so only the register is stale -- the case that left 22 pins behind.
func TestSourceDriftReportsAFileAheadOfItsRow(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	root := project(t, map[string]string{
		"docs/de/a.md":          now,
		"docs/wiki/topics/b.md": page("B", testPin{"docs/de/a.md", "01A", hashOf(old), 3}),
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3})
	want := []SourceFinding{{RuleRegisterStale, "docs/de/a.md", "changed since its register row (revision 3); cited by topics/b.md, topics/t.md"}}
	if got := drift(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("findings %+v\nwant %+v", got, want)
	}
}

// pin-stale: the register moved on (a reindex in the registry's checkout),
// the page did not.
func TestSourceDriftReportsAPinBehindItsRow(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	root := project(t, map[string]string{
		"docs/de/a.md":          now,
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(now), Revision: 4})
	want := []SourceFinding{{RulePinStale, "topics/t.md", "sources[0]: docs/de/a.md pinned at revision 3, the register row is at revision 4"}}
	if got := drift(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("findings %+v\nwant %+v", got, want)
	}
}

// Same revision, another hash: still behind, and the message says what
// differs, since the numbers alone would read as equal.
func TestSourceDriftReportsAPinWithAnotherHashAtTheSameRevision(t *testing.T) {
	doc := "# A\n"
	root := project(t, map[string]string{
		"docs/de/a.md":          doc,
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf("# other\n"), 6}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(doc), Revision: 6})
	want := []SourceFinding{{RulePinStale, "topics/t.md", "sources[0]: docs/de/a.md pinned with another content_hash than its register row (revision 6)"}}
	if got := drift(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("findings %+v\nwant %+v", got, want)
	}
}

// The findings repin cannot fix (row-missing, source-missing, doc-id) stand
// alone: a pin that has one gets no other.
func TestSourceDriftReportsWhatRepinCannotFix(t *testing.T) {
	doc := "# A\n"
	root := project(t, map[string]string{
		"docs/de/a.md": doc,
		"docs/wiki/topics/t.md": page("T",
			testPin{"docs/de/new.md", "01N", hashOf(doc), 1},
			testPin{"docs/de/gone.md", "01G", hashOf(doc), 2},
			testPin{"docs/de/a.md", "01WRONG", hashOf("# x\n"), 9}),
	},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(doc), Revision: 7},
		identity.Identity{DocID: "01G", Relative: "docs/de/gone.md", ContentHash: hashOf(doc), Revision: 2},
	)
	want := []SourceFinding{
		{RuleDocID, "topics/t.md", "sources[2]: doc_id 01WRONG, the register has 01A for docs/de/a.md"},
		{RuleRowMissing, "topics/t.md", "sources[0]: docs/de/new.md is not in the register"},
		{RuleSourceMissing, "topics/t.md", "sources[1]: docs/de/gone.md does not exist"},
	}
	if got := drift(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("findings %+v\nwant %+v", got, want)
	}
}

// A pin of another area, an incomplete entry (house/source-incomplete owns
// it) and a page whose frontmatter does not load (the okf axis owns it) are
// none of this check's business; neither is a changed file nobody cites.
func TestSourceDriftLeavesOtherAxesAndScopesAlone(t *testing.T) {
	doc, uncited := "# A\n", "# U\n"
	foreign := "---\ntype: Topic\ntitle: F\ndescription: d\nsources:\n  - id: f\n    resource: brain://shared/kb/x.md\n    doc_id: 01F\n    content_hash: \"sha256:00\"\n    revision: 1\n---\n"
	incomplete := "---\ntype: Topic\ntitle: I\ndescription: d\nsources:\n  - id: i\n    resource: brain://project/x/docs/de/a.md\n    doc_id: 01A\n---\n"
	broken := "---\ntitle: [unclosed\n---\n"
	root := project(t, map[string]string{
		"docs/de/a.md":          doc,
		"docs/de/u.md":          uncited,
		"docs/wiki/topics/f.md": foreign,
		"docs/wiki/topics/i.md": incomplete,
		"docs/wiki/topics/z.md": broken,
	},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(doc), Revision: 2},
		identity.Identity{DocID: "01U", Relative: "docs/de/u.md", ContentHash: hashOf("# U old\n"), Revision: 8},
	)
	if got := drift(t, root); len(got) != 0 {
		t.Fatalf("findings %+v", got)
	}
}

// A CRLF checkout of a file whose row was hashed with LF is no change.
func TestSourceDriftFoldsCRLF(t *testing.T) {
	root := project(t, map[string]string{
		"docs/de/a.md":          "# A\r\nline\r\n",
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf("# A\nline\n"), 4}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf("# A\nline\n"), Revision: 4})
	if got := drift(t, root); len(got) != 0 {
		t.Fatalf("findings %+v", got)
	}
}

// Pages that pin each other can never both be current; a page that pins
// itself neither. One finding per cycle, naming its pages.
func TestSourceDriftReportsCitationCycles(t *testing.T) {
	a := page("A", testPin{"docs/wiki/topics/b.md", "01B", "sha256:b", 2})
	b := page("B", testPin{"docs/wiki/topics/a.md", "01A", "sha256:a", 3})
	self := page("S", testPin{"docs/wiki/topics/s.md", "01S", "sha256:s", 4})
	root := project(t, map[string]string{
		"docs/wiki/topics/a.md": a,
		"docs/wiki/topics/b.md": b,
		"docs/wiki/topics/s.md": self,
	},
		identity.Identity{DocID: "01A", Relative: "docs/wiki/topics/a.md", ContentHash: "sha256:a", Revision: 3},
		identity.Identity{DocID: "01B", Relative: "docs/wiki/topics/b.md", ContentHash: "sha256:b", Revision: 2},
		identity.Identity{DocID: "01S", Relative: "docs/wiki/topics/s.md", ContentHash: "sha256:s", Revision: 4},
	)
	var cycles []SourceFinding
	for _, f := range drift(t, root) {
		if f.Rule == RuleCycle {
			cycles = append(cycles, f)
		}
	}
	want := []SourceFinding{
		{RuleCycle, "topics/a.md", "pages cite each other: topics/a.md, topics/b.md"},
		{RuleCycle, "topics/s.md", "pages cite each other: topics/s.md"},
	}
	if !reflect.DeepEqual(cycles, want) {
		t.Fatalf("cycles %+v\nwant %+v", cycles, want)
	}
}

// A chain without a cycle is no cycle: a synthesis citing a topic citing a doc.
func TestSourceDriftFindsNoCycleInAChain(t *testing.T) {
	doc := "# A\n"
	topic := page("T", testPin{"docs/de/a.md", "01A", hashOf(doc), 2})
	root := project(t, map[string]string{
		"docs/de/a.md":             doc,
		"docs/wiki/topics/t.md":    topic,
		"docs/wiki/syntheses/s.md": page("S", testPin{"docs/wiki/topics/t.md", "01T", hashOf(topic), 5}),
	},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(doc), Revision: 2},
		identity.Identity{DocID: "01T", Relative: "docs/wiki/topics/t.md", ContentHash: hashOf(topic), Revision: 5},
	)
	if got := drift(t, root); len(got) != 0 {
		t.Fatalf("findings %+v", got)
	}
}

// Nothing to judge: no scope, no wiki, no register.
func TestSourceDriftWithoutScopeWikiOrRegister(t *testing.T) {
	root := project(t, map[string]string{
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", "sha256:a", 1}),
	})
	wikiRoot := filepath.Join(root, "docs", "wiki")
	for _, c := range []struct{ wiki, scope string }{{wikiRoot, ""}, {"", testScope}, {wikiRoot, testScope}} {
		got, err := SourceDrift(root, c.wiki, c.scope)
		if err != nil || got != nil {
			t.Fatalf("%+v: findings %+v, err %v", c, got, err)
		}
	}
}

// A lane that cannot read is the lane's failure, never its silence.
func TestSourceDriftReportsWhatItCannotRead(t *testing.T) {
	root := project(t, map[string]string{"docs/de/a.md": "# A\n"},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: "sha256:a", Revision: 1})
	if _, err := SourceDrift(root, filepath.Join(root, "docs", "missing"), testScope); err == nil {
		t.Fatal("a wiki root that does not exist must fail")
	}
	writeAt(t, root, "_identities.tsv", "doc_id\tpfad\tcontent_hash\trevision\nbroken\n")
	if _, err := SourceDrift(root, filepath.Join(root, "docs"), testScope); err == nil {
		t.Fatal("a register that does not read must fail")
	}
}

// Reading a page or a cited file can fail for reasons the tree cannot
// provoke on every platform; the seam stands in for them.
func TestSourceDriftReportsAReadFailure(t *testing.T) {
	doc := "# A\n"
	root := project(t, map[string]string{
		"docs/de/a.md":          doc,
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf(doc), 2}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(doc), Revision: 2})
	boom := errors.New("denied")
	for _, target := range []string{"t.md", "a.md"} {
		saved := readSource
		readSource = func(path string) ([]byte, error) {
			if filepath.Base(path) == target {
				return nil, boom
			}
			return os.ReadFile(path)
		}
		_, err := SourceDrift(root, filepath.Join(root, "docs", "wiki"), testScope)
		readSource = saved
		if !errors.Is(err, boom) {
			t.Fatalf("%s: err %v", target, err)
		}
	}
}

// rawPage is a page whose sources[] entries are written by hand, so a test
// can leave a field out; each entry is the text of one list item.
func rawPage(entries ...string) string {
	return "---\ntype: Topic\ntitle: R\ndescription: d\nsources:\n" + strings.Join(entries, "") + "---\n"
}

func cyclesOf(t *testing.T, root string) []SourceFinding {
	t.Helper()
	var cycles []SourceFinding
	for _, f := range drift(t, root) {
		if f.Rule == RuleCycle {
			cycles = append(cycles, f)
		}
	}
	return cycles
}

// With a register to read, each of scope, wiki and the register alone keeps
// the check silent: a pin written for the empty scope ("brain:///…") would
// match the prefix of one.
func TestSourceDriftJudgesNothingWithoutScopeOrWiki(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	stale := rawPage("  - id: s\n    resource: brain:///docs/de/a.md\n    doc_id: 01A\n    content_hash: \"" + hashOf(old) + "\"\n    revision: 3\n")
	root := project(t, map[string]string{
		"docs/de/a.md":          now,
		"docs/wiki/topics/t.md": stale,
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(now), Revision: 4})
	wikiRoot := filepath.Join(root, "docs", "wiki")
	for _, c := range []struct{ wiki, scope string }{{wikiRoot, ""}, {"", testScope}} {
		got, err := SourceDrift(root, c.wiki, c.scope)
		if err != nil || got != nil {
			t.Fatalf("%+v: findings %+v, err %v", c, got, err)
		}
	}
}

// The scaffold files, a file that is no page and a directory that merely has
// the name of one carry no pins of the wiki, however stale a pin in them is.
func TestSourceDriftReadsPagesOnly(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	stale := page("X", testPin{"docs/de/a.md", "01A", hashOf(old), 3})
	root := project(t, map[string]string{
		"docs/de/a.md":                     now,
		"docs/wiki/index.md":               stale,
		"docs/wiki/topics/notes.txt":       stale,
		"docs/wiki/topics/dir.md/inner.md": "no frontmatter\n",
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(now), Revision: 4})
	if got := drift(t, root); len(got) != 0 {
		t.Fatalf("findings %+v", got)
	}
}

// An entry that lacks one of doc_id, content_hash and revision is
// house/source-incomplete's, and it lacks exactly one so that no other
// condition hides the one under test.
func TestSourceDriftLeavesAnEntryWithAFieldMissingAlone(t *testing.T) {
	doc := "# A\n"
	h := hashOf(doc)
	root := project(t, map[string]string{
		"docs/de/a.md": doc,
		"docs/wiki/topics/t.md": rawPage(
			"  - id: a\n    resource: brain://"+testScope+"/docs/de/a.md\n    content_hash: \""+h+"\"\n    revision: 4\n",
			"  - id: b\n    resource: brain://"+testScope+"/docs/de/a.md\n    doc_id: 01A\n    revision: 4\n",
			"  - id: c\n    resource: brain://"+testScope+"/docs/de/a.md\n    doc_id: 01A\n    content_hash: \""+h+"\"\n",
		),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: h, Revision: 4})
	if got := drift(t, root); len(got) != 0 {
		t.Fatalf("findings %+v", got)
	}
}

// A doc_id written with a stray space is the register's doc_id all the same.
func TestSourceDriftTrimsTheDocID(t *testing.T) {
	doc := "# A\n"
	root := project(t, map[string]string{
		"docs/de/a.md": doc,
		"docs/wiki/topics/t.md": rawPage(
			"  - id: a\n    resource: brain://" + testScope + "/docs/de/a.md\n    doc_id: \"01A \"\n    content_hash: \"" + hashOf(doc) + "\"\n    revision: 4\n"),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(doc), Revision: 4})
	if got := drift(t, root); len(got) != 0 {
		t.Fatalf("findings %+v", got)
	}
}

// One page citing a file twice is one citer of it.
func TestSourceDriftNamesAPageOnceWhenItCitesAFileTwice(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	pin := testPin{"docs/de/a.md", "01A", hashOf(old), 3}
	root := project(t, map[string]string{
		"docs/de/a.md":          now,
		"docs/wiki/topics/t.md": page("T", pin, pin),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3})
	want := []SourceFinding{{RuleRegisterStale, "docs/de/a.md", "changed since its register row (revision 3); cited by topics/t.md"}}
	if got := drift(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("findings %+v\nwant %+v", got, want)
	}
}

// Findings come in page order first, whatever their rules are.
func TestSourceDriftSortsByPageBeforeRule(t *testing.T) {
	doc := "# A\n"
	root := project(t, map[string]string{
		"docs/de/a.md":          doc,
		"docs/wiki/topics/a.md": page("A", testPin{"docs/de/a.md", "01A", hashOf(doc), 3}),
		"docs/wiki/topics/z.md": page("Z", testPin{"docs/de/a.md", "01WRONG", hashOf(doc), 7}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(doc), Revision: 7})
	want := []SourceFinding{
		{RulePinStale, "topics/a.md", "sources[0]: docs/de/a.md pinned at revision 3, the register row is at revision 7"},
		{RuleDocID, "topics/z.md", "sources[0]: doc_id 01WRONG, the register has 01A for docs/de/a.md"},
	}
	if got := drift(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("findings %+v\nwant %+v", got, want)
	}
}

// A page that cites a finished page and also takes part in a cycle: the
// finished one is not on the stack any more and must not lower the cycle's
// root, or the cycle goes unreported.
func TestSourceDriftFindsACycleBesideAFinishedPage(t *testing.T) {
	root := project(t, map[string]string{
		"docs/wiki/topics/a.md": page("A"),
		"docs/wiki/topics/b.md": page("B",
			testPin{"docs/wiki/topics/c.md", "01C", "sha256:c", 1},
			testPin{"docs/wiki/topics/a.md", "01A", "sha256:a", 1}),
		"docs/wiki/topics/c.md": page("C", testPin{"docs/wiki/topics/b.md", "01B", "sha256:b", 1}),
	}, identity.Identity{DocID: "01A", Relative: "docs/wiki/topics/a.md", ContentHash: "sha256:a", Revision: 1})
	want := []SourceFinding{{RuleCycle, "topics/b.md", "pages cite each other: topics/b.md, topics/c.md"}}
	if got := cyclesOf(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("cycles %+v\nwant %+v", got, want)
	}
}

// A cycle of three whose first page is not where the walk meets it again:
// the low link of the middle page has to come from the last.
func TestSourceDriftFindsACycleOfThree(t *testing.T) {
	root := project(t, map[string]string{
		"docs/wiki/topics/a.md": page("A"),
		"docs/wiki/topics/b.md": page("B", testPin{"docs/wiki/topics/c.md", "01C", "sha256:c", 1}),
		"docs/wiki/topics/c.md": page("C", testPin{"docs/wiki/topics/d.md", "01D", "sha256:d", 1}),
		"docs/wiki/topics/d.md": page("D", testPin{"docs/wiki/topics/b.md", "01B", "sha256:b", 1}),
	}, identity.Identity{DocID: "01A", Relative: "docs/wiki/topics/a.md", ContentHash: "sha256:a", Revision: 1})
	want := []SourceFinding{{RuleCycle, "topics/b.md", "pages cite each other: topics/b.md, topics/c.md, topics/d.md"}}
	if got := cyclesOf(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("cycles %+v\nwant %+v", got, want)
	}
}
