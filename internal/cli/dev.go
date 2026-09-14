package cli

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/xidus90/loomux/internal/dev/covergate"
	"github.com/xidus90/loomux/internal/dev/swap"
)

const module = "github.com/xidus90/loomux"

var coverFunc = runCoverFunc

//coverage:exempt starts the go toolchain; the gate's logic is tested through coverFunc
func runCoverFunc(profile string) ([]byte, error) {
	return exec.Command("go", "tool", "cover", "-func="+profile).Output()
}

var devCommands = map[string]command{
	"covergate":   devCovergate,
	"swap-binary": devSwapBinary,
}

func devCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "loomux dev: subcommand required")
		return 2
	}
	sub, ok := devCommands[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "loomux dev: unknown subcommand %q\n", args[0])
		return 2
	}
	return sub(args[1:], stdin, stdout, stderr)
}

func devCovergate(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev covergate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	profile := fs.String("profile", "coverage.out", "coverage profile written by go test")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	out, err := coverFunc(*profile)
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev covergate: %v\n", err)
		return 1
	}
	lines, err := covergate.Parse(bytes.NewReader(out))
	if err != nil {
		fmt.Fprintf(stderr, "loomux dev covergate: %v\n", err)
		return 1
	}
	return covergate.Gate(lines, module, os.ReadFile, stdout)
}

func devSwapBinary(args []string, _ io.Reader, _, stderr io.Writer) int {
	fs := flag.NewFlagSet("dev swap-binary", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", "bin", "directory holding loomux.new.exe")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := swap.Swap(*dir); err != nil {
		fmt.Fprintf(stderr, "loomux dev swap-binary: %v\n", err)
		return 1
	}
	return 0
}
