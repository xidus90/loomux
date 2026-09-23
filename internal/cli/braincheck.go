package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/xidus90/loomux/internal/brain/check"
	checkrun "github.com/xidus90/loomux/internal/brain/check/run"
	"github.com/xidus90/loomux/internal/config"
)

// brainCheckCommand is `loomux brain check file|bundle|all`, the three widths
// of the check run. Its reference is the Go binary of ultra-brain, not the
// Python form, which has no `check`; that is why it stands beside the argparse
// emulation of the other brain verbs instead of inside it, and why `code`, the
// fourth width there, is an unknown width here: the check chain of
// `loomux check` owns the code lanes.
//
// The exit code has three values. 0 is checked and clean, 1 is at least one
// error finding, and 2 is a run that did not happen -- an unknown width, a
// missing path, an unreadable registry, a scope nobody registered. A caller
// that could not tell "the bundle is broken" from "the registry could not be
// read" would report the first when it meant the second.
func brainCheckCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "error: 'loomux brain check' needs file, bundle or all")
		return 2
	}
	width := args[0]
	flags := flag.NewFlagSet("check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	notes := flags.Bool("notes", false, "include the notes")
	scope := flags.String("scope", "", "the area to check")
	free, err := parseInterspersed(flags, args[1:])
	if err != nil {
		return 2
	}

	lookup := config.NewArtifactLookup()
	var findings []check.Finding
	switch width {
	case "file":
		if len(free) == 0 {
			fmt.Fprintln(stderr, "error: 'loomux brain check file' needs a path")
			return 2
		}
		path, ok := readablePath(free[0], stderr)
		if !ok {
			return 2
		}
		findings = checkrun.CheckFile(path, lookup)
	case "bundle":
		if *scope == "" {
			fmt.Fprintln(stderr, "error: 'loomux brain check bundle' needs --scope <area>")
			return 2
		}
		// `all` is the one scope Targets widens to every area, and a bundle
		// run checks exactly one: that question is the width `all`.
		if *scope == "all" {
			fmt.Fprintln(stderr, "error: 'loomux brain check bundle' checks one area; use 'loomux brain check all'")
			return 2
		}
		named, all, ok := checkTargets(*scope, lookup, stderr)
		if !ok {
			return 2
		}
		findings = checkrun.CheckBundle(named[0], all, lookup)
	case "all":
		named, all, ok := checkTargets("all", lookup, stderr)
		if !ok {
			return 2
		}
		findings = checkrun.CheckAll(named, all, lookup)
	default:
		fmt.Fprintf(stderr, "error: unknown width %q\n", width)
		return 2
	}
	io.WriteString(stdout, check.Render(findings, *notes))
	return check.ExitCode(findings)
}

// readablePath is the absolute path of a file the run can open.
//
// Absolute, because `checkrun.CheckFile` walks the parents of its path for the
// declaration above it, and the parents of a relative path end at ".". A file
// that cannot be opened stops the run with 2 and not 0: 0 means checked and
// clean, and nothing here was read. A directory answers Stat as well, so being
// a regular file is asked separately.
func readablePath(given string, stderr io.Writer) (string, bool) {
	path, err := filepath.Abs(given)
	if err != nil {
		// On Windows Abs asks the system for the full path, which refuses a
		// name it could never open, such as one holding a NUL byte.
		fmt.Fprintf(stderr, "error: %v\n", err)
		return "", false
	}
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(stderr, "error: no such file: %s\n", path)
		return "", false
	}
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return "", false
	}
	if !info.Mode().IsRegular() {
		fmt.Fprintf(stderr, "error: %s is not a file\n", path)
		return "", false
	}
	return path, true
}

// checkTargets reads the registry and selects the areas a run covers. It
// answers both lists: the selection says which bundles are read, the whole
// registry what the federation rules judge them against.
func checkTargets(scope string, lookup config.ArtifactLookup, stderr io.Writer) (named, all []config.Area, ok bool) {
	all, err := config.ReadRegistry(lookup.Primary)
	if err == nil {
		named, err = checkrun.Targets(all, scope)
	}
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return nil, nil, false
	}
	return named, all, true
}
