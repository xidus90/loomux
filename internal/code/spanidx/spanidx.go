// Package spanidx keeps the symbol spans of every file in one mapped sidecar,
// so a search finds the enclosing symbol of a hit without loading the graph.
//
// Layout, little endian: magic "LGSP", u32 version, 32-byte corpus digest,
// u32 n, n × u32 entry offsets sorted by path; each entry is u16 path length,
// path, u32 block length, block. A block is the file's spans as JSON, in
// model.FileSpans order. Blocks are decoded only when a file is asked for.
package spanidx

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/mmap"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/lock"
)

// Name is the sidecar under the graph's cache directory.
const Name = "grepspans.idx"

const (
	magic   = "LGSP"
	version = 1
	// headerSize is magic, version, digest and n.
	headerSize = 4 + 4 + 32 + 4
)

// Errors a reader distinguishes: no sidecar was written, it belongs to another
// corpus or format, or it is broken.
var (
	ErrNoSpans  = errors.New("no symbol spans")
	ErrMismatch = errors.New("symbol spans do not match the corpus")
	ErrCorrupt  = errors.New("symbol spans are corrupt")
)

// span is one entry of a block. The node goes without its body text, which
// model.Node never serializes.
type span struct {
	Node     model.Node `json:"node"`
	From     int        `json:"from"`
	To       int        `json:"to"`
	InDegree int        `json:"in_degree"`
}

// Write stores the spans of g bound to digest, atomically.
func Write(root string, digest [32]byte, g *model.Graph) error {
	raw, err := encode(digest, g)
	if err != nil {
		return err
	}
	final := store.CachePath(root, Name)
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

func encode(digest [32]byte, g *model.Graph) ([]byte, error) {
	x := blast.New(g)
	files := model.FileSpans(g)
	paths := make([]string, 0, len(files))
	for p := range files {
		if len(p) > math.MaxUint16 {
			return nil, fmt.Errorf("symbol spans: path longer than %d bytes: %.40s...", math.MaxUint16, p)
		}
		paths = append(paths, p)
	}
	sort.Strings(paths)

	out := make([]byte, 0, headerSize+4*len(paths))
	out = append(out, magic...)
	out = binary.LittleEndian.AppendUint32(out, version)
	out = append(out, digest[:]...)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(paths)))
	out = out[:headerSize+4*len(paths)]
	// ponytail: u32 offsets cap the file at 4 GiB, far beyond any graph's spans.
	for i, p := range paths {
		binary.LittleEndian.PutUint32(out[headerSize+4*i:], uint32(len(out)))
		list := make([]span, len(files[p]))
		for j, s := range files[p] {
			list[j] = span{Node: *s.Node, From: s.From, To: s.To, InDegree: x.InDegree(s.Node.ID)}
		}
		// Strings, ints and bools only: Marshal cannot fail.
		block, _ := json.Marshal(list)
		out = binary.LittleEndian.AppendUint16(out, uint16(len(p)))
		out = append(out, p...)
		out = binary.LittleEndian.AppendUint32(out, uint32(len(block)))
		out = append(out, block...)
	}
	return out, nil
}

// Reader looks up the spans of one file at a time. It is not safe for
// concurrent use.
type Reader struct {
	mapped *mmap.File
	raw    []byte
	n      int
	spans  map[string][]model.SymbolSpan
	degree map[model.NodeID]int
	broken bool
}

// Open maps the sidecar and checks every entry's bounds, so that no lookup can
// read past the end. The caller closes it.
func Open(root string, digest [32]byte) (*Reader, error) {
	path := store.CachePath(root, Name)
	mapped, err := mmap.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrNoSpans, path)
		}
		return nil, fmt.Errorf("read symbol spans: %w", err)
	}
	r := &Reader{mapped: mapped, raw: mapped.Bytes(),
		spans: map[string][]model.SymbolSpan{}, degree: map[model.NodeID]int{}}
	if err := r.check(digest); err != nil {
		mapped.Close()
		return nil, err
	}
	return r, nil
}

func (r *Reader) check(digest [32]byte) error {
	raw := r.raw
	if len(raw) < 8 || string(raw[:4]) != magic {
		return fmt.Errorf("%w: no header", ErrCorrupt)
	}
	if v := binary.LittleEndian.Uint32(raw[4:]); v != version {
		return fmt.Errorf("%w: version %d, want %d", ErrMismatch, v, version)
	}
	if len(raw) < headerSize {
		return fmt.Errorf("%w: header cut short", ErrCorrupt)
	}
	if [32]byte(raw[8:40]) != digest {
		return fmt.Errorf("%w: another corpus digest", ErrMismatch)
	}
	r.n = int(binary.LittleEndian.Uint32(raw[40:]))
	if headerSize+4*r.n > len(raw) {
		return fmt.Errorf("%w: offset table cut short", ErrCorrupt)
	}
	for i := range r.n {
		if _, _, ok := r.entry(i); !ok {
			return fmt.Errorf("%w: entry %d out of range", ErrCorrupt, i)
		}
	}
	return nil
}

// entry reads entry i of the offset table; ok is false when any part of it
// lies outside the file.
func (r *Reader) entry(i int) (path string, block []byte, ok bool) {
	raw := r.raw
	at := int(binary.LittleEndian.Uint32(raw[headerSize+4*i:]))
	if at+2 > len(raw) {
		return "", nil, false
	}
	pathEnd := at + 2 + int(binary.LittleEndian.Uint16(raw[at:]))
	if pathEnd+4 > len(raw) {
		return "", nil, false
	}
	end := pathEnd + 4 + int(binary.LittleEndian.Uint32(raw[pathEnd:]))
	if end > len(raw) {
		return "", nil, false
	}
	return string(raw[at+2 : pathEnd]), raw[pathEnd+4 : end], true
}

// Spans returns the spans of path in model.FileSpans order, nil for a file
// without symbols. A block that does not decode yields nil and marks the
// reader Broken.
func (r *Reader) Spans(path string) []model.SymbolSpan {
	if list, ok := r.spans[path]; ok {
		return list
	}
	// Open checked every entry's bounds, so ok holds here.
	i := sort.Search(r.n, func(i int) bool { p, _, _ := r.entry(i); return p >= path })
	var list []model.SymbolSpan
	if i < r.n {
		if p, block, _ := r.entry(i); p == path {
			list = r.decode(block)
		}
	}
	r.spans[path] = list
	return list
}

func (r *Reader) decode(block []byte) []model.SymbolSpan {
	var entries []span
	if err := json.Unmarshal(block, &entries); err != nil {
		r.broken = true
		return nil
	}
	list := make([]model.SymbolSpan, len(entries))
	for j := range entries {
		e := &entries[j]
		list[j] = model.SymbolSpan{Node: &e.Node, From: e.From, To: e.To}
		r.degree[e.Node.ID] = e.InDegree
	}
	return list
}

// InDegree is the in-degree of a symbol of a file Spans has returned; any
// other id counts 0.
func (r *Reader) InDegree(id model.NodeID) int { return r.degree[id] }

// Broken reports whether a block failed to decode.
func (r *Reader) Broken() bool { return r.broken }

// Close releases the mapping.
func (r *Reader) Close() error { return r.mapped.Close() }
