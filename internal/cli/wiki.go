package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/xidus90/loomux/internal/brain/wiki"
)

var getwd = os.Getwd

func lintCommand(args []string, _ io.Reader, _, stderr io.Writer) int {
	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; the working directory when empty")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "loomux lint: file path required: loomux lint [--root DIR] <file>")
		return 2
	}
	target := fs.Arg(0)
	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux lint: %v\n", err)
		return 1
	}
	wikiRoot := wiki.Root(project)
	if wikiRoot == "" {
		wikiRoot = filepath.Dir(target)
	}
	return wiki.LintReport(target, wikiRoot, stderr)
}

func wikiGateCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("wiki-gate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; the working directory when empty")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux wiki-gate: %v\n", err)
		return 1
	}
	return wiki.GateReport(project, stdout, stderr)
}

func projectRoot(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	return getwd()
}
