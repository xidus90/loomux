package cli

import (
	"fmt"
	"io"

	"github.com/xidus90/loomux/internal/brain/answer"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// brainPorts are the engines the command line asks. It is the seam the tests
// of this package replace, and it stays unexported: a test of another caller
// brings its own ports to answer.RunWith and never reaches in here. The value
// is a function, not a call: nothing is built before a brain command runs.
var brainPorts = answer.DefaultPorts

// brainCommand is `loomux brain`: the five read commands of brain-mcp
// (cli.py:712-810) with their argument forms, exit codes and error line.
func brainCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		brainRefuse(stderr, brainTopUsage(), "loomux brain", "the following arguments are required: command")
		return 2
	}
	// Before the argparse emulation and not among its verbs: `check` comes
	// from the Go binary and speaks its own usage, and adding it to the
	// choices would change what the other five answer to an unknown verb.
	if args[0] == "check" {
		return brainCheckCommand(args[1:], stdout, stderr)
	}
	parser, known := brainParserFor(args[0])
	if !known {
		brainRefuse(stderr, brainTopUsage(), "loomux brain",
			"argument command: invalid choice: "+pytext.Repr(args[0])+" (choose from "+brainChoices(brainSubcommands)+")")
		return 2
	}
	parsed, refused := parser.parse(args[1:])
	if refused != nil {
		if refused.top {
			brainRefuse(stderr, brainTopUsage(), "loomux brain", refused.message)
		} else {
			brainRefuse(stderr, parser.usage(), "loomux brain "+parser.name, refused.message)
		}
		return 2
	}
	// The parser let through only the two channel names, so the conversion
	// cannot produce a third; and only the five command names, so the answer
	// never has to refuse one here.
	req := answer.Request{
		Command: parser.name,
		Query:   parsed.positional,
		Scope:   parsed.values["--scope"],
		Profile: parsed.values["--profile"],
		Count:   parsed.n,
		Section: parsed.values["--section"],
		Channel: privacy.Channel(parsed.values["--channel"]),
	}
	registryDir := config.StateDir()
	// Nothing reaches stdout before the answer stands: a failure after half an
	// answer would leave the reader holding lines that look complete.
	out, notes, err := answer.RunWith(brainPorts(), req, registryDir, func(message string) {
		fmt.Fprintf(stderr, "note: %s\n", message)
	})
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	io.WriteString(stdout, out)
	for _, note := range notes {
		fmt.Fprintf(stderr, "note: %s\n", note)
	}
	return 0
}

// brainRefuse writes argparse's usage error: the usage line, then which parser
// refused and why.
func brainRefuse(stderr io.Writer, usage, prog, message string) {
	fmt.Fprintf(stderr, "%s\n%s: error: %s\n", usage, prog, message)
}
