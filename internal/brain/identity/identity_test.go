package identity

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNewDocID(t *testing.T) {
	id1 := NewDocID()
	time.Sleep(2 * time.Millisecond)
	id2 := NewDocID()

	if len(id1) != 26 {
		t.Fatalf("expected ULID length 26, got %d (%s)", len(id1), id1)
	}
	if len(id2) != 26 {
		t.Fatalf("expected ULID length 26, got %d (%s)", len(id2), id2)
	}

	for _, c := range id1 {
		if !strings.ContainsRune(CrockfordAlphabet, c) {
			t.Errorf("character %c not in Crockford alphabet", c)
		}
	}

	if id1 == id2 {
		t.Error("expected distinct doc IDs")
	}

	if id1 >= id2 {
		t.Errorf("expected id1 (%s) < id2 (%s) due to time-ordering", id1, id2)
	}
}

func TestContentHash(t *testing.T) {
	tmp := t.TempDir()
	lfFile := filepath.Join(tmp, "lf.md")
	crlfFile := filepath.Join(tmp, "crlf.md")

	contentLF := []byte("# Heading\nLine 2\nLine 3\n")
	contentCRLF := []byte("# Heading\r\nLine 2\r\nLine 3\r\n")

	if err := os.WriteFile(lfFile, contentLF, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(crlfFile, contentCRLF, 0o644); err != nil {
		t.Fatal(err)
	}

	hashLF, err := ContentHash(lfFile)
	if err != nil {
		t.Fatalf("ContentHash failed: %v", err)
	}
	hashCRLF, err := ContentHash(crlfFile)
	if err != nil {
		t.Fatalf("ContentHash failed: %v", err)
	}

	if !strings.HasPrefix(hashLF, "sha256:") {
		t.Errorf("expected sha256: prefix, got %s", hashLF)
	}

	// Normalisation must yield byte-identical hashes across line ending styles (Spec 5.6)
	if hashLF != hashCRLF {
		t.Errorf("expected identical hash for LF and CRLF, got %s vs %s", hashLF, hashCRLF)
	}

	// Missing file error
	_, err = ContentHash(filepath.Join(tmp, "nonexistent.md"))
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestReadIdentities(t *testing.T) {
	tmp := t.TempDir()
	tsvPath := filepath.Join(tmp, "_identities.tsv")

	// Missing file returns empty map
	m, err := ReadIdentities(tsvPath)
	if err != nil {
		t.Fatalf("expected nil error on missing file, got %v", err)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}

	// Valid file with blank line
	tsvContent := "doc_id\tpfad\tcontent_hash\trevision\n" +
		"01JABCDEFGHJKMNPQRSTVWXYZ0\tnote1.md\tsha256:1111\t1\n" +
		"\n" +
		"01JABCDEFGHJKMNPQRSTVWXYZ1\tnote2.md\tsha256:2222\t3\n"
	if err := os.WriteFile(tsvPath, []byte(tsvContent), 0o644); err != nil {
		t.Fatal(err)
	}

	identities, err := ReadIdentities(tsvPath)
	if err != nil {
		t.Fatalf("ReadIdentities failed: %v", err)
	}
	if len(identities) != 2 {
		t.Fatalf("expected 2 identities, got %d", len(identities))
	}
	id1 := identities["note1.md"]
	if id1.DocID != "01JABCDEFGHJKMNPQRSTVWXYZ0" || id1.Relative != "note1.md" || id1.ContentHash != "sha256:1111" || id1.Revision != 1 {
		t.Errorf("unexpected identity 1: %+v", id1)
	}
	id2 := identities["note2.md"]
	if id2.DocID != "01JABCDEFGHJKMNPQRSTVWXYZ1" || id2.Relative != "note2.md" || id2.ContentHash != "sha256:2222" || id2.Revision != 3 {
		t.Errorf("unexpected identity 2: %+v", id2)
	}

	// Corrupt file: wrong column count
	badCols := "doc_id\tpfad\tcontent_hash\trevision\n" +
		"01JABC\tnote1.md\tsha256:1111\n"
	if err := os.WriteFile(tsvPath, []byte(badCols), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadIdentities(tsvPath); err == nil {
		t.Error("expected error for 3 columns instead of 4")
	}

	// Corrupt file: non-numeric revision
	badRev := "doc_id\tpfad\tcontent_hash\trevision\n" +
		"01JABC\tnote1.md\tsha256:1111\tnotanumber\n"
	if err := os.WriteFile(tsvPath, []byte(badRev), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadIdentities(tsvPath); err == nil {
		t.Error("expected error for non-numeric revision")
	}

	// Read error when path is a directory
	if _, err := ReadIdentities(tmp); err == nil {
		t.Error("expected error reading directory as file")
	}
}

func TestMatchRenames(t *testing.T) {
	prev := map[string]Identity{
		"old.md": {
			DocID:       "01JAAA",
			Relative:    "old.md",
			ContentHash: "sha256:same",
			Revision:    2,
		},
		"stable.md": {
			DocID:       "01JBBB",
			Relative:    "stable.md",
			ContentHash: "sha256:stable",
			Revision:    1,
		},
		"ambig1.md": {
			DocID:       "01JCCC",
			Relative:    "ambig1.md",
			ContentHash: "sha256:duplicate",
			Revision:    1,
		},
		"ambig2.md": {
			DocID:       "01JDDD",
			Relative:    "ambig2.md",
			ContentHash: "sha256:duplicate",
			Revision:    1,
		},
	}

	current := map[string]string{
		"new.md":         "sha256:same",      // Unambiguous rename: old.md -> new.md
		"stable.md":      "sha256:stable",    // Unchanged
		"arrived_dup.md": "sha256:duplicate", // Ambiguous: 2 vanished candidates share this hash
		"fresh.md":       "sha256:fresh",     // Completely new file
	}

	matched := MatchRenames(prev, current)
	if len(matched) != 1 {
		t.Fatalf("expected exactly 1 rename match, got %d (%v)", len(matched), matched)
	}

	renamed, ok := matched["new.md"]
	if !ok {
		t.Fatalf("expected new.md to be matched")
	}
	if renamed.DocID != "01JAAA" {
		t.Errorf("expected DocID 01JAAA preserved, got %s", renamed.DocID)
	}
	if renamed.Relative != "new.md" {
		t.Errorf("expected Relative new.md, got %s", renamed.Relative)
	}
	if renamed.ContentHash != "sha256:same" {
		t.Errorf("expected ContentHash sha256:same, got %s", renamed.ContentHash)
	}
	if renamed.Revision != 2 {
		t.Errorf("expected Revision 2 preserved, got %d", renamed.Revision)
	}

	// Ambiguity on arrival side: two fresh files share the same vanished hash
	prevSingle := map[string]Identity{
		"vanished.md": {
			DocID:       "01JEEE",
			Relative:    "vanished.md",
			ContentHash: "sha256:copy",
			Revision:    1,
		},
	}
	currDouble := map[string]string{
		"copy1.md": "sha256:copy",
		"copy2.md": "sha256:copy",
	}
	if m := MatchRenames(prevSingle, currDouble); len(m) != 0 {
		t.Errorf("expected 0 matches when two new files claim same hash, got %v", m)
	}
}

func TestRenderIdentities(t *testing.T) {
	identities := map[string]Identity{
		"b.md": {DocID: "01JB", Relative: "b.md", ContentHash: "sha256:2", Revision: 1},
		"a.md": {DocID: "01JA", Relative: "a.md", ContentHash: "sha256:1", Revision: 2},
	}

	rendered := RenderIdentities(identities)
	expected := "doc_id\tpfad\tcontent_hash\trevision\n" +
		"01JA\ta.md\tsha256:1\t2\n" +
		"01JB\tb.md\tsha256:2\t1\n"

	if rendered != expected {
		t.Fatalf("RenderIdentities mismatch:\ngot:\n%s\nwant:\n%s", rendered, expected)
	}

	// Round-trip verification
	tmp := t.TempDir()
	p := filepath.Join(tmp, "_identities.tsv")
	if err := os.WriteFile(p, []byte(rendered), 0o644); err != nil {
		t.Fatal(err)
	}
	readBack, err := ReadIdentities(p)
	if err != nil {
		t.Fatalf("ReadIdentities round-trip failed: %v", err)
	}
	if !reflect.DeepEqual(readBack, identities) {
		t.Errorf("round-trip mismatch: got %+v, want %+v", readBack, identities)
	}
}
