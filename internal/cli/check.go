package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/xidus90/loomux/internal/verify"
	"github.com/xidus90/loomux/internal/verify/commit"
)

func checkCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "loomux check: subcommand required: commit-msg, gofmt")
		return 2
	}
	switch args[0] {
	case "commit-msg":
		if len(args) != 2 {
			fmt.Fprintln(stderr, "loomux check commit-msg: exactly one message file required")
			return 2
		}
		content, err := os.ReadFile(args[1])
		if err != nil {
			fmt.Fprintf(stderr, "loomux check commit-msg: %v\n", err)
			return 1
		}
		if err := commit.ValidateCommitMessage(string(content)); err != nil {
			fmt.Fprintf(stderr, "loomux check commit-msg: %v\n", err)
			return 1
		}
		return 0
	case "gofmt":
		paths := args[1:]
		if len(paths) == 0 {
			paths = []string{"."}
		}
		unformatted, err := verify.CheckGoFormat(paths)
		if err != nil {
			fmt.Fprintf(stderr, "loomux check gofmt: %v\n", err)
			return 1
		}
		for _, file := range unformatted {
			fmt.Fprintln(stdout, file)
		}
		if len(unformatted) > 0 {
			return 1
		}
		return 0
	}
	fmt.Fprintf(stderr, "loomux check: unknown subcommand %q\n", args[0])
	return 2
}
