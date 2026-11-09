package corpus

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"os"
	"strings"
)

// IndexTracked is TrackedSet read from the git index the record names, without
// starting git. An index the reader does not understand with certainty is left
// to git: a missing file, a bad checksum (a SHA-256 repository or a
// skipped-hash index among them), an unknown version, an unmerged path or a
// required extension such as a split or sparse index.
func IndexTracked(rec Record, root string) ([]string, error) {
	if abs := GitAbs(rec, rec.GitPaths.Index); abs != "" {
		if data, err := os.ReadFile(abs); err == nil {
			if paths, ok := parseIndex(data, rec.GitPaths.Prefix); ok {
				return paths, nil
			}
		}
	}
	return TrackedSet(root)
}

// parseIndex lists the paths of a git index (DIRC version 2, 3 or 4) below
// prefix, relative to it, as `git ls-files -z --cached` run in prefix lists
// them. The index keeps its entries sorted by name, so the list is sorted.
// ok is false for anything git would read differently or refuse.
//
// Format: https://git-scm.com/docs/index-format
func parseIndex(data []byte, prefix string) (paths []string, ok bool) {
	const header = 12
	if len(data) < header+sha1.Size || string(data[:4]) != "DIRC" {
		return nil, false
	}
	body := data[:len(data)-sha1.Size]
	if sum := sha1.Sum(body); !bytes.Equal(sum[:], data[len(body):]) {
		return nil, false
	}
	version := binary.BigEndian.Uint32(data[4:])
	if version < 2 || version > 4 {
		return nil, false
	}
	if prefix != "" {
		prefix += "/"
	}
	off, name := header, ""
	for range binary.BigEndian.Uint32(data[8:]) {
		// 62 bytes of stat data, object id and flags, then at least a NUL and
		// a pad byte (v2, v3) or a strip count and a NUL (v4).
		if off+64 > len(body) {
			return nil, false
		}
		flags := binary.BigEndian.Uint16(body[off+60:])
		// A stage marks an unmerged path, which git lists once per stage.
		if flags&0x3000 != 0 {
			return nil, false
		}
		start := off + 62
		if flags&0x4000 != 0 {
			if version < 3 {
				return nil, false
			}
			start += 2
		}
		if version == 4 {
			strip, n := indexVarint(body[start:])
			if n == 0 || strip > uint64(len(name)) {
				return nil, false
			}
			start += n
			end := bytes.IndexByte(body[start:], 0)
			if end < 0 {
				return nil, false
			}
			name = name[:len(name)-int(strip)] + string(body[start:start+end])
			off = start + end + 1
		} else {
			end := bytes.IndexByte(body[start:], 0)
			if end < 0 {
				return nil, false
			}
			name = string(body[start : start+end])
			// The entry is NUL-padded to a multiple of eight bytes.
			off += (start - off + end + 8) &^ 7
			if off > len(body) {
				return nil, false
			}
		}
		if rel, below := strings.CutPrefix(name, prefix); below {
			paths = append(paths, rel)
		}
	}
	for off < len(body) {
		if off+8 > len(body) {
			return nil, false
		}
		// An extension whose signature does not start with A-Z is one git
		// must understand to read the index (link, sdir, ...).
		if sig := body[off]; sig < 'A' || sig > 'Z' {
			return nil, false
		}
		size := uint64(binary.BigEndian.Uint32(body[off+4:]))
		if uint64(off)+8+size > uint64(len(body)) {
			return nil, false
		}
		off += 8 + int(size)
	}
	return paths, true
}

// indexVarint decodes the offset varint of a v4 index entry: each
// continuation byte adds one before the shift. n is 0 for a cut or
// overflowing number.
func indexVarint(b []byte) (v uint64, n int) {
	for i, c := range b {
		if i > 0 {
			if v >= 1<<57-1 {
				return 0, 0
			}
			v = (v + 1) << 7
		}
		v |= uint64(c & 0x7f)
		if c&0x80 == 0 {
			return v, i + 1
		}
	}
	return 0, 0
}
