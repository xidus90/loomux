package sessions

import (
	"os"
	"path/filepath"
	"testing"
)

// A file that cannot be read counts as empty, exactly as state.py decides:
// raising would end every turn with an internal error over a counter whose
// worst case is three extra rounds.
func TestReadStateOfBrokenFileIsEmpty(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, filepath.FromSlash(StateDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "s1.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := ReadState(root, "s1")

	if got.Blocks != 0 || got.Base != "" {
		t.Fatalf("a damaged file reads as empty, got %+v", got)
	}
}

func TestReadStateOfMissingFileIsEmpty(t *testing.T) {
	if got := ReadState(t.TempDir(), "nobody"); got.Blocks != 0 || got.Base != "" {
		t.Fatalf("an absent file reads as empty, got %+v", got)
	}
}

// The shape check is state.py's: a `blocks` that is not a number makes the
// whole file empty rather than half-trusted.
func TestReadStateOfWrongShapeIsEmpty(t *testing.T) {
	root := t.TempDir()
	writeRaw(t, root, "s1", `{"blocks": "three", "base": null}`)

	if got := ReadState(root, "s1"); got.Blocks != 0 {
		t.Fatalf("a wrong shape reads as empty, got %+v", got)
	}
}

// state.py refuses a `base` that is there but is not a string, and so must
// this: the file is damaged, not merely old.
func TestReadStateOfANonStringBaseIsEmpty(t *testing.T) {
	root := t.TempDir()
	writeRaw(t, root, "s1", `{"blocks": 2, "base": 7}`)

	if got := ReadState(root, "s1"); got.Blocks != 0 {
		t.Fatalf("a numeric base reads as empty, got %+v", got)
	}
}

// The counter has been in the file since it existed, so a file without it is
// damage and not age -- and damage reads as empty, base and all, rather than
// as a session whose row of blocks starts over at zero.
func TestReadStateWithoutBlocksIsEmpty(t *testing.T) {
	root := t.TempDir()
	writeRaw(t, root, "s1", `{"base": "b", "green": "g"}`)

	if got := ReadState(root, "s1"); got != (SessionState{}) {
		t.Fatalf("a file without blocks reads as empty, got %+v", got)
	}
}

// Blocks is the only key a file must carry. Since stage 2c the snapshots
// live in a file each, so their absence is the ordinary shape rather than
// damage, and a file written before `base` existed was always merely older --
// state.py reads that field with `get` and not `[...]` for this reason.
func TestReadStateOfBlocksAloneKeepsThem(t *testing.T) {
	root := t.TempDir()
	writeRaw(t, root, "s1", `{"blocks": 2}`)

	got := ReadState(root, "s1")

	if got.Blocks != 2 || got.Base != "" {
		t.Fatalf("expected blocks 2 and no base, got %+v", got)
	}
}

// A file from before stage 2c carries snapshots and no green; it still holds
// a counter and a base worth keeping.
func TestReadStateOfAnOlderFile(t *testing.T) {
	root := t.TempDir()
	writeRaw(t, root, "s1", `{"base": "b", "blocks": 1, "snapshots": {"a": "x"}}`)

	if got := ReadState(root, "s1"); got != (SessionState{Blocks: 1, Base: "b"}) {
		t.Fatalf("got %+v", got)
	}
}

func TestStateKeepsGreen(t *testing.T) {
	root := t.TempDir()
	want := SessionState{Blocks: 2, Base: "b", Green: "g"}

	if err := WriteState(root, "s1", want); err != nil {
		t.Fatal(err)
	}

	if got := ReadState(root, "s1"); got != want {
		t.Fatalf("got %+v", got)
	}
}

func TestWriteStateThenReadBack(t *testing.T) {
	root := t.TempDir()
	want := SessionState{Blocks: 1, Base: "deadbeef"}

	if err := WriteState(root, "s1", want); err != nil {
		t.Fatal(err)
	}
	got := ReadState(root, "s1")

	if got.Blocks != 1 || got.Base != "deadbeef" {
		t.Fatalf("round trip lost something: %+v", got)
	}
}

// An absent base is JSON null and never "", and a green that was never found
// is no key at all rather than an empty string.
func TestWriteStateLeavesGreenOutWhenEmpty(t *testing.T) {
	root := t.TempDir()
	if err := WriteState(root, "s1", SessionState{Blocks: 0}); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(StateDir), "s1.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"base":null,"blocks":0}`; string(raw) != want {
		t.Fatalf("got %s, want %s", raw, want)
	}
}

// The id comes from outside, so it may not decide where the file lands. Same
// rule as safeName, checked through the public door.
func TestWriteStateKeepsATraversingIdInside(t *testing.T) {
	root := t.TempDir()
	if err := WriteState(root, "../../escape", SessionState{Blocks: 1}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(StateDir), "escape.json")); err != nil {
		t.Fatalf("expected the file inside the state directory: %v", err)
	}
}

// A directory that cannot be made is reported, unlike everything on the
// reading side: a state that was meant to be kept and was not is a real loss,
// and the caller decides what to do about it. A plain file where the first
// path segment belongs makes MkdirAll refuse on every platform.
func TestWriteStateReportsADirectoryItCannotCreate(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, filepath.FromSlash(StateDir))
	if err := os.MkdirAll(filepath.Dir(blocker), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteState(root, "s1", SessionState{}); err == nil {
		t.Fatal("WriteState = nil, want an error")
	}
}

// The same for the file itself: a directory standing where the state file goes
// makes WriteFile refuse, on every platform.
func TestWriteStateReportsAFileItCannotWrite(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(StateDir), "s1.json"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := WriteState(root, "s1", SessionState{}); err == nil {
		t.Fatal("WriteState = nil, want an error")
	}
}

func writeRaw(t *testing.T, root, sessionID, body string) {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(StateDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, sessionID+".json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
