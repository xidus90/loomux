package serve

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/config"
)

// BrokeAwayEnv carries to the child what only the parent can know: whether the
// start left the parent's job object. The child writes serve.json, Spawn knows
// which attempt succeeded, so the answer has to travel.
//
// An environment variable rather than an argument, because the flags of the
// serve command belong to the command line and an unknown one would turn a
// lost breakaway into a failed start.
const BrokeAwayEnv = "LOOMUX_BROKE_AWAY"

// brokeAwayYes is the only value that means yes. Anything else -- a leftover
// "0", a "false", an empty string -- means the service is in its host's job
// object and dies with it, which is the safe reading of a value we did not
// write ourselves.
const brokeAwayYes = "1"

// executablePath is the seam for naming the running program. The failure of
// os.Executable needs a platform that cannot name its own process, and this
// package has no other way to reach that arm.
var executablePath = os.Executable

// absolutePath is the seam for making the state directory absolute.
// filepath.Abs fails only where os.Getwd does -- a working directory that was
// removed under the running process -- and nothing else in this package
// reaches that arm.
var absolutePath = filepath.Abs

// BrokeAwayFromEnv is what a serve run reads back out of what Spawn wrote.
func BrokeAwayFromEnv() bool { return os.Getenv(BrokeAwayEnv) == brokeAwayYes }

// Spawn starts a detached service and reports whether it left the caller's job
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
func Spawn(stateDir string, spawner func(cmd *exec.Cmd) error) (bool, error) {
	// Absolute before anything uses it. The child runs in this directory and
	// inherits the same path as LOOMUX_STATE_DIR; a relative one would be
	// resolved by the child against its new working directory -- which is the
	// directory itself -- and the service would write its state one level
	// deeper than the caller that is waiting for it looks.
	stateDir, err := absolutePath(stateDir)
	if err != nil {
		return false, fmt.Errorf("resolve the state directory: %w", err)
	}
	executable, err := executablePath()
	if err != nil {
		return false, fmt.Errorf("find the running program: %w", err)
	}
	// LogPath names the file; making its directory is the caller's, and on a
	// fresh machine nothing below the state directory exists yet.
	logPath := LogPath(stateDir)
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
		return false, fmt.Errorf("create %s: %w", filepath.Dir(logPath), err)
	}
	log, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return false, fmt.Errorf("open %s: %w", logPath, err)
	}
	// The child holds a descriptor of its own once it is started; this one has
	// no further use here, and holding it would keep the file open in the
	// caller for as long as the caller lives.
	defer log.Close()

	withBreakaway := spawner(command(executable, stateDir, log, true))
	if withBreakaway == nil {
		return true, nil
	}
	if err := spawner(command(executable, stateDir, log, false)); err != nil {
		return false, fmt.Errorf("start %s serve: %w", executable,
			errors.Join(err, fmt.Errorf("the attempt with breakaway: %w", withBreakaway)))
	}
	return false, nil
}

// command is one attempt's process, built and not started.
//
// The child is started with --foreground, because it is the service and not
// another starter: a bare `loomux serve` is the command that spawns, and a
// child given that would spawn a child of its own without end.
//
// Stdin and Stdout stay nil, which os/exec turns into os.DevNull. The caller is
// usually a bridge whose stdout is its host's MCP pipe: an inherited descriptor
// wrecks that pipe's framing and holds it open after the bridge exits.
// search/daemon.go does the same for qmd.
//
// Stderr is the log file itself and not a wrapper around it. os/exec builds a
// pipe and a copying goroutine for every writer that is not an *os.File, and
// only Wait reaps those; Spawn never waits.
//
// The working directory is the state directory rather than the caller's: a
// detached service that inherited a worktree's directory would hold it open,
// and on Windows that is a worktree nobody can remove afterwards. Nothing the
// service resolves comes from the working directory.
func command(executable, stateDir string, log *os.File, breakaway bool) *exec.Cmd {
	cmd := exec.Command(executable, "serve", "--foreground")
	cmd.Dir = stateDir
	cmd.Env = childEnv(os.Environ(), stateDir, breakaway)
	cmd.Stderr = log
	cmd.SysProcAttr = detachAttrs(breakaway)
	return cmd
}

// childEnv is the caller's environment with the two variables this start
// decides replaced rather than appended. os/exec does keep the last of two
// equal names, but a block that names a variable twice is a block a human
// reading it cannot judge.
//
// The comparison is case-insensitive, because Windows environment names are.
func childEnv(base []string, stateDir string, brokeAway bool) []string {
	out := make([]string, 0, len(base)+2)
	for _, entry := range base {
		if names(entry, config.StateDirEnv) || names(entry, BrokeAwayEnv) {
			continue
		}
		out = append(out, entry)
	}
	out = append(out, config.StateDirEnv+"="+stateDir)
	if brokeAway {
		out = append(out, BrokeAwayEnv+"="+brokeAwayYes)
	}
	return out
}

// names reports whether an environment entry sets this variable. An entry
// without '=' sets nothing: Windows passes such entries through its own
// bookkeeping, and they are none of our business.
func names(entry, variable string) bool {
	name, _, found := strings.Cut(entry, "=")
	return found && strings.EqualFold(name, variable)
}
