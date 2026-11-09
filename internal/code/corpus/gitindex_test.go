package corpus

import (
	"crypto/sha1"
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
)

// indexRepo is a repository whose index holds paths with umlauts, spaces, a
// subdirectory with a sibling that shares its name as a prefix, and from
// version 3 on an intent-to-add entry (an extended flag, which git refuses to
// write in version 2 and answers by writing version 3), in the given version.
func indexRepo(t *testing.T, version int) string {
	t.Helper()
	root := newRepo(t, map[string]string{
		"a.go":           "package a\n",
		"b c.txt":        "x",
		"über/größe.go":  "package g\n",
		"sub/d.go":       "package d\n",
		"sub/deep/e.go":  "package e\n",
		"sub/later.go":   "package l\n",
		"subx/f.go":      "package f\n",
		"zz/ä b/ü.txt":   "y",
		"untracked.txt":  "u",
		"sub/ignored.go": "package i\n",
	}, []string{"a.go", "b c.txt", "über/größe.go", "sub/d.go", "sub/deep/e.go", "subx/f.go", "zz/ä b/ü.txt"}, nil)
	if version > 2 {
		gitRun(t, root, "add", "-N", "sub/later.go")
	}
	gitRun(t, root, "update-index", "--index-version", strconv.Itoa(version))
	return root
}

// indexRecord is the part of a record IndexTracked reads.
func indexRecord(root, prefix string) Record {
	return Record{GitPaths: GitPaths{Root: root, Prefix: prefix, Index: filepath.Join(".git", "index")}}
}

// TestIndexTrackedMatchesGit pins that the index read in-process lists what
// git ls-files lists, in every index version git writes, from the top and
// from a subdirectory.
func TestIndexTrackedMatchesGit(t *testing.T) {
	for _, version := range []int{2, 3, 4} {
		root := indexRepo(t, version)
		for _, prefix := range []string{"", "sub"} {
			dir := filepath.Join(root, filepath.FromSlash(prefix))
			want, err := TrackedSet(dir)
			if err != nil {
				t.Fatalf("TrackedSet: %v", err)
			}
			data, err := os.ReadFile(filepath.Join(root, ".git", "index"))
			if err != nil {
				t.Fatal(err)
			}
			if written := binary.BigEndian.Uint32(data[4:8]); written != uint32(version) {
				t.Fatalf("index written in version %d, want %d", written, version)
			}
			got, ok := parseIndex(data, prefix)
			if !ok || !slices.Equal(got, want) || len(want) < 2 {
				t.Errorf("v%d prefix %q: parseIndex = %q, %v, want %q", version, prefix, got, ok, want)
			}
			if got, err := IndexTracked(indexRecord(root, prefix), dir); err != nil || !slices.Equal(got, want) {
				t.Errorf("v%d prefix %q: IndexTracked = %q, %v, want %q", version, prefix, got, err, want)
			}
		}
	}
}

// TestIndexTrackedAsksGitWhenTheIndexIsGone pins the fallback for an index
// the record names but the disk does not have: git's answer, here none.
func TestIndexTrackedAsksGitWhenTheIndexIsGone(t *testing.T) {
	root := newRepo(t, map[string]string{"a.go": "package a\n"}, nil, nil)
	got, err := IndexTracked(indexRecord(root, ""), root)
	if err != nil || len(got) != 0 {
		t.Fatalf("IndexTracked without an index = %q, %v, want git's empty set", got, err)
	}
	gitRun(t, root, "add", "a.go")
	if got, err := IndexTracked(Record{}, root); err != nil || !slices.Equal(got, []string{"a.go"}) {
		t.Fatalf("IndexTracked of a record without git paths = %q, %v, want git's [a.go]", got, err)
	}
}

// TestIndexTrackedAsksGitForASplitIndex pins the fallback for an index the
// reader does not understand, here a split index: git still lists it.
func TestIndexTrackedAsksGitForASplitIndex(t *testing.T) {
	root := indexRepo(t, 2)
	gitRun(t, root, "update-index", "--split-index")
	data, err := os.ReadFile(filepath.Join(root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := parseIndex(data, ""); ok {
		t.Fatal("parseIndex read a split index")
	}
	want, err := TrackedSet(root)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := IndexTracked(indexRecord(root, ""), root); err != nil || !slices.Equal(got, want) {
		t.Fatalf("IndexTracked of a split index = %q, %v, want %q", got, err, want)
	}
}

// seal appends the trailing checksum to an index body a test edited.
func seal(body []byte) []byte {
	sum := sha1.Sum(body)
	return append(body, sum[:]...)
}

// firstFlags is the offset of the first entry's flags field.
const firstFlags = 12 + 60

// TestParseIndexRefusesWhatItCannotRead pins every case the reader leaves to
// git: each is an index git wrote and the test then changed.
func TestParseIndexRefusesWhatItCannotRead(t *testing.T) {
	read := func(version int) []byte {
		root := newRepo(t, map[string]string{"a.go": "a", "b.go": "b"}, []string{"a.go", "b.go"}, nil)
		gitRun(t, root, "update-index", "--index-version", strconv.Itoa(version))
		data, err := os.ReadFile(filepath.Join(root, ".git", "index"))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	v2, v4 := read(2), read(4)
	// edit changes the body, the index without its checksum, and seals it.
	edit := func(data []byte, f func(b []byte) []byte) []byte {
		return seal(f(slices.Clone(data[:len(data)-sha1.Size])))
	}
	// entriesEnd is where the extensions start in v2: two entries of 62
	// bytes and a four-byte name, each padded to 72.
	entriesEnd := 12 + 72 + 72
	ext := func(sig string, size uint32) func([]byte) []byte {
		return func(b []byte) []byte {
			head := append([]byte(sig), binary.BigEndian.AppendUint32(nil, size)...)
			return slices.Concat(b[:entriesEnd], head, b[entriesEnd:])
		}
	}
	cases := map[string][]byte{
		"too short":      v2[:20],
		"not an index":   edit(v2, func(b []byte) []byte { copy(b, "DIRX"); return b }),
		"bad checksum":   append(slices.Clone(v2[:len(v2)-1]), v2[len(v2)-1]^1),
		"version 1":      edit(v2, func(b []byte) []byte { b[7] = 1; return b }),
		"version 5":      edit(v2, func(b []byte) []byte { b[7] = 5; return b }),
		"unmerged entry": edit(v2, func(b []byte) []byte { b[firstFlags] |= 0x10; return b }),
		"extended in v2": edit(v2, func(b []byte) []byte { b[firstFlags] |= 0x40; return b }),
		"entries cut":    edit(v2, func(b []byte) []byte { b[11] = 200; return b }),
		"name unended": edit(v2, func(b []byte) []byte {
			b[11] = 3
			return slices.Concat(b[:entriesEnd], make([]byte, 62), []byte("abc"))
		}),
		"padding cut":     edit(v2, func(b []byte) []byte { return b[:entriesEnd-1] }),
		"link extension":  edit(v2, ext("link", 0)),
		"sdir extension":  edit(v2, ext("sdir", 0)),
		"extension cut":   edit(v2, ext("TREE", 1000)),
		"trailing bytes":  edit(v2, func(b []byte) []byte { return append(b, 0, 0, 0) }),
		"v4 strips more":  edit(v4, func(b []byte) []byte { b[12+62] = 9; return b }),
		"v4 varint cut":   edit(v4, func(b []byte) []byte { b[11] = 1; return append(b[:12+62], 0x80) }),
		"v4 name unended": edit(v4, func(b []byte) []byte { b[11] = 1; return append(b[:12+62], 0, 'a') }),
	}
	for name, data := range cases {
		if got, ok := parseIndex(data, ""); ok {
			t.Errorf("%s: parseIndex = %q, want a refusal", name, got)
		}
	}
	// An optional extension is skipped, as git skips it.
	if got, ok := parseIndex(edit(v2, ext("ZZZZ", 0)), ""); !ok || !slices.Equal(got, []string{"a.go", "b.go"}) {
		t.Errorf("optional extension: parseIndex = %q, %v", got, ok)
	}
}

// TestIndexVarint pins git's offset varint: each continuation byte adds one
// before the shift, so 0x80 0x00 is 128, not 0.
func TestIndexVarint(t *testing.T) {
	cases := []struct {
		in   []byte
		want uint64
		n    int
	}{
		{[]byte{0x05}, 5, 1},
		{[]byte{0x7f}, 127, 1},
		{[]byte{0x80, 0x00}, 128, 2},
		{[]byte{0x81, 0x7f}, 383, 2},
		{[]byte{0x80}, 0, 0},
		{nil, 0, 0},
		{[]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}, 0, 0},
	}
	for _, c := range cases {
		if got, n := indexVarint(c.in); got != c.want || n != c.n {
			t.Errorf("indexVarint(% x) = %d, %d, want %d, %d", c.in, got, n, c.want, c.n)
		}
	}
}
