package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/apply"
	"github.com/xidus90/loomux/internal/brain/maintenance"
)

// Seams: a test decides without a vault that proves a proposal, and names
// the account without depending on who runs the suite.
var (
	approveCase = apply.Approve
	currentUser = user.Current
)

// approveCommand is `loomux approve <id> [--amend PATH | --reject | --defer]`
// (`_approve`, cli.py:1334-1387): find the case, hand the decision to apply,
// report it.
//
// The three flags exclude one another because each names one decision, and
// picking one of two silently would decide for the reviewer at the one gate
// that exists so a human decides.
func approveCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("loomux approve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	amend := flags.String("amend", "", "approve this file instead")
	reject := flags.Bool("reject", false, "discard the proposal")
	postpone := flags.Bool("defer", false, "leave the case for later")
	free, err := parseInterspersed(flags, args)
	if err != nil {
		return 2
	}
	switch {
	case len(free) == 0:
		fmt.Fprintf(stderr, "%s: the following arguments are required: id\n", flags.Name())
		return 2
	case len(free) > 1:
		fmt.Fprintf(stderr, "%s: unrecognized arguments: %s\n", flags.Name(), strings.Join(free[1:], " "))
		return 2
	}
	if chosen := decisionsGiven(flags, args); len(chosen) > 1 {
		fmt.Fprintf(stderr, "%s: argument --%s: not allowed with argument --%s\n", flags.Name(), chosen[1], chosen[0])
		return 2
	}

	vault, root, err := reviewCentre()
	if err != nil {
		return reportReconcileError(stderr, err)
	}
	directory, err := maintenance.FindCase(root, free[0])
	if err != nil {
		return reportReconcileError(stderr, err)
	}
	if *postpone {
		return deferCase(stdout, stderr, directory)
	}
	reviewer, err := reviewerName()
	if err != nil {
		return reportReconcileError(stderr, err)
	}
	decision := "approve"
	if *reject {
		decision = "reject"
	}
	// `--amend=` names a file too: Python reads it as `Path("")`, which is
	// `.`, and apply refuses to read a directory. Left empty it would mean
	// "no amendment" and approve the case's own proposal instead.
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "amend" && *amend == "" {
			*amend = "."
		}
	})
	result, err := approveCase(filepath.Join(directory, "case.toml"), vault.areas, apply.Options{
		Amend:    *amend,
		Decision: decision,
		Reviewer: reviewer,
		Now:      time.Now().UTC(),
		// The directory vcs makes its private scratch indexes in; shared by
		// every vault, which is why each commit gets an index of its own.
		Scratch: filepath.Join(vault.lookup.Primary, "maintenance"),
		Lookup:  vault.lookup,
	})
	if err != nil {
		reportAbort(stderr, err)
		return 1
	}
	reportDecision(stdout, stderr, vault, result)
	return 0
}

// decisionsGiven is the decision flags the command line set, in the order it
// names them, so a collision is reported as argparse reports it: the later
// flag is "not allowed with" the earlier one. A switch set to false decides
// nothing and collides with nothing.
func decisionsGiven(flags *flag.FlagSet, args []string) []string {
	set := map[string]bool{}
	flags.Visit(func(f *flag.Flag) {
		set[f.Name] = f.Name == "amend" || f.Value.String() == "true"
	})
	var chosen []string
	for _, arg := range args {
		if arg == "--" {
			break
		}
		name, _, _ := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		if strings.HasPrefix(arg, "-") && set[name] {
			chosen = append(chosen, name)
			set[name] = false
		}
	}
	return chosen
}

// deferCase puts a case off, which is the absence of a decision: apply knows
// approve and reject, and `case.toml` has no deferred state to write. The file
// is read all the same, so a broken one is named now rather than at the next
// pass, and the line names the directory, the key `case` answers to.
func deferCase(stdout, stderr io.Writer, directory string) int {
	read, err := maintenance.ReadCase(filepath.Join(directory, "case.toml"))
	if err != nil {
		return reportReconcileError(stderr, err)
	}
	warnRenamed(stderr, directory, read)
	fmt.Fprintf(stdout, "Fall %s zurückgestellt; er bleibt unverändert in der Warteschlange.\n", filepath.Base(directory))
	return 0
}

// reviewerName is `_reviewer`: `human:` and the account running the command.
// No flag names it, because apply checks only the prefix, and a flag would
// let any caller type any name into an assurance about a person. The domain
// is cut off, which leaves the name `getpass.getuser()` gives.
func reviewerName() (string, error) {
	account, err := currentUser()
	if err != nil {
		return "", err
	}
	name := account.Username
	if at := strings.LastIndex(name, `\`); at >= 0 {
		name = name[at+1:]
	}
	return "human:" + name, nil
}

// reportAbort names a decision that was broken off, and every file it had
// already changed. The hint hangs on the files, never on the kind of error:
// the kind says why the run stopped, not whether the vault was touched.
// Nothing on an abort path is committed, so a file in the vault stands
// unversioned until some other tool's auto-commit sweeps it up. A file
// outside the vault (the register of a readonly or out-of-tree area) is an
// absolute path in the list and is in no vault commit at all, so it is named
// under its own hint. (A relative state directory would make such a path
// relative and file it with the vault's.)
func reportAbort(stderr io.Writer, err error) {
	fmt.Fprintf(stderr, "error: %v\n", err)
	var touched interface{ DirtyFiles() []string }
	if !errors.As(err, &touched) {
		return
	}
	var inVault, outside []string
	for _, file := range touched.DirtyFiles() {
		if filepath.IsAbs(file) {
			outside = append(outside, file)
		} else {
			inVault = append(inVault, file)
		}
	}
	if len(inVault) > 0 {
		fmt.Fprintln(stderr, "Hinweis: der Abbruch hat diese Dateien im Vault bereits geändert und nichts "+
			"committet. Sie stehen unversioniert im Arbeitsbaum, bis der nächste Auto-Commit "+
			"eines fremden Werkzeugs (etwa obsidian-git) sie einsammelt:")
		for _, file := range inVault {
			fmt.Fprintf(stderr, "  %s\n", file)
		}
	}
	if len(outside) > 0 {
		fmt.Fprintln(stderr, "Hinweis: außerhalb des Vaults hat der Abbruch diese Dateien bereits geändert; "+
			"kein Commit erfasst sie:")
		for _, file := range outside {
			fmt.Fprintf(stderr, "  %s\n", file)
		}
	}
}

// reportDecision is `_report_decision`: what was decided on stdout, and "not
// committed" on stderr, kept apart from "not decided". The exit is 0 either
// way, because the vault changed and running the command again would not
// improve matters.
//
// A written approval is followed by the technical update, and only that one:
// a rejection leaves the page's text as it was and moves only its `sources[]`
// and the register.
func reportDecision(stdout, stderr io.Writer, vault registered, result apply.Result) {
	fmt.Fprintf(stdout, "Fall %s: %s\n", result.Case.ID, result.Decision)
	for _, dropped := range result.Dropped {
		fmt.Fprintf(stdout, "  verworfene Behauptung: %s\n", dropped)
	}
	if result.Commit != "" {
		fmt.Fprintf(stdout, "committet als %s\n", result.Commit)
	} else {
		done := "entschieden"
		if result.Written {
			done = "geschrieben"
		}
		fmt.Fprintf(stderr, "%s, aber nicht committet: %s\n", done, result.Warning)
	}
	if result.Written {
		technicalUpdate(stderr, vault)
	}
}

// technicalUpdate is `_technical_update` (cli.py:1432-1516): a catch-up pass,
// then the index run. It never fails the decision, which is already on disk.
//
// The catch-up runs first because the index run advances the register of
// every source whose hash moved, not only this case's; a source changed while
// the case was being decided would otherwise pass the review gate unseen.
// Any failure of it stops the update, a missing review centre included --
// unlike `reindex`, which warns and indexes on there: indexing past a failed
// catch-up is the very swallow the pass prevents, and on this path the
// reference makes no exception. The words say what the reader is left with,
// a stale index, and not whether the commit landed, which was said above.
//
// The index run is called without its own catch-up, so the vault is passed
// over once.
func technicalUpdate(stderr io.Writer, vault registered) {
	report, root, err := catchUp(vault.areas, vault.lookup)
	if err != nil {
		fmt.Fprintf(stderr, "warning: die Aufholung vor der technischen Aktualisierung ist "+
			"fehlgeschlagen (%v). Die Freigabe steht, aber der Suchindex des gesamten Vaults ist "+
			"ab jetzt veraltet: die Suche liefert weiter den alten Text. Ursache beheben, dann "+
			"`loomux reconcile` und danach `loomux reindex`.\n", err)
		return
	}
	reportCatchUp(stderr, root, report)
	registry := filepath.Join(vault.lookup.Primary, "registry.toml")
	if indexAreas(registry, vault.lookup, io.Discard, stderr) != 0 {
		fmt.Fprintln(stderr, "warning: die technische Aktualisierung ist fehlgeschlagen; die Freigabe steht, "+
			"der Index hinkt ihr hinterher -- `loomux reindex` holt das nach")
	}
}
