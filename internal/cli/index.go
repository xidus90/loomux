package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/brain/index"
	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/config"
)

// qmdInstallCommand is how qmd gets onto a machine. It is named whenever the
// binary is missing, because a command that only says "not found" leaves the
// reader to search for a package name this program already knows.
const qmdInstallCommand = "npm install -g @tobilu/qmd"

// indexLook is the PATH lookup of the two index commands. It is a seam: a
// test has no qmd to install and no way to uninstall the one that may lie on
// the machine running it.
var indexLook = exec.LookPath

// reindexCommand is `loomux reindex`: it rebuilds the catalogs, the link
// graph and the identity register of every registered area and tells qmd
// which collections exist -- after a reconcile pass over the same areas, for
// the reason catchUpBeforeIndexing gives.
//
// Both places the run needs are resolved here and handed in: the state
// directory is the one thing written to, the legacy directory the one a
// read-only area's stock and the record of qmd collections are still read
// from until `loomux migrate` moves them.
// This is the composition site, and nothing deeper asks the environment --
// `internal/serve` promises that everything hangs off the state directory it
// was handed, and a lookup one layer down would break that promise for every
// caller that is not this command line.
func reindexCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("loomux reindex", flag.ContinueOnError)
	flags.SetOutput(stderr)
	registry := flags.String("registry", "", "the registry to read; the one in the state directory when empty")
	if err := flags.Parse(args); err != nil || refusesArguments(flags, stderr) {
		return 2
	}

	lookup := config.NewArtifactLookup()
	path, named := registryPath(*registry, lookup.Primary)
	if silent, code := reportEmptyState(path, named, "reindex", "index", stdout, stderr); silent {
		return code
	}

	// The registry this run is about to walk, and not the registered one:
	// `--registry` may name another, and a pass over a different set of areas
	// would assure nothing about the sources being indexed. registryPath
	// always ends in the file's own name, so its directory is the one
	// ReadRegistry wants. Outside the catch-up's refusal, as in the
	// reference: a registry that does not read is no failed catch-up but a
	// run that never had anything to walk.
	areas, err := config.ReadRegistry(filepath.Dir(path))
	if err != nil {
		return reportReconcileError(stderr, err)
	}
	if !catchUpBeforeIndexing(areas, lookup, stderr) {
		return 1
	}

	return indexAreas(path, lookup, stdout, stderr)
}

// indexAreas is the index run itself, without the catch-up in front of it:
// reindexCommand runs its own first, and `approve` runs one in its own words
// before it calls this, so neither passes over the vault twice.
func indexAreas(path string, lookup config.ArtifactLookup, stdout, stderr io.Writer) int {
	code, _ := index.ReindexWithOutput(path, lookup.Primary, lookup.Fallback, brainPorts().Status(), stderr)
	if code == 0 {
		fmt.Fprintf(stdout, "indexed the areas of %s\n", path)
	}
	return code
}

// catchUpBeforeIndexing runs a reconcile pass ahead of the index run and says
// whether the run may go on.
//
// It runs first and it is not optional: the index run writes a new identity
// for every source whose hash moved, so a source changed since the last pass
// would have its hash advanced here without ever becoming a case, and the next
// `loomux reconcile` would find it identical to its register -- a knowledge
// change past the review gate with nobody doing anything wrong.
//
// It is no gate of its own, though, and two of its three outcomes go on:
//
//   - A green pass. Its cases go to stderr, because stdout carries what was
//     asked for and these are a side effect of it; a broken case file is named
//     the same way and, unlike under `reconcile`, scores nothing -- the
//     reference drops that exit too.
//   - No review centre. There is no gate to walk around, and refusing to index
//     would tie this command to a declaration nothing else in the index run
//     reads. It warns all the same, because the state looks exactly like a
//     declaring vault whose manifest is merely out of reach.
//   - Any other failure stops the command. The index run has not started, so
//     stopping loses nothing, and indexing past a failed pass is the very
//     swallow the pass is there to prevent.
//
// Both texts are German and word for word the reference's: they are part of
// the behaviour, not a translation of it.
func catchUpBeforeIndexing(areas []config.Area, lookup config.ArtifactLookup, stderr io.Writer) bool {
	report, root, err := catchUp(areas, lookup)
	if errors.Is(err, maintenance.ErrNoReviewCentre) {
		fmt.Fprintln(stderr, "warning: kein registrierter Bereich erklärt ein Prüfzentrum "+
			"([layout] review) -- es wurde nicht abgeglichen. Der Indexlauf schreibt "+
			"geänderte Quellen ins Register, ohne dass ein Prüffall entsteht. Fehlt das "+
			"Manifest des erklärenden Bereichs nur (etwa weil sein Laufwerk nicht "+
			"eingehängt ist), sind diese Änderungen danach nicht mehr auffindbar.")
		return true
	}
	if err != nil {
		fmt.Fprintf(stderr, "error: die Aufholung vor dem Indexlauf ist fehlgeschlagen (%v). Es wurde "+
			"nicht indiziert: ein Indexlauf ohne vorherigen Abgleich schreibt geänderte "+
			"Quellen ins Register, ohne dass je ein Prüffall entsteht. Ursache beheben, "+
			"dann `loomux reconcile` und danach `loomux reindex`.\n", err)
		return false
	}
	reportCatchUp(stderr, root, report)
	return true
}

// reportCatchUp names what a green catch-up found, on stderr: the cases it
// opened under one heading, then every case file it could not read. Neither
// scores anything -- the reference drops that exit on both paths that catch
// up, `reindex` and the update after an approval.
func reportCatchUp(stderr io.Writer, root string, report maintenance.Report) {
	if len(report.Cases) > 0 {
		fmt.Fprintf(stderr, "warning: %s durch die Aufholung eröffnet:\n",
			germanCount(len(report.Cases), "neuer Fall", "neue Fälle"))
		listCases(stderr, root, report.Cases)
	}
	reportUnreadable(stderr, report.Unreadable)
}

// embedCommand is `loomux embed`: it asks the search engine for the vectors
// the index run leaves pending.
//
// The engine is asked before the registry is read. A machine without qmd
// cannot embed anything, whatever the registry says, and the reader is owed
// the name of the missing program rather than a complaint about a file that
// was never the problem.
func embedCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("loomux embed", flag.ContinueOnError)
	flags.SetOutput(stderr)
	registry := flags.String("registry", "", "the registry to read; the one in the state directory when empty")
	if err := flags.Parse(args); err != nil || refusesArguments(flags, stderr) {
		return 2
	}

	if _, err := indexLook("qmd"); err != nil {
		fmt.Fprintf(stderr, "loomux embed: qmd is not on PATH; install it with: %s\n", qmdInstallCommand)
		return 1
	}

	stateDir := config.StateDir()
	path, named := registryPath(*registry, stateDir)
	if silent, code := reportEmptyState(path, named, "embed", "embed", stdout, stderr); silent {
		return code
	}

	code, _ := index.EmbedWithOutput(path, stateDir, brainPorts().Status(), stderr)
	return code
}

// refusesArguments turns away what is left on the command line once the flags
// are read, the way argparse does: exit 2 and the words it did not know.
//
// None of the commands that call it takes a positional, and the flag package
// would otherwise keep the rest and let the command run on -- against the state
// the environment names, which is the real one of the machine whenever the
// caller meant the word as a directory. That is how a review once ended up
// writing into the live state directory with `loomux reconcile somewhere`.
func refusesArguments(flags *flag.FlagSet, stderr io.Writer) bool {
	if flags.NArg() == 0 {
		return false
	}
	fmt.Fprintf(stderr, "%s: unrecognized arguments: %s\n", flags.Name(), strings.Join(flags.Args(), " "))
	return true
}

// registryPath is the file the run will read, and whether the caller named it
// themselves. It mirrors what index.Reindex would resolve, so the check below
// asks about the very file the run would open.
func registryPath(named, stateDir string) (string, bool) {
	if named == "" {
		return filepath.Join(stateDir, "registry.toml"), false
	}
	if filepath.Base(named) == "registry.toml" {
		return named, true
	}
	return filepath.Join(named, "registry.toml"), true
}

// reportEmptyState decides whether there is anything to do at all.
//
// A registry that is simply not there is the state of a machine on which no
// area has been registered yet: nothing to index, and saying so is the whole
// answer. A registry the caller named by hand and that is missing is a typo
// instead, and a registry that exists is the run's business -- both go on and
// end red down there if they must.
func reportEmptyState(path string, named bool, name, verb string, stdout, stderr io.Writer) (bool, int) {
	if _, err := os.Stat(path); err == nil || !errors.Is(err, fs.ErrNotExist) {
		return false, 0
	}
	if named {
		fmt.Fprintf(stderr, "loomux %s: %s does not exist\n", name, path)
		return true, 1
	}
	fmt.Fprintf(stdout, "no areas registered in %s; nothing to %s\n", path, verb)
	return true, 0
}
