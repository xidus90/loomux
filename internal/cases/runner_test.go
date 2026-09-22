package cases_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/sessions"
	"github.com/xidus90/loomux/internal/testlock"
)

func TestStageWorld(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src")
	dst := filepath.Join(t.TempDir(), "dst")

	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeCaseFile(t, src, "root.txt", []byte("root"))
	writeCaseFile(t, filepath.Join(src, "sub"), "nested.txt", []byte("nested"))

	if err := cases.StageWorld(src, dst); err != nil {
		t.Fatalf("unexpected error staging world: %v", err)
	}

	rootBytes, err := os.ReadFile(filepath.Join(dst, "root.txt"))
	if err != nil || string(rootBytes) != "root" {
		t.Fatalf("expected root file copied, got %v / %q", err, string(rootBytes))
	}
	nestedBytes, err := os.ReadFile(filepath.Join(dst, "sub", "nested.txt"))
	if err != nil || string(nestedBytes) != "nested" {
		t.Fatalf("expected nested file copied, got %v / %q", err, string(nestedBytes))
	}
}

func TestCompareTrees(t *testing.T) {
	actual := filepath.Join(t.TempDir(), "actual")
	expected := filepath.Join(t.TempDir(), "expected")
	if err := os.MkdirAll(actual, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(expected, 0o755); err != nil {
		t.Fatal(err)
	}

	// Case 1: Identical
	writeCaseFile(t, actual, "a.txt", []byte("hello"))
	writeCaseFile(t, expected, "a.txt", []byte("hello"))
	diffs, err := cases.CompareTrees(actual, expected)
	if err != nil || len(diffs) != 0 {
		t.Fatalf("expected identical trees, got %v (err: %v)", diffs, err)
	}

	// Case 2: Content mismatch
	writeCaseFile(t, actual, "a.txt", []byte("changed"))
	diffs, err = cases.CompareTrees(actual, expected)
	if err != nil || len(diffs) != 1 || diffs[0] != "content mismatch: a.txt" {
		t.Fatalf("expected content mismatch, got %v", diffs)
	}

	// Case 3: Missing in actual
	writeCaseFile(t, expected, "b.txt", []byte("missing"))
	writeCaseFile(t, actual, "a.txt", []byte("hello"))
	diffs, err = cases.CompareTrees(actual, expected)
	if err != nil || len(diffs) != 1 || diffs[0] != "missing file in actual: b.txt" {
		t.Fatalf("expected missing file mismatch, got %v", diffs)
	}

	// Case 4: Extra in actual
	_ = os.Remove(filepath.Join(expected, "b.txt"))
	writeCaseFile(t, actual, "extra.txt", []byte("extra"))
	diffs, err = cases.CompareTrees(actual, expected)
	if err != nil || len(diffs) != 1 || diffs[0] != "unexpected extra file in actual: extra.txt" {
		t.Fatalf("expected extra file mismatch, got %v", diffs)
	}
}

func TestSplitCommand(t *testing.T) {
	tokens, err := cases.SplitCommand(`brain guard "path with spaces" 'single quote'`)
	if err != nil {
		t.Fatalf("unexpected error splitting command: %v", err)
	}
	expected := []string{"brain", "guard", "path with spaces", "single quote"}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %v", len(expected), tokens)
	}
	for i, exp := range expected {
		if tokens[i] != exp {
			t.Errorf("token %d: expected %q, got %q", i, exp, tokens[i])
		}
	}

	tokensEmpty, err := cases.SplitCommand(`brain search "" ''`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedEmpty := []string{"brain", "search", "", ""}
	if len(tokensEmpty) != len(expectedEmpty) {
		t.Fatalf("expected %d tokens for empty quotes, got %v", len(expectedEmpty), tokensEmpty)
	}
	for i, exp := range expectedEmpty {
		if tokensEmpty[i] != exp {
			t.Errorf("empty token %d: expected %q, got %q", i, exp, tokensEmpty[i])
		}
	}

	_, err = cases.SplitCommand(`brain "unclosed`)
	if err == nil {
		t.Fatal("expected error on unclosed quote, got nil")
	}
}

func TestRunCaseSubstitutesTheWorldAndComparesInProcess(t *testing.T) {
	dir := t.TempDir()
	caseDir := filepath.Join(dir, "verb", "one")
	os.MkdirAll(filepath.Join(caseDir, "world"), 0o755)
	os.WriteFile(filepath.Join(caseDir, "world", "a.txt"), []byte("at {{WORLD}}\n"), 0o644)
	os.WriteFile(filepath.Join(caseDir, "cmd"), []byte("loomux echo {{WORLD}}/a.txt\n"), 0o644)
	os.WriteFile(filepath.Join(caseDir, "exit"), []byte("0\n"), 0o644)
	os.WriteFile(filepath.Join(caseDir, "stdout"), []byte("{{WORLD}}/a.txt\n"), 0o644)
	c, err := cases.LoadCase(caseDir)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := cases.RunCase(c, func(args []string, world string, _ io.Reader, stdout, _ io.Writer) int {
		data, _ := os.ReadFile(filepath.Join(world, "a.txt"))
		if string(data) != "at "+filepath.ToSlash(world)+"\n" {
			t.Errorf("world not substituted: %q", data)
		}
		fmt.Fprintln(stdout, args[1])
		return 0
	})
	if err != nil || !outcome.Passed {
		t.Fatalf("%v %+v", err, outcome)
	}
}

func TestAMessageCaseIgnoresStdout(t *testing.T) {
	c := &cases.Case{Verb: "v", Name: "n", Path: t.TempDir(), Cmd: "loomux x", ExitCode: 2, Stdout: []byte("old words"), Compare: "message"}
	os.MkdirAll(filepath.Join(c.Path, "world"), 0o755)
	outcome, err := cases.RunCase(c, func([]string, string, io.Reader, io.Writer, io.Writer) int { return 2 })
	if err != nil || !outcome.Passed {
		t.Fatalf("%v %+v", err, outcome)
	}
}

func TestACommandThatIsNotLoomuxIsRefused(t *testing.T) {
	c := &cases.Case{Verb: "v", Name: "n", Path: t.TempDir(), Cmd: "brain guard"}
	os.MkdirAll(filepath.Join(c.Path, "world"), 0o755)
	if _, err := cases.RunCase(c, nil); err == nil {
		t.Fatal("want error")
	}
}

func TestRunCaseReportsExitAndStdoutMismatches(t *testing.T) {
	c := &cases.Case{Verb: "v", Name: "n", Path: t.TempDir(), Cmd: "loomux x", ExitCode: 0, Stdout: []byte("want"), Compare: "data"}
	os.MkdirAll(filepath.Join(c.Path, "world"), 0o755)
	outcome, err := cases.RunCase(c, func(_ []string, _ string, _ io.Reader, stdout, _ io.Writer) int {
		fmt.Fprint(stdout, "got")
		return 2
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Passed || len(outcome.Mismatches) != 2 {
		t.Fatalf("%+v", outcome.Mismatches)
	}
	if outcome.ActualExit != 2 || string(outcome.ActualStdout) != "got" {
		t.Fatalf("exit %d stdout %q", outcome.ActualExit, outcome.ActualStdout)
	}
}

// What loomux said while failing belongs in the report. The run's stderr went
// to io.Discard, so a failing case named the exit codes and nothing else -- and
// the one channel carrying the refusal that produced them was thrown away.
func TestAFailingCaseCarriesWhatLoomuxSaid(t *testing.T) {
	c := &cases.Case{Verb: "v", Name: "n", Path: t.TempDir(), Cmd: "loomux x", ExitCode: 0, Compare: "message"}
	os.MkdirAll(filepath.Join(c.Path, "world"), 0o755)
	outcome, err := cases.RunCase(c, func(_ []string, _ string, _ io.Reader, _, stderr io.Writer) int {
		fmt.Fprintln(stderr, "loomux refuses: the registry is unreadable")
		return 2
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outcome.ActualStderr), "the registry is unreadable") {
		t.Fatalf("stderr %q", outcome.ActualStderr)
	}
	report := strings.Join(outcome.Mismatches, "\n")
	if !strings.Contains(report, "the registry is unreadable") {
		t.Fatalf("the report does not carry what loomux said: %q", report)
	}
}

// A case that passes reports nothing, stderr or not: the channel is evidence
// about a failure, not a second expectation.
func TestAPassingCaseReportsNoStderr(t *testing.T) {
	c := &cases.Case{Verb: "v", Name: "n", Path: t.TempDir(), Cmd: "loomux x", ExitCode: 0, Compare: "message"}
	os.MkdirAll(filepath.Join(c.Path, "world"), 0o755)
	outcome, err := cases.RunCase(c, func(_ []string, _ string, _ io.Reader, _, stderr io.Writer) int {
		fmt.Fprintln(stderr, "a warning nobody asked about")
		return 0
	})
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Passed || len(outcome.Mismatches) != 0 {
		t.Fatalf("%+v", outcome.Mismatches)
	}
}

func TestRunCaseSubstitutesTheWorldInStdinAndComparesTheTreeAfter(t *testing.T) {
	c := &cases.Case{
		Verb: "v", Name: "n", Path: t.TempDir(), Cmd: "loomux x",
		Stdin: []byte(`{"path":"{{WORLD}}/made.txt"}`), HasWorldAfter: true, Compare: "data",
	}
	os.MkdirAll(filepath.Join(c.Path, "world"), 0o755)
	os.MkdirAll(filepath.Join(c.Path, "world_after"), 0o755)
	os.WriteFile(filepath.Join(c.Path, "world_after", "made.txt"), []byte("{{WORLD}}\n"), 0o644)
	outcome, err := cases.RunCase(c, func(_ []string, world string, stdin io.Reader, _, _ io.Writer) int {
		payload, _ := io.ReadAll(stdin)
		want := `{"path":"` + filepath.ToSlash(world) + `/made.txt"}`
		if string(payload) != want {
			t.Errorf("stdin %q, want %q", payload, want)
		}
		os.WriteFile(filepath.Join(world, "made.txt"), []byte(filepath.ToSlash(world)+"\n"), 0o644)
		return 0
	})
	if err != nil || !outcome.Passed {
		t.Fatalf("%v %+v", err, outcome)
	}
}

func TestRunCaseRefusesACommandItCannotSplit(t *testing.T) {
	c := &cases.Case{Verb: "v", Name: "n", Path: t.TempDir(), Cmd: `loomux "unclosed`}
	os.MkdirAll(filepath.Join(c.Path, "world"), 0o755)
	if _, err := cases.RunCase(c, nil); err == nil {
		t.Fatal("want error")
	}
}

func TestRunCaseRefusesAnEmptyCommand(t *testing.T) {
	c := &cases.Case{Verb: "v", Name: "n", Path: t.TempDir()}
	os.MkdirAll(filepath.Join(c.Path, "world"), 0o755)
	if _, err := cases.RunCase(c, nil); err == nil {
		t.Fatal("want error")
	}
}

func TestRunCaseReportsAnUnstageableWorld(t *testing.T) {
	c := &cases.Case{Verb: "v", Name: "n", Path: filepath.Join(t.TempDir(), "gone"), Cmd: "loomux x"}
	if _, err := cases.RunCase(c, nil); err == nil {
		t.Fatal("want error")
	}
}

func TestRunCaseReportsAnUnreadableTreeAfter(t *testing.T) {
	c := &cases.Case{Verb: "v", Name: "n", Path: t.TempDir(), Cmd: "loomux x", HasWorldAfter: true}
	os.MkdirAll(filepath.Join(c.Path, "world"), 0o755)
	locked := filepath.Join(c.Path, "world_after", "secret.txt")
	os.MkdirAll(filepath.Dir(locked), 0o755)
	os.WriteFile(locked, []byte("x"), 0o644)
	testlock.Lock(t, locked)
	if _, err := cases.RunCase(c, func([]string, string, io.Reader, io.Writer, io.Writer) int { return 0 }); err == nil {
		t.Fatal("want error")
	}
}

func TestNormalizeReplacesEverySpellingOfTheWorld(t *testing.T) {
	dir := `C:\tmp\case-1`
	data := []byte(`C:/tmp/case-1 and C:\tmp\case-1 and "C:\\tmp\\case-1"`)
	got := string(cases.Normalize(data, dir))
	want := `{{WORLD}} and {{WORLD}} and "{{WORLD}}"`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestStageWorldSubstitutesTheWorldToken(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src")
	dst := filepath.Join(t.TempDir(), "dst")
	os.MkdirAll(src, 0o755)
	writeCaseFile(t, src, "a.txt", []byte("at {{WORLD}}\n"))
	if err := cases.StageWorld(src, dst); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dst, "a.txt"))
	if err != nil || string(data) != "at "+filepath.ToSlash(dst)+"\n" {
		t.Fatalf("%v %q", err, data)
	}
}

func TestCompareTreesNormalizesTheActualTree(t *testing.T) {
	actual := filepath.Join(t.TempDir(), "actual")
	expected := filepath.Join(t.TempDir(), "expected")
	os.MkdirAll(actual, 0o755)
	os.MkdirAll(expected, 0o755)
	writeCaseFile(t, actual, "a.txt", []byte(filepath.ToSlash(actual)+"\n"))
	writeCaseFile(t, expected, "a.txt", []byte("{{WORLD}}\n"))
	diffs, err := cases.CompareTrees(actual, expected)
	if err != nil || len(diffs) != 0 {
		t.Fatalf("%v %v", err, diffs)
	}
}

func TestSplitCommandKeepsAnEscapedCharacter(t *testing.T) {
	tokens, err := cases.SplitCommand(`loomux a\ b`)
	if err != nil || len(tokens) != 2 || tokens[1] != "a b" {
		t.Fatalf("%v %q", err, tokens)
	}
	if _, err := cases.SplitCommand(`loomux a\`); err == nil {
		t.Fatal("want error on a trailing escape")
	}
}

func TestStageWorldReportsAWorldItCannotRead(t *testing.T) {
	if err := cases.StageWorld(filepath.Join(t.TempDir(), "gone"), t.TempDir()); err == nil {
		t.Fatal("want error for a missing source")
	}

	blocked := filepath.Join(t.TempDir(), "file")
	os.WriteFile(blocked, []byte("x"), 0o644)
	if err := cases.StageWorld(t.TempDir(), filepath.Join(blocked, "dst")); err == nil {
		t.Fatal("want error for a destination under a file")
	}

	src := t.TempDir()
	secret := filepath.Join(src, "secret.txt")
	os.WriteFile(secret, []byte("x"), 0o644)
	testlock.Lock(t, secret)
	if err := cases.StageWorld(src, filepath.Join(t.TempDir(), "dst")); err == nil {
		t.Fatal("want error for an unreadable source file")
	}
}

func TestCompareTreesReportsATreeItCannotRead(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "gone")
	diffs, err := cases.CompareTrees(missing, missing)
	if err != nil || len(diffs) != 0 {
		t.Fatalf("a missing tree has no files: %v %v", err, diffs)
	}

	actual := t.TempDir()
	secret := filepath.Join(actual, "secret.txt")
	os.WriteFile(secret, []byte("x"), 0o644)
	testlock.Lock(t, secret)
	if _, err := cases.CompareTrees(actual, t.TempDir()); err == nil {
		t.Fatal("want error for an unreadable actual tree")
	}
}

// A lanes case compares the exit code and the verdict per kind, never the
// bytes: the old chain printed one line per kind, loomux one per lane.
func TestALanesCaseComparesVerdictsPerKind(t *testing.T) {
	c := &cases.Case{Verb: "v", Name: "n", Path: t.TempDir(), Cmd: "loomux x", ExitCode: 1,
		Stdout: []byte("lint: ok [preset]\nfaketool: ruff\ntest: failed [preset]\n"), Compare: "lanes"}
	os.MkdirAll(filepath.Join(c.Path, "world"), 0o755)
	same := func(_ []string, _ string, _ io.Reader, stdout, _ io.Writer) int {
		fmt.Fprint(stdout, "lint/python: ok [preset] 0.1s\ntest/python: failed [preset] 1.0s\nE1\n")
		return 1
	}
	outcome, err := cases.RunCase(c, same)
	if err != nil || !outcome.Passed {
		t.Fatalf("%v %+v", err, outcome)
	}

	differs := func(_ []string, _ string, _ io.Reader, stdout, _ io.Writer) int {
		fmt.Fprint(stdout, "lint/python: ok [preset] 0.1s\ntest/python: ok [preset] 1.0s\ncoverage/python: ok [preset] 0.1s\n")
		return 1
	}
	outcome, err = cases.RunCase(c, differs)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"lanes: coverage absent != ok", "lanes: test red != ok"}
	if outcome.Passed || strings.Join(outcome.Mismatches, "|") != strings.Join(want, "|") {
		t.Fatalf("%q", outcome.Mismatches)
	}
}

func TestRunCaseComparesTheStateAStopGateLeft(t *testing.T) {
	c := &cases.Case{Verb: "v", Name: "n", Path: t.TempDir(), Cmd: "loomux x", HasWorldAfter: true, Compare: "state"}
	os.MkdirAll(filepath.Join(c.Path, "world"), 0o755)
	after := filepath.Join(c.Path, "world_after")
	os.MkdirAll(after, 0o755)
	sessions.WriteState(after, "s1", sessions.SessionState{Blocks: 1, Base: "b"})

	// The green tree is loomux's own and no mismatch; so is the extra file
	// the run leaves in the world, which a state case does not compare.
	outcome, err := cases.RunCase(c, func(_ []string, world string, _ io.Reader, _, _ io.Writer) int {
		sessions.WriteState(world, "s1", sessions.SessionState{Blocks: 1, Base: "b", Green: "t"})
		os.WriteFile(filepath.Join(world, "scratch.txt"), []byte("x"), 0o644)
		return 0
	})
	if err != nil || !outcome.Passed {
		t.Fatalf("%v %+v", err, outcome)
	}

	outcome, err = cases.RunCase(c, func(_ []string, world string, _ io.Reader, _, _ io.Writer) int {
		sessions.WriteState(world, "s1", sessions.SessionState{Blocks: 2, Base: "b"})
		return 0
	})
	if err != nil || outcome.Passed || len(outcome.Mismatches) != 1 {
		t.Fatalf("%v %q", err, outcome.Mismatches)
	}
}

func TestRunCaseRefusesAStateCaseWithoutAWorldAfter(t *testing.T) {
	c := &cases.Case{Verb: "v", Name: "n", Path: t.TempDir(), Cmd: "loomux x", Compare: "state"}
	os.MkdirAll(filepath.Join(c.Path, "world"), 0o755)
	outcome, err := cases.RunCase(c, func([]string, string, io.Reader, io.Writer, io.Writer) int { return 0 })
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Passed || strings.Join(outcome.Mismatches, "|") != "a state case needs a world_after" {
		t.Fatalf("%q", outcome.Mismatches)
	}
}

func TestRunCaseComparesWhatASubagentHookFound(t *testing.T) {
	c := &cases.Case{Verb: "v", Name: "n", Path: t.TempDir(), Cmd: "loomux x",
		Stdout: []byte("subagent a1: origin x is new at c\n"), Compare: "finding"}
	os.MkdirAll(filepath.Join(c.Path, "world"), 0o755)

	outcome, err := cases.RunCase(c, func(_ []string, world string, _ io.Reader, _, _ io.Writer) int {
		sessions.WriteAgent(world, "s1", "a1", sessions.AgentFile{Finding: []string{"origin x is new at c"}})
		return 0
	})
	if err != nil || !outcome.Passed {
		t.Fatalf("%v %+v", err, outcome)
	}

	outcome, err = cases.RunCase(c, func(_ []string, world string, _ io.Reader, _, _ io.Writer) int {
		sessions.WriteAgent(world, "s1", "a1", sessions.AgentFile{Finding: []string{"origin x moved to d"}})
		return 0
	})
	if err != nil || outcome.Passed || len(outcome.Mismatches) != 1 {
		t.Fatalf("%v %q", err, outcome.Mismatches)
	}
}

// A suite hands RunCase a RunFunc that moves the working directory into the
// staged world, and it stays there until the subtest's cleanup -- long after
// RunCase has compared. The case still has to find its own world_after, and
// it was discovered under a relative path, as every suite here discovers one.
// The assertion is that the mismatch is reported: a case that cannot find its
// expectation compares against nothing and passes on anything.
func TestACaseComparesAfterTheRunChangedTheWorkingDirectory(t *testing.T) {
	corpus := t.TempDir()
	dir := filepath.Join(corpus, "v", "n")
	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeCaseFile(t, dir, "cmd", []byte("loomux x"))
	writeCaseFile(t, dir, "exit", []byte("0"))
	writeCaseFile(t, dir, "stdout", nil)
	writeCaseFile(t, dir, "compare", []byte("state"))
	after := filepath.Join(dir, "world_after")
	if err := os.MkdirAll(after, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := sessions.WriteState(after, "s1", sessions.SessionState{Blocks: 3, Base: "b"}); err != nil {
		t.Fatal(err)
	}

	t.Chdir(corpus)
	c, err := cases.LoadCase(filepath.Join("v", "n"))
	if err != nil {
		t.Fatal(err)
	}

	outcome, err := cases.RunCase(c, func(_ []string, world string, _ io.Reader, _, _ io.Writer) int {
		t.Chdir(world)
		if err := sessions.WriteState(world, "s1", sessions.SessionState{Blocks: 0, Base: "b"}); err != nil {
			t.Error(err)
		}
		return 0
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Passed {
		t.Fatal("a state that differs by three blocks compared clean: the case no longer finds its own world_after")
	}
}
