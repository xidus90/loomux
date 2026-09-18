// Package freshness is the cheap "has the working tree moved?" probe.
//
// Every retrieval runs it, so the unchanged path has to be almost free: one
// stat per source file, no reads and no parsing. Only a file whose size or
// mtime disagrees with the record gets read and hashed, which is what keeps a
// touch -- or a checkout that restores identical bytes -- from costing a
// rebuild.
//
// It does NOT import the extractor. That is deliberate and it is what lets the
// hook path use this package under the dependency gate: the stamp arrives as a
// string. The same separation is why the file set lives in
// internal/code/sourceset -- the probe and the build must enumerate identical
// files, and two enumerations that can drift are two answers.
//
// `graph check` is the other mechanism and does something else entirely: it
// re-extracts and diffs node ids and body hashes. A stat may decide whether a
// query bothers rebuilding; it may not decide what the rebuild looks at.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/graph/fingerprint.ts.
package freshness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/xidus90/loomux/internal/code/sourceset"
	"github.com/xidus90/loomux/internal/code/store"
)

// recordVersion is the shape of the sidecar. A record of another version is a
// record this code cannot read, which is the same as no record.
const recordVersion = 1

// print is one file's fingerprint: size, modification time in nanoseconds, and
// the hash of its bytes.
//
// mtime is an int64 and stays one all the way through JSON. Graft's field is a
// JavaScript number and so a double; here that would be a silent loss of
// precision on a field whose equality decides whether a rebuild happens.
type print struct {
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime"`
	Hash  string `json:"hash"`
}

// record is the sidecar.
type record struct {
	Version   int              `json:"version"`
	Extractor string           `json:"extractor"`
	Files     map[string]print `json:"files"`
}

// Drift is what moved since the last build. Three empty slices mean nothing to
// do.
type Drift struct {
	Changed []string
	Added   []string
	Removed []string
}

// Clean reports whether nothing moved.
func (d Drift) Clean() bool { return d.Count() == 0 }

// Count is how many files moved, in any category.
func (d Drift) Count() int { return len(d.Changed) + len(d.Added) + len(d.Removed) }

// Path is where the record lives.
func Path(root string) string { return store.CachePath(root, "fingerprint.json") }

// Write records the tree as the build just saw it.
//
// hashes is keyed by the same repo-relative path sourceset uses. A file the
// build could not read is recorded with an empty hash, which puts it back on
// the slow path every time -- the only way to learn it is readable again is to
// try.
//
//coverage:exempt MarshalIndent's error arm needs a value json cannot encode (a channel, a func, a cyclic pointer, or a NaN/Inf float); record is built entirely of strings, ints and a map[string]print, which are the same -- no record this function can build makes it fail
func Write(root, extractor string, files []sourceset.SourceFile, hashes map[string]string) error {
	rec := record{Version: recordVersion, Extractor: extractor, Files: map[string]print{}}
	for _, f := range files {
		rec.Files[f.Rel] = print{Size: f.Size, MTime: f.MTime, Hash: hashes[f.Rel]}
	}
	body, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(Path(root)), 0o755); err != nil {
		return err
	}
	return os.WriteFile(Path(root), body, 0o644)
}

// Probe diffs the working tree against the last build's record.
//
// A nil Drift with a nil error means UNKNOWN: never built, built by a version
// that wrote no record, a record that cannot be read, or a record from another
// extractor. A caller treats that as "rebuild", never as "clean".
func Probe(root, extractor string) (*Drift, error) {
	files, err := sourceset.Stat(root)
	if err != nil {
		return nil, err
	}
	rec, ok := read(root, extractor)
	if !ok {
		return nil, nil
	}

	d := &Drift{}
	seen := make(map[string]bool, len(files))
	for _, f := range files {
		seen[f.Rel] = true
		p, known := rec.Files[f.Rel]
		if !known {
			d.Added = append(d.Added, f.Rel)
			continue
		}
		if trusted(p, f) {
			continue
		}
		// Suspect. Confirm by bytes, so a touch does not cost a rebuild.
		sum, err := hashFile(f.Abs)
		if err != nil {
			// Unreadable right now: leave it to the next probe rather than
			// reporting drift a rebuild could not repair either.
			continue
		}
		if sum != p.Hash {
			d.Changed = append(d.Changed, f.Rel)
		}
	}
	for rel := range rec.Files {
		if !seen[rel] {
			d.Removed = append(d.Removed, rel)
		}
	}
	sort.Strings(d.Changed)
	sort.Strings(d.Added)
	sort.Strings(d.Removed)
	return d, nil
}

// read loads the record, or reports that there is none this code may use.
//
// A sidecar is a cache: a broken one costs the next probe its fast path and
// nothing more, so it is never an error a caller dies on.
func read(root, extractor string) (record, bool) {
	b, err := os.ReadFile(Path(root))
	if err != nil {
		return record{}, false
	}
	var rec record
	if err := json.Unmarshal(b, &rec); err != nil {
		return record{}, false
	}
	if rec.Version != recordVersion || rec.Files == nil || rec.Extractor != extractor {
		return record{}, false
	}
	return rec, true
}

// trusted reports whether a recorded print may stand for the file on disk
// without reading it.
//
// This is the PROBE's rule only. A build reads and hashes every file, every
// time: a stat may decide whether a query rebuilds, but not what the rebuild
// looks at -- otherwise `check`, which always re-extracts, could report drift
// that the `build` it recommends then refuses to repair.
//
// An empty hash means the last build never got the bytes, so it is never
// trusted.
func trusted(p print, f sourceset.SourceFile) bool {
	return p.Hash != "" && p.Size == f.Size && p.MTime == f.MTime
}

// hashFile is sha256 over a file's bytes, as full hex.
func hashFile(abs string) (string, error) {
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
