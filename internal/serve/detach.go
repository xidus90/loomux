package serve

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DetachSpec is one detached start of this program.
type DetachSpec struct {
	// StateDir is the child's working directory and its LOOMUX_STATE_DIR.
	StateDir string
	// Args is the command line behind the program, for example serve
	// --foreground. The child must be the foreground form of its command: a
	// child that spawned again would never end.
	Args []string
	// LogPath is the file the child's stderr goes to. Its directory is made.
	LogPath string
	// Stdin is what the child reads, nil for nothing. It is read once, up
	// front, so a second attempt hands the child the same bytes.
	Stdin io.Reader
	// Console gives the child a console without a window on Windows, which
	// every console program it starts shares. Without it the child has none,
	// and Windows opens a visible one for each such program.
	Console bool
}

// Detach starts a detached child and reports whether it left the caller's job
// object. It builds the command, the spawner starts it -- usually
// (*exec.Cmd).Start, in a test something that starts nothing and only looks.
//
// There is deliberately no default spawner: a nil that fell back to Start
// would be an arm no test can take without this very binary spawning itself.
//
// The first attempt asks to break away. Any failure of it is answered with a
// second attempt without that request, and the two cannot be told apart by the
// error: a refused breakaway arrives as a plain access denial, and the error a
// spawner returns is whatever the platform gave it. So a genuinely unstartable
// program is tried twice and both failures are reported. That is the cheaper
// mistake -- the alternative is matching on error text, which would silently
// stop working in another language or another Windows build. Both errors,
// because only the first one can carry the evidence that the breakaway was
// what was refused, and whether a host puts us in a job object is this stage's
// one unverified assumption.
func Detach(spec DetachSpec, spawner func(cmd *exec.Cmd) error) (bool, error) {
	// Absolute before anything uses it. The child runs in this directory and
	// inherits the same path as LOOMUX_STATE_DIR; a relative one would be
	// resolved by the child against its new working directory -- which is the
	// directory itself -- and the service would write its state one level
	// deeper than the caller that is waiting for it looks.
	stateDir, err := absolutePath(spec.StateDir)
	if err != nil {
		return false, fmt.Errorf("resolve the state directory: %w", err)
	}
	executable, err := executablePath()
	if err != nil {
		return false, fmt.Errorf("find the running program: %w", err)
	}
	var stdin []byte
	if spec.Stdin != nil {
		if stdin, err = io.ReadAll(spec.Stdin); err != nil {
			return false, fmt.Errorf("read the child's input: %w", err)
		}
	}
	// LogPath names the file; making its directory is the caller's, and on a
	// fresh machine nothing below the state directory exists yet.
	if err := os.MkdirAll(filepath.Dir(spec.LogPath), 0o700); err != nil {
		return false, fmt.Errorf("create %s: %w", filepath.Dir(spec.LogPath), err)
	}
	log, err := os.OpenFile(spec.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return false, fmt.Errorf("open %s: %w", spec.LogPath, err)
	}
	// The child holds a descriptor of its own once it is started; this one has
	// no further use here, and holding it would keep the file open in the
	// caller for as long as the caller lives.
	defer log.Close()

	attempt := func(breakaway bool) error {
		return spawner(command(executable, stateDir, spec, stdin, log, breakaway))
	}
	withBreakaway := attempt(true)
	if withBreakaway == nil {
		return true, nil
	}
	if err := attempt(false); err != nil {
		return false, fmt.Errorf("start %s %s: %w", executable, strings.Join(spec.Args, " "),
			errors.Join(err, fmt.Errorf("the attempt with breakaway: %w", withBreakaway)))
	}
	return false, nil
}

// command is one attempt's process, built and not started.
//
// The child is the foreground form of its command, because it is the service
// and not another starter: a bare `loomux serve` is the command that spawns,
// and a child given that would spawn a child of its own without end.
//
// Stdout stays nil, which os/exec turns into os.DevNull. The caller is usually
// a bridge whose stdout is its host's MCP pipe: an inherited descriptor wrecks
// that pipe's framing and holds it open after the bridge exits.
// search/daemon.go does the same for qmd. Stdin is nil for the same reason
// unless the caller hands the child something, and then it is that and only
// that, never the caller's own.
//
// Stderr is the log file itself and not a wrapper around it. os/exec builds a
// pipe and a copying goroutine for every writer that is not an *os.File, and
// only Wait reaps those; Detach never waits. Stdin, when there is one, is a
// reader that is no *os.File either and gets the same pipe and goroutine; its
// copy ends with the bytes it was given.
//
// The working directory is the state directory rather than the caller's: a
// detached service that inherited a worktree's directory would hold it open,
// and on Windows that is a worktree nobody can remove afterwards. Nothing the
// service resolves comes from the working directory.
func command(executable, stateDir string, spec DetachSpec, stdin []byte, log *os.File, breakaway bool) *exec.Cmd {
	cmd := exec.Command(executable, spec.Args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	cmd.Dir = stateDir
	cmd.Env = childEnv(os.Environ(), stateDir, breakaway)
	cmd.Stderr = log
	cmd.SysProcAttr = detachAttrs(breakaway, spec.Console)
	return cmd
}
