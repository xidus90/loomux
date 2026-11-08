// Package query is the graph's question layer: build, check and ask, with the
// answers as text a terminal and a tool call both carry.
//
// It exists so the command line and serve reach the same code. Before it, the
// build sequence lived in internal/cli, where serve cannot import it; a second
// copy there would be a second place to forget the sidecar.
package query

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
	"github.com/xidus90/loomux/internal/code/sourceset"
	"github.com/xidus90/loomux/internal/code/spanidx"
	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/lock"
)

// noticeRecord announces a record that could not be written. A lost mirror
// leaves the record in place and costs only the fast path, so it says so.
func noticeRecord(notice func(string), what string, err error) {
	if errors.Is(err, corpus.ErrMirror) {
		notice(fmt.Sprintf("%s mirror not written: %v", what, err))
		return
	}
	notice(fmt.Sprintf("%s not written: %v", what, err))
}

// Stats is what a build learned on the way, for the report and for the
// freshness record.
type Stats struct {
	Files       []sourceset.SourceFile
	Hashes      map[string]string
	NoSymbol    int
	PerLanguage map[string]LangStats // key: extract.Language.Name()

	// entries is the cache entry of every extracted file, parsed or reused,
	// for the build to write back. A file that is gone has none, so it leaves
	// the cache with the first build that no longer finds it.
	entries map[string]cacheEntry

	// bodies are the bytes this build read, keyed by repo-relative path, so
	// the search corpus hashes the graph's files without a second read.
	bodies map[string][]byte

	// Corpus is what the search corpus selection came to: how many files it
	// holds and why it left the others out.
	Corpus CorpusStats
}

// LangStats is what one language's files came to in a build.
type LangStats struct {
	Files, Parsed, Reused, ParseErrors int
	ErrorFiles                         []string // sorted, the files with ParseErrors > 0
}

// ExtractOptions says what Extract may take over from the last build.
type ExtractOptions struct {
	Reuse  bool         // read cache/extract.json and skip files whose hash and language version match
	Notice func(string) // cache problems; nil means silent
}

// BuildOptions are the choices of one build.
type BuildOptions struct {
	// NoReuse parses every file, whatever the extract cache holds. It is the
	// way out when an extractor changed without its Version: the cache's
	// entries still match then, and a plain build keeps them.
	NoReuse bool

	// MaxFilesize is the resolved search corpus limit; nil means the default
	// and 0 means unlimited. The caller resolves it from flag and
	// configuration so that the index, the record and the next probe agree.
	MaxFilesize *int64
}

// Build is BuildWith and the defaults: the extraction of every unchanged file
// is reused.
func Build(root string, notice func(string)) (*model.Graph, Stats, error) {
	return BuildWith(root, BuildOptions{}, notice)
}

// BuildWith is one whole build: the graph, the ask sidecar, the freshness
// record and the extract cache.
//
// Every caller comes through here -- `graph build`, and the rebuild a query
// triggers, on the command line and over MCP. Two copies of this sequence
// would be two places where the sidecar can be forgotten, and a forgotten
// sidecar is a silently worse answer.
//
// The two writes a question reads are errors; the record and the cache are
// not. Once the graph and the sidecar are on disk, a record that could not be
// written costs the next probe its fast path, and a cache the next build its
// reuse, and nothing more -- so each is announced through notice and the
// build stands. Turning either into a failure would let a build that produced
// everything a question needs report that it produced nothing.
func BuildWith(root string, opts BuildOptions, notice func(string)) (*model.Graph, Stats, error) {
	if notice == nil {
		notice = func(string) {}
	}
	g, stats, err := Extract(root, ExtractOptions{Reuse: !opts.NoReuse, Notice: notice})
	if err != nil {
		return nil, Stats{}, err
	}
	// The spans belong to this graph but carry the corpus digest, so the old
	// ones go first and the new ones come last: a build that stops in between
	// leaves none, and grep falls back to the graph. The digest does not cover
	// the graph, so spans that cannot be removed fail the build.
	if err := lock.RemoveEventually(store.CachePath(root, spanidx.Name), lock.SidecarBudget); err != nil {
		return nil, Stats{}, fmt.Errorf("remove stale symbol spans: %w", err)
	}
	if err := store.Write(root, g); err != nil {
		return nil, Stats{}, err
	}
	// From the graph in memory, which still carries the body text: the written
	// file has it stripped, and that is what makes the sidecar necessary rather
	// than redundant.
	if err := lexicon.Write(root, lexicon.Build(g)); err != nil {
		return nil, Stats{}, err
	}
	if err := freshness.Write(root, all.Version(), stats.Files, stats.Hashes); err != nil {
		noticeRecord(notice, "freshness record", err)
	}
	if err := writeCache(root, stats.entries); err != nil {
		notice(fmt.Sprintf("extract cache not written: %v", err))
	}
	maxFilesize, err := corpus.ResolveMaxFilesize(opts.MaxFilesize, nil)
	if err != nil {
		return nil, Stats{}, err
	}
	stats.Corpus = writeCorpus(root, maxFilesize, stats.bodies, notice)
	if stats.Corpus.Indexed {
		if err := spanidx.Write(root, stats.Corpus.Digest, g); err != nil {
			notice(fmt.Sprintf("symbol spans not written: %v", err))
		}
	}
	return g, stats, nil
}

// Extract reads, parses and resolves the whole tree.
//
// It reads and hashes every file, every time -- never the probe's stat fast
// path. A stat may decide whether a query rebuilds; it may not decide what the
// rebuild looks at, or `check` could report drift that the `build` it
// recommends refuses to repair. The extract cache spares a file its parse and
// never its read: an entry is taken only when the hash of the bytes just read
// and the language's Version both match it.
func Extract(root string, opts ExtractOptions) (*model.Graph, Stats, error) {
	files, err := sourceset.Stat(root)
	if err != nil {
		return nil, Stats{}, err
	}
	stats := Stats{
		Files: files, Hashes: map[string]string{},
		PerLanguage: map[string]LangStats{}, entries: map[string]cacheEntry{},
		bodies: map[string][]byte{},
	}

	var cached map[string]cacheEntry
	if opts.Reuse {
		cached, err = readCache(root)
		if err != nil && opts.Notice != nil {
			opts.Notice(fmt.Sprintf("extract cache ignored, parsing every file: %v", err))
		}
	}

	var results []extract.Result
	for _, f := range files {
		b, err := readFile(f.Abs)
		if err != nil {
			return nil, Stats{}, err
		}
		stats.bodies[f.Rel] = b
		sum := sha256.Sum256(b)
		hash := hex.EncodeToString(sum[:])
		stats.Hashes[f.Rel] = hash

		// Hashed before the skip, so the freshness record covers every file
		// the walk listed, as the probe expects.
		lang, ok := languageFor(f.Rel)
		if !ok {
			continue
		}
		var r extract.Result
		e, hit := cached[f.Rel]
		reuse := hit && e.SHA256 == hash && e.Extractor == lang.Version()
		if reuse {
			r = e.Result
		} else {
			r, err = lang.File(f.Rel, string(b))
			if err != nil {
				return nil, Stats{}, err
			}
		}
		tally(stats.PerLanguage, f.Rel, r, !reuse)
		if len(r.Nodes) == 1 {
			stats.NoSymbol++
		}
		stats.entries[f.Rel] = cacheEntry{Extractor: lang.Version(), SHA256: hash, Result: r}
		// Whether parsed or reused, the result takes its place in the walk's
		// order, so the resolver sees the same sequence either way.
		results = append(results, r)
	}

	mods, err := resolve.Modules(root, goModPaths(root))
	if err != nil {
		return nil, Stats{}, err
	}
	return resolve.Graph(results, mods, all.Version()), stats, nil
}

// languageFor is a seam: the lookup of the extractor for one file.
//
// sourceset lists only extensions some language claims, and extract/all's
// test keeps the two lists equal, so no real walk reaches a file without one.
// A test swaps this to show that such a file is skipped rather than handed to
// a nil extractor, for the day the lists part.
var languageFor = all.For

// goModPaths are the repo-relative go.mod files, found by walking for them --
// sourceset lists source files and a go.mod is not one. This walk shares
// sourceset.SkipDir with sourceset.Stat rather than keeping its own,
// narrower list: the two used to disagree about testdata, which let a
// fixture go.mod there reach module resolution after its .go files had
// already been excluded from the file set.
func goModPaths(root string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && sourceset.SkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "go.mod" {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err == nil {
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// tally counts one file under the language of each of its parts: a page
// with an inline script is a file of both languages, parsed or reused as a
// whole, and a parse error in its script is the script language's, listed
// under the page's path.
func tally(per map[string]LangStats, rel string, r extract.Result, parsed bool) {
	for _, p := range r.Flatten() {
		ls := per[p.Language]
		ls.Files++
		if parsed {
			ls.Parsed++
		} else {
			ls.Reused++
		}
		if p.ParseErrors > 0 {
			ls.ParseErrors += p.ParseErrors
			// Sorted without a sort: the walk hands the files over in order.
			ls.ErrorFiles = append(ls.ErrorFiles, rel)
		}
		per[p.Language] = ls
	}
}
