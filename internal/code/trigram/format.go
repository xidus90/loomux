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

	"github.com/xidus90/loomux/internal/code/store"
)

// Format constants. The layout is the one in the design's "Bytes" section:
// a variable header, the file table, the trigram table and the posting buffer,
// all integers little endian.
const (
	magic         = "LTGI"
	formatVersion = 1
	// maxExtractor bounds the extractor identity the header carries: the
	// combined all.Version() string names every extractor and is longer than
	// the first format's 64 bytes allowed.
	maxExtractor = 256
	// indexName is the sidecar under the graph's cache directory.
	indexName = "trigram.idx"
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

// Index is a parsed trigram index.
type Index struct {
	files     []string
	extractor string
	settings  []byte
	digest    [32]byte
	grams     map[uint32]posting
	buffer    []uint32
}

// posting is where one trigram's file ids sit in the buffer.
type posting struct {
	offset uint32
	count  uint32
}

// Files returns the corpus paths in the index's order, a copy the caller owns.
func (ix *Index) Files() []string {
	return append([]string(nil), ix.files...)
}

// Postings returns the file ids whose bytes contain gram, ascending, or nil
// when the index has no such trigram.
func (ix *Index) Postings(gram uint32) []uint32 {
	p, ok := ix.grams[gram]
	if !ok {
		return nil
	}
	out := make([]uint32, p.count)
	copy(out, ix.buffer[p.offset:p.offset+p.count])
	return out
}

// Settings returns the canonical settings the index was built with.
func (ix *Index) Settings() []byte {
	return append([]byte(nil), ix.settings...)
}

// Extractor returns the extractor version the index was built with.
func (ix *Index) Extractor() string { return ix.extractor }

// Digest returns the corpus digest the index was built for.
func (ix *Index) Digest() [32]byte { return ix.digest }

// Build serializes the corpus. The bytes depend on the corpus content, the
// extractor version, the settings and the digest, never on the order the
// caller passes the files in.
func Build(files []File, extractor string, settings []byte, digest [32]byte) ([]byte, error) {
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
// must leave no orphan.
func Write(root string, digest [32]byte, extractor string, settings []byte, files []File) error {
	raw, err := Build(files, extractor, settings, digest)
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
	if err := os.Rename(tmp, final); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// Read loads the index from the graph's cache directory.
//
// A missing file is ErrNoIndex, so "never built" and "built and broken" stay
// distinguishable; a broken file is ErrCorrupt and a newer format ErrStale.
func Read(root string) (*Index, error) {
	raw, err := os.ReadFile(store.CachePath(root, indexName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrNoIndex, store.CachePath(root, indexName))
		}
		return nil, fmt.Errorf("read trigram index: %w", err)
	}
	return decode(raw)
}

// decode parses and validates an index. Every length and offset is checked
// before it is used, so a truncated or hostile file cannot panic.
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

	if uint64(gramCount)*16 > uint64(len(raw)-pos) {
		return nil, fmt.Errorf("%w: trigram table", ErrCorrupt)
	}
	grams := make(map[uint32]posting, gramCount)
	tableEnd := pos + int(gramCount)*16
	var lastGram uint32
	for i := uint32(0); i < gramCount; i++ {
		entry := raw[pos+int(i)*16:]
		gram := binary.LittleEndian.Uint32(entry)
		off := binary.LittleEndian.Uint64(entry[4:])
		count := binary.LittleEndian.Uint32(entry[12:])
		if gram > 0xFFFFFF {
			return nil, fmt.Errorf("%w: trigram %#x out of range", ErrCorrupt, gram)
		}
		if i > 0 && gram <= lastGram {
			return nil, fmt.Errorf("%w: trigram table not sorted at %d", ErrCorrupt, i)
		}
		lastGram = gram
		if off%4 != 0 || off+uint64(count)*4 > uint64(bufferBytes) {
			return nil, fmt.Errorf("%w: postings of %#x out of range", ErrCorrupt, gram)
		}
		grams[gram] = posting{offset: uint32(off / 4), count: count}
	}
	pos = tableEnd

	if uint64(postingsCount)*4 != uint64(bufferBytes) {
		return nil, fmt.Errorf("%w: posting count and buffer size disagree", ErrCorrupt)
	}
	if uint64(len(raw)-pos) != uint64(bufferBytes) {
		return nil, fmt.Errorf("%w: posting buffer has the wrong length", ErrCorrupt)
	}
	buffer := make([]uint32, 0, postingsCount)
	for i := uint64(0); i < postingsCount; i++ {
		buffer = append(buffer, binary.LittleEndian.Uint32(raw[pos+int(i)*4:]))
	}
	for gram, p := range grams {
		for i := uint32(1); i < p.count; i++ {
			prev := buffer[p.offset+i-1]
			if buffer[p.offset+i] <= prev {
				return nil, fmt.Errorf("%w: postings of %#x not ascending", ErrCorrupt, gram)
			}
		}
		if p.count > 0 && buffer[p.offset+p.count-1] >= fileCount {
			return nil, fmt.Errorf("%w: postings of %#x name no file", ErrCorrupt, gram)
		}
	}
	return &Index{
		files:     files,
		extractor: extractor,
		settings:  settings,
		digest:    digest,
		grams:     grams,
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
