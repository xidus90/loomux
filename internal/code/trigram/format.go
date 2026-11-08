package trigram

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/xidus90/loomux/internal/code/mmap"
	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/lock"
)

// Format constants. The layout is the one in the design's "Bytes" section:
// a variable header, the file table, the trigram table and the posting buffer,
// all integers little endian.
const (
	magic = "LTGI"
	// formatVersion 2 added the selection counters to the header.
	formatVersion = 2
	// maxExtractor bounds the extractor identity the header carries: the
	// combined all.Version() string names every extractor and is longer than
	// the first format's 64 bytes allowed.
	maxExtractor = 256
	// indexName is the sidecar under the graph's cache directory.
	indexName = "trigram.idx"
	// entrySize is one trigram table entry: u32 gram, u64 offset, u32 count.
	entrySize = 16
)

// Errors a reader distinguishes: no index was ever built, the index is from
// another format version and has to be rebuilt, or the file is broken.
var (
	ErrNoIndex = errors.New("no trigram index")
	ErrStale   = errors.New("trigram index is stale")
	ErrCorrupt = errors.New("trigram index is corrupt")
)

// File is one text file of the search corpus: its repository-relative path
// with forward slashes, and its bytes.
type File struct {
	Rel  string
	Data []byte
}

// Counters are what the corpus selection left out and the case setting it ran
// with, carried in the header so a reader can report them without the corpus
// record.
type Counters struct {
	Ignored, SkippedSize, SkippedBinary int
	IgnoreCase                          bool
}

// Index is a parsed trigram index. The header and the file table are decoded
// when it is read; the trigram table and the postings stay in raw and are
// looked up and checked one trigram at a time.
type Index struct {
	files     []string
	extractor string
	settings  []byte
	digest    [32]byte
	counters  Counters

	raw       []byte
	table     int
	gramCount int
	buffer    int
	mapped    *mmap.File
}

// Files returns the corpus paths in the index's order, a copy the caller owns.
func (ix *Index) Files() []string {
	return append([]string(nil), ix.files...)
}

// Postings returns the file ids whose bytes contain gram, ascending. A trigram
// the index does not carry is nil, true; ok == false means the table or the
// posting list around gram is broken and the index cannot answer for it.
// Not valid after Close.
func (ix *Index) Postings(gram uint32) (ids []uint32, ok bool) {
	lo, hi := 0, ix.gramCount
	for lo < hi {
		mid := (lo + hi) / 2
		if ix.gramAt(mid) < gram {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo == ix.gramCount {
		return nil, true
	}
	// The search only holds on a sorted table of 24-bit trigrams: the entry it
	// lands on and the next one are checked, which catches a corrupt or
	// duplicated trigram exactly where it would change the answer.
	found := ix.gramAt(lo)
	if found > 0xFFFFFF || (lo+1 < ix.gramCount && ix.gramAt(lo+1) <= found) {
		return nil, false
	}
	if found != gram {
		return nil, true
	}
	entry := ix.raw[ix.table+lo*entrySize:]
	off := binary.LittleEndian.Uint64(entry[4:])
	count := uint64(binary.LittleEndian.Uint32(entry[12:]))
	size := uint64(len(ix.raw) - ix.buffer)
	if off%4 != 0 || off > size || count*4 > size-off {
		return nil, false
	}
	ids = make([]uint32, count)
	at := ix.buffer + int(off)
	for i := range ids {
		ids[i] = binary.LittleEndian.Uint32(ix.raw[at+i*4:])
		if (i > 0 && ids[i] <= ids[i-1]) || ids[i] >= uint32(len(ix.files)) {
			return nil, false
		}
	}
	return ids, true
}

// gramAt is the trigram of table entry i.
func (ix *Index) gramAt(i int) uint32 {
	return binary.LittleEndian.Uint32(ix.raw[ix.table+i*entrySize:])
}

// Close releases the mapping Read made. An index from decode has none.
func (ix *Index) Close() error {
	ix.raw = nil
	if ix.mapped == nil {
		return nil
	}
	return ix.mapped.Close()
}

// Settings returns the canonical settings the index was built with.
func (ix *Index) Settings() []byte {
	return append([]byte(nil), ix.settings...)
}

// Extractor returns the extractor version the index was built with.
func (ix *Index) Extractor() string { return ix.extractor }

// Digest returns the corpus digest the index was built for.
func (ix *Index) Digest() [32]byte { return ix.digest }

// Counters returns the selection counters the index was built with.
func (ix *Index) Counters() Counters { return ix.counters }

// Build serializes the corpus. The bytes depend on the corpus content, the
// extractor version, the settings, the digest and the counters, never on the
// order the caller passes the files in.
func Build(files []File, extractor string, settings []byte, digest [32]byte, c Counters) ([]byte, error) {
	if len(extractor) > maxExtractor {
		return nil, fmt.Errorf("extractor %q: longer than %d bytes", extractor, maxExtractor)
	}
	sorted := make([]File, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Rel < sorted[j].Rel })

	// File ids follow the sorted table, and every trigram's ids are appended in
	// that order, so each posting list is ascending without a later sort.
	byGram := make(map[uint32][]uint32, len(sorted)*64)
	for id := range sorted {
		for _, gram := range Grams(sorted[id].Data) {
			byGram[gram] = append(byGram[gram], uint32(id))
		}
	}
	grams := make([]uint32, 0, len(byGram))
	for gram := range byGram {
		grams = append(grams, gram)
	}
	sort.Slice(grams, func(i, j int) bool { return grams[i] < grams[j] })

	var whole bytes.Buffer
	whole.WriteString(magic)
	writeU32(&whole, formatVersion)
	writeU16(&whole, uint16(len(extractor)))
	whole.WriteString(extractor)
	writeU32(&whole, uint32(len(settings)))
	whole.Write(settings)
	whole.Write(digest[:])
	writeU32(&whole, uint32(len(sorted)))
	writeU32(&whole, uint32(len(grams)))
	var total uint64
	for _, gram := range grams {
		total += uint64(len(byGram[gram]))
	}
	writeU64(&whole, total)
	writeU32(&whole, uint32(total*4))
	writeU32(&whole, uint32(c.Ignored))
	writeU32(&whole, uint32(c.SkippedSize))
	writeU32(&whole, uint32(c.SkippedBinary))
	if c.IgnoreCase {
		whole.WriteByte(1)
	} else {
		whole.WriteByte(0)
	}

	for _, f := range sorted {
		writeU16(&whole, uint16(len(f.Rel)))
		whole.WriteString(f.Rel)
	}
	var offset uint32
	for _, gram := range grams {
		writeU32(&whole, gram)
		writeU64(&whole, uint64(offset))
		writeU32(&whole, uint32(len(byGram[gram])))
		offset += uint32(len(byGram[gram])) * 4
	}
	for _, gram := range grams {
		for _, id := range byGram[gram] {
			writeU32(&whole, id)
		}
	}
	return whole.Bytes(), nil
}

// Write builds the index and writes it atomically to the graph's cache
// directory. Temp plus rename with the pid in the temp name, as store.Write
// does: a fixed temp name would let two runs share scratch, and a failed rename
// must leave no orphan. The rename waits for a reader that still has the old
// index mapped, which Windows would otherwise refuse.
func Write(root string, digest [32]byte, extractor string, settings []byte, c Counters, files []File) error {
	raw, err := Build(files, extractor, settings, digest, c)
	if err != nil {
		return err
	}
	final := store.CachePath(root, indexName)
	if err := os.MkdirAll(filepath.Dir(final), 0o755); err != nil {
		return err
	}
	tmp := final + "." + strconv.Itoa(os.Getpid()) + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := lock.ReplaceEventually(tmp, final, lock.SidecarBudget); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// Read maps the index from the graph's cache directory. The caller closes it.
//
// A missing file is ErrNoIndex, so "never built" and "built and broken" stay
// distinguishable; a broken file is ErrCorrupt and another format ErrStale.
func Read(root string) (*Index, error) {
	mapped, err := mmap.Open(store.CachePath(root, indexName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrNoIndex, store.CachePath(root, indexName))
		}
		return nil, fmt.Errorf("read trigram index: %w", err)
	}
	ix, err := decode(mapped.Bytes())
	if err != nil {
		mapped.Close()
		return nil, err
	}
	ix.mapped = mapped
	return ix, nil
}

// decode parses the header and the file table and checks that the trigram
// table and the posting buffer have the lengths the header gives. Every length
// is checked before it is used, so a truncated or hostile file cannot panic;
// the entries themselves are checked when Postings looks one up.
func decode(raw []byte) (*Index, error) {
	if len(raw) < 4 || string(raw[:4]) != magic {
		return nil, fmt.Errorf("%w: bad magic", ErrCorrupt)
	}
	if len(raw) < 8 {
		return nil, fmt.Errorf("%w: truncated header", ErrCorrupt)
	}
	if v := binary.LittleEndian.Uint32(raw[4:]); v != formatVersion {
		return nil, fmt.Errorf("%w: format version %d", ErrStale, v)
	}
	pos := 8
	extLen, pos, ok := takeU16(raw, pos)
	if !ok {
		return nil, fmt.Errorf("%w: extractor length", ErrCorrupt)
	}
	extractor, pos, ok := takeN(raw, pos, extLen, maxExtractor)
	if !ok {
		return nil, fmt.Errorf("%w: extractor", ErrCorrupt)
	}
	settingsLen, next, ok := takeU32(raw, pos)
	if !ok {
		return nil, fmt.Errorf("%w: settings length", ErrCorrupt)
	}
	pos = next
	if settingsLen > uint32(len(raw)) {
		return nil, fmt.Errorf("%w: settings", ErrCorrupt)
	}
	settingsRaw, pos, ok := takeN(raw, pos, int(settingsLen), len(raw))
	if !ok {
		return nil, fmt.Errorf("%w: settings", ErrCorrupt)
	}
	settings := []byte(settingsRaw)
	if len(raw) < pos+32 {
		return nil, fmt.Errorf("%w: digest", ErrCorrupt)
	}
	var digest [32]byte
	copy(digest[:], raw[pos:pos+32])
	pos += 32
	fileCount, pos, ok := takeU32(raw, pos)
	if !ok {
		return nil, fmt.Errorf("%w: file count", ErrCorrupt)
	}
	gramCount, pos, ok := takeU32(raw, pos)
	if !ok {
		return nil, fmt.Errorf("%w: trigram count", ErrCorrupt)
	}
	postingsCount, pos, ok := takeU64(raw, pos)
	if !ok {
		return nil, fmt.Errorf("%w: posting count", ErrCorrupt)
	}
	bufferBytes, pos, ok := takeU32(raw, pos)
	if !ok {
		return nil, fmt.Errorf("%w: buffer size", ErrCorrupt)
	}
	var counts [3]uint32
	for i := range counts {
		if counts[i], pos, ok = takeU32(raw, pos); !ok {
			return nil, fmt.Errorf("%w: counters", ErrCorrupt)
		}
	}
	if pos >= len(raw) {
		return nil, fmt.Errorf("%w: ignorecase flag", ErrCorrupt)
	}
	counters := Counters{
		Ignored: int(counts[0]), SkippedSize: int(counts[1]), SkippedBinary: int(counts[2]),
		IgnoreCase: raw[pos] != 0,
	}
	pos++

	files := make([]string, 0, fileCount)
	for i := uint32(0); i < fileCount; i++ {
		pathLen, next, ok := takeU16(raw, pos)
		if !ok {
			return nil, fmt.Errorf("%w: file path %d", ErrCorrupt, i)
		}
		p, next, ok := takeN(raw, next, pathLen, len(raw))
		pos = next
		if !ok || p == "" {
			return nil, fmt.Errorf("%w: file path %d", ErrCorrupt, i)
		}
		if i > 0 && files[i-1] >= p {
			return nil, fmt.Errorf("%w: file table not sorted at %d", ErrCorrupt, i)
		}
		files = append(files, p)
	}

	if uint64(gramCount)*entrySize > uint64(len(raw)-pos) {
		return nil, fmt.Errorf("%w: trigram table", ErrCorrupt)
	}
	buffer := pos + int(gramCount)*entrySize
	if postingsCount*4 != uint64(bufferBytes) {
		return nil, fmt.Errorf("%w: posting count and buffer size disagree", ErrCorrupt)
	}
	if uint64(len(raw)-buffer) != uint64(bufferBytes) {
		return nil, fmt.Errorf("%w: posting buffer has the wrong length", ErrCorrupt)
	}
	return &Index{
		files:     files,
		extractor: extractor,
		settings:  settings,
		digest:    digest,
		counters:  counters,
		raw:       raw,
		table:     pos,
		gramCount: int(gramCount),
		buffer:    buffer,
	}, nil
}

func writeU16(buf *bytes.Buffer, v uint16) {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], v)
	buf.Write(b[:])
}

func writeU32(buf *bytes.Buffer, v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	buf.Write(b[:])
}

func writeU64(buf *bytes.Buffer, v uint64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], v)
	buf.Write(b[:])
}

// takeU32 reads one little-endian u32 and returns the position after it.
func takeU32(raw []byte, pos int) (uint32, int, bool) {
	if pos < 0 || pos+4 > len(raw) {
		return 0, pos, false
	}
	return binary.LittleEndian.Uint32(raw[pos:]), pos + 4, true
}

// takeU64 reads one little-endian u64 and returns the position after it.
func takeU64(raw []byte, pos int) (uint64, int, bool) {
	if pos < 0 || pos+8 > len(raw) {
		return 0, pos, false
	}
	return binary.LittleEndian.Uint64(raw[pos:]), pos + 8, true
}

// takeU16 reads one little-endian u16 and returns the position after it.
func takeU16(raw []byte, pos int) (int, int, bool) {
	if pos < 0 || pos+2 > len(raw) {
		return 0, pos, false
	}
	return int(binary.LittleEndian.Uint16(raw[pos:])), pos + 2, true
}

// takeN reads n bytes at pos, bounded by limit, and returns the position
// after them.
func takeN(raw []byte, pos, n, limit int) (string, int, bool) {
	if pos < 0 || n < 0 || n > limit || n > len(raw)-pos {
		return "", pos, false
	}
	return string(raw[pos : pos+n]), pos + n, true
}
