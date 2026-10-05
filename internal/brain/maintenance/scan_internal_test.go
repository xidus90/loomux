package maintenance

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// decode carries bytes that are not UTF-8 through as U+FFFD, the way
// `errors="replace"` does -- and one replacement per maximal subpart, not one
// per byte and not one per run.
func TestDecodeReplacesEachMaximalSubpart(t *testing.T) {
	cases := []struct {
		name string
		raw  []byte
		want string
	}{
		{"two lone bytes are two replacements", []byte{0xFF, 0xFF}, "��"},
		{"a truncated three-byte sequence is one", []byte{0xE2, 0x82}, "�"},
		{"and the byte after it stands", []byte{0xE2, 0x82, 'A'}, "�A"},
		{"a lead whose second byte is out of range breaks at one", []byte{0xE0, 0x80, 0x80}, "���"},
		{"a surrogate lead breaks at one", []byte{0xED, 0xA0, 0x80}, "���"},
		{"an overlong lead is never a prefix", []byte{0xC0, 0xAF}, "��"},
		{"a lead beyond U+10FFFF is never a prefix", []byte{0xF5, 0x80}, "��"},
		{"a truncated two-byte sequence is one", []byte{0xC2}, "�"},
		{"a truncated four-byte sequence is one", []byte{0xF0, 0x9F, 0x92}, "�"},
		{"a four-byte sequence broken at the third byte", []byte{0xF1, 0x80, 'A'}, "�A"},
		{"the ceiling of the four-byte range", []byte{0xF4, 0x90}, "��"},
		{"a lead above the surrogate block", []byte{0xEE, 'A'}, "�A"},
		{"a continuation byte on its own", []byte{0x80}, "�"},
		{"valid text beside broken bytes", []byte("ä\xffb"), "ä�b"},
		// The edges of every lead range and of the second-byte ranges, each a
		// truncated sequence and so one replacement -- put to CPython 3.14.7 on
		// 2026-09-22 like the rows above, and each one answered a single U+FFFD.
		{"the lowest ordinary three-byte lead", []byte{0xE1, 0x80}, "�"},
		{"the highest lead below the surrogates", []byte{0xEC, 0x80}, "�"},
		{"the lowest lead above the surrogates", []byte{0xEE, 0x80}, "�"},
		{"the highest three-byte lead", []byte{0xEF, 0x80}, "�"},
		{"the highest ordinary four-byte lead", []byte{0xF3, 0x80}, "�"},
		{"a second byte at the top of its range", []byte{0xE1, 0xBF}, "�"},
		{"a second byte at the top of the ED range", []byte{0xED, 0x9F}, "�"},
		{"a third byte at the bottom of its range", []byte{0xF1, 0x80, 0x80}, "�"},
		{"a third byte at the top of its range", []byte{0xF1, 0x80, 0xBF}, "�"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := decode(c.raw); got != c.want {
				t.Fatalf("decode(% x) = %q, want %q", c.raw, got, c.want)
			}
		})
	}
}

// A whole well-formed sequence is the one answer decode never asks for -- it
// only calls maximalSubpart where DecodeRune already refused -- so the arm is
// measured here directly rather than left standing unmeasured.
func TestMaximalSubpartMeasuresAWholeSequence(t *testing.T) {
	if got := maximalSubpart([]byte{0xE2, 0x82, 0xAC}); got != 3 {
		t.Fatalf("maximalSubpart = %d, want 3", got)
	}
}

// The fold happens before the decoding, so a CRLF split across the boundary of
// a broken byte is still a CRLF.
func TestDecodeFoldsBeforeItReplaces(t *testing.T) {
	if got := decode([]byte("a\r\n\xffb")); got != "a\n�b" {
		t.Fatalf("decode = %q", got)
	}
}

// The seams: three reads of one source that only a race between the walk and
// the read can break. Each is its own case, because one that covered all three
// would stop at the first.
func TestScanCarriesAFailedStat(t *testing.T) {
	withSeam(t, &statFn, func(string) (os.FileInfo, error) { return nil, errors.New("stat broke") })
	if err := scanErr(t); err == nil || err.Error() != "stat broke" {
		t.Fatalf("err = %v, want the stat's own", err)
	}
}

func TestScanCarriesAFailedHash(t *testing.T) {
	withSeam(t, &contentHashFn, func(string) (string, error) { return "", errors.New("hash broke") })
	if err := scanErr(t); err == nil || err.Error() != "hash broke" {
		t.Fatalf("err = %v, want the hash's own", err)
	}
}

func TestScanCarriesAFailedRead(t *testing.T) {
	withSeam(t, &readFileFn, func(string) ([]byte, error) { return nil, errors.New("read broke") })
	if err := scanErr(t); err == nil || err.Error() != "read broke" {
		t.Fatalf("err = %v, want the read's own", err)
	}
}

// withSeam swaps one seam for the length of the test and puts the real one
// back, so the order the cases run in cannot matter.
func withSeam[T any](t *testing.T, seam *T, replacement T) {
	t.Helper()
	standing := *seam
	*seam = replacement
	t.Cleanup(func() { *seam = standing })
}

// scanErr is a scan over one area whose single source no longer matches its
// register -- the path that reaches all three seams.
func scanErr(t *testing.T) error {
	t.Helper()
	root := t.TempDir()
	area := config.Area{Scope: "project/one", Path: filepath.Join(root, "area")}
	if err := os.MkdirAll(area.Path, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(area.Path, "a.md"), []byte("second\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	register := "doc_id\tpfad\tcontent_hash\trevision\n" +
		"00000000000000000000000000\ta.md\tsha256:0000\t1\n"
	if err := os.WriteFile(filepath.Join(area.Path, identitiesName), []byte(register), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, _, _, err := Scan(area, nil, filepath.Join(root, "state"))
	return err
}

// The stats file is read byte for byte by the Python side, so its bytes are
// pinned against a golden the reference itself wrote.
// testdata/stats.golden.tsv came out of `_write_stats`; the script that
// produced it is in the archive release archive/parity-recordings.
func TestRenderStatsMatchesThePythonWriter(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("testdata", "stats.golden.tsv"))
	if err != nil {
		t.Fatalf("ReadFile golden: %v", err)
	}
	if got := renderStats(goldenStats()); got != string(want) {
		t.Fatalf("renderStats =\n%q\nwant\n%q", got, string(want))
	}
}

// goldenStats are the inputs of testdata/stats.golden.tsv: a path with a space
// in it, an upper and a lower case pair and a non-ASCII name, all of which pin
// the order, plus a zero and a stamp too large for 32 bits.
func goldenStats() map[string]stamp {
	return map[string]stamp{
		"Ärger.md":     {mtimeNs: 1758300000000000000, size: 12},
		"a b.md":       {mtimeNs: 0, size: 0},
		"a.md":         {mtimeNs: 1, size: 7},
		"B.md":         {mtimeNs: 4294967296, size: 4294967296},
		"z/tief/ä.md":  {mtimeNs: 1758300000123456789, size: 1},
		"z/tief/zz.md": {mtimeNs: 2, size: 3},
	}
}

// An empty cache is the header and nothing else, not an empty file: the reader
// drops the first line unconditionally.
func TestRenderStatsWritesTheHeaderAlone(t *testing.T) {
	if got := renderStats(map[string]stamp{}); got != statHeader+"\n" {
		t.Fatalf("renderStats = %q", got)
	}
}

// Every way a cache line can be damaged, one case each: green coverage over a
// chain of `continue`s says nothing about which of them a line died on.
func TestReadStatsDropsDamagedLines(t *testing.T) {
	cases := []struct {
		name string
		line string
	}{
		{"too few fields", "a.md\t1"},
		{"too many fields", "a.md\t1\t2\t3"},
		{"an mtime that is no number", "a.md\tx\t2"},
		{"a size that is no number", "a.md\t1\tx"},
		{"an mtime with a sign", "a.md\t+1\t2"},
		{"a size out of range", "a.md\t1\t99999999999999999999"},
		{"an empty mtime", "a.md\t\t2"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "stats.tsv")
			if err := os.WriteFile(path, []byte(statHeader+"\n"+c.line+"\n"), 0o644); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}
			if got := readStats(path); len(got) != 0 {
				t.Fatalf("readStats = %+v, want the line dropped", got)
			}
		})
	}
}

// And a good line survives beside a damaged one, which is what makes the drop
// a drop rather than a refusal of the whole file.
func TestReadStatsKeepsTheGoodLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.tsv")
	text := statHeader + "\na.md\tx\t2\nb.md\t11\t22\n"
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got := readStats(path)
	if len(got) != 1 || got["b.md"] != (stamp{mtimeNs: 11, size: 22}) {
		t.Fatalf("readStats = %+v", got)
	}
}

// Bytes that are not UTF-8 are not checked at all: they travel into the key,
// which then matches no real path, and the good lines around them still work.
// The reference raises here and takes the whole run down; this pins that the
// Go side does neither that nor the throwing-away the parity record used to
// claim of it.
func TestReadStatsCarriesBrokenBytesIntoTheKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.tsv")
	text := statHeader + "\n\xff\xfe.md\t1\t2\nb.md\t3\t4\n"
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got := readStats(path)
	if len(got) != 2 {
		t.Fatalf("readStats = %+v, want both lines kept", got)
	}
	if got["b.md"] != (stamp{mtimeNs: 3, size: 4}) {
		t.Fatalf("the good line beside the broken one was lost: %+v", got)
	}
	// Verbatim, not through `decode`: the key is compared against a path of
	// the walk's own output, and no real path carries these bytes, so the line
	// is spent on nothing and costs exactly one hash.
	if got["\xff\xfe.md"] != (stamp{mtimeNs: 1, size: 2}) {
		t.Fatalf("the broken key did not arrive as its own bytes: %+v", got)
	}
}

// A file that exists but cannot be opened is an empty cache. Python's
// `read_text` raises the PermissionError on; this side spends one hash per
// source instead, which is what the cache is allowed to cost.
func TestReadStatsIsEmptyWhenTheFileCannotBeOpened(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.tsv")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if got := readStats(path); len(got) != 0 {
		t.Fatalf("readStats = %+v, want empty", got)
	}
}

// The header is dropped by position and not by its text: a cache whose first
// line happens to look like a record loses that record, which costs one hash,
// where reading it would make the file's own header a phantom source.
func TestReadStatsDropsTheFirstLineWhateverItSays(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.tsv")
	if err := os.WriteFile(path, []byte("a.md\t1\t2\nb.md\t3\t4\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got := readStats(path)
	if len(got) != 1 {
		t.Fatalf("readStats = %+v, want only the second line", got)
	}
}
