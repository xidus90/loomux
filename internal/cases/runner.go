package cases

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/shellwords"
)

// WorldToken stands for the staged world directory in a case's files.
const WorldToken = "{{WORLD}}"

// RunFunc is the entry point under test: the command's arguments without the
// program name, the staged world, and the three streams.
type RunFunc func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int

// The staging directory is the one thing a test cannot take away from the
// runner, so the call is a seam.
var defaultMkdirTemp = os.MkdirTemp

var mkdirTemp = defaultMkdirTemp

// Normalize replaces every spelling of dir -- slashed, native and
// JSON-escaped -- with WorldToken, so a recording matches on any machine.
func Normalize(data []byte, dir string) []byte {
	token := []byte(WorldToken)
	data = bytes.ReplaceAll(data, []byte(strings.ReplaceAll(dir, `\`, `\\`)), token)
	data = bytes.ReplaceAll(data, []byte(dir), token)
	return bytes.ReplaceAll(data, []byte(filepath.ToSlash(dir)), token)
}

// RunOutcome describes the outcome of running a single case.
type RunOutcome struct {
	Case         *Case
	Passed       bool
	ActualExit   int
	ActualStdout []byte
	// What the run said while it was failing. No case compares it -- the
	// corpus pins exit codes and stdout -- but a failing case that named only
	// the codes left the one channel carrying the refusal unread.
	ActualStderr []byte
	Mismatches   []string
}

// StageWorld copies the directory tree from src into dst, putting the staged
// path in place of WorldToken in every file's content.
//
//coverage:exempt the filepath.Rel arm needs a path WalkDir found below src that is not below src, which no filesystem produces
func StageWorld(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		data = bytes.ReplaceAll(data, []byte(WorldToken), []byte(filepath.ToSlash(dst)))
		return os.WriteFile(target, data, 0o644)
	})
}

// CompareTrees compares regular files between actualDir and expectedDir. The
// actual contents are normalised first: the expected tree keeps WorldToken.
func CompareTrees(actualDir, expectedDir string) ([]string, error) {
	mismatches, _, err := compareTrees(actualDir, expectedDir, "", nil, [2][]byte{})
	return mismatches, err
}

// stdoutKey carries a side's stdout through the normalizer beside its tree. No
// path a walk yields contains a NUL, so it can meet no file.
const stdoutKey = "\x00stdout"

// compareTrees is CompareTrees with a normalizer run over both sides, fed the
// recorded world in worldDir. Without one, worldDir is not read.
//
// stdouts are the actual and the recorded stdout. They go through the
// normalizer with their tree and come back normalized: the day a case id
// carries stands on stdout too, and only the tree beside it knows which day
// the run's own is.
func compareTrees(actualDir, expectedDir, worldDir string, normalize Normalizer, stdouts [2][]byte) ([]string, [2][]byte, error) {
	actualFiles, err := collectFiles(actualDir)
	if err != nil {
		return nil, stdouts, err
	}
	for rel, data := range actualFiles {
		actualFiles[rel] = Normalize(data, actualDir)
	}
	expectedFiles, err := collectFiles(expectedDir)
	if err != nil {
		return nil, stdouts, err
	}
	if normalize != nil {
		world, err := collectFiles(worldDir)
		if err != nil {
			return nil, stdouts, err
		}
		actualFiles[stdoutKey], expectedFiles[stdoutKey] = stdouts[0], stdouts[1]
		actualFiles = normalize(world, actualFiles)
		expectedFiles = normalize(world, expectedFiles)
		stdouts = [2][]byte{actualFiles[stdoutKey], expectedFiles[stdoutKey]}
		delete(actualFiles, stdoutKey)
		delete(expectedFiles, stdoutKey)
	}

	var mismatches []string
	for rel, expectedContent := range expectedFiles {
		actualContent, ok := actualFiles[rel]
		if !ok {
			mismatches = append(mismatches, "missing file in actual: "+rel)
		} else if !bytes.Equal(actualContent, expectedContent) {
			mismatches = append(mismatches, "content mismatch: "+rel)
		}
	}
	for rel := range actualFiles {
		if _, ok := expectedFiles[rel]; !ok {
			mismatches = append(mismatches, "unexpected extra file in actual: "+rel)
		}
	}
	sort.Strings(mismatches)
	return mismatches, stdouts, nil
}

//coverage:exempt the filepath.Rel arm needs a path below dir that is not below dir, and the remaining WalkDir err arm a directory the OS refuses to list while its parent reads
func collectFiles(dir string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		// The repositories a git world is built into are the bench, not the
		// world: git rewrites its index on a read, which is no change anybody
		// made.
		if rel, _ := filepath.Rel(dir, path); InfraPath(filepath.ToSlash(rel)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = data
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// SplitCommand splits a recorded command line; the rules live in shellwords
// because verify splits configured commands the same way.
func SplitCommand(s string) ([]string, error) { return shellwords.Split(s) }

// RunCase runs a single case in process, in an isolated staged world.
func RunCase(c *Case, run RunFunc) (*RunOutcome, error) {
	return RunCaseWith(c, run, nil)
}

// RunCaseWith is RunCase with a world comparison that normalizes both sides
// first, and that compares a case without a world_after against its world. A
// nil normalizer compares as RunCase does.
func RunCaseWith(c *Case, run RunFunc, normalize Normalizer) (*RunOutcome, error) {
	tmpDir, err := mkdirTemp("", "case-run-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	srcWorld := filepath.Join(c.Path, "world")
	if err := StageWorld(srcWorld, tmpDir); err != nil {
		return nil, err
	}
	// A world may declare a repository; it is built after staging, so the
	// SHAs are the same every time and {{COMMIT:<n>}} can stand for them.
	repo, err := BuildGitWorldAt(tmpDir)
	if err != nil {
		return nil, err
	}
	world := filepath.ToSlash(tmpDir)

	tokens, err := SplitCommand(c.Cmd)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty command in case %s/%s", c.Verb, c.Name)
	}
	// The corpus belongs to one binary; anything else is a recording that was
	// never translated.
	if tokens[0] != "loomux" {
		return nil, fmt.Errorf("case %s/%s does not call loomux: %s", c.Verb, c.Name, c.Cmd)
	}
	for i := range tokens {
		tokens[i] = strings.ReplaceAll(tokens[i], WorldToken, world)
	}

	stdin := bytes.ReplaceAll(c.Stdin, []byte(WorldToken), []byte(world))
	var stdoutBuf, stderrBuf bytes.Buffer
	actualExit := run(tokens[1:], tmpDir, bytes.NewReader(stdin), &stdoutBuf, &stderrBuf)
	actualStdout := Normalize(stdoutBuf.Bytes(), tmpDir)
	actualStderr := Normalize(stderrBuf.Bytes(), tmpDir)

	// With a normalizer the whole world is compared: a case without a
	// world_after is held against the world it started from. The recorder
	// writes a world_after only where the run changed the world, so its
	// absence is the recording's claim that nothing changed -- and a refusal
	// that writes after all is exactly what such a case exists to catch.
	expected := filepath.Join(c.Path, "world_after")
	if !c.HasWorldAfter {
		expected = srcWorld
	}
	// A commit is compared only where the case asks for it: the git cases
	// recorded before git.after existed hold none, and would find one extra.
	if repo != "" {
		rel, _ := filepath.Rel(tmpDir, repo)
		if _, err := os.Stat(filepath.Join(expected, rel, GitAfterName)); err == nil {
			if err := WriteGitAfter(repo); err != nil {
				return nil, err
			}
		}
	}
	// A state and a finding case read loomux's state alone, so no tree
	// comparison runs for them.
	var diffs []string
	stdouts := [2][]byte{actualStdout, c.Stdout}
	treeCompared := c.Compare != "state" && c.Compare != "finding"
	if treeCompared && (c.HasWorldAfter || normalize != nil) {
		diffs, stdouts, err = compareTrees(tmpDir, expected, srcWorld, normalize, stdouts)
		if err != nil {
			return nil, err
		}
	}

	var mismatches []string
	if actualExit != c.ExitCode {
		mismatches = append(mismatches, fmt.Sprintf("exit code: expected %d, got %d", c.ExitCode, actualExit))
	}
	// A message case pins the exit code alone: its wording is loomux's own. A
	// lanes case pins the verdict per kind. A state case pins what a stop
	// gate decided, a finding case what a subagent hook found; both read
	// loomux's state, and the rest of the tree is theirs to differ in.
	switch c.Compare {
	case "lanes":
		mismatches = append(mismatches, compareLanes(c.Stdout, actualStdout)...)
	case "state":
		// Always against world_after: a git world has its commit tokens
		// replaced at every staging, so the recorder always wrote one, and
		// the world itself holds tokens where the SHAs belong.
		if !c.HasWorldAfter {
			mismatches = append(mismatches, "a state case needs a world_after")
			break
		}
		mismatches = append(mismatches, compareState(filepath.Join(c.Path, "world_after"), tmpDir)...)
	case "finding":
		mismatches = append(mismatches, compareFindings(c.Stdout, tmpDir)...)
	case "message":
	default:
		if !bytes.Equal(stdouts[0], stdouts[1]) {
			mismatches = append(mismatches, fmt.Sprintf("stdout mismatch: expected %d bytes, got %d bytes", len(c.Stdout), len(actualStdout)))
		}
	}
	mismatches = append(mismatches, diffs...)

	// Last, and only where something is wrong: a run that matched has nothing
	// to explain, and a warning on stderr is no mismatch.
	if len(mismatches) > 0 && len(actualStderr) > 0 {
		mismatches = append(mismatches, "stderr:\n"+string(actualStderr))
	}

	return &RunOutcome{
		Case:         c,
		Passed:       len(mismatches) == 0,
		ActualExit:   actualExit,
		ActualStdout: actualStdout,
		ActualStderr: actualStderr,
		Mismatches:   mismatches,
	}, nil
}
