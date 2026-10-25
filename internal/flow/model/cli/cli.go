// Package cli is the shared core of the model adapters that drive a vendor
// CLI as a child process: the process seam, the time limit and the error text.
package cli

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Command is one process to run. Env holds additions to os.Environ() only.
type Command struct {
	Name  string
	Args  []string
	Env   []string
	Dir   string
	Stdin []byte
}

// Exec runs a Command; tests replace it.
type Exec func(ctx context.Context, c Command) (stdout, stderr []byte, err error)

// Options configure run. A zero Timeout means no time limit.
type Options struct {
	Root    string
	Timeout time.Duration
	Exec    Exec
}

// stderrLines is how much of a failed process's stderr an error carries.
const stderrLines = 5

// System is the real Exec: it starts the process and feeds Stdin to it.
func System(ctx context.Context, c Command) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	cmd.Dir = c.Dir
	cmd.Env = append(os.Environ(), c.Env...)
	cmd.Stdin = bytes.NewReader(c.Stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// denial is one entry of permission_denials in claude's and qodercli's JSON.
type denial struct {
	ToolName string `json:"tool_name"`
}

// deniedError names the tools a CLI denied in headless mode, or is nil when
// it denied none.
func deniedError(cli, settings string, denied []denial) error {
	if len(denied) == 0 {
		return nil
	}
	names := make([]string, len(denied))
	for i, d := range denied {
		names[i] = cmp.Or(d.ToolName, "a tool")
	}
	return fmt.Errorf("%s denied %s in headless mode and returned no reply; allow them in %s", cli, strings.Join(names, ", "), settings)
}

// run executes c in o.Root under o.Timeout and returns stdout.
func run(ctx context.Context, o Options, c Command) ([]byte, error) {
	if o.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.Timeout)
		defer cancel()
	}
	c.Dir = o.Root
	stdout, stderr, err := o.Exec(ctx, c)
	switch {
	case err == nil:
		return stdout, nil
	case o.Timeout > 0 && errors.Is(ctx.Err(), context.DeadlineExceeded):
		return nil, fmt.Errorf("%s timed out after %s; raise [agent] timeout", c.Name, o.Timeout)
	case errors.Is(err, exec.ErrNotFound):
		return nil, fmt.Errorf("%s CLI not found on PATH", c.Name)
	}
	lines := strings.Split(strings.TrimSpace(string(stderr)), "\n")
	if len(lines) > stderrLines {
		lines = lines[:stderrLines]
	}
	if strings.TrimSpace(string(stderr)) == "" {
		return nil, fmt.Errorf("%s failed: %w", c.Name, err)
	}
	return nil, fmt.Errorf("%s failed: %w: %s", c.Name, err, strings.Join(lines, "\n"))
}
