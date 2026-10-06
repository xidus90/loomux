package identity

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// writeRegister writes one register file into a fresh directory and returns its path.
func writeRegister(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "_identities.tsv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadIdentitiesSplitsLinesLikePython(t *testing.T) {
	// Python: read_text folds \r to \n, and splitlines also breaks at \x1c.
	path := writeRegister(t, "h\nA\ta.md\tsha256:1\t1\rB\tb.md\tsha256:2\t2\x1cC\tc.md\tsha256:3\t3")
	got, err := ReadIdentities(path)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]Identity{
		"a.md": {DocID: "A", Relative: "a.md", ContentHash: "sha256:1", Revision: 1},
		"b.md": {DocID: "B", Relative: "b.md", ContentHash: "sha256:2", Revision: 2},
		"c.md": {DocID: "C", Relative: "c.md", ContentHash: "sha256:3", Revision: 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestReadIdentitiesCountsLinesLikePython(t *testing.T) {
	path := writeRegister(t, "h\nA\ta.md\tsha256:1\t1\x1cbroken")
	_, err := ReadIdentities(path)
	want := path + ": line 3: expected 4 tab-separated fields, found 1"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
}

func TestReadIdentitiesSkipsLinesPythonStripsEmpty(t *testing.T) {
	// "\x1f\u3000".strip() is "" in Python; strings.TrimSpace keeps the \x1f.
	path := writeRegister(t, "h\n\x1f\u3000\nA\ta.md\tsha256:1\t1\n")
	got, err := ReadIdentities(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["a.md"].DocID != "A" {
		t.Fatalf("got %+v", got)
	}
}

func TestReadIdentitiesOfAnEmptyOrHeaderOnlyFileIsEmpty(t *testing.T) {
	for _, content := range []string{"", "doc_id\tpfad\tcontent_hash\trevision\n"} {
		got, err := ReadIdentities(writeRegister(t, content))
		if err != nil || len(got) != 0 {
			t.Fatalf("%q: got %+v, %v", content, got, err)
		}
	}
}

func TestReadIdentitiesNamesTheFieldCount(t *testing.T) {
	for _, tc := range []struct {
		row   string
		found string
	}{
		{"A\ta.md\tsha256:1", "3"},
		{"A\ta.md\tsha256:1\t1\tx", "5"},
	} {
		path := writeRegister(t, "h\n"+tc.row+"\n")
		_, err := ReadIdentities(path)
		want := path + ": line 2: expected 4 tab-separated fields, found " + tc.found
		if err == nil || err.Error() != want {
			t.Errorf("got %v, want %q", err, want)
		}
	}
}

func TestReadIdentitiesQuotesTheRevisionLikeRepr(t *testing.T) {
	for _, tc := range []struct {
		revision string
		quoted   string
	}{
		{"notanumber", "'notanumber'"},
		{"+5", "'+5'"},
		{"-1", "'-1'"},
		{"", "''"},
		{"it's", `"it's"`},
		{"\u0663", "'\u0663'"},
		{"\u00b2", "'\u00b2'"},
		{"99999999999999999999", "'99999999999999999999'"},
	} {
		path := writeRegister(t, "h\nA\ta.md\tsha256:1\t"+tc.revision+"\n")
		_, err := ReadIdentities(path)
		want := path + ": line 2: revision " + tc.quoted + " is not a number"
		if err == nil || err.Error() != want {
			t.Errorf("%q: got %v, want %q", tc.revision, err, want)
		}
	}
}

func TestReadIdentitiesKeepsTheLastRowOfAPath(t *testing.T) {
	path := writeRegister(t, "h\nA\ta.md\tsha256:1\t1\nB\ta.md\tsha256:2\t2\n")
	got, err := ReadIdentities(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["a.md"].DocID != "B" || got["a.md"].Revision != 2 {
		t.Fatalf("got %+v", got)
	}
}

func TestReadIdentitiesRefusesARegisterThatCannotBeInspected(t *testing.T) {
	// A NUL byte makes the stat fail with something other than "not found":
	// an unreadable register is not an empty one, and a caller that takes it
	// for empty reports an area without a source or overwrites the rows.
	path := filepath.Join(t.TempDir(), "a\x00b")
	got, err := ReadIdentities(path)
	if err == nil || got != nil {
		t.Fatalf("got %+v, %v", got, err)
	}
	if errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), "invalid argument") {
		t.Fatalf("an invalid path is not an absent one: %v", err)
	}
}

func TestReadIdentitiesTakesAMissingRegisterForEmpty(t *testing.T) {
	got, err := ReadIdentities(filepath.Join(t.TempDir(), "_identities.tsv"))
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestReadIdentitiesRefusesInvalidUTF8(t *testing.T) {
	path := writeRegister(t, "h\nA\ta.md\tsha256:\xff\t1\n")
	_, err := ReadIdentities(path)
	want := path + ": not valid UTF-8"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
}

func TestReadIdentitiesTakesAPathThroughARegularFileForEmpty(t *testing.T) {
	// Linux answers ENOTDIR for it, Windows a plain "not found": either way
	// nothing can be there.
	file := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadIdentities(filepath.Join(file, "_identities.tsv"))
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("got %+v, %v", got, err)
	}
}
