package cli

import (
	"cmp"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/config"
)

// reconcileCommand is `loomux reconcile`: it compares every registered source
// against its register entry and opens a case for every wiki page a change
// reached.
//
// Both places the pass needs come in through one lookup built here and are
// handed down; nothing deeper asks the environment. That is the promise
// `internal/serve` makes about the state directory it was given, and a lookup
// one layer down would break it for every caller that is not this command
// line. It is also why the command takes no directory of its own: the state
// model of this stage is the environment, `LOOMUX_STATE_DIR` and
// `LOOMUX_LEGACY_BRAIN_DIR`, and a flag beside it would be a second one.
//
// A case is not a failure. The pass ran to the end and the queue it produced
// is its result, so the exit code is scored by the broken case files alone --
// `lint`'s scheme, one step further: an exit that tripped over an ordinary
// finding teaches the reader to stop looking.
func reconcileCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("loomux reconcile", flag.ContinueOnError)
	flags.SetOutput(stderr)
	if err := flags.Parse(args); err != nil || refusesArguments(flags, stderr) {
		return 2
	}

	lookup := config.NewArtifactLookup()
	// The registry is read from the new place alone, the way `reindex`
	// resolves it: it is the file this machine's own registration writes, not
	// an artefact of the pass that the old directory could still hold.
	areas, err := config.ReadRegistry(lookup.Primary)
	if err != nil {
		return reportReconcileError(stderr, err)
	}
	report, root, err := catchUp(areas, lookup)
	if err != nil {
		return reportReconcileError(stderr, err)
	}

	fmt.Fprintf(stdout, "%s geprüft, %d davon gehasht\n",
		germanCount(report.Checked, "Quelle", "Quellen"), report.Hashed)
	listCases(stdout, root, report.Cases)
	fmt.Fprintln(stdout, germanCount(len(report.Cases), "Fall", "Fälle"))
	return reportUnreadable(stderr, report.Unreadable)
}

// catchUp is one reconcile pass and the review centre it landed its cases in,
// for both commands that run one: `reconcile` itself and `reindex` ahead of
// its index run.
func catchUp(areas []config.Area, lookup config.ArtifactLookup) (maintenance.Report, string, error) {
	// Before the pass and not after it, where the reference asks: the answer
	// is the same either way, because Reconcile resolves the very same review
	// centre before it scans or writes anything at all. Asked afterwards, a
	// refusal here could only come from a manifest that changed mid-run --
	// unreachable, and an error path nothing can reach is one nothing checks.
	root, err := maintenance.ReviewRoot(areas, lookup)
	if err != nil {
		return maintenance.Report{}, "", err
	}
	// UTC and not the machine's zone, as `datetime.now(UTC)` is. Three things
	// downstream carry the zone rather than fold it away: CaseID takes the
	// date into the directory name, WriteCase renders `created` through
	// pytext.IsoFormat, which prints the offset it is given, and the same goes
	// for the `.last-run` stamp. A local clock would therefore name a case
	// differently from the reference for two hours of every day.
	report, err := maintenance.Reconcile(areas, lookup, time.Now().UTC())
	if err != nil {
		return maintenance.Report{}, "", err
	}
	return report, root, nil
}

// listCases is `_list_cases`: every case of a pass on its own line, under the
// directory that addresses it. The writer is the caller's, because the same
// cases are the result of `reconcile` and a side effect of `reindex`.
func listCases(w io.Writer, root string, cases []maintenance.Case) {
	addresses := caseAddresses(root)
	// cmp.Or and not a branch: every case in the report was either just
	// written into the review centre or read back out of it, so the lookup
	// answers for all of them and the id behind it is unreachable today. It
	// stays as the value rather than as a second path, because an `else`
	// nothing can enter is an `else` nothing can check.
	for _, item := range cases {
		fmt.Fprintf(w, "  %s\n",
			caseLine(cmp.Or(addresses[caseKey{item.Area, item.Target}], item.ID), item))
	}
}

// reportUnreadable is `_report_unreadable`: every case file the pass could not
// read, named on stderr -- whoever reads stdout for the queue must never meet
// a line that is not part of it -- and the exit they are worth.
func reportUnreadable(stderr io.Writer, entries []string) int {
	for _, entry := range entries {
		fmt.Fprintf(stderr, "unreadable case: %s\n", entry)
	}
	if len(entries) > 0 {
		return 1
	}
	return 0
}

// reportReconcileError is the one shape a refusal of this command takes: a
// broken registry, a vault that declares no review centre and a pass that
// tripped over the state directory are all human errors in a file, and each
// is owed one line rather than a stack.
func reportReconcileError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "error: %v\n", err)
	return 1
}

// caseKey is what actually addresses a case in the review centre: one case per
// page per area. The id does not, because a standing case is handed back as it
// was read off the disk, hand-edited field and all.
type caseKey struct {
	area, target string
}

// caseAddresses is every case standing in the review centre, keyed by the pair
// that names it, and valued by the directory that answers to it.
//
// Two levels and not a walk of the whole tree, against the reference's
// `rglob`: `standingCase` only ever looks in `<review>/<scope>/*/case.toml`,
// so a case a reader moved deeper is already invisible to the pass that raised
// this listing -- it opened a fresh case beside it. The deeper walk is not
// without effect, though: a moved case with the same area and target as that
// fresh one competes for its key, and where its path sorts later the
// reference prints the moved directory as the new case's address. Two levels
// give the address of the case the pass just raised.
//
// Every failure is silence: a review centre before its first case does not
// exist yet, a reviewer's own file lying between the scope directories is not
// one, and a case file that does not read is reported by the pass itself --
// here it simply carries no address to offer.
func caseAddresses(root string) map[caseKey]string {
	found := map[caseKey]string{}
	scopes, err := os.ReadDir(root)
	if err != nil {
		return found
	}
	for _, scope := range scopes {
		directories, err := os.ReadDir(filepath.Join(root, scope.Name()))
		if err != nil {
			continue
		}
		for _, directory := range directories {
			read, err := maintenance.ReadCase(filepath.Join(root, scope.Name(), directory.Name(), "case.toml"))
			if err != nil {
				continue
			}
			found[caseKey{read.Area, read.Target}] = directory.Name()
		}
	}
	return found
}

// caseLine is one case as a single line, in the order a reader scans it.
//
// The address is passed in rather than read off the case, because the two can
// disagree and only one of them addresses anything.
func caseLine(address string, item maintenance.Case) string {
	line := address + "\t" + item.Area + "\t" + item.Target + "\t" + item.State
	if item.Manual {
		line += "\tmanuell"
	}
	return line
}

// germanCount is `1 Fall`, `2 Fälle`. German has no bare plural for one, and
// the same block prints two counts: `1 Quellen` beside `1 Fall` reads as a
// defect in the count rather than as grammar.
func germanCount(number int, singular, plural string) string {
	if number == 1 {
		return fmt.Sprintf("%d %s", number, singular)
	}
	return fmt.Sprintf("%d %s", number, plural)
}
