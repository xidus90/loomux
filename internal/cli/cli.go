// Package cli is the one entry point of loomux: it maps the first argument to
// a command and returns the exit code instead of exiting, so tests and the
// case suite can drive every command in-process.
package cli

import (
	"fmt"
	"io"
	"sort"
)

// Version is what `loomux --version` answers. Nothing in this branch sets it:
// the gate and the bootstrap both build with a plain `go build`, so every
// binary of stage 1a says 0.0.0-dev. A release build would overwrite it with
// -ldflags "-X github.com/xidus90/loomux/internal/cli.Version=…"; until such a
// build exists, the literal below is the whole answer.
var Version = "0.0.0-dev"

type command func(args []string, stdin io.Reader, stdout, stderr io.Writer) int

// Run dispatches one invocation and returns its exit code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "loomux %s\n", Version)
		return 0
	case "help", "--help", "-h":
		usage(stdout)
		return 0
	}
	cmd, ok := commands[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "loomux: unknown command %q\n", args[0])
		usage(stderr)
		return 2
	}
	return cmd(args[1:], stdin, stdout, stderr)
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "Usage: loomux <command> [arguments]")
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(w, "  %s\n", name)
	}
}
