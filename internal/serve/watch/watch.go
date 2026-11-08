// Package watch keeps the graphs of the registered areas fresh in the
// background: a change under an area's root, once the tree has been quiet for
// a moment, runs grep's drift check and rebuilds only on drift.
package watch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/config"
)

const (
	// quietPeriod is how long an area must see no relevant change before it
	// is checked, so a burst of writes costs one check.
	quietPeriod = 2 * time.Second
	// maxDelay bounds the wait from the first change of a burst: a log or a
	// build output written every second would never leave a quiet period.
	maxDelay = 10 * time.Second
	// resyncInterval is how often the registry is read again.
	resyncInterval = time.Minute
)

// Event is one change a Source saw. Root is the area it belongs to, also for
// the extra git files outside the root; Path is the changed file. Overflow
// means changes were lost and the area must be checked whatever they were.
// Err means the source stopped watching Root; Path is then empty.
type Event struct {
	Root     string
	Path     string
	Overflow bool
	Err      error
}

// Source watches the roots it is given. extra are single files outside root,
// reported with Root set to root. Events stays open until Close.
type Source interface {
	Add(root string, extra []string) error
	Remove(root string)
	Events() <-chan Event
	Close() error
}

// Watcher refreshes the areas Areas names whenever Source reports a relevant
// change. All fields are required.
type Watcher struct {
	Source   Source
	Areas    func() ([]string, error)
	Refresh  func(root string) (bool, error)
	After    func(time.Duration) <-chan time.Time
	Log      io.Writer
	Relevant func(root, path string) bool

	logMu sync.Mutex
}

// area is the worker of one watched root; a kick asks it to check the root
// after the quiet period. Kicks that arrive while it is busy collapse into one.
type area struct {
	kick   chan struct{}
	cancel context.CancelFunc
}

// Run watches until ctx ends, then waits for a running refresh and closes the
// source.
func (w *Watcher) Run(ctx context.Context) {
	var wg sync.WaitGroup
	areas := map[string]*area{}
	// failed are the areas the source could not watch or stopped watching;
	// they stay unwatched.
	failed := map[string]bool{}
	w.sync(ctx, &wg, areas, failed)
	resync := w.After(resyncInterval)
	events := w.Source.Events()
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			if err := w.Source.Close(); err != nil {
				w.logf("watch: source not closed: %v", err)
			}
			return
		case <-resync:
			w.sync(ctx, &wg, areas, failed)
			resync = w.After(resyncInterval)
		case e := <-events:
			a, ok := areas[e.Root]
			switch {
			case !ok:
			case e.Err != nil:
				w.logf("watch %s: no longer watched: %v", e.Root, e.Err)
				w.drop(areas, e.Root)
				failed[e.Root] = true
			case e.Overflow || w.Relevant(e.Root, e.Path):
				kick(a.kick)
			}
		}
	}
}

// sync brings the watched areas in line with the registry. An area the source
// cannot watch, or stopped watching, is logged once and left to grep's own
// check until serve restarts: the usual cause, an exhausted inotify limit,
// does not pass in a minute.
func (w *Watcher) sync(ctx context.Context, wg *sync.WaitGroup, areas map[string]*area, failed map[string]bool) {
	roots, err := w.Areas()
	if err != nil {
		w.logf("watch: registry not read: %v", err)
		return
	}
	want := map[string]bool{}
	for _, root := range roots {
		want[root] = true
	}
	for root := range areas {
		if !want[root] {
			w.drop(areas, root)
		}
	}
	for _, root := range roots {
		if areas[root] != nil || failed[root] {
			continue
		}
		rec, _ := corpus.Read(root)
		if err := w.Source.Add(root, ExtraPaths(rec)); err != nil {
			w.logf("watch %s: not watched: %v", root, err)
			failed[root] = true
			continue
		}
		actx, cancel := context.WithCancel(ctx)
		a := &area{kick: make(chan struct{}, 1), cancel: cancel}
		areas[root] = a
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.work(actx, root, a.kick)
		}()
	}
}

// drop stops watching root and ends its worker.
func (w *Watcher) drop(areas map[string]*area, root string) {
	areas[root].cancel()
	w.Source.Remove(root)
	delete(areas, root)
}

// kick asks without waiting; a kick already pending stands for this one.
func kick(c chan struct{}) {
	select {
	case c <- struct{}{}:
	default:
	}
}

// work checks root after every quiet period that follows a kick. A held lock
// starts the quiet period over; a graph from another extractor is left alone,
// since a service and a CLI of different versions would otherwise rebuild it
// in turn.
func (w *Watcher) work(ctx context.Context, root string, kicks chan struct{}) {
	warned := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-kicks:
		}
		if !w.quiet(ctx, kicks) {
			return
		}
		if ext := extractor(root); ext != "" && ext != all.Version() {
			if !warned {
				w.logf("watch %s: graph built by extractor %q, this service runs %q; not rebuilding", root, ext, all.Version())
				warned = true
			}
			continue
		}
		_, err := w.Refresh(root)
		var locked *ask.LockedError
		switch {
		case errors.As(err, &locked):
			kick(kicks)
		case err != nil:
			w.logf("watch %s: %v", root, err)
		}
	}
}

// quiet waits until no kick arrived for a whole quiet period, or maxDelay
// passed since it began; false means ctx ended first.
func (w *Watcher) quiet(ctx context.Context, kicks chan struct{}) bool {
	deadline := w.After(maxDelay)
	for {
		select {
		case <-ctx.Done():
			return false
		case <-kicks:
		case <-w.After(quietPeriod):
			return true
		case <-deadline:
			return true
		}
	}
}

// extractor names the extractor of the graph's fingerprint, "" when there is
// none to read.
func extractor(root string) string {
	body, _ := os.ReadFile(freshness.Path(root))
	var rec struct {
		Extractor string `json:"extractor"`
	}
	json.Unmarshal(body, &rec)
	return rec.Extractor
}

func (w *Watcher) logf(format string, args ...any) {
	w.logMu.Lock()
	defer w.logMu.Unlock()
	fmt.Fprintf(w.Log, format+"\n", args...)
}

// gitFiles are the record's git files that decide what grep sees, wherever
// they lie.
func gitFiles(rec corpus.Record) []string {
	return []string{
		corpus.GitAbs(rec, rec.GitPaths.Index),
		corpus.GitAbs(rec, rec.GitPaths.Config),
		corpus.GitAbs(rec, rec.GitPaths.Exclude),
	}
}

// isGitFile reports whether path is one of the record's git files.
func isGitFile(rec corpus.Record, path string) bool {
	for _, p := range gitFiles(rec) {
		// Git and the Windows notifier may spell the drive letter in different
		// case; a false match costs one probe.
		if strings.EqualFold(p, path) {
			return true
		}
	}
	return false
}

// Relevant reports whether a change at path can move grep's view of root: the
// record's git files wherever they lie, and every file under root outside a
// directory whose name starts with a dot, as grep's walk skips those. Dot
// files count.
func Relevant(rec corpus.Record, root, path string) bool {
	path = filepath.Clean(path)
	if isGitFile(rec, path) {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	// A path outside root starts with "..", which this check refuses too.
	dirs := strings.Split(filepath.ToSlash(rel), "/")
	for _, d := range dirs[:len(dirs)-1] {
		if strings.HasPrefix(d, ".") {
			return false
		}
	}
	return true
}

// ExtraPaths are the record's git files outside the area's root, as in a
// linked worktree or a root below the repository's top, which a Source has to
// watch on their own.
func ExtraPaths(rec corpus.Record) []string {
	root := filepath.Join(rec.GitPaths.Root, filepath.FromSlash(rec.GitPaths.Prefix))
	var out []string
	for _, p := range gitFiles(rec) {
		rel, err := filepath.Rel(root, p)
		if p != "" && (err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			out = append(out, p)
		}
	}
	return out
}

// Areas lists the registered areas that have a graph, read from the registry
// in registryDir on every call. No registry means no areas.
func Areas(registryDir string) func() ([]string, error) {
	return func() ([]string, error) {
		areas, err := config.ReadRegistry(registryDir)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		var roots []string
		for _, a := range areas {
			root := filepath.Clean(filepath.FromSlash(a.Path))
			if _, err := os.Stat(store.WiringPath(root)); err == nil {
				roots = append(roots, root)
			}
		}
		return roots, nil
	}
}

// readRecord is a seam: the tests count the record reads of Judge.
var readRecord = corpus.Read

// Judge is a Watcher's Relevant: Relevant over each area's record, less the
// paths the record's ignore rules leave out of the corpus, such as a log or a
// build output written every second.
//
// Run calls it for every event on one goroutine, so it keeps the record and
// its rules per area and reads them again only when the record file moved.
//
// ponytail: keyed on size and mtime of corpus.json, which a coarse-mtime file
// system can miss; a missed rewrite costs a few stale verdicts until the next.
func Judge() func(root, path string) bool {
	type judged struct {
		stamp corpus.Stamp
		rec   corpus.Record
		rules *corpus.Rules
	}
	areas := map[string]judged{}
	return func(root, path string) bool {
		stamp := corpus.FileStamp(corpus.RecordPath(root))
		a, ok := areas[root]
		if !ok || a.stamp != stamp {
			rec, _ := readRecord(root)
			a = judged{stamp: stamp, rec: rec, rules: corpus.RulesFromRecord(rec)}
			areas[root] = a
		}
		if !Relevant(a.rec, root, path) {
			return false
		}
		path = filepath.Clean(path)
		// The git files and the ignore files decide what is ignored; a
		// whitelist pattern such as `*` matches them too.
		if a.rules == nil || isGitFile(a.rec, path) || filepath.Base(path) == ".gitignore" {
			return true
		}
		rel, _ := filepath.Rel(root, path)
		return !a.rules.Ignored(filepath.ToSlash(rel), false)
	}
}
