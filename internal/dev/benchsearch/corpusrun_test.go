package benchsearch

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/index"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// isolate points qmd's config and cache at directories of the test's own,
// gives the user a model choice to copy, and returns a lock directory.
func isolate(t *testing.T) (lockDir string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	mkdir(t, filepath.Dir(index.QmdConfigPath()))
	mkdir(t, index.QmdCacheDir())
	writeFile(t, index.QmdConfigPath(), "models:\n  embed: E\n  generate: G\n  rerank: R\n")
	return t.TempDir()
}

func mkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func fixed(s string) func() string { return func() string { return s } }

// logTo hands warnings to the test log; failOn makes any warning a failure.
func logTo(t *testing.T) func(string)  { return func(s string) { t.Log(s) } }
func failOn(t *testing.T) func(string) { return func(s string) { t.Fatal(s) } }

func gone(t *testing.T, paths ...string) {
	t.Helper()
	for _, path := range paths {
		if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s left behind", path)
		}
	}
}

func present(t *testing.T, paths ...string) {
	t.Helper()
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("%s is gone: %v", path, err)
		}
	}
}

func benchLeftovers(t *testing.T) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(index.QmdConfigPath()), indexPrefix+"*"))
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

func TestPrepareCorpusWritesStateAndANamedConfig(t *testing.T) {
	lockDir := isolate(t)
	p, cleanup, err := PrepareCorpus(corpusDir(t), lockDir, fixed("a1"), logTo(t))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if p.Index != "loomux-bench-a1" || p.Scope != CorpusScope || p.Stand != corpusDir(t) {
		t.Fatalf("%+v", p)
	}
	if p.StateDir != filepath.Join(lockDir, p.Index) {
		t.Fatalf("state = %s", p.StateDir)
	}
	areas, err := config.ReadRegistry(p.StateDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 1 || !areas[0].ReadOnly || areas[0].Path != filepath.ToSlash(filepath.Join(p.Stand, "notes")) {
		t.Fatalf("areas = %+v", areas)
	}
	visible, err := privacy.VisibleAreas(p.StateDir, "", CorpusScope, privacy.ChannelLocal)
	if err != nil || len(visible) != 1 {
		t.Fatalf("visible = %+v, %v", visible, err)
	}
	reg, err := identity.ReadIdentities(filepath.Join(config.ManifestDir(areas[0], p.StateDir), "_identities.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(reg) != 100 {
		t.Fatalf("register holds %d", len(reg))
	}
	note := reg["baustatik-01.md"]
	want, _ := identity.ContentHash(filepath.Join(p.Stand, "notes", "baustatik-01.md"))
	if note.ContentHash != want || note.Revision != 1 || note.DocID == "" {
		t.Fatalf("baustatik-01.md = %+v", note)
	}
	if m := index.Models(index.QmdConfigPathFor(p.Index)); m["embedding"] != "E" || m["query_expansion"] != "G" || m["rerank"] != "R" {
		t.Fatalf("models = %v", m)
	}
	named, _ := os.ReadFile(index.QmdConfigPathFor(p.Index))
	if !strings.Contains(string(named), filepath.ToSlash(filepath.Join(p.Stand, "notes"))) || !strings.Contains(string(named), CorpusScope+":") {
		t.Fatalf("named config lacks the collection:\n%s", named)
	}
	real, _ := os.ReadFile(index.QmdConfigPath())
	if strings.Contains(string(real), CorpusScope) {
		t.Fatal("the real index.yml was touched")
	}
}

func TestPrepareCorpusLeavesUnknownModelsToQmd(t *testing.T) {
	lockDir := isolate(t)
	writeFile(t, index.QmdConfigPath(), "models:\n  embed: E\n")
	p, cleanup, err := PrepareCorpus(corpusDir(t), lockDir, fixed("a1"), logTo(t))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	named, _ := os.ReadFile(index.QmdConfigPathFor(p.Index))
	if !strings.Contains(string(named), "embed: E") || strings.Contains(string(named), "rerank") || strings.Contains(string(named), "generate") {
		t.Fatalf("named config:\n%s", named)
	}
}

func TestCleanupRemovesEverythingOfTheRun(t *testing.T) {
	lockDir := isolate(t)
	var warnings []string
	p, cleanup, err := PrepareCorpus(corpusDir(t), lockDir, fixed("a1"), func(s string) { warnings = append(warnings, s) })
	if err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{".sqlite", ".sqlite-wal", ".sqlite-shm"} {
		writeFile(t, filepath.Join(index.QmdCacheDir(), p.Index+suffix), "")
	}
	writeFile(t, filepath.Join(index.QmdCacheDir(), ".qmd-embed.lock"), "")
	present(t, index.QmdConfigPathFor(p.Index)+".brain-backup")
	cleanup()
	cleanup()
	gone(t, p.StateDir, index.QmdConfigPathFor(p.Index), index.QmdConfigPathFor(p.Index)+".brain-backup",
		filepath.Join(index.QmdCacheDir(), p.Index+".sqlite"), filepath.Join(index.QmdCacheDir(), p.Index+".sqlite-wal"),
		filepath.Join(index.QmdCacheDir(), p.Index+".sqlite-shm"), filepath.Join(lockDir, p.Index+".lock"))
	present(t, index.QmdConfigPath(), filepath.Join(index.QmdCacheDir(), ".qmd-embed.lock"))
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
}

func TestCleanupWarnsAboutWhatItCannotRemove(t *testing.T) {
	lockDir := isolate(t)
	var warnings []string
	warn := func(s string) { warnings = append(warnings, s) }
	p, cleanup, err := PrepareCorpus(corpusDir(t), lockDir, fixed("a1"), warn)
	if err != nil {
		t.Fatal(err)
	}
	stuck := filepath.Join(index.QmdCacheDir(), p.Index+".sqlite-wal")
	writeFile(t, filepath.Join(mkdirAll(t, stuck), "x"), "")
	cleanup()
	if len(warnings) != 1 || !strings.Contains(warnings[0], stuck) {
		t.Fatalf("warnings = %v", warnings)
	}
	gone(t, p.StateDir, index.QmdConfigPathFor(p.Index))
	// The lock file is what lets the next sweep find the stuck file.
	present(t, filepath.Join(lockDir, p.Index+".lock"))
	if err := os.RemoveAll(stuck); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(index.QmdCacheDir(), p.Index+".sqlite"), "")
	SweepStale(lockDir, failOn(t))
	gone(t, filepath.Join(index.QmdCacheDir(), p.Index+".sqlite"))
}

func mkdirAll(t *testing.T, dir string) string {
	t.Helper()
	mkdir(t, dir)
	return dir
}

func TestAFailedPreparationCleansUpToo(t *testing.T) {
	lockDir := isolate(t)
	// The ownership record is a directory: SyncCollections fails reading it,
	// after <name>.yml has been written.
	mkdir(t, filepath.Join(lockDir, "loomux-bench-a1", "qmd-owned.txt"))
	_, cleanup, err := PrepareCorpus(corpusDir(t), lockDir, fixed("a1"), logTo(t))
	if err == nil {
		t.Fatal("no error")
	}
	present(t, index.QmdConfigPathFor("loomux-bench-a1"))
	cleanup()
	if left := benchLeftovers(t); len(left) != 0 {
		t.Fatalf("left behind: %v", left)
	}
	gone(t, filepath.Join(lockDir, "loomux-bench-a1"), filepath.Join(lockDir, "loomux-bench-a1.lock"))
}

func TestPrepareCorpusRefusesWhatItCannotWrite(t *testing.T) {
	cases := map[string]func(t *testing.T, lockDir string) (stand string){
		"declaration directory": func(t *testing.T, lockDir string) string {
			writeFile(t, filepath.Join(mkdirAll(t, filepath.Join(lockDir, "loomux-bench-a1")), "areas"), "")
			return corpusDir(t)
		},
		"unreadable note": func(t *testing.T, lockDir string) string {
			stand := copyCorpus(t)
			mkdir(t, filepath.Join(stand, "notes", "zz.md"))
			return stand
		},
	}
	for name, arrange := range cases {
		t.Run(name, func(t *testing.T) {
			lockDir := isolate(t)
			stand := arrange(t, lockDir)
			_, cleanup, err := PrepareCorpus(stand, lockDir, fixed("a1"), logTo(t))
			if err == nil {
				t.Fatal("no error")
			}
			cleanup()
			if left := benchLeftovers(t); len(left) != 0 {
				t.Fatalf("left behind: %v", left)
			}
			gone(t, filepath.Join(lockDir, "loomux-bench-a1"), filepath.Join(lockDir, "loomux-bench-a1.lock"))
		})
	}
}

func TestPrepareCorpusWithoutALock(t *testing.T) {
	cases := map[string]func(t *testing.T) (lockDir string){
		"lock directory is a file": func(t *testing.T) string {
			file := filepath.Join(t.TempDir(), "file")
			writeFile(t, file, "")
			return filepath.Join(file, "locks")
		},
		"lock file is a directory": func(t *testing.T) string {
			lockDir := t.TempDir()
			mkdir(t, filepath.Join(lockDir, "loomux-bench-a1.lock"))
			return lockDir
		},
	}
	for name, arrange := range cases {
		t.Run(name, func(t *testing.T) {
			isolate(t)
			_, cleanup, err := PrepareCorpus(corpusDir(t), arrange(t), fixed("a1"), failOn(t))
			if err == nil {
				t.Fatal("no error")
			}
			cleanup()
			if left := benchLeftovers(t); len(left) != 0 {
				t.Fatalf("left behind: %v", left)
			}
		})
	}
}

func TestSweepStaleLeavesAHeldIndexAlone(t *testing.T) {
	lockDir := isolate(t)
	h, err := lock.Acquire(filepath.Join(lockDir, "loomux-bench-a.lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer h.Release()
	mkdir(t, filepath.Join(lockDir, "loomux-bench-a"))
	writeFile(t, index.QmdConfigPathFor("loomux-bench-a"), "")
	writeFile(t, filepath.Join(index.QmdCacheDir(), "loomux-bench-a.sqlite"), "")
	writeFile(t, index.QmdConfigPathFor("loomux-bench-b"), "")
	writeFile(t, index.QmdConfigPathFor("loomux-bench-b")+".brain-backup", "")
	writeFile(t, filepath.Join(index.QmdCacheDir(), "loomux-bench-b.sqlite"), "")
	writeFile(t, filepath.Join(index.QmdCacheDir(), "loomux-bench-b.sqlite-wal"), "")
	mkdir(t, filepath.Join(lockDir, "loomux-bench-b"))
	writeFile(t, filepath.Join(lockDir, "loomux-bench-b.lock"), "")
	SweepStale(lockDir, failOn(t))
	present(t, index.QmdConfigPathFor("loomux-bench-a"), filepath.Join(index.QmdCacheDir(), "loomux-bench-a.sqlite"),
		filepath.Join(lockDir, "loomux-bench-a"))
	gone(t, index.QmdConfigPathFor("loomux-bench-b"), index.QmdConfigPathFor("loomux-bench-b")+".brain-backup",
		filepath.Join(index.QmdCacheDir(), "loomux-bench-b.sqlite"), filepath.Join(index.QmdCacheDir(), "loomux-bench-b.sqlite-wal"),
		filepath.Join(lockDir, "loomux-bench-b"))
	present(t, filepath.Join(lockDir, "loomux-bench-a.lock"), filepath.Join(lockDir, "loomux-bench-b.lock"))
}

func TestSweepStaleFindsWhatACrashLeftWithoutAConfig(t *testing.T) {
	lockDir := isolate(t)
	// a database without its config or state, and a state that never got an index
	writeFile(t, filepath.Join(index.QmdCacheDir(), "loomux-bench-c.sqlite"), "")
	writeFile(t, filepath.Join(lockDir, "loomux-bench-c.lock"), "")
	mkdir(t, filepath.Join(lockDir, "loomux-bench-d"))
	writeFile(t, filepath.Join(lockDir, "loomux-bench-d.lock"), "")
	SweepStale(lockDir, failOn(t))
	gone(t, filepath.Join(index.QmdCacheDir(), "loomux-bench-c.sqlite"), filepath.Join(lockDir, "loomux-bench-d"))
	present(t, filepath.Join(lockDir, "loomux-bench-c.lock"), filepath.Join(lockDir, "loomux-bench-d.lock"))
}

func TestSweepStaleLeavesARunningRunWhole(t *testing.T) {
	lockDir := isolate(t)
	p, cleanup, err := PrepareCorpus(corpusDir(t), lockDir, fixed("a1"), failOn(t))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	SweepStale(lockDir, failOn(t))
	present(t, p.StateDir, index.QmdConfigPathFor(p.Index), filepath.Join(lockDir, p.Index+".lock"))
}

// lock.Acquire creates the lock file before it locks it. A sweep that finds
// the file free in that gap must leave it in place, or the starting run would
// lock an orphaned inode and a later sweep would find its index free.
func TestSweepStaleKeepsTheLockFileOfARunThatIsStarting(t *testing.T) {
	lockDir := isolate(t)
	lockFile := filepath.Join(lockDir, "loomux-bench-f.lock")
	opened, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	SweepStale(lockDir, failOn(t))
	onDisk, err := os.Stat(lockFile)
	if err != nil {
		t.Fatalf("the sweep removed the lock file: %v", err)
	}
	mine, _ := opened.Stat()
	if !os.SameFile(onDisk, mine) {
		t.Fatal("the lock path names another file than the run opened")
	}
	h, err := lock.Acquire(lockFile)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Release()
	writeFile(t, index.QmdConfigPathFor("loomux-bench-f"), "")
	SweepStale(lockDir, failOn(t))
	present(t, index.QmdConfigPathFor("loomux-bench-f"))
}

func TestCleanupKeepsTheLockFileWhileTheStateStays(t *testing.T) {
	lockDir := isolate(t)
	var warnings []string
	p, cleanup, err := PrepareCorpus(corpusDir(t), lockDir, fixed("a1"), func(s string) { warnings = append(warnings, s) })
	if err != nil {
		t.Fatal(err)
	}
	unblock := blockRemoval(t, p.StateDir)
	cleanup()
	if len(warnings) == 0 || !strings.Contains(strings.Join(warnings, "\n"), p.StateDir) {
		t.Fatalf("warnings = %v", warnings)
	}
	present(t, filepath.Join(lockDir, p.Index+".lock"))
	gone(t, index.QmdConfigPathFor(p.Index))
	unblock()
	SweepStale(lockDir, failOn(t))
	gone(t, p.StateDir)
}

// blockRemoval makes os.RemoveAll of dir fail until the returned function
// runs: an open file on Windows, a directory without write permission
// elsewhere.
func blockRemoval(t *testing.T, dir string) func() {
	t.Helper()
	inner := mkdirAll(t, filepath.Join(dir, "blocked"))
	pinned := filepath.Join(inner, "pinned")
	writeFile(t, pinned, "")
	if runtime.GOOS == "windows" {
		f, err := os.Open(pinned)
		if err != nil {
			t.Fatal(err)
		}
		return func() { f.Close() }
	}
	if err := os.Chmod(inner, 0o500); err != nil {
		t.Fatal(err)
	}
	return func() { os.Chmod(inner, 0o755) }
}

func TestSweepStaleWarnsWhenItCannotAskTheLock(t *testing.T) {
	lockDir := isolate(t)
	writeFile(t, index.QmdConfigPathFor("loomux-bench-e"), "")
	mkdir(t, filepath.Join(lockDir, "loomux-bench-e.lock"))
	var warnings []string
	SweepStale(lockDir, func(s string) { warnings = append(warnings, s) })
	if len(warnings) != 1 || !strings.Contains(warnings[0], "loomux-bench-e") {
		t.Fatalf("warnings = %v", warnings)
	}
	present(t, index.QmdConfigPathFor("loomux-bench-e"))
}

func TestSweepStaleNeverTouchesTheUsersOwnIndexes(t *testing.T) {
	lockDir := isolate(t)
	writeFile(t, filepath.Join(index.QmdCacheDir(), "index.sqlite"), "")
	writeFile(t, index.QmdConfigPathFor("work"), "")
	writeFile(t, filepath.Join(index.QmdCacheDir(), ".qmd-embed.lock"), "")
	SweepStale(lockDir, failOn(t))
	present(t, index.QmdConfigPath(), index.QmdConfigPathFor("work"), filepath.Join(index.QmdCacheDir(), "index.sqlite"),
		filepath.Join(index.QmdCacheDir(), ".qmd-embed.lock"))
}

// Names come from the lock directory alone. A run of another state directory
// shares qmd's directories but holds its lock elsewhere; asked here, its lock
// would answer free.
func TestSweepStaleLeavesTheRunsOfAnotherStateDirAlone(t *testing.T) {
	lockDir := isolate(t)
	other := t.TempDir()
	p, cleanup, err := PrepareCorpus(corpusDir(t), other, fixed("x"), failOn(t))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	database := filepath.Join(index.QmdCacheDir(), p.Index+".sqlite")
	writeFile(t, database, "")
	writeFile(t, index.QmdConfigPathFor("loomux-bench-y"), "")
	writeFile(t, filepath.Join(index.QmdCacheDir(), "loomux-bench-y.sqlite"), "")
	SweepStale(lockDir, failOn(t))
	present(t, index.QmdConfigPathFor(p.Index), database, p.StateDir,
		index.QmdConfigPathFor("loomux-bench-y"), filepath.Join(index.QmdCacheDir(), "loomux-bench-y.sqlite"))
	if made, _ := filepath.Glob(filepath.Join(lockDir, "*")); len(made) != 0 {
		t.Fatalf("the sweep made %v", made)
	}
}
