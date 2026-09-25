package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os/exec"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/gitenv"
)

const mergeHookUsage = "usage: loomux merge-hook install|status|remove|record"

// nothingConsents is the answer of a run that found no line to print. The
// reference says the same in German (`_hook` in src/brain/cli.py).
const nothingConsents = "no area consents with [maintenance] on_merge = true, and no hook is installed"

// runGit runs git in dir and answers its trimmed stdout. stderr is dropped:
// record runs inside git merge, where nothing may print.
func runGit(dir string, args ...string) (string, error) {
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	// A hook exports GIT_DIR, which outranks -C; see gitenv.
	command.Env = gitenv.Environ()
	out, err := command.Output()
	return strings.TrimSpace(string(out)), err
}

// mergeHookCommands are the three subcommands that report, one line per
// repository.
var mergeHookCommands = map[string]func([]config.Area, config.ArtifactLookup, maintenance.Git) ([]maintenance.HookState, error){
	"install": maintenance.InstallHooks,
	"status":  maintenance.HookStatus,
	"remove":  maintenance.RemoveHooks,
}

// mergeHookCommand is `loomux merge-hook`, `brain-mcp hook` of the reference
// under a new name: `hook` is the namespace of the host hooks here.
//
// install, status and remove print a line per repository and fail only when
// the tool was asked to do something and did not (maintenance.HookFailed);
// an orphan or a deleted hook is a finding, not a failed run. record is what
// the hook itself calls.
func mergeHookCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "record" {
		return mergeHookRecord()
	}
	var run func([]config.Area, config.ArtifactLookup, maintenance.Git) ([]maintenance.HookState, error)
	if len(args) == 1 {
		run = mergeHookCommands[args[0]]
	}
	if run == nil {
		fmt.Fprintln(stderr, mergeHookUsage)
		return 2
	}
	// A machine that never registered an area has no registry and nothing
	// installed, as for init and record.
	areas, err := config.ReadRegistry(config.StateDir())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(stderr, "loomux merge-hook %s: %v\n", args[0], err)
		return 1
	}
	found, err := run(areas, config.NewArtifactLookup(), runGit)
	if err != nil {
		fmt.Fprintf(stderr, "loomux merge-hook %s: %v\n", args[0], err)
		return 1
	}
	if len(found) == 0 {
		fmt.Fprintln(stdout, nothingConsents)
		return 0
	}
	printHookStates(stdout, found)
	if maintenance.HookFailed(found) {
		return 1
	}
	return 0
}

// printHookStates writes one line per repository.
func printHookStates(w io.Writer, found []maintenance.HookState) {
	for _, one := range found {
		where := ""
		if one.Detail != "" {
			where = " [" + one.Detail + "]"
		}
		fmt.Fprintf(w, "%s: %s — %s%s\n", one.State, one.Scope, one.Repo, where)
	}
}

// mergeHookRecord is `loomux merge-hook record`: one event for the merge that
// just landed in the working directory (git runs a hook at the top level
// of the working tree), if an area wants it.
//
// It runs inside the user's git merge, so it says nothing and answers 0
// whatever happens -- no registry, no repository, a state directory it cannot
// write, and arguments it was not meant to get. A merge must never fail or
// print over bookkeeping. Nothing here waits: git rev-parse does not, and the
// event is appended without a lock.
func mergeHookRecord() int {
	areas, err := config.ReadRegistry(config.StateDir())
	if err != nil {
		return 0
	}
	_, _ = maintenance.RecordMerge(".", areas, config.NewArtifactLookup(), runGit, time.Now())
	return 0
}
