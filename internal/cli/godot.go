package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/verify/godotsuite"
)

var godotRescan = godotsuite.Rescan

func godotCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "rescan" {
		fmt.Fprintln(stderr, "loomux godot: name a subcommand: rescan")
		return 2
	}
	return godotRescanCommand(args[1:], stderr)
}

// godotStatus is `git status --porcelain -- <root>`, run from the checkout.
func godotStatus(checkout string) func(string) (string, error) {
	return func(root string) (string, error) {
		r := checkStart(child.Spec{Dir: checkout, Argv: []string{"git", "status", "--porcelain", "--", root}})
		if r.Err != nil {
			return "", r.Err
		}
		if r.Code != 0 {
			return "", errors.New("git status failed: " + r.Stderr)
		}
		return r.Stdout, nil
	}
}

// godotRescanCommand refreshes the class cache of a Godot project without
// leaving its rewritten project.godot behind.
func godotRescanCommand(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("godot rescan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	godot := fs.String("godot", "", "Godot binary; found like {godot} when empty")
	rootFlag := fs.String("root", ".", "the Godot project (the directory holding project.godot)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "loomux godot rescan: unexpected argument %q\n", fs.Arg(0))
		return 2
	}
	root, checkout, _ := godotTarget(*rootFlag)
	bin, finding, err := godotBinary(*godot, root, checkout)
	if err == nil && finding != "" {
		err = errors.New(finding)
	}
	var notes []string
	if err == nil {
		notes, err = godotRescan(godotsuite.RescanOptions{Root: root, Godot: bin, Start: checkStart, Status: godotStatus(checkout)})
	}
	for _, n := range notes {
		fmt.Fprintf(stderr, "loomux godot rescan: note: %s\n", n)
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux godot rescan: %v\n", err)
		return 1
	}
	return 0
}
