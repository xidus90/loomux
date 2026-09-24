package cli

import (
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/hooks"
	"github.com/xidus90/loomux/internal/hosts"
)

// Exit codes when a hook call is malformed: a write barrier refuses (2);
// everything else ends with 1, which blocks nothing -- a stop gate that
// cannot read its call must not hold the turn over it.
var malformed = map[string]int{
	"pre-tool-use":   hooks.ExitDenied,
	"post-tool-use":  hooks.ExitInternal,
	"session-start":  hooks.ExitInternal,
	"stop":           hooks.ExitInternal,
	"subagent-start": hooks.ExitInternal,
	"subagent-stop":  hooks.ExitInternal,
}

// The seams a test uses to see what the flags handed on.
var (
	postToolUse = hooks.PostToolUse
	stopHook    = hooks.Stop
)

func hookCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: loomux hook <event> --host <host> [--root <dir>]")
		return 2
	}
	event := args[0]
	failure, known := malformed[event]
	if !known {
		fmt.Fprintf(stderr, "loomux hook: unknown event %q\n", event)
		return 2
	}
	flags := flag.NewFlagSet("loomux hook "+event, flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "path to the project root; found upwards when empty")
	host := flags.String("host", "", "the harness calling: claude, antigravity or codex")
	// post-edit and the stop gate run lanes, so only they have a budget.
	budget := new(time.Duration)
	switch event {
	case "post-tool-use":
		budget = flags.Duration("budget", hooks.DefaultBudget, "how long the post-edit lanes may take in all")
	case "stop":
		budget = flags.Duration("budget", hooks.DefaultStopBudget, "how long the stop gate's lanes may take in all")
	}
	if err := flags.Parse(args[1:]); err != nil {
		return failure
	}
	if *host == "" {
		fmt.Fprintf(stderr, "loomux hook %s: --host is required: expected claude, antigravity or codex\n", event)
		return failure
	}
	if _, err := hosts.ParseHost(*host); err != nil {
		fmt.Fprintf(stderr, "loomux hook %s: %v\n", event, err)
		return failure
	}
	resolved := *root
	if resolved == "" {
		found, err := hosts.FindRoot(".")
		if err != nil && event != "pre-tool-use" {
			fmt.Fprintf(stderr, "loomux hook %s: %v\n", event, err)
			return failure
		}
		// The barrier is global and needs no project; without a root it
		// judges against the registry alone.
		resolved = found
	}
	// The guard is not a module: the write barrier is global and protects
	// other repositories' read-only areas, so no project switches it off.
	if event != "pre-tool-use" {
		modules, err := config.ReadModules(resolved)
		if err != nil {
			fmt.Fprintf(stderr, "loomux hook %s: %v\n", event, err)
			return failure
		}
		if !modules.Hooks {
			return 0
		}
	}
	switch event {
	case "pre-tool-use":
		return hooks.PreToolUse(stdin, stdout, stderr, resolved, config.StateDir())
	case "post-tool-use":
		return postToolUse(stdin, stdout, stderr, resolved, *budget)
	case "stop":
		return stopHook(stdin, stderr, resolved, *host, *budget)
	case "subagent-start":
		return hooks.SubagentStart(stdin, stderr, resolved, *host)
	case "subagent-stop":
		return hooks.SubagentStop(stdin, stderr, resolved, *host)
	default:
		return hooks.SessionStart(stdin, stdout, stderr, resolved, *host)
	}
}

func statusCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("loomux status", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "path to the project root")
	if err := flags.Parse(args); err != nil {
		return hooks.ExitInternal
	}
	return hooks.Status(stdout, stderr, *root)
}

func worktreeCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: loomux worktree link|unlink [--root <dir>] | remove <worktree path>")
		return 2
	}
	// The worktree comes as an argument and not as `--root`: this one is run by
	// hand, and the caller is not standing in the directory that is about to
	// disappear. The arity is checked because that argument is the whole
	// instruction. The empty string is turned away here as well: it is a path
	// nothing can stat, so every identity check downstream answers "no" about
	// it and it would reach the wrong refusal with no path in the message.
	if args[0] == "remove" {
		if len(args) != 2 || args[1] == "" {
			fmt.Fprintln(stderr, "usage: loomux worktree remove <worktree path>")
			return hooks.ExitInternal
		}
		return hooks.WorktreeRemove(stdout, stderr, args[1])
	}
	flags := flag.NewFlagSet("loomux worktree "+args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "path to the project root")
	if err := flags.Parse(args[1:]); err != nil {
		return hooks.ExitInternal
	}
	switch args[0] {
	case "link":
		return hooks.WorktreeLink(stdout, stderr, *root)
	case "unlink":
		return hooks.WorktreeUnlink(stdout, stderr, stdin, *root)
	}
	fmt.Fprintf(stderr, "loomux worktree: unknown subcommand %q\n", args[0])
	return 2
}
