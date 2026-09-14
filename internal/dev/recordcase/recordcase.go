// Package recordcase records a case of the corpus from one of the old
// binaries: it stages a world, runs the binary in it, and writes down the
// command, the exit code, the output and the world the run left behind.
package recordcase

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/cases"
)

// The staging directory is the one thing a test cannot take away, so the call
// is a seam.
var mkdirTemp = os.MkdirTemp

// Spec is one recording: which old binary, which command, which world.
type Spec struct {
	Exe     string // absolute path of the old binary; replaces the command's first token
	Cmd     string // command line with {{WORLD}}, first token the old name ("ulguard", "brain", "ulinit")
	World   string // directory to stage
	Stdin   string // file with the payload, may contain {{WORLD}}; "" for none
	Out     string // case directory to create
	Notes   string // text for notes.md: tag, binary, what the case shows
	Compare string // "" (data) or "message"
}

// Record runs the old binary in a staged copy of the world and writes the case.
func Record(s Spec) error {
	tmp, err := mkdirTemp("", "record-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if err := cases.StageWorld(s.World, tmp); err != nil {
		return err
	}
	world := filepath.ToSlash(tmp)
	tokens, err := cases.SplitCommand(s.Cmd)
	if err != nil {
		return err
	}
	for i := range tokens {
		tokens[i] = strings.ReplaceAll(tokens[i], cases.WorldToken, world)
	}
	var stdin []byte
	if s.Stdin != "" {
		if stdin, err = os.ReadFile(s.Stdin); err != nil {
			return err
		}
	}
	cmd := exec.Command(s.Exe, tokens[1:]...)
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), "BRAIN_STATE_DIR="+tmp)
	cmd.Stdin = bytes.NewReader(bytes.ReplaceAll(stdin, []byte(cases.WorldToken), []byte(world)))
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	exit := 0
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return fmt.Errorf("running %s: %w", s.Exe, err)
		}
		exit = exitErr.ExitCode()
	}
	files := map[string][]byte{
		"cmd":      []byte(s.Cmd + "\n"),
		"exit":     []byte(fmt.Sprintf("%d\n", exit)),
		"stdout":   cases.Normalize(stdout.Bytes(), tmp),
		"notes.md": []byte(s.Notes + "\n"),
	}
	if stdin != nil {
		files["stdin"] = stdin
	}
	if s.Compare != "" {
		files["compare"] = []byte(s.Compare + "\n")
	}
	if err := os.MkdirAll(s.Out, 0o755); err != nil {
		return err
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(s.Out, name), data, 0o644); err != nil {
			return err
		}
	}
	if err := copyTree(s.World, filepath.Join(s.Out, "world"), nil); err != nil {
		return err
	}
	return writeWorldAfter(tmp, s.World, s.Out)
}

// writeWorldAfter records the staged tree, but only where the run changed it:
// a world_after that equals the world would make every case compare a tree
// nothing touched.
//
//coverage:exempt the CompareTrees arm needs a tree that staged from a readable world and is unreadable a moment later, which only an OS intervention produces
func writeWorldAfter(tmp, world, out string) error {
	diffs, err := cases.CompareTrees(tmp, world)
	if err != nil {
		return err
	}
	if len(diffs) == 0 {
		return nil
	}
	return copyTree(tmp, filepath.Join(out, "world_after"), func(data []byte) []byte { return cases.Normalize(data, tmp) })
}

// copyTree copies src to dst, passing every file's content through transform.
//
//coverage:exempt the filepath.Rel arm needs a path WalkDir found below src that is not below src, and the WalkDir err and ReadFile arms a source the OS stops handing out after the world was staged from it
func copyTree(src, dst string, transform func([]byte) []byte) error {
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
		if transform != nil {
			data = transform(data)
		}
		return os.WriteFile(target, data, 0o644)
	})
}
