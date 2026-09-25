package sessions

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func state(t *testing.T, root, id string, age time.Duration) string {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(StateDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, id+".json")
	if err := os.WriteFile(path, []byte(`{"blocks":0,"snapshots":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	when := time.Now().Add(-age)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOthersCountsEveryOtherLivingSession(t *testing.T) {
	root := t.TempDir()
	state(t, root, "mine", 0)
	state(t, root, "yours", time.Minute)

	got, err := Others(root, "mine", 12*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("Others = %d, want 1", got)
	}
}

// Nobody deletes these files today, so an old one must not hold a junction
// hostage for ever.
func TestOthersIgnoresAStaleFile(t *testing.T) {
	root := t.TempDir()
	state(t, root, "mine", 0)
	state(t, root, "ancient", 48*time.Hour)

	got, err := Others(root, "mine", 12*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Fatalf("Others = %d, want 0", got)
	}
}

func TestOthersOfADirectoryWithoutStateIsZero(t *testing.T) {
	got, err := Others(t.TempDir(), "mine", 12*time.Hour)
	if err != nil || got != 0 {
		t.Fatalf("Others = %d, %v; want 0 and nil", got, err)
	}
}

// Only the session files are counted. The Python hooks are not the only writer
// in a project, and a directory or a file with another suffix beside them is
// nobody's session.
func TestOthersCountsOnlySessionFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, filepath.FromSlash(StateDir))
	if err := os.MkdirAll(filepath.Join(dir, "sub.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Others(root, "mine", 12*time.Hour)
	if err != nil || got != 0 {
		t.Fatalf("Others = %d, %v; want 0 and nil", got, err)
	}
}

// A count that could not be taken is not a count of zero: read as zero, the
// caller would take a junction out from under whoever is still there. Absence
// is the one error that reads as zero, and this is not absence -- `?` is
// illegal in a Windows filename, so os.ReadDir answers with a bad syntax and
// os.IsNotExist is false for it. Measured on 2026-09-07.
func TestOthersReportsADirectoryItCannotRead(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the illegal-name trick this leans on is a Windows rule")
	}
	if _, err := Others(filepath.Join(t.TempDir(), "bad?name"), "mine", 12*time.Hour); err == nil {
		t.Fatal("Others = nil error, want one")
	}
}

// A retired session no longer counts for anybody, keeps its state and its
// subagents' findings for a resume under the same id, and stays ended when a
// stop that was still running rewrites its file afterwards.
func TestRetireEndsTheSessionAndKeepsItsState(t *testing.T) {
	root := t.TempDir()
	if err := WriteState(root, "s1", SessionState{Base: "abc", Blocks: 2}); err != nil {
		t.Fatal(err)
	}
	writeAgent(t, root, "s1", "a", AgentFile{Finding: []string{"x"}})
	state(t, root, "s2", 0)

	if err := Retire(root, "s1"); err != nil {
		t.Fatal(err)
	}
	if err := WriteState(root, "s1", SessionState{Base: "abc", Blocks: 3}); err != nil {
		t.Fatal(err)
	}
	if n, err := Others(root, "s2", time.Hour); err != nil || n != 0 {
		t.Fatalf("others = %d, %v; the retired session still counts", n, err)
	}
	if got := ReadState(root, "s1"); got.Base != "abc" || got.Blocks != 3 {
		t.Fatalf("state %+v", got)
	}
	if _, err := os.Stat(agentPath(root, "s1", "a")); err != nil {
		t.Fatalf("agent file: %v", err)
	}
}

// A revived session counts again; reviving one never retired is no error.
func TestReviveCountsTheSessionAgain(t *testing.T) {
	root := t.TempDir()
	state(t, root, "s1", 0)
	if err := Retire(root, "s1"); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := Revive(root, "s1"); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := Others(root, "s2", time.Hour); err != nil || n != 1 {
		t.Fatalf("others = %d, %v", n, err)
	}
}

// A marker that cannot be put down or taken away is an error. A directory
// with something in it is the portable way to make both refuse.
func TestRetireAndReviveReportAMarkerTheyCannotTouch(t *testing.T) {
	root := t.TempDir()
	state(t, root, "s1", 0)
	busy := endedPath(root, "s1")
	if err := os.MkdirAll(filepath.Join(busy, "inside"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Retire(root, "s1"); err == nil {
		t.Fatal("Retire: want an error")
	}
	if err := Revive(root, "s1"); err == nil {
		t.Fatal("Revive: want an error")
	}
	// A marker directory is no marker: the session still counts.
	if n, err := Others(root, "s2", time.Hour); err != nil || n != 1 {
		t.Fatalf("others = %d, %v", n, err)
	}
}

// A session killed without a SessionEnd has no marker; when it resumes after
// a day its file is made young, and its content stays.
func TestReviveMakesAnUnmarkedOldFileYoung(t *testing.T) {
	root := t.TempDir()
	if err := WriteState(root, "s1", SessionState{Base: "abc", Blocks: 2}); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(statePath(root, "s1"), old, old); err != nil {
		t.Fatal(err)
	}
	if err := Revive(root, "s1"); err != nil {
		t.Fatal(err)
	}
	if n, err := Others(root, "s2", 24*time.Hour); err != nil || n != 1 {
		t.Fatalf("others = %d, %v", n, err)
	}
	if got := ReadState(root, "s1"); got.Base != "abc" || got.Blocks != 2 {
		t.Fatalf("state %+v", got)
	}
}

// A session that never wrote its file gets no marker.
func TestRetireMarksNothingWithoutAFile(t *testing.T) {
	root := t.TempDir()
	if err := Retire(root, "s1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(endedPath(root, "s1")); !os.IsNotExist(err) {
		t.Fatalf("marker: %v", err)
	}
}

// A revived session starts a new row of blocks, keeps its base, and is young
// again however long it rested.
func TestReviveResetsTheBlocksAndMakesTheFileYoung(t *testing.T) {
	root := t.TempDir()
	if err := WriteState(root, "s1", SessionState{Base: "abc", Green: "t", Blocks: 3}); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(statePath(root, "s1"), old, old); err != nil {
		t.Fatal(err)
	}
	if err := Retire(root, "s1"); err != nil {
		t.Fatal(err)
	}
	if err := Revive(root, "s1"); err != nil {
		t.Fatal(err)
	}
	if got := ReadState(root, "s1"); got.Base != "abc" || got.Green != "t" || got.Blocks != 0 {
		t.Fatalf("state %+v", got)
	}
	if n, err := Others(root, "s2", 24*time.Hour); err != nil || n != 1 {
		t.Fatalf("others = %d, %v", n, err)
	}
}

// Forget takes the end marker along with the file.
func TestForgetTakesTheMarkerWith(t *testing.T) {
	root := t.TempDir()
	state(t, root, "s1", 0)
	if err := Retire(root, "s1"); err != nil {
		t.Fatal(err)
	}
	if err := Forget(root, "s1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(endedPath(root, "s1")); !os.IsNotExist(err) {
		t.Fatalf("marker: %v", err)
	}
}

// The id comes from outside, so it may not decide which file is read -- the
// same reasoning as ultraloom/hooks/state.py's own path builder.
func TestForgetRemovesOnlyThisSessionsFile(t *testing.T) {
	root := t.TempDir()
	mine := state(t, root, "mine", 0)
	yours := state(t, root, "yours", 0)

	if err := Forget(root, "mine"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mine); !os.IsNotExist(err) {
		t.Fatalf("the file survived: %v", err)
	}
	if _, err := os.Stat(yours); err != nil {
		t.Fatalf("somebody else's file was removed: %v", err)
	}
}

// The subagents' files sit in a directory beside the session file; a session
// that ends leaves neither behind.
func TestForgetTakesTheAgentsWith(t *testing.T) {
	root := t.TempDir()
	if err := WriteState(root, "s1", SessionState{}); err != nil {
		t.Fatal(err)
	}
	writeAgent(t, root, "s1", "a", AgentFile{Finding: []string{"x"}})

	if err := Forget(root, "s1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(StateDir), "s1")); !os.IsNotExist(err) {
		t.Fatalf("agents dir: %v", err)
	}
}

// A subagent's file that will not go is not a session that ended: leaving it
// would let the next session read a finding nobody left for it. Windows only,
// like the unreadable-directory case above -- Go's Open asks for no
// FILE_SHARE_DELETE, so an open handle makes the removal refuse, while POSIX
// unlinks an open file happily.
func TestForgetReportsAnAgentsDirectoryItCannotRemove(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("an open handle only blocks a removal on Windows")
	}
	root := t.TempDir()
	writeAgent(t, root, "s1", "a", AgentFile{Finding: []string{"x"}})
	held, err := os.Open(filepath.Join(root, filepath.FromSlash(StateDir), "s1", "agents", "a.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()

	if err := Forget(root, "s1"); err == nil {
		t.Fatal("Forget = nil, want an error")
	}
}

func TestForgetAnUnknownSessionIsNotAnError(t *testing.T) {
	if err := Forget(t.TempDir(), "nobody"); err != nil {
		t.Fatalf("Forget = %v, want nil", err)
	}
}

func TestASeparatorInTheIdCannotClimbOut(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "outside.json")
	if err := os.WriteFile(outside, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Forget(root, "../outside"); err != nil {
		t.Fatalf("Forget = %v, want nil", err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("a file outside the state directory was removed: %v", err)
	}
}

// state.py keeps `char.isalnum()`, and that answers true for a letter outside
// ASCII: measured on 2026-09-07 with CPython 3.13, `'ä'.isalnum()` is
// True. So the Python side writes `<letter>.json` for such an id, and a Go
// rule that dropped it would look for `unnamed.json` and find nothing.
func TestAnIdOutsideAsciiIsTheSameNameOnBothSides(t *testing.T) {
	root := t.TempDir()
	mine := state(t, root, "ä٣", 0)

	if err := Forget(root, "ä٣"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mine); !os.IsNotExist(err) {
		t.Fatalf("the file survived: %v", err)
	}
}

// An id with nothing keepable in it lands on one shared name rather than on
// the state directory itself -- state.py's `or "unnamed"`, and the reason it
// is there.
func TestAnIdWithNothingKeepableInItBecomesUnnamed(t *testing.T) {
	root := t.TempDir()
	mine := state(t, root, "unnamed", 0)
	// A neighbour nobody asked about. Without the shared name, Forget's
	// RemoveAll would be aimed at the state directory itself and take this
	// file with it -- and the removal of `mine` above would then look right
	// for the wrong reason.
	other := state(t, root, "other", 0)

	if err := Forget(root, "!!!"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mine); !os.IsNotExist(err) {
		t.Fatalf("the file survived: %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("another session's file was removed: %v", err)
	}
}

// A file that is there and will not go is not the same as one that was never
// written, and reading it as "done" would leave the junction standing with
// nobody to take it out. A directory with something in it is the portable way
// to make os.Remove refuse: measured on 2026-09-07 it answers "directory not
// empty", for which os.IsNotExist is false.
func TestForgetReportsAFileItCannotRemove(t *testing.T) {
	root := t.TempDir()
	busy := filepath.Join(root, filepath.FromSlash(StateDir), "mine.json")
	if err := os.MkdirAll(busy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(busy, "inside"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Forget(root, "mine"); err == nil {
		t.Fatal("Forget = nil, want an error")
	}
}
