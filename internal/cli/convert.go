package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/xidus90/loomux/internal/brain/convert"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/hosts"
)

// convertTools is the seam to pdftotext and yt-dlp; the replay of the
// recorded cases answers from the world's fixture instead.
var convertTools = convert.SystemTools()

// convertCommand turns the inboxes, or one named file, into Markdown. It
// writes into an area, so it stands at the top level beside reindex; the
// exit code is 1 as soon as anything is left for a person.
func convertCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("loomux convert", flag.ContinueOnError)
	flags.SetOutput(stderr)
	free, err := parseInterspersed(flags, args)
	if err != nil {
		return 2
	}
	if len(free) > 1 {
		fmt.Fprintf(stderr, "loomux convert: unrecognized arguments: %s\n", strings.Join(free[1:], " "))
		return 2
	}
	if code, off := brainModuleOff("convert", stderr); off {
		return code
	}
	ctx := context.Background()
	var outcome convert.Outcome
	var stopped error
	if len(free) == 1 {
		target, message := convert.ConvertFile(ctx, convertTools, free[0])
		if target != "" {
			outcome.Written = []string{target}
		}
		if message != "" {
			outcome.Skipped = []string{message}
		}
	} else {
		lookup := config.NewArtifactLookup()
		areas, err := config.ReadRegistry(lookup.Primary)
		if err != nil {
			return reportReconcileError(stderr, err)
		}
		entries, err := convert.Areas(areas, lookup.Primary, lookup.Fallback)
		if err != nil {
			return reportReconcileError(stderr, err)
		}
		// An inbox that cannot be listed stops the run with what the inboxes
		// before it did; that is printed first, so nothing written goes
		// unlisted.
		outcome, stopped = convert.ConvertAll(ctx, convertTools, entries, lookup.Primary)
	}
	for _, written := range outcome.Written {
		fmt.Fprintln(stdout, written)
	}
	// Beside the written paths, not among the leftovers: a suggestion is no
	// work waiting.
	for _, suggestion := range outcome.Suggested {
		fmt.Fprintf(stdout, "suggested: %s\n", suggestion)
	}
	for _, message := range outcome.Skipped {
		fmt.Fprintf(stderr, "skipped: %s\n", message)
	}
	if stopped != nil {
		return reportReconcileError(stderr, stopped)
	}
	if len(outcome.Skipped) > 0 {
		return 1
	}
	return 0
}

// brainModuleOff refuses a command of the brain module where the project
// found above the working directory switched the module off. Outside a
// project nothing is switched off.
func brainModuleOff(name string, stderr io.Writer) (int, bool) {
	root, err := hosts.FindRoot(".")
	if err != nil {
		return 0, false
	}
	modules, err := config.ReadModules(root)
	if err != nil {
		return reportReconcileError(stderr, err), true
	}
	if modules.Brain {
		return 0, false
	}
	fmt.Fprintf(stderr, "loomux %s: the brain module is off in %s ([modules] brain = false)\n", name, config.ManifestPath(root))
	return 1, true
}
