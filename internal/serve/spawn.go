package serve

import (
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

// Spawn starts a detached `loomux serve --foreground` and reports whether it
// left the caller's job object; Detach says how.
func Spawn(stateDir string, spawner func(cmd *exec.Cmd) error) (bool, error) {
	// LogPath is taken from the absolute directory: Detach resolves the same
	// one for the child, and a log under a relative one would be the caller's.
	abs, err := absolutePath(stateDir)
	if err != nil {
		return false, fmt.Errorf("resolve the state directory: %w", err)
	}
	return Detach(DetachSpec{StateDir: abs, Args: []string{"serve", "--foreground"}, LogPath: LogPath(abs)}, spawner)
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
