package benchsearch

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/index"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// CorpusScope is the one area a corpus run registers.
const CorpusScope = "loomux-bench-corpus"

// indexPrefix names every qmd index a corpus run makes, and nothing else:
// the sweep removes what carries it and trusts that nobody else uses it.
const indexPrefix = "loomux-bench-"

// declaration is the corpus area's own config: every note is indexed.
const declaration = "[area]\nscope = \"" + CorpusScope + "\"\n\n[index]\ninclude = [\"**/*.md\"]\n"

// Prepared is a corpus registered for one run.
type Prepared struct {
	Stand    string // absolute
	StateDir string // throwaway registry dir
	Index    string // qmd index name
	Scope    string // CorpusScope
}

// PrepareCorpus takes the lock, writes the throwaway state and the named
// qmd config. The returned cleanup is always safe to call, also after an
// error, and reports what it could not remove through warn.
//
// The state lives beside the lock rather than in TEMP: Go runs no defer on
// Ctrl+C, and a later SweepStale then removes it together with the index.
// random must answer a plain file name part: it names files in lockDir and
// in qmd's directories.
func PrepareCorpus(stand, lockDir string, random func() string, warn func(string)) (*Prepared, func(), error) {
	name := indexPrefix + random()
	abs, absErr := filepath.Abs(stand)
	if err := errors.Join(absErr, os.MkdirAll(lockDir, 0o755)); err != nil {
		return nil, func() {}, err
	}
	held, err := lock.Acquire(filepath.Join(lockDir, name+".lock"))
	if err != nil {
		return nil, func() {}, err
	}
	var once sync.Once
	cleanup := func() { once.Do(func() { removeRun(lockDir, name, held, warn) }) }
	p := &Prepared{Stand: abs, StateDir: filepath.Join(lockDir, name), Index: name, Scope: CorpusScope}
	if err := p.write(); err != nil {
		return nil, cleanup, err
	}
	return p, cleanup, nil
}

// write lays down the registry, the area's declaration and register, and
// the named qmd config with the corpus as its one collection.
func (p *Prepared) write() error {
	notes := filepath.Join(p.Stand, "notes")
	area := config.ManifestDir(config.Area{Scope: CorpusScope, ReadOnly: true}, p.StateDir)
	register, err := freshRegister(p.Stand)
	if err != nil {
		return err
	}
	files := []struct{ path, body string }{
		{filepath.Join(p.StateDir, "registry.toml"),
			fmt.Sprintf("[[area]]\nscope = %q\npath = %q\nreadonly = true\n", CorpusScope, filepath.ToSlash(notes))},
		{filepath.Join(area, ".loomux", "config.toml"), declaration},
		{filepath.Join(area, "_identities.tsv"), identity.RenderIdentities(register)},
		{index.QmdConfigPathFor(p.Index), modelChoice(index.Models(index.QmdConfigPath()))},
	}
	for _, f := range files {
		if err := writeCreating(f.path, f.body); err != nil {
			return err
		}
	}
	owned := filepath.Join(p.StateDir, "qmd-owned.txt")
	_, err = index.SyncCollections(index.QmdConfigPathFor(p.Index),
		map[string]index.CollectionSpec{search.CollectionName(CorpusScope): {Path: notes, Pattern: "**/*.md", Ignore: index.AlwaysExcludes}},
		index.OwnershipRecord{Read: owned, Write: owned})
	return err
}

// freshRegister gives every note a new identity at revision 1, as if the
// corpus had just been added.
func freshRegister(stand string) (map[string]identity.Identity, error) {
	register := map[string]identity.Identity{}
	for _, name := range noteNames(stand) {
		hash, err := identity.ContentHash(filepath.Join(stand, "notes", name))
		if err != nil {
			return nil, err
		}
		register[name] = identity.Identity{DocID: identity.NewDocID(), Relative: name, ContentHash: hash, Revision: 1}
	}
	return register, nil
}

// modelChoice is the user's models block for the named config. A model
// Models reports unknown is left out, so qmd takes its own default.
func modelChoice(models map[string]string) string {
	chosen := map[string]string{}
	for _, key := range [][2]string{{"embedding", "embed"}, {"query_expansion", "generate"}, {"rerank", "rerank"}} {
		if model := models[key[0]]; model != "unknown" {
			chosen[key[1]] = model
		}
	}
	// A map of strings always marshals.
	data, _ := yaml.Marshal(map[string]any{"models": chosen})
	return string(data)
}

// writeCreating writes body to path, creating the directories above it.
func writeCreating(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

// SweepStale removes a loomux-bench-* index nobody holds a lock on.
//
// Names come from lockDir alone: a run's lock file is the first thing it
// makes and the last thing it removes (see removeRun), so it finds a crash
// wherever it stopped, after the state, the config or the database. A name
// in qmd's directories without a lock file here belongs to a run of another
// state directory, whose lock this sweep cannot ask. qmd's own files in the
// shared cache, .qmd-embed.lock among them, never carry the prefix and are
// never touched.
func SweepStale(lockDir string, warn func(string)) {
	locks, _ := filepath.Glob(filepath.Join(lockDir, indexPrefix+"*.lock"))
	for _, lockFile := range locks {
		name := strings.TrimSuffix(filepath.Base(lockFile), ".lock")
		held, free, err := lock.TryAcquire(lockFile)
		if err != nil {
			warn(fmt.Sprintf("left %s alone: %v", name, err))
			continue
		}
		if free {
			removeLeftovers(lockDir, name, warn)
			report(warn, "release "+lockFile, held.Release())
		}
	}
}

// removeRun is a run's own cleanup: its leftovers, its lock, and -- only
// once everything else is gone -- its lock file. The lock file is the one
// thing that lets a later sweep find what could not be removed.
//
// Only the run itself ever removes its lock file. lock.Acquire creates the
// file before it locks it; on POSIX a sweep that unlinked the file in that
// gap would leave the starting run locking an orphaned inode, and the next
// sweep would find its live index free.
func removeRun(lockDir, name string, held *lock.Handle, warn func(string)) {
	allGone := removeLeftovers(lockDir, name, warn)
	lockFile := filepath.Join(lockDir, name+".lock")
	report(warn, "release "+lockFile, held.Release())
	if allGone {
		report(warn, "clean up", os.Remove(lockFile))
	}
}

// removeLeftovers removes the state and the qmd index of this name, and
// reports whether all of it is gone.
func removeLeftovers(lockDir, name string, warn func(string)) bool {
	qmdConfig := index.QmdConfigPathFor(name)
	database := filepath.Join(index.QmdCacheDir(), name+".sqlite")
	allGone := report(warn, "clean up", os.RemoveAll(filepath.Join(lockDir, name)))
	for _, path := range []string{qmdConfig, qmdConfig + ".brain-backup", database, database + "-wal", database + "-shm"} {
		allGone = report(warn, "clean up", os.Remove(path)) && allGone
	}
	return allGone
}

// report warns about a step of the cleanup that failed, and answers whether
// it succeeded; what is already gone is not a failure. The errors of os name
// their path.
func report(warn func(string), step string, err error) bool {
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		warn(fmt.Sprintf("could not %s: %v", step, err))
		return false
	}
	return true
}
