package trigram

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/lock"
)

// testFiles is the corpus of the format tests: a Go file, a Python file with
// CRLF line endings and a file in a subdirectory.
func testFiles() []File {
	return []File{
		{Rel: "a.go", Data: []byte("func main() {}\n")},
		{Rel: "b.py", Data: []byte("def main():\r\n    pass\r\n")},
		{Rel: "z/c.txt", Data: []byte("nothing here")},
	}
}

const testExtractor = "loomux-test"

var testSettings = []byte("max_filesize=67108864;binary=nul-v1;ignore=git-v1;ignorecase=false")

func testDigest() [32]byte {
	var d [32]byte
	for i := range d {
		d[i] = byte(i)
	}
	return d
}

// testCounters are the selection counters the tests' header carries.
var testCounters = Counters{Ignored: 3, SkippedSize: 4, SkippedBinary: 5, IgnoreCase: true}

// update rewrites testdata/golden.idx:
//
//	go test ./internal/code/trigram/ -run TestGolden -count=1 -update
var update = flag.Bool("update", false, "rewrite testdata/golden.idx")

// buildTest builds the format tests' index and fails the test if Build does.
func buildTest(t testing.TB, files []File) []byte {
	t.Helper()
	raw, err := Build(files, testExtractor, testSettings, testDigest(), testCounters)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return raw
}

// readRaw puts raw where Read looks, reads it back and closes it when the
// test ends.
func readRaw(t *testing.T, raw []byte) *Index {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, ".loomux", "state", "graph", "cache", "trigram.idx")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	ix, err := Read(root)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	t.Cleanup(func() { ix.Close() })
	return ix
}

// wantFallback pins that a broken posting does not fail the read but turns
// the candidate search into "search everything" for every literal.
func wantFallback(t *testing.T, raw []byte, literal string) {
	t.Helper()
	ix := readRaw(t, raw)
	files, used := ix.Candidates([][]string{{literal}})
	if used {
		t.Fatalf("Candidates(%q) used the broken index", literal)
	}
	if !reflect.DeepEqual(files, ix.Files()) {
		t.Fatalf("Candidates(%q) = %v, want every file %v", literal, files, ix.Files())
	}
}

// gramName is the text of a packed trigram, for a literal that looks it up.
func gramName(gram uint32) string {
	return string([]byte{byte(gram >> 16), byte(gram >> 8), byte(gram)})
}

// headerSize is the byte offset where the file table starts, for the fixed
// extractor and settings of the tests.
func headerSize() int {
	return 4 + 4 + 2 + len(testExtractor) + 4 + len(testSettings) + 32 + 4 + 4 + 8 + 4 + 4 + 4 + 4 + 1
}

// fileTableSize is the byte length of the file table of files, in the order
// Build writes it (sorted by path).
func fileTableSize(files []File) int {
	n := 0
	for _, f := range files {
		n += 2 + len(f.Rel)
	}
	return n
}

// TestRoundTrip pins that an index answers the postings it was built from.
func TestRoundTrip(t *testing.T) {
	ix, err := decode(buildTest(t, testFiles()))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got, want := ix.Files(), []string{"a.go", "b.py", "z/c.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Files() = %v, want %v", got, want)
	}
	ids := map[string]uint32{"a.go": 0, "b.py": 1, "z/c.txt": 2}
	for _, f := range testFiles() {
		for _, gram := range Grams(f.Data) {
			got, ok := ix.Postings(gram)
			if !ok {
				t.Fatalf("Postings(%#x) reported a broken posting", gram)
			}
			found := false
			for _, id := range got {
				if id == ids[f.Rel] {
					found = true
				}
			}
			if !found {
				t.Fatalf("Postings(%#x) = %v misses %s", gram, got, f.Rel)
			}
		}
	}
	if got, ok := ix.Postings(0); got != nil || !ok {
		t.Fatalf("Postings(0) = %v, %v; want nil, true", got, ok)
	}
	if got, ok := ix.Postings(0xFFFFFF); got != nil || !ok {
		t.Fatalf("Postings(0xFFFFFF) = %v, %v; want nil, true", got, ok)
	}
	if !bytes.Equal(ix.Settings(), testSettings) {
		t.Fatalf("Settings() = %q, want %q", ix.Settings(), testSettings)
	}
	if ix.Digest() != testDigest() {
		t.Fatalf("Digest() = %x, want %x", ix.Digest(), testDigest())
	}
	if got := ix.Counters(); got != testCounters {
		t.Fatalf("Counters() = %+v, want %+v", got, testCounters)
	}
	if err := ix.Close(); err != nil {
		t.Fatalf("Close of an unmapped index: %v", err)
	}
}

// TestPostingsOfAnEmptyIndex pins that a corpus without trigrams answers
// "absent", not "broken".
func TestPostingsOfAnEmptyIndex(t *testing.T) {
	ix, err := decode(buildTest(t, nil))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got, ok := ix.Postings(0x616263); got != nil || !ok {
		t.Fatalf("Postings = %v, %v; want nil, true", got, ok)
	}
}

// TestBuildSortsFiles pins that the index numbers files by path, whatever
// order the caller passes.
func TestBuildSortsFiles(t *testing.T) {
	unsorted := []File{testFiles()[2], testFiles()[0], testFiles()[1]}
	ix, err := decode(buildTest(t, unsorted))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got, want := ix.Files(), []string{"a.go", "b.py", "z/c.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Files() = %v, want %v", got, want)
	}
	// b.py is second by path, so the trigram "def" has to name id 1.
	gram, ok := Pack("def")
	if !ok {
		t.Fatal("Pack(def) not packable")
	}
	if got, want := first(ix.Postings(gram)), []uint32{1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Postings(def) = %v, want %v", got, want)
	}
}

// first drops a posting lookup's validity flag.
func first(ids []uint32, _ bool) []uint32 { return ids }

// TestBuildDeterministic pins that the bytes depend on the content only.
func TestBuildDeterministic(t *testing.T) {
	first := buildTest(t, testFiles())
	second := buildTest(t, testFiles())
	if !bytes.Equal(first, second) {
		t.Fatal("two builds of the same corpus differ")
	}
	reversed := []File{testFiles()[2], testFiles()[1], testFiles()[0]}
	if !bytes.Equal(first, buildTest(t, reversed)) {
		t.Fatal("a different input order changes the bytes")
	}
}

// TestBuildRefusesLongExtractor pins the cap maxExtractor promises.
func TestBuildRefusesLongExtractor(t *testing.T) {
	long := strings.Repeat("x", maxExtractor+1)
	if _, err := Build(testFiles(), long, testSettings, testDigest(), testCounters); err == nil {
		t.Fatalf("Build accepted a %d-byte extractor", len(long))
	}
}

// TestParseBadMagic pins that a foreign file is corrupt, not stale.
func TestParseBadMagic(t *testing.T) {
	raw := buildTest(t, testFiles())
	raw[0] = 'X'
	if _, err := decode(raw); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("decode = %v, want ErrCorrupt", err)
	}
}

// TestParseVersionMismatch pins that another format is stale, not corrupt: the
// index can be rebuilt, the file is not broken.
func TestParseVersionMismatch(t *testing.T) {
	raw := buildTest(t, testFiles())
	binary.LittleEndian.PutUint32(raw[4:], 1)
	if _, err := decode(raw); !errors.Is(err, ErrStale) {
		t.Fatalf("decode = %v, want ErrStale", err)
	}
}

// TestParseTruncated pins that every prefix of a valid index is refused and
// none panics.
func TestParseTruncated(t *testing.T) {
	raw := buildTest(t, testFiles())
	for i := 0; i < len(raw); i++ {
		if _, err := decode(raw[:i]); !errors.Is(err, ErrCorrupt) && !errors.Is(err, ErrStale) {
			t.Fatalf("decode(first %d bytes) = %v, want ErrCorrupt", i, err)
		}
	}
}

// gramTable is where the trigram table of the tests' index starts.
func gramTable() int { return headerSize() + fileTableSize(testFiles()) }

// gramCountOf is the trigram count in raw's header.
func gramCountOf(raw []byte) int {
	return int(binary.LittleEndian.Uint32(raw[4+4+2+len(testExtractor)+4+len(testSettings)+32+4:]))
}

// TestParseOffsetOutOfRange pins that a posting table entry pointing outside
// the posting buffer is found when the trigram is looked up.
func TestParseOffsetOutOfRange(t *testing.T) {
	raw := buildTest(t, testFiles())
	base := gramTable()
	binary.LittleEndian.PutUint64(raw[base+4:], 1<<40)
	wantFallback(t, raw, gramName(binary.LittleEndian.Uint32(raw[base:])))
}

// TestParseOffsetNotAligned pins that an offset inside a file id is broken.
func TestParseOffsetNotAligned(t *testing.T) {
	raw := buildTest(t, testFiles())
	base := gramTable()
	binary.LittleEndian.PutUint64(raw[base+4:], 2)
	wantFallback(t, raw, gramName(binary.LittleEndian.Uint32(raw[base:])))
}

// TestParseLengthOutOfRange pins that a posting count running past the
// posting buffer is found when the trigram is looked up.
func TestParseLengthOutOfRange(t *testing.T) {
	raw := buildTest(t, testFiles())
	base := gramTable()
	binary.LittleEndian.PutUint32(raw[base+12:], 1<<30)
	wantFallback(t, raw, gramName(binary.LittleEndian.Uint32(raw[base:])))
}

// TestParseSettingsLengthOutOfRange pins that a settings length beyond the
// file is corrupt.
func TestParseSettingsLengthOutOfRange(t *testing.T) {
	raw := buildTest(t, testFiles())
	binary.LittleEndian.PutUint32(raw[8+2+len(testExtractor):], 1<<30)
	if _, err := decode(raw); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("decode = %v, want ErrCorrupt", err)
	}
}

// TestParsePathTooLong pins that a path length that would run past the file is
// corrupt.
func TestParsePathTooLong(t *testing.T) {
	raw := buildTest(t, testFiles())
	binary.LittleEndian.PutUint16(raw[headerSize():], 0xFFFF)
	if _, err := decode(raw); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("decode = %v, want ErrCorrupt", err)
	}
}

// TestParsePostingsNotAscending pins that the ascending posting lists the
// format promises are checked on read: an index that names a file twice would
// make the candidate set wrong, so the lookup reports it broken.
func TestParsePostingsNotAscending(t *testing.T) {
	raw := buildTest(t, testFiles())
	base := gramTable()
	gramCount := gramCountOf(raw)
	postingsBase := base + gramCount*16
	for i := 0; i < gramCount; i++ {
		entry := base + i*16
		count := int(binary.LittleEndian.Uint32(raw[entry+12:]))
		if count < 2 {
			continue
		}
		off := postingsBase + int(binary.LittleEndian.Uint64(raw[entry+4:]))
		first := binary.LittleEndian.Uint32(raw[off:])
		second := binary.LittleEndian.Uint32(raw[off+4:])
		binary.LittleEndian.PutUint32(raw[off:], second)
		binary.LittleEndian.PutUint32(raw[off+4:], first)
		wantFallback(t, raw, gramName(binary.LittleEndian.Uint32(raw[entry:])))
		return
	}
	t.Fatal("test corpus has no trigram with two postings")
}

// TestWriteReadRoundTrips pins the file on disk: Write, Read, same answers.
func TestWriteReadRoundTrips(t *testing.T) {
	root := t.TempDir()
	if err := Write(root, testDigest(), testExtractor, testSettings, testCounters, testFiles()); err != nil {
		t.Fatalf("Write: %v", err)
	}
	ix, err := Read(root)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	defer ix.Close()
	if got, want := ix.Files(), []string{"a.go", "b.py", "z/c.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Files() = %v, want %v", got, want)
	}
	if got, used := ix.Candidates([][]string{{"main"}}); !used || !reflect.DeepEqual(got, []string{"a.go", "b.py"}) {
		t.Fatalf("Candidates(main) = %v, %v; want a.go and b.py", got, used)
	}
	if _, err := os.Stat(filepath.Join(root, ".loomux", "state", "graph", "cache", "trigram.idx")); err != nil {
		t.Fatalf("index file missing: %v", err)
	}
}

// TestReadWithoutIndex pins the difference between "never built" and "broken".
func TestReadWithoutIndex(t *testing.T) {
	if _, err := Read(t.TempDir()); !errors.Is(err, ErrNoIndex) {
		t.Fatalf("Read(empty) = %v, want ErrNoIndex", err)
	}
}

// TestWriteAtomicLeavesNoTempFile pins that a failed write cleans up.
func TestWriteAtomicLeavesNoTempFile(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Join(root, ".loomux", "state", "graph", "cache")
	if err := os.MkdirAll(filepath.Dir(cache), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(cache, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	if err := Write(root, testDigest(), testExtractor, testSettings, testCounters, testFiles()); err == nil {
		t.Fatal("Write succeeded although the cache path is a file")
	}
	entries, err := os.ReadDir(filepath.Dir(cache))
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("temporary file left behind: %s", e.Name())
		}
	}
}

// TestHeaderCarriesSettings pins that the settings are part of the bytes.
func TestHeaderCarriesSettings(t *testing.T) {
	first := buildTest(t, testFiles())
	other := append([]byte(nil), testSettings...)
	other[0] = 'M'
	raw, err := Build(testFiles(), testExtractor, other, testDigest(), testCounters)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if bytes.Equal(first, raw) {
		t.Fatal("different settings produce identical bytes")
	}
	ix, err := decode(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !bytes.Equal(ix.Settings(), other) {
		t.Fatalf("Settings() = %q, want %q", ix.Settings(), other)
	}
	if got := ix.Counters(); got != testCounters {
		t.Fatalf("Counters() = %+v, want %+v", got, testCounters)
	}
}

// TestGolden pins the format against a file in the repository: if the format
// changes, this fails before a reader in the wild does. Rewrite the file with
// the command at update.
func TestGolden(t *testing.T) {
	got := buildTest(t, testFiles())
	golden := filepath.Join("testdata", "golden.idx")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("index bytes differ from testdata/golden.idx (%d vs %d bytes)", len(got), len(want))
	}
}

// TestHeaderCarriesExtractor pins the extractor version in the header.
func TestHeaderCarriesExtractor(t *testing.T) {
	ix, err := decode(buildTest(t, testFiles()))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got := ix.Extractor(); got != testExtractor {
		t.Fatalf("Extractor() = %q, want %q", got, testExtractor)
	}
}

// TestReadFailsWhenIndexIsADirectory pins that a read error which is not
// "missing" is reported, not turned into ErrNoIndex.
func TestReadFailsWhenIndexIsADirectory(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".loomux", "state", "graph", "cache", "trigram.idx")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := Read(root); err == nil || errors.Is(err, ErrNoIndex) {
		t.Fatalf("Read(directory) = %v, want a read error", err)
	}
}

// TestWriteErrors pins that every failing step of Write reports instead of
// leaving a half-written index.
func TestWriteErrors(t *testing.T) {
	t.Run("build error", func(t *testing.T) {
		long := strings.Repeat("x", maxExtractor+1)
		if err := Write(t.TempDir(), testDigest(), long, testSettings, testCounters, testFiles()); err == nil {
			t.Fatalf("Write accepted a %d-byte extractor", len(long))
		}
	})
	t.Run("mkdir error", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(root, []byte("x"), 0o644); err != nil {
			t.Fatalf("write blocker: %v", err)
		}
		if err := Write(root, testDigest(), testExtractor, testSettings, testCounters, testFiles()); err == nil {
			t.Fatal("Write succeeded below a file")
		}
	})
	t.Run("write error", func(t *testing.T) {
		root := t.TempDir()
		final := filepath.Join(root, ".loomux", "state", "graph", "cache", "trigram.idx")
		if err := os.MkdirAll(filepath.Dir(final), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		tmp := final + "." + strconv.Itoa(os.Getpid()) + ".tmp"
		if err := os.MkdirAll(tmp, 0o755); err != nil {
			t.Fatalf("mkdir tmp: %v", err)
		}
		if err := Write(root, testDigest(), testExtractor, testSettings, testCounters, testFiles()); err == nil {
			t.Fatal("Write succeeded although the temp path is a directory")
		}
	})
	t.Run("rename error", func(t *testing.T) {
		old := lock.SidecarBudget
		lock.SidecarBudget = 50 * time.Millisecond
		t.Cleanup(func() { lock.SidecarBudget = old })
		root := t.TempDir()
		final := filepath.Join(root, ".loomux", "state", "graph", "cache", "trigram.idx")
		if err := os.MkdirAll(final, 0o755); err != nil {
			t.Fatalf("mkdir final: %v", err)
		}
		if err := Write(root, testDigest(), testExtractor, testSettings, testCounters, testFiles()); err == nil {
			t.Fatal("Write succeeded although the final path is a directory")
		}
	})
}

// TestParseFileTableNotSorted pins the sorted file table as a format promise
// the reader enforces.
func TestParseFileTableNotSorted(t *testing.T) {
	raw := buildTest(t, testFiles())
	off := headerSize() + 2 + len("a.go") + 2
	copy(raw[off:], "a.go")
	if _, err := decode(raw); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("decode = %v, want ErrCorrupt", err)
	}
}

// TestParseGramNotSorted pins the ascending trigram table: a trigram listed
// twice is found when it is looked up.
func TestParseGramNotSorted(t *testing.T) {
	raw := buildTest(t, testFiles())
	base := gramTable()
	copy(raw[base+16:], raw[base:base+4])
	wantFallback(t, raw, gramName(binary.LittleEndian.Uint32(raw[base:])))
}

// TestParseGramOutOfRange pins that a trigram with a byte above the format's
// 24 bits is found when the search lands on it.
func TestParseGramOutOfRange(t *testing.T) {
	raw := buildTest(t, testFiles())
	base := gramTable()
	smallest := binary.LittleEndian.Uint32(raw[base:])
	binary.LittleEndian.PutUint32(raw[base:], 0x01000000)
	wantFallback(t, raw, gramName(smallest))
}

// TestParseCountMismatch pins that the header's posting count has to agree
// with its buffer size.
func TestParseCountMismatch(t *testing.T) {
	raw := buildTest(t, testFiles())
	off := 4 + 4 + 2 + len(testExtractor) + 4 + len(testSettings) + 32 + 4 + 4
	binary.LittleEndian.PutUint64(raw[off:], 1<<20)
	if _, err := decode(raw); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("decode = %v, want ErrCorrupt", err)
	}
}

// TestParseBufferWrongLength pins that trailing bytes are refused.
func TestParseBufferWrongLength(t *testing.T) {
	raw := buildTest(t, testFiles())
	raw = append(raw, 0)
	if _, err := decode(raw); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("decode = %v, want ErrCorrupt", err)
	}
}

// TestParsePostingNamesNoFile pins that a file id outside the file table is
// broken when the trigram is looked up, not an empty answer.
func TestParsePostingNamesNoFile(t *testing.T) {
	raw := buildTest(t, testFiles())
	base := gramTable()
	gramCount := gramCountOf(raw)
	postingsBase := base + gramCount*16
	for i := 0; i < gramCount; i++ {
		entry := base + i*16
		if binary.LittleEndian.Uint32(raw[entry+12:]) != 1 {
			continue
		}
		off := postingsBase + int(binary.LittleEndian.Uint64(raw[entry+4:]))
		binary.LittleEndian.PutUint32(raw[off:], 3)
		wantFallback(t, raw, gramName(binary.LittleEndian.Uint32(raw[entry:])))
		return
	}
	t.Fatal("test corpus has no trigram with one posting")
}
