package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
	"github.com/xidus90/loomux/internal/selfupdate"
	"github.com/xidus90/loomux/internal/serve"
)

// The four seams of this file. Each stands for something that must not happen
// in a test: a service that binds ports and blocks, a process that outlives
// the run, a stop that reaches a real listener, and a signal registration that
// would swallow the suite's own interrupt.
var (
	serveRun    = serve.Run
	serveStatus = serve.Status
	serveStop   = serve.Stop
	serveSpawn  = serve.Spawn
	serveNotify = signal.NotifyContext
)

// startProcess is what a detached start really does. It is handed to
// serve.Spawn rather than defaulted there: internal/serve deliberately has no
// default spawner, because a nil that fell back to Start would be an arm no
// test can take without this very binary spawning itself.
var startProcess = (*exec.Cmd).Start

// serveSubcommands are the verbs of `loomux serve`, in the order the usage
// line names them.
var serveSubcommands = []string{"status", "stop"}

func serveUsage() string       { return "usage: loomux serve [--foreground]" }
func serveStatusUsage() string { return "usage: loomux serve status" }
func serveStopUsage() string   { return "usage: loomux serve stop [--force]" }

// serveCommand is `loomux serve` and its two control verbs.
//
// A first argument that does not look like an option is a subcommand, and an
// unknown one is refused rather than read as a flag of the start: `loomux
// serve stat` must say what it should have said, not start a service.
func serveCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "status":
			return serveStatusCommand(args[1:], stdout, stderr)
		case "stop":
			return serveStopCommand(args[1:], stderr)
		}
		brainRefuse(stderr, serveUsage(), "loomux serve", "argument command: invalid choice: "+
			pytext.Repr(args[0])+" (choose from "+brainChoices(serveSubcommands)+")")
		return 2
	}
	foreground, extra := onlyFlag(args, "--foreground")
	if extra != "" {
		brainRefuse(stderr, serveUsage(), "loomux serve", "unrecognized arguments: "+extra)
		return 2
	}
	if foreground {
		return serveForeground(stderr)
	}
	return serveDetached(stdout, stderr)
}

// onlyFlag reads an argument list that knows exactly one boolean flag. It
// answers whether the flag was given and what was not the flag, joined the way
// argparse joins unrecognized arguments.
func onlyFlag(args []string, name string) (bool, string) {
	given := false
	var extra []string
	for _, arg := range args {
		if arg == name {
			given = true
			continue
		}
		extra = append(extra, arg)
	}
	return given, strings.Join(extra, " ")
}

// serveForeground runs the service in this process. It is the only way a human
// sees a failed start: a detached service writes into a log file nobody opens
// until something is wrong, and by then the terminal has moved on.
//
// It is also what the detached child runs -- spawn starts `serve --foreground`
// with its stderr pointing at the log file, so the same path serves both.
//
// BrokeAway comes out of the environment, because only the parent that spawned
// this service knows whether the breakaway held; a foreground run started by a
// human has no such parent, says no, and is right: it dies with its terminal.
func serveForeground(stderr io.Writer) int {
	// Ctrl+C ends the service through Run's own shutdown rather than killing
	// the process: a killed service leaves serve.json behind naming a port
	// nobody listens on any more.
	ctx, stop := serveNotify(context.Background(), os.Interrupt)
	defer stop()
	err := serveRun(ctx, serve.Options{
		StateDir:    config.StateDir(),
		RegistryDir: config.StateDir(),
		Foreground:  true,
		BrokeAway:   serve.BrokeAwayFromEnv(),
		Update:      func(ctx context.Context) { selfUpdateRun(ctx, selfUpdateOptions(selfupdate.SourceServe)) },
	})
	if err != nil {
		fmt.Fprintf(stderr, "loomux serve: %v\n", err)
		return 1
	}
	return 0
}

// serveDetached starts the service beside this process and returns at once.
func serveDetached(stdout, stderr io.Writer) int {
	stateDir := config.StateDir()
	if alreadyRunning(stateDir) {
		fmt.Fprintf(stderr, "loomux serve: %v; run `loomux serve status`\n", serve.ErrAlreadyRunning)
		return 1
	}
	brokeAway, err := serveSpawn(stateDir, startProcess)
	if err != nil {
		fmt.Fprintf(stderr, "loomux serve: %v\n", err)
		return 1
	}
	// Where the log is, because the service's whole voice goes there: a start
	// that fails after the spawn says so in that file and nowhere else.
	fmt.Fprintf(stdout, "loomux serve started; it writes to %s\n", serve.LogPath(stateDir))
	if !brokeAway {
		fmt.Fprintln(stdout, "note: the breakaway was refused, so this service dies with its host")
	}
	return 0
}

// alreadyRunning asks serve.lock whether a service is there, by taking it as
// serve status does. It is asked before a detached start so that a second
// `loomux serve` says so here, rather than leaving the refusal in a log file
// the user has no reason to open.
//
// It is a hint and not a guarantee -- the lock is free again the moment this
// returns -- and it does not have to be one: the spawned service takes the
// lock itself and gives up if it lost the race.
//
// A lock that cannot even be opened is no answer and counts as nothing
// running: on a fresh machine the state directory does not exist yet, and
// serve.Spawn makes it on the way -- before any service runs, and after this
// question has been asked.
func alreadyRunning(stateDir string) bool {
	handle, free, err := lock.TryAcquire(serve.LockPath(stateDir))
	if err != nil {
		return false
	}
	if !free {
		return true
	}
	// Taking it was the question; keeping it would refuse the service this
	// very call is about to start.
	_ = handle.Release()
	return false
}

func serveStatusCommand(args []string, stdout, stderr io.Writer) int {
	if extra := strings.Join(args, " "); extra != "" {
		brainRefuse(stderr, serveStatusUsage(), "loomux serve status", "unrecognized arguments: "+extra)
		return 2
	}
	report, err := serveStatus(config.StateDir())
	if err != nil {
		fmt.Fprintf(stderr, "loomux serve status: %v\n", err)
		return 1
	}
	io.WriteString(stdout, report)
	return 0
}

// serveStopCommand ends the service. It writes nothing on success: a stop that
// worked is the silence the shell expects.
func serveStopCommand(args []string, stderr io.Writer) int {
	force, extra := onlyFlag(args, "--force")
	if extra != "" {
		brainRefuse(stderr, serveStopUsage(), "loomux serve stop", "unrecognized arguments: "+extra)
		return 2
	}
	if err := serveStop(config.StateDir(), force); err != nil {
		// Nothing to stop is a state, not a breakdown: the caller wanted no
		// service and there is none, and a script that stops before it starts
		// must not die of it. Two ways to have none: no state file at all, and
		// a state file that outlived its service -- serve.Run leaves it
		// behind, so the second is what a repeated stop meets.
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, serve.ErrNotRunning) {
			fmt.Fprintln(stderr, "loomux serve stop: nothing is running")
			return 0
		}
		fmt.Fprintf(stderr, "loomux serve stop: %v\n", err)
		return 1
	}
	return 0
}
