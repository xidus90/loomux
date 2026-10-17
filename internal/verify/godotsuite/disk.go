package godotsuite

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/verify/linecov"
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

// manifest.json is the addon's own record of a disk instrumentation: every
// rewritten source with its backup and the hash of the original. loomux reads
// it instead of asking the engine to put the sources back, because the engine
// has to be started by a process the caller may terminate at any moment: a
// lane killed on its timeout or on the hook's budget dies with its whole
// process tree, and a terminated process runs no deferred restore. What the
// addon left on disk is the only thing a later run can still work with, and
// reading it needs no binary.
const (
	manifestName  = "manifest.json"
	backupSetting = "general/backup_dir"
	defaultBackup = "res://.nano_coverage"
)

// manifestEntry is one rewritten source as the addon records it.
type manifestEntry struct {
	Path   string `json:"path"`
	Backup string `json:"backup_path"`
	Hash   string `json:"original_hash"`
}

// resolve turns a res:// path into one relative to the project root; the
// addon writes no other kind, and anything else names itself in a finding. A
// manifest is a file like any other, so a path in it that leaves the project
// must not become a path loomux writes to. This check is textual only: every
// access goes through an os.Root, which also refuses a link out of the
// project.
func resolve(p string) (string, bool) {
	rel, ok := strings.CutPrefix(p, "res://")
	if !ok {
		return "", false
	}
	r := filepath.Clean(filepath.FromSlash(rel))
	if !filepath.IsLocal(r) {
		return "", false
	}
	return r, true
}

// backupDir is where the addon keeps the manifest and the backups of a disk
// instrumentation; a project may move it.
func backupDir(root *os.Root) (string, error) {
	cfg, err := root.ReadFile("project.godot")
	if err != nil {
		return "", err
	}
	nano, _, err := linecov.GodotCfg(cfg, "nano_coverage")
	if err != nil {
		return "", err
	}
	dir := defaultBackup
	if v := nano[backupSetting]; len(v) == 1 {
		dir = v[0]
	}
	p, ok := resolve(dir)
	if !ok {
		return "", errors.New(backupSetting + " does not name a path inside the project: " + dir)
	}
	return p, nil
}

// hasMarker says whether data starts with the addon's instrumentation marker.
// A CR before the newline is the writer's, not the file's.
func hasMarker(data []byte, marker string) bool {
	first, _, _ := bytes.Cut(bytes.TrimPrefix(data, []byte(bom)), []byte("\n"))
	return string(bytes.TrimSuffix(first, []byte("\r"))) == marker
}

// restoreOne writes one backup back over its source and says whether that
// source is settled afterwards. A source without the marker is one somebody
// has put back already and stays as it is; only a source that is there and
// carries the marker is written to.
func restoreOne(root *os.Root, marker string, e manifestEntry) (string, bool) {
	src, okSrc := resolve(e.Path)
	bkp, okBkp := resolve(e.Backup)
	if !okSrc || !okBkp {
		return "the manifest names a path outside the project: " + cmp.Or(e.Path, e.Backup), false
	}
	data, err := root.ReadFile(src)
	if err != nil {
		return e.Path + " cannot be written back: " + err.Error(), false
	}
	if !hasMarker(data, marker) {
		return "", true
	}
	want, err := root.ReadFile(bkp)
	if err != nil {
		return e.Path + ": its backup is gone, " + e.Backup, false
	}
	sum := sha256.Sum256(want)
	if err := root.WriteFile(src, want, 0o644); err != nil {
		return e.Path + " cannot be written back: " + err.Error(), false
	}
	// Written first, reported second: a backup is the only copy of the
	// original there is, and refusing it over its hash would leave the source
	// instrumented.
	if got := "sha256:" + hex.EncodeToString(sum[:]); got != e.Hash {
		return e.Path + ": its backup does not match the hash the manifest records", true
	}
	return "", true
}

// RestoreManifest puts every source the addon's manifest names back from its
// backup, in this process and without an engine, and then drops the manifest
// and the backups. Without a manifest there is nothing to put back and no
// finding: whoever cares whether the sources are instrumented has asked
// already. A source that stays instrumented keeps the manifest, so a later
// run can try again.
func RestoreManifest(project string) ([]string, error) {
	root, err := os.OpenRoot(project)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	dir, err := backupDir(root)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, manifestName)
	data, err := root.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var m struct{ Files []manifestEntry }
	if err := json.Unmarshal(data, &m); err != nil {
		return []string{manifestName + " of the instrumentation does not read: " + err.Error()}, nil
	}
	marker, err := instrumentedMarker()
	if err != nil {
		return nil, err
	}
	var findings []string
	settled := true
	for _, e := range m.Files {
		f, done := restoreOne(root, marker, e)
		if f != "" {
			findings = append(findings, f)
		}
		settled = settled && done
	}
	sort.Strings(findings)
	if !settled {
		return findings, nil
	}
	// Best effort: a manifest that survives makes the addon refuse the next
	// instrumentation, which is a finding of its own and not a silent loss.
	root.RemoveAll(filepath.Join(dir, "backup"))
	root.Remove(path)
	root.Remove(dir)
	return findings, nil
}
