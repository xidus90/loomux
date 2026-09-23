package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/brain/wiki"
)

var getwd = os.Getwd

// lintCommand is `brain lint`: one page when it is given a file or a name
// ending in `.md` -- the Go form of 1a, by `wiki.LintReport` -- and otherwise
// the sweep over the registered areas, `--scope all` or one of them, by the
// rules of `lint.py`.
func lintCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; the working directory when empty")
	file := fs.String("file", "", "single markdown file to lint")
	scope := fs.String("scope", "all", "the area to lint, or all")
	free, err := parseInterspersed(fs, args)
	if err != nil {
		return 2
	}
	if len(free) > 1 {
		fmt.Fprintf(stderr, "loomux lint: unrecognized arguments: %s\n", strings.Join(free[1:], " "))
		return 2
	}
	target := *file
	if target == "" && len(free) == 1 {
		target = free[0]
	}
	// Python's test: a path that is a file, or any name ending in `.md`.
	// Everything else -- nothing, a directory, a name that is neither -- is
	// the sweep.
	info, statErr := os.Stat(target)
	single := statErr == nil && info.Mode().IsRegular() || strings.HasSuffix(target, ".md")
	if !single {
		return lintSweep(*scope, stdout, stderr)
	}
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
