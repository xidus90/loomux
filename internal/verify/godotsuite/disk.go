package godotsuite

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Snapshot is what each project .gd looked like before the engine
// instrumented it on disk: content hash and modification time.
type Snapshot map[string]struct {
	Sum   [32]byte
	MTime time.Time
}

// Take records every .gd walkGD visits. The hash covers the content after
// walkGD stripped a BOM; Settle strips it the same way. Line endings are
// not normalised, the engine restores byte for byte.
func Take(root string) (Snapshot, error) {
	s := Snapshot{}
	err := walkGD(root, func(rel string, data []byte) error {
		e := s[rel]
		e.Sum = sha256.Sum256(data)
		// A file that vanished since the walk keeps the zero time and is
		// reported by Settle if it stays gone.
		if fi, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err == nil {
			e.MTime = fi.ModTime()
		}
		s[rel] = e
		return nil
	})
	return s, err
}

// Settle compares the restored sources with the snapshot (E4.4). A file
// with other content, or gone, is a finding; an identical one gets its old
// mtime back, so the restore leaves no trace in the working tree.
func (s Snapshot) Settle(root string) []string {
	var findings []string
	for rel, was := range s {
		p := filepath.Join(root, filepath.FromSlash(rel))
		data, err := os.ReadFile(p)
		if err != nil || sha256.Sum256(bytes.TrimPrefix(data, []byte(bom))) != was.Sum {
			findings = append(findings, fmt.Sprintf("%s differs after restoring the instrumentation", rel))
			continue
		}
		_ = os.Chtimes(p, was.MTime, was.MTime) // best effort: content is what counts
	}
	sort.Strings(findings)
	return findings
}
