package watch

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/query"
	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/gitenv"
)

// fakeSource is a Source over a channel the test feeds.
type fakeSource struct {
	events   chan Event
	addErr   map[string]error
	attempts map[string]int

	mu      sync.Mutex
	added   []string
	removed []string
	closed  chan struct{}
	// closeErr is what Close returns.
	closeErr error
}

func newFakeSource() *fakeSource {
	return &fakeSource{events: make(chan Event), addErr: map[string]error{}, attempts: map[string]int{}, closed: make(chan struct{})}
}

func (s *fakeSource) Add(root string, extra []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempts[root]++
	if err := s.addErr[root]; err != nil {
		return err
	}
	s.added = append(s.added, root)
	return nil
}

func (s *fakeSource) Remove(root string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removed = append(s.removed, root)
}

func (s *fakeSource) Events() <-chan Event { return s.events }

func (s *fakeSource) Close() error {
	close(s.closed)
	return s.closeErr
}

// rig runs a Watcher over fakes: quiet ticks and registry ticks are the
// test's to send, every armed quiet period is reported on armed, every
// refresh on started, and a refresh answers what results holds.
type rig struct {
	t       *testing.T
	w       *Watcher
	src     *fakeSource
	quiet   chan time.Time
	minute  chan time.Time
	armed   chan struct{}
	resync  chan struct{}
	started chan string
	results chan error
	log     bytes.Buffer
	cancel  context.CancelFunc
	done    chan struct{}

	mu        sync.Mutex
	areas     []string
	areasErr  error
	refreshes int
}

func newRig(t *testing.T, roots ...string) *rig {
	t.Helper()
	r := &rig{
		t:       t,
		src:     newFakeSource(),
		quiet:   make(chan time.Time),
		minute:  make(chan time.Time),
		armed:   make(chan struct{}, 100),
		resync:  make(chan struct{}, 100),
		started: make(chan string, 100),
		results: make(chan error, 100),
		areas:   roots,
	}
	r.w = &Watcher{
		Source: r.src,
		Areas: func() ([]string, error) {
			r.mu.Lock()
			defer r.mu.Unlock()
			return slices.Clone(r.areas), r.areasErr
		},
		Refresh: func(root string) (bool, error) {
			r.mu.Lock()
			r.refreshes++
			r.mu.Unlock()
			r.started <- root
			err := <-r.results
			return err == nil, err
		},
		After: func(d time.Duration) <-chan time.Time {
			switch d {
			case 2 * time.Second:
				r.armed <- struct{}{}
				return r.quiet
			case time.Minute:
				r.resync <- struct{}{}
				return r.minute
			case maxDelay:
				// The rig's bursts end by themselves.
				return nil
			}
			t.Errorf("After(%v), want the quiet period or the registry interval", d)
			return nil
		},
		Log:      &r.log,
		Relevant: func(root, path string) bool { return path != "skip" },
	}
	return r
}

func (r *rig) start() {
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel, r.done = cancel, make(chan struct{})
	go func() {
		r.w.Run(ctx)
		close(r.done)
	}()
	<-r.resync
}

// stop ends Run and waits for it; the counters are safe to read after.
func (r *rig) stop() {
	r.cancel()
	<-r.done
}

// event sends a relevant change under root.
func (r *rig) event(root string) {
	r.src.events <- Event{Root: root, Path: filepath.Join(root, "a.go")}
}

// flush returns once Run has handled every event sent before it: an
// irrelevant event is received only after the one ahead of it is through.
func (r *rig) flush(root string) { r.src.events <- Event{Root: root, Path: "skip"} }

func (r *rig) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.refreshes
}

func TestOneRefreshAfterTheQuietPeriod(t *testing.T) {
	r := newRig(t, "a")
	r.start()
	for range 3 {
		r.event("a")
		<-r.armed
	}
	r.flush("a")
	r.src.events <- Event{Root: "unknown", Path: "x.go"}
	r.quiet <- time.Time{}
	r.results <- nil
	<-r.started
	// Armed again, then ended before the quiet period is over.
	r.event("a")
	<-r.armed
	r.stop()
	if n := r.count(); n != 1 {
		t.Fatalf("%d refreshes, want 1", n)
	}
}

func TestEventsDuringARefreshCauseOneMore(t *testing.T) {
	r := newRig(t, "a")
	r.start()
	r.event("a")
	<-r.armed
	r.quiet <- time.Time{}
	<-r.started
	for range 3 {
		r.event("a")
	}
	r.flush("a")
	r.results <- nil
	<-r.armed
	r.quiet <- time.Time{}
	<-r.started
	r.results <- nil
	r.stop()
	if n := r.count(); n != 2 {
		t.Fatalf("%d refreshes, want 2", n)
	}
}

func TestOverflowRefreshes(t *testing.T) {
	r := newRig(t, "a")
	r.w.Relevant = func(string, string) bool { return false }
	r.start()
	r.src.events <- Event{Root: "a", Overflow: true}
	<-r.armed
	r.quiet <- time.Time{}
	if root := <-r.started; root != "a" {
		t.Fatalf("refreshed %q, want a", root)
	}
	r.results <- nil
	r.stop()
}

func TestLockedRefreshRearms(t *testing.T) {
	r := newRig(t, "a")
	r.start()
	r.event("a")
	<-r.armed
	r.quiet <- time.Time{}
	<-r.started
	r.results <- &ask.LockedError{Path: "lock"}
	// No new event: the lock alone arms the quiet period again.
	<-r.armed
	r.quiet <- time.Time{}
	<-r.started
	r.results <- nil
	r.stop()
	if n := r.count(); n != 2 || r.log.Len() != 0 {
		t.Fatalf("%d refreshes, log %q; want 2 and a silent log", n, r.log.String())
	}
}

func TestRefreshErrorIsLogged(t *testing.T) {
	r := newRig(t, "a")
	r.start()
	r.event("a")
	<-r.armed
	r.quiet <- time.Time{}
	<-r.started
	r.results <- errors.New("disk full")
	r.stop()
	lines := strings.Split(strings.TrimSpace(r.log.String()), "\n")
	if len(lines) != 1 || !strings.Contains(lines[0], "disk full") || !strings.Contains(lines[0], "a") {
		t.Fatalf("log %q, want one line naming the area and the error", r.log.String())
	}
}

func TestForeignExtractorIsNotRebuilt(t *testing.T) {
	root := t.TempDir()
	path := freshness.Path(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"version":1,"extractor":"other/1","files":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	r := newRig(t, root)
	r.start()
	for range 2 {
		r.event(root)
		<-r.armed
		r.quiet <- time.Time{}
	}
	// The next arm proves the second tick is through.
	r.event(root)
	<-r.armed
	r.stop()
	lines := strings.Split(strings.TrimSpace(r.log.String()), "\n")
	if n := r.count(); n != 0 || len(lines) != 1 || !strings.Contains(lines[0], "other/1") {
		t.Fatalf("%d refreshes, log %q; want none and one line naming the extractor", n, r.log.String())
	}
}

func TestAreasFollowTheRegistry(t *testing.T) {
	r := newRig(t, "a", "b", "broken")
	r.src.addErr["broken"] = errors.New("inotify limit")
	r.start()
	r.mu.Lock()
	r.areas = []string{"a", "broken"}
	r.mu.Unlock()
	r.minute <- time.Time{}
	<-r.resync
	r.mu.Lock()
	r.areasErr = errors.New("registry unreadable")
	r.mu.Unlock()
	r.minute <- time.Time{}
	<-r.resync
	r.stop()
	if !slices.Equal(r.src.added, []string{"a", "b"}) || !slices.Equal(r.src.removed, []string{"b"}) {
		t.Fatalf("added %v, removed %v; want a and b added, b removed", r.src.added, r.src.removed)
	}
	log := r.log.String()
	// An area the source cannot watch is given up, not retried every minute.
	if strings.Count(log, "inotify limit") != 1 || r.src.attempts["broken"] != 1 || !strings.Contains(log, "registry unreadable") {
		t.Fatalf("log %q, %d attempts on broken; want the failed area once and the failed registry", log, r.src.attempts["broken"])
	}
}

func TestAnAreaWhoseWatchFailsIsGivenUp(t *testing.T) {
	r := newRig(t, "a")
	r.start()
	r.src.events <- Event{Root: "a", Err: errors.New("network name deleted")}
	// A second error from a sibling handle arrives after the area is gone.
	r.src.events <- Event{Root: "a", Err: errors.New("again")}
	r.event("a")
	r.minute <- time.Time{}
	<-r.resync
	r.flush("a")
	r.stop()
	log := r.log.String()
	if !slices.Equal(r.src.added, []string{"a"}) || !slices.Equal(r.src.removed, []string{"a"}) || r.count() != 0 {
		t.Fatalf("added %v, removed %v, %d refreshes; want a added and removed once, no refresh", r.src.added, r.src.removed, r.count())
	}
	if strings.Count(log, "\n") != 1 || !strings.Contains(log, "network name deleted") {
		t.Fatalf("log %q, want one line naming the error", log)
	}
}

func TestRunWaitsForTheRefreshBeforeItReturns(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, "a")
		r.start()
		r.event("a")
		<-r.armed
		r.quiet <- time.Time{}
		<-r.started
		r.cancel()
		// Every goroutine is now blocked for good: Run in its wait for the
		// refresh, or past Close if it does not wait.
		synctest.Wait()
		select {
		case <-r.src.closed:
			t.Error("the source closed while a refresh was running")
		default:
		}
		r.results <- nil
		<-r.done
		select {
		case <-r.src.closed:
		default:
			t.Fatal("Run returned without closing the source")
		}
	})
}

func TestRunLogsAFailedClose(t *testing.T) {
	r := newRig(t, "a")
	r.src.closeErr = errors.New("handle leaked")
	r.start()
	r.stop()
	if log := r.log.String(); !strings.Contains(log, "handle leaked") {
		t.Fatalf("log %q, want the close error", log)
	}
}

func TestRelevantSkipsDotDirectories(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	git := filepath.Join(root, ".git")
	rec := corpus.Record{GitPaths: corpus.GitPaths{
		Root:    root,
		Index:   filepath.Join(".git", "index"),
		Config:  filepath.Join(git, "config"),
		Exclude: filepath.Join(".git", "info", "exclude"),
	}}
	in := func(rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }
	for path, want := range map[string]bool{
		in(".git/objects/x"):                  false,
		in(".loomux/state/graph/wiring.json"): false,
		in(".claude/worktrees/a/b.go"):        false,
		in(".gitignore"):                      true,
		in("sub/.gitignore"):                  true,
		in("src/a.go"):                        true,
		in(".git/index"):                      true,
		in(".git/config"):                     true,
		in(".git/info/exclude"):               true,
		filepath.Join(t.TempDir(), "x.go"):    false,
		"relative.go":                         false,
	} {
		if got := Relevant(rec, root, path); got != want {
			t.Errorf("Relevant(%q) = %v, want %v", path, got, want)
		}
	}
	outside := filepath.Join(t.TempDir(), "main", ".git", "worktrees", "wt", "index")
	rec.GitPaths.Index = outside
	if !Relevant(rec, root, outside) {
		t.Errorf("Relevant(%q) = false, want true for the record's index", outside)
	}
}

// gitRepo is an empty repository with one commit, under a git that reads no
// configuration of this machine.
func gitRepo(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	root := filepath.Join(t.TempDir(), "main")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "init", "-q")
	runGit(t, root, "add", "a.go")
	runGit(t, root, "-c", "user.name=x", "-c", "user.email=x@x", "commit", "-q", "-m", "x")
	return root
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = append(gitenv.Environ(),
		"GIT_CONFIG_NOSYSTEM="+os.Getenv("GIT_CONFIG_NOSYSTEM"),
		"GIT_CONFIG_GLOBAL="+os.Getenv("GIT_CONFIG_GLOBAL"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// containsPath reports whether paths holds want, the way Windows compares.
func containsPath(paths []string, want string) bool {
	return slices.ContainsFunc(paths, func(p string) bool { return strings.EqualFold(p, filepath.Clean(want)) })
}

func TestExtraPathsOfAWorktree(t *testing.T) {
	main := gitRepo(t)
	wt := filepath.Join(filepath.Dir(main), "wt")
	runGit(t, main, "worktree", "add", "-q", wt)
	if _, _, err := query.BuildWith(wt, query.BuildOptions{}, nil); err != nil {
		t.Fatal(err)
	}
	rec, ok := corpus.Read(wt)
	if !ok {
		t.Fatal("no corpus record in the worktree")
	}
	extra := ExtraPaths(rec)
	index := corpus.GitAbs(rec, rec.GitPaths.Index)
	config := corpus.GitAbs(rec, rec.GitPaths.Config)
	if !containsPath(extra, index) || !containsPath(extra, config) {
		t.Fatalf("ExtraPaths = %v, want the index %q and the config %q", extra, index, config)
	}
	for _, p := range extra {
		if rel, err := filepath.Rel(wt, p); err == nil && !strings.HasPrefix(rel, "..") {
			t.Errorf("%q lies inside the worktree", p)
		}
	}
}

func TestExtraPathsOfASubdirectoryRoot(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	rec := corpus.Record{GitPaths: corpus.GitPaths{Root: repo, Prefix: "sub/dir", Index: filepath.Join(".git", "index")}}
	if got, want := ExtraPaths(rec), []string{filepath.Join(repo, ".git", "index")}; !slices.Equal(got, want) {
		t.Fatalf("ExtraPaths = %v, want %v", got, want)
	}
	rec.GitPaths.Prefix = ""
	if got := ExtraPaths(rec); len(got) != 0 {
		t.Fatalf("ExtraPaths = %v, want none for git files under the root", got)
	}
}

// registry writes a registry naming the given areas under a fresh state
// directory and returns it.
func registry(t *testing.T, areas map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	var body strings.Builder
	for scope, path := range areas {
		if err := os.MkdirAll(filepath.Join(path, ".loomux"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, ".loomux", "config.toml"), []byte("[area]\nscope = \""+scope+"\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		body.WriteString("[[area]]\nscope = \"" + scope + "\"\npath = " + strconv.Quote(filepath.ToSlash(path)) + "\n\n")
	}
	if err := os.WriteFile(filepath.Join(dir, "registry.toml"), []byte(body.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestAreasListsTheRegisteredGraphs(t *testing.T) {
	withGraph, without := t.TempDir(), t.TempDir()
	wiring := store.WiringPath(withGraph)
	if err := os.MkdirAll(filepath.Dir(wiring), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wiring, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := registry(t, map[string]string{"project/with": withGraph, "project/without": without})
	roots, err := Areas(dir)()
	if err != nil || !slices.Equal(roots, []string{filepath.Clean(withGraph)}) {
		t.Fatalf("Areas = %v, %v; want only %q", roots, err, withGraph)
	}
}

func TestAreasWithoutARegistryIsEmpty(t *testing.T) {
	if roots, err := Areas(t.TempDir())(); err != nil || len(roots) != 0 {
		t.Fatalf("Areas = %v, %v; want none", roots, err)
	}
}

func TestAreasReportsABrokenRegistry(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "registry.toml"), []byte("[[area"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Areas(dir)(); err == nil {
		t.Fatal("want the registry's error")
	}
}

// A change every second never leaves a quiet period; the area is checked
// anyway once maxDelay has passed since the first change of the burst.
func TestABusyAreaIsCheckedWithinTheMaxDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		src := newFakeSource()
		refreshed := make(chan time.Duration, 10)
		start := time.Now()
		w := &Watcher{
			Source:  src,
			Areas:   func() ([]string, error) { return []string{"a"}, nil },
			Refresh: func(string) (bool, error) { refreshed <- time.Since(start); return true, nil },
			After:   time.After,
			Log:     io.Discard,
			// Every change counts.
			Relevant: func(string, string) bool { return true },
		}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			w.Run(ctx)
			close(done)
		}()
		for range 30 {
			src.events <- Event{Root: "a", Path: "a.go"}
			time.Sleep(time.Second)
		}
		cancel()
		<-done
		select {
		case at := <-refreshed:
			if at > maxDelay+time.Second {
				t.Fatalf("first check after %v, want within %v", at, maxDelay)
			}
		default:
			t.Fatalf("no check in 30 s of changes, want one within %v", maxDelay)
		}
	})
}

func TestJudgeDropsWhatTheRecordIgnores(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	rec := corpus.Record{
		GitPaths: corpus.GitPaths{Root: root, Index: filepath.Join(".git", "index")},
		Sources:  []corpus.Source{{Rel: ".gitignore", Kind: "gitignore", Body: []byte("*.log\nbin/\n")}},
		Tracked:  []string{"keep.log"},
	}
	stubRecord(t, rec)
	judge := Judge()
	in := func(rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }
	for path, want := range map[string]bool{
		in("logs/app.log"): false,
		in("bin/x.o"):      false,
		in("keep.log"):     true,
		in("src/a.go"):     true,
		in(".gitignore"):   true,
		in(".git/index"):   true,
		in(".loomux/x"):    false,
	} {
		if got := judge(root, path); got != want {
			t.Errorf("judge(%q) = %v, want %v", path, got, want)
		}
	}
}

// Without a record there are no rules: every change outside a dot directory
// counts.
func TestJudgeWithoutARecord(t *testing.T) {
	root := t.TempDir()
	stubRecord(t, corpus.Record{})
	if !Judge()(root, filepath.Join(root, "x.log")) {
		t.Fatal("a change without a record must count")
	}
}

// The record is read once per area, and again only when its file moved.
func TestJudgeReadsTheRecordOncePerVersion(t *testing.T) {
	root := t.TempDir()
	reads := stubRecord(t, corpus.Record{})
	judge := Judge()
	for range 3 {
		judge(root, filepath.Join(root, "a.go"))
	}
	if *reads != 1 {
		t.Fatalf("%d reads, want 1", *reads)
	}
	path := corpus.RecordPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	judge(root, filepath.Join(root, "a.go"))
	if *reads != 2 {
		t.Fatalf("%d reads after the record moved, want 2", *reads)
	}
}

// stubRecord makes every record read return rec and counts the reads.
func stubRecord(t *testing.T, rec corpus.Record) *int {
	t.Helper()
	n := 0
	old := readRecord
	readRecord = func(string) (corpus.Record, bool) {
		n++
		return rec, rec.GitPaths.Root != ""
	}
	t.Cleanup(func() { readRecord = old })
	return &n
}

// A whitelist ignore matches the git files and every .gitignore too; those
// still decide what grep sees, so they still count.
func TestJudgeKeepsGitFilesUnderAWhitelistIgnore(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	rec := corpus.Record{
		GitPaths: corpus.GitPaths{
			Root: root, Index: filepath.Join(".git", "index"),
			Config: filepath.Join(".git", "config"), Exclude: filepath.Join(".git", "info", "exclude"),
		},
		Sources: []corpus.Source{{Rel: ".gitignore", Kind: "gitignore", Body: []byte("*\n!*.go\n!*/\n")}},
	}
	stubRecord(t, rec)
	judge := Judge()
	in := func(rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }
	for path, want := range map[string]bool{
		in(".git/index"):      true,
		in(".git/config"):     true,
		in("sub/.gitignore"):  true,
		in("x.log"):           false,
		in("a.go"):            true,
		in(".git/objects/ab"): false,
	} {
		if got := judge(root, path); got != want {
			t.Errorf("judge(%q) = %v, want %v", path, got, want)
		}
	}
}
