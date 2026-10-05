package cases

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The tests the mutation round of stage 1b-1 asked for. They are gathered
// here rather than scattered because what they have in common is how they
// were found: each one is the case a surviving mutant proved nobody had
// asked for. `internal/cases` has no Python reference -- it is loomux's own
// corpus tool -- so every expectation below is measured against the shell
// quoting the splitter mirrors or read from the recorded corpus under
// `testdata/cases/`, never derived from the Go code that has to satisfy it.
//
// The file is the internal test package because one case reaches the
// unexported `mkdirTemp` seam.

// writeCase lays down the four files LoadCase demands, minus the ones named
// in skip.
func writeCase(t *testing.T, dir string, skip ...string) {
	t.Helper()
	dropped := map[string]bool{}
	for _, s := range skip {
		dropped[s] = true
	}
	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"cmd": "loomux x", "exit": "0", "stdout": ""} {
		if dropped[name] {
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAMissingCmdIsNotAnEmptyOne(t *testing.T) {
	// A command case directory carries `cmd`, `exit`, `stdout` and `world/`; the
	// corpus has all four in every such case (`ls testdata/cases/*/*`).
	// The file that is not there and the file that is there and empty are
	// two different faults, and the reader says which.
	dir := filepath.Join(t.TempDir(), "c")
	writeCase(t, dir, "cmd")
	_, err := LoadCase(dir)
	if err == nil || !strings.Contains(err.Error(), "missing cmd") {
		t.Fatalf("LoadCase err = %v, want one that says the cmd is missing", err)
	}
}

func TestAMissingExitIsNotAnUnreadableOne(t *testing.T) {
	// The same distinction on `exit`: no file at all is a missing
	// recording, an unparseable one is a broken recording.
	dir := filepath.Join(t.TempDir(), "c")
	writeCase(t, dir, "exit")
	_, err := LoadCase(dir)
	if err == nil || !strings.Contains(err.Error(), "missing exit") {
		t.Fatalf("LoadCase err = %v, want one that says the exit is missing", err)
	}
}

func TestDataIsAComparisonTheCorpusWrites(t *testing.T) {
	// `data` is the default and one of the two words a `compare` file
	// may carry, and the corpus writes it out: `grep -rl '^data$'
	// testdata/cases --include=compare` names ten files, among them
	// `testdata/cases/1a/check-commit-msg/english/compare`. A reader that
	// only knew `message` would refuse them.
	dir := filepath.Join(t.TempDir(), "c")
	writeCase(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "compare"), []byte("data\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadCase(dir)
	if err != nil {
		t.Fatalf("LoadCase: %v", err)
	}
	if c.Compare != "data" {
		t.Fatalf("Compare = %q, want data", c.Compare)
	}
}

func TestCasesAreOrderedByVerbAndThenByName(t *testing.T) {
	// The verb is the name of the directory above the case, so cases of
	// one verb can sit under different parents and the walk hands them
	// over in the order of those parents, not of the verbs. The order the
	// reader promises is verb first, name second -- both halves.
	root := t.TempDir()
	for _, rel := range []string{"a/dup/zzz", "b/beta/a2", "m/dup/aaa", "q/alpha/z1"} {
		writeCase(t, filepath.Join(root, filepath.FromSlash(rel)))
	}
	got, err := DiscoverCases(root, "")
	if err != nil {
		t.Fatalf("DiscoverCases: %v", err)
	}
	want := []string{"alpha/z1", "beta/a2", "dup/aaa", "dup/zzz"}
	if len(got) != len(want) {
		t.Fatalf("found %d cases, want %d", len(got), len(want))
	}
	for i, w := range want {
		if have := got[i].Verb + "/" + got[i].Name; have != w {
			t.Fatalf("case %d is %s, want %s", i, have, w)
		}
	}
}

func TestABackslashInsideSingleQuotesIsACharacter(t *testing.T) {
	// Measured: `shlex.split(r"loomux 'a\b'")` answers
	// `['loomux', 'a\b']`. Single quotes take the escape out of the
	// backslash, as they do in the shell the cmd files are written for.
	got, err := SplitCommand(`loomux 'a\b'`)
	if err != nil {
		t.Fatalf("SplitCommand: %v", err)
	}
	if len(got) != 2 || got[1] != `a\b` {
		t.Fatalf("SplitCommand = %q, want [loomux a\b]", got)
	}
}

func TestRunsOfSeparatorsOpenNoEmptyTokens(t *testing.T) {
	// Measured: `shlex.split('loomux  a')` and `shlex.split('loomux a ')`
	// both answer `['loomux', 'a']`. Neither the gap between two spaces
	// nor the space at the end is a word.
	for _, s := range []string{"loomux  a", "loomux a "} {
		got, err := SplitCommand(s)
		if err != nil {
			t.Fatalf("SplitCommand(%q): %v", s, err)
		}
		if len(got) != 2 || got[0] != "loomux" || got[1] != "a" {
			t.Fatalf("SplitCommand(%q) = %q, want [loomux a]", s, got)
		}
	}
}

func TestTheTempDirectoryFailureIsTheOneReported(t *testing.T) {
	// The seam's error is what the caller needs to read: a run that went
	// on without a directory would report whatever failed next instead,
	// and the cause would be gone.
	mkdirTemp = func(string, string) (string, error) { return "", errors.New("no temp") }
	defer func() { mkdirTemp = defaultMkdirTemp }()
	c := &Case{Verb: "v", Name: "n", Path: t.TempDir(), Cmd: "loomux x"}
	_, err := RunCase(c, nil)
	if err == nil || !strings.Contains(err.Error(), "no temp") {
		t.Fatalf("RunCase err = %v, want the temp directory failure", err)
	}
}

func TestACommandThatCannotBeSplitIsNotAnEmptyCommand(t *testing.T) {
	// A cmd file with an unclosed quote is a broken recording, and an
	// empty one is a missing recording: two faults with two repairs. A
	// run that dropped the splitter's word would report the second for
	// the first and send the reader to the wrong file.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := &Case{Verb: "v", Name: "n", Path: dir, Cmd: "loomux " + "'" + "x"}
	run := func(_ []string, _ string, _ io.Reader, _, _ io.Writer) int { return 0 }
	_, err := RunCase(c, run)
	if err == nil || !strings.Contains(err.Error(), "unclosed quote") {
		t.Fatalf("RunCase err = %v, want the unclosed quote", err)
	}
}
