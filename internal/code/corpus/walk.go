// Package corpus runs the one directory walk the graph and the search index
// share.
//
// The graph cares about seven extensions and one megabyte; the search index
// cares about every text file it can read. Both need the same answer to "which
// paths does this repository have", and the same answer to "is this path
// ignored" -- two walks that could drift would be two answers to one question,
// the failure mode the sourceset package was already built to avoid for the
// freshness probe alone.
//
// Walk enumerates; it does not decide what belongs to the graph or to the
// search corpus. Filtering is the caller's, on the Entry list it returns.
package corpus

import (
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
)

// pruneDirs are directories a walk does not enter unless the caller says a
// tracked file lives below them: dependency output, build output, and fixture
// input Go's own toolchain does not build -- not necessarily "never a symbol
// the graph should have a node for" in general, since go/parser reads a .go
// file under testdata/ as readily as any other. The comparison is against a
// single path segment.
//
// testdata is the third kind, not the first two: it holds fixtures, not
// dependency or build output, and go/build ignores it for builds by convention
// rather than by content. On THIS repository none of its fixtures are .go files
// (they are .go.txt or recorded case corpora) and its .py files are the worlds
// of recorded cases, not code of this repository, so skipping it costs nothing
// here -- but a repository whose testdata/ held real Go it wanted indexed would
// need this entry removed, the same argument the design spec makes for keeping
// _test.go files in the graph. A probe that still walked testdata/ paid for
// 1,826 directories below the 3 named testdata roots, out of 1,910 directories
// in this repository's tree; see docs/en/benchmarks.md, 2026-09-18.
//
// venv names one virtualenv; site-packages names the installed packages of
// every one, whatever the environment is called (env/Lib/site-packages on
// Windows, <env>/lib/python3.x/site-packages on POSIX). A dot directory such
// as .venv is skipped anyway.
//
// obj is MSBuild's intermediate output (generated sources such as
// *.AssemblyInfo.cs). bin is not listed: the list serves every language, and
// bin/ holds hand-written scripts in many repositories.
//
// coverage and htmlcov hold the HTML and CSS of coverage reports, istanbul's
// coverage/lcov-report and coverage.py's htmlcov.
//
// The list is shared with the graph: sourceset.SkipDir delegates here, so the
// hook path, the graph build and the search index prune the same names.
var pruneDirs = map[string]bool{
	"node_modules":  true,
	"dist":          true,
	"build":         true,
	"_build":        true,
	"out":           true,
	"target":        true,
	"vendor":        true,
	"coverage":      true,
	"htmlcov":       true,
	"__pycache__":   true,
	"venv":          true,
	"site-packages": true,
	"testdata":      true,
	"obj":           true,
}

// SkipSegment reports whether one path segment names a directory the walk
// prunes: a dot directory such as .git or .loomux, or one of pruneDirs.
func SkipSegment(name string) bool {
	return strings.HasPrefix(name, ".") || pruneDirs[name]
}

// PruneDirs are the names of pruneDirs in byte order, a copy of its own.
func PruneDirs() []string { return slices.Sorted(maps.Keys(pruneDirs)) }

// PrunedDir reports whether a directory name is one of pruneDirs, without the
// dot rule. A caller that filters an already enumerated path list checks this
// per segment; a dot segment never reaches it because Walk does not enter one.
func PrunedDir(name string) bool {
	return pruneDirs[name]
}

// Matcher answers the two ignore questions a walk needs. Implementations come
// from the git ignore rules (Load); a nil Matcher means "no ignore evaluation",
// which is what a directory outside a repository gets.
type Matcher interface {
	// Ignored reports whether the path is excluded by an ignore rule.
	Ignored(rel string, isDir bool) bool
	// Tracked reports whether the file is in the index.
	Tracked(rel string) bool
	// HasTrackedUnder reports whether a directory contains a tracked file. A
	// pruned directory with one is walked, because the file is the graph's and
	// the search corpus' no matter what an ignore rule says.
	HasTrackedUnder(dir string) bool
}

// Entry is one file of the walk.
type Entry struct {
	Rel     string
	Abs     string
	Size    int64
	MTime   int64
	Ignored bool
	Tracked bool
	// Link marks a symbolic link. Walk never follows one, so the entry
	// describes the link, not its target.
	Link bool
}

// Listing is the result of a walk.
type Listing struct {
	Entries []Entry
	// Repo reports whether a Matcher was consulted, that is, whether the walk
	// ran inside a git repository.
	Repo bool
}

// walkReaders bounds the directories a walk reads at once. Throughput was
// measured saturated at 16 concurrent readers.
const walkReaders = 16

// readDir reads one directory of the walk; tests replace it to observe which
// reads overlap.
var readDir = os.ReadDir

// Walk enumerates every file below root, sorted by path.
//
// It does not enter dot directories, and does not enter a pruned directory
// whose content is ignored and has no tracked file. A directory that an ignore
// rule excludes but that is not pruned is entered: its files carry Ignored, and
// a caller that counts ignored files needs to see them.
//
// Directories are read concurrently, at most walkReaders at once. The Matcher
// is called behind one lock: *Rules loads .gitignore files lazily and is not
// safe for concurrent use. Its answers do not depend on the order of the
// calls, so the listing is the one a serial walk produces.
//
// A read error of one directory does not abort the walk; the entries read
// before the error are kept, the rest of the directory is treated as absent,
// the way filepath.WalkDir does. A root that is not a directory, a symbolic
// link among them, is listed as the one entry "." and not entered.
//
// A file that vanishes between its directory entry and the stat is dropped.
//
//coverage:exempt the d.Info() arm needs a file to vanish between directory entry and stat, the os.Lstat arm needs root to vanish right after the os.Stat that found it
func Walk(root string, m Matcher) (Listing, error) {
	if _, err := os.Stat(root); err != nil {
		return Listing{}, err
	}
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return Listing{}, err
	}
	var (
		mu      sync.Mutex // guards entries and every Matcher call
		entries []Entry
		wg      sync.WaitGroup
		readers = make(chan struct{}, walkReaders)
	)
	add := func(abs, rel string, d fs.DirEntry) {
		info, err := d.Info()
		if err != nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if d.Name() == ".gitignore" {
			// Load the file even when every neighbour is tracked: the
			// record must name every visited ignore source, or the probe
			// reports it as new on every run.
			if loader, ok := m.(interface{ LoadIgnore(string) }); ok {
				loader.LoadIgnore(rel)
			}
		}
		entry := Entry{
			Rel:   rel,
			Abs:   abs,
			Size:  info.Size(),
			MTime: info.ModTime().UnixNano(),
			Link:  d.Type()&fs.ModeSymlink != 0,
		}
		if m != nil {
			entry.Ignored = m.Ignored(rel, false)
			entry.Tracked = m.Tracked(rel)
		}
		entries = append(entries, entry)
	}
	var visit func(abs, rel string)
	visit = func(abs, rel string) {
		defer wg.Done()
		readers <- struct{}{}
		dirEntries, _ := readDir(abs)
		<-readers
		for _, d := range dirEntries {
			name := d.Name()
			childAbs, childRel := filepath.Join(abs, name), path.Join(rel, name)
			if !d.IsDir() {
				add(childAbs, childRel, d)
				continue
			}
			if strings.HasPrefix(name, ".") {
				continue
			}
			mu.Lock()
			pruned := m != nil && pruneDirs[name] && m.Ignored(childRel, true) && !m.HasTrackedUnder(childRel)
			mu.Unlock()
			if pruned {
				continue
			}
			wg.Add(1)
			go visit(childAbs, childRel)
		}
	}
	if rootInfo.IsDir() {
		wg.Add(1)
		visit(root, "")
		wg.Wait()
	} else {
		add(root, ".", fs.FileInfoToDirEntry(rootInfo))
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Rel < entries[j].Rel })
	return Listing{Entries: entries, Repo: m != nil}, nil
}
