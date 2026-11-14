package serve_test

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/xidus90/loomux/internal/serve"
)

func TestDetachBuildsTheCommandFromTheSpec(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "logs", "other.log")
	var kept []*exec.Cmd
	brokeAway, err := serve.Detach(serve.DetachSpec{
		StateDir: dir,
		Args:     []string{"ui", "--foreground"},
		LogPath:  log,
		Stdin:    strings.NewReader("secret\n"),
	}, keepCmd(&kept))
	if err != nil || !brokeAway {
		t.Fatalf("Detach = %v, %v, want a start that broke away", brokeAway, err)
	}
	cmd := kept[0]
	if len(cmd.Args) != 3 || cmd.Args[1] != "ui" || cmd.Args[2] != "--foreground" {
		t.Errorf("args are %v, want ui --foreground", cmd.Args)
	}
	if file, ok := cmd.Stderr.(*os.File); !ok || file.Name() != log {
		t.Errorf("stderr is %v, want the log file %s", cmd.Stderr, log)
	}
	input, _ := io.ReadAll(cmd.Stdin)
	if string(input) != "secret\n" {
		t.Errorf("the child reads %q, want what the spec carried", input)
	}
	if cmd.Stdout != nil {
		t.Error("the child inherited stdout")
	}
}

func TestDetachHandsTheSecondAttemptTheSameInput(t *testing.T) {
	calls := 0
	var inputs []string
	_, err := serve.Detach(serve.DetachSpec{
		StateDir: t.TempDir(),
		Args:     []string{"ui", "--foreground"},
		LogPath:  filepath.Join(t.TempDir(), "x.log"),
		Stdin:    strings.NewReader("once"),
	}, func(cmd *exec.Cmd) error {
		calls++
		got, _ := io.ReadAll(cmd.Stdin)
		inputs = append(inputs, string(got))
		if calls == 1 {
			return errors.New("access is denied")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Detach: %v", err)
	}
	if len(inputs) != 2 || inputs[0] != "once" || inputs[1] != "once" {
		t.Errorf("the attempts read %q, want the bytes twice", inputs)
	}
}

func TestDetachWithoutInputLeavesStdinNil(t *testing.T) {
	var kept []*exec.Cmd
	if _, err := serve.Detach(serve.DetachSpec{
		StateDir: t.TempDir(), Args: []string{"x"}, LogPath: filepath.Join(t.TempDir(), "x.log"),
	}, keepCmd(&kept)); err != nil {
		t.Fatalf("Detach: %v", err)
	}
	if kept[0].Stdin != nil {
		t.Error("a child without input must not get one")
	}
}

func TestDetachRefusesInputThatCannotBeRead(t *testing.T) {
	_, err := serve.Detach(serve.DetachSpec{
		StateDir: t.TempDir(), Args: []string{"x"}, LogPath: filepath.Join(t.TempDir(), "x.log"),
		Stdin: iotest.ErrReader(errors.New("broken pipe")),
	}, func(*exec.Cmd) error {
		t.Error("the spawner ran without the input")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "broken pipe") {
		t.Fatalf("Detach: %v, want the failure of reading the input", err)
	}
}
