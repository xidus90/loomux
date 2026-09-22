package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// The lines `case` prints between a case's head and its files, spelt to the
// letter of the reference (`cli.py:921-939`): the review skill halts at the
// first two, and the case corpus compares them byte for byte.
const (
	haltLine = "local_only: der Quelldiff dieses Bereichs darf kein Cloud-Modell " +
		"erreichen — kein Skill-Pfad in einer Wolkensitzung"
	manualLine = "manuell: für diesen Fall wird kein Skill-Pfad angeboten"

	// unknownModeLine stands under the halt line when the area's mode cannot
	// be read at all: not registered, or registered without a declaration.
	// Fixed text without a cause or a path, so a corpus entry can compare it.
	unknownModeLine = "Datenschutzmodus unbekannt: Der Bereich ist nicht registriert oder hat kein " +
		"lesbares Manifest und gilt deshalb als local_only"
)

// casesCommand is `loomux cases`: every case standing in the review centre,
// one line each, in the order the reference lists them.
//
// Scored like `reconcile`: a waiting case is the normal product of a pass, and
// only a case file that does not read makes the exit 1 -- it means the queue
// holds two entries for one page, and only a person can settle which stands.
func casesCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("loomux cases", flag.ContinueOnError)
	flags.SetOutput(stderr)
	if err := flags.Parse(args); err != nil || refusesArguments(flags, stderr) {
		return 2
	}
	_, root, err := reviewCentre()
	if err != nil {
		return reportReconcileError(stderr, err)
	}
	listed := 0
	var broken []string
	for _, path := range maintenance.CaseFiles(root) {
		read, err := maintenance.ReadCase(path)
		if err != nil {
			broken = append(broken, err.Error())
			continue
		}
		directory := filepath.Dir(path)
		fmt.Fprintln(stdout, caseLine(filepath.Base(directory), read))
		warnRenamed(stderr, directory, read)
		listed++
	}
	if listed == 0 {
		fmt.Fprintln(stdout, "keine offenen Fälle")
	}
	return reportUnreadable(stderr, broken)
}

// caseCommand is `loomux case <id> [--package]`: everything the human gate
// has to see before it decides -- the case, its package and its proposal.
//
// The proposal belongs beside the package: the evidence binding checks each
// claim's quote against a package segment, never the diff the claim carries,
// so only the person reading both catches a diff nobody vouched for.
//
// A closed case withholds both files unless `--package` is typed: a
// proposal quotes the package verbatim, and the package carries the source
// diff a cloud model must not see.
func caseCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("loomux case", flag.ContinueOnError)
	flags.SetOutput(stderr)
	lift := flags.Bool("package", false, "print the withheld package and proposal of a local_only case")
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
	if err := showCase(stdout, stderr, free[0], *lift); err != nil {
		return reportReconcileError(stderr, err)
	}
	return 0
}

// reviewCentre is the registry and the review centre it declares, the two
// things every case command resolves before it looks at a case.
func reviewCentre() (registered, string, error) {
	lookup := config.NewArtifactLookup()
	areas, err := config.ReadRegistry(lookup.Primary)
	if err != nil {
		return registered{}, "", err
	}
	root, err := maintenance.ReviewRoot(areas, lookup)
	return registered{areas, lookup}, root, err
}

// registered is the registry together with the state it was read from, which
// is what a declaration is looked up against.
type registered struct {
	areas  []config.Area
	lookup config.ArtifactLookup
}

// showCase prints one case the way `_case` does: its head, the lines the
// review skill halts at, and then either its files or where they are withheld.
func showCase(stdout, stderr io.Writer, identifier string, lift bool) error {
	vault, root, err := reviewCentre()
	if err != nil {
		return err
	}
	directory, err := maintenance.FindCase(root, identifier)
	if err != nil {
		return err
	}
	read, err := maintenance.ReadCase(filepath.Join(directory, "case.toml"))
	if err != nil {
		return err
	}
	warnRenamed(stderr, directory, read)
	// The case's mark is a snapshot, so the area's current declaration closes
	// the case as well, and no declaration at all closes it too: a switch that
	// opens for want of an answer is not one.
	manifest := maintenance.AreaManifest(vault.areas, vault.lookup, read.Area)
	withheld := read.LocalOnly || manifest == nil || manifest.PrivacyMode == "local_only"

	name := filepath.Base(directory)
	fmt.Fprintf(stdout, "Fall %s (%s, ausgelöst durch %s, Gewicht %s)\n", name, read.State, read.Trigger, read.Weight)
	fmt.Fprintf(stdout, "Bereich: %s\n", read.Area)
	fmt.Fprintf(stdout, "Ziel: %s\n", read.Target)
	for _, source := range read.Sources {
		fmt.Fprintf(stdout, "Quelle: %s (Revision %d, %s)\n", source.DocID, source.Revision, source.ContentHash)
	}
	// The halt line stands before `manuell:` because it carries the sharper
	// condition: the one refuses a proposal, the other refuses a reader.
	if withheld {
		fmt.Fprintln(stdout, haltLine)
	}
	if manifest == nil {
		fmt.Fprintln(stdout, unknownModeLine)
	}
	if read.Manual {
		fmt.Fprintln(stdout, manualLine)
	}
	if read.Note != "" {
		fmt.Fprintf(stdout, "Vermerk: %s\n", read.Note)
	}
	if withheld && !lift {
		printWithheld(stdout, root, directory)
		return nil
	}
	files := []string{"package.md", "proposal.md"}
	if read.SupersededProposal != "" {
		files = append(files, "superseded-proposal.md")
	}
	for _, file := range files {
		if err := showFile(stdout, filepath.Join(directory, file)); err != nil {
			return err
		}
	}
	return nil
}

// printWithheld says where the withheld files are and how to open them
// deliberately (`_withheld`).
//
// The place is relative to the review centre and slash-separated, because in
// that form it is the same on every run and every system; the absolute path
// would never repeat in a recorded expectation. The command is loomux's own
// where the reference names `brain case`.
func printWithheld(stdout io.Writer, root, directory string) {
	// FindCase answered a directory it walked to beneath root, so the two
	// always relate and Rel cannot fail.
	relative, _ := filepath.Rel(root, directory)
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "===== zurückgehalten =====")
	fmt.Fprintln(stdout, "Die Falldateien bleiben ungedruckt: Der Quelldiff dieses Bereichs darf kein "+
		"Cloud-Modell erreichen, und ein Vorschlag zitiert ihn wörtlich.")
	fmt.Fprintf(stdout, "Im Prüfzentrum unter: %s\n", filepath.ToSlash(relative))
	fmt.Fprintf(stdout, "Bewusst ausgeben: loomux case --package %s\n", filepath.Base(directory))
}

// showFile prints one file of the case under a heading, or a line saying it
// is not there (`_show`). Named rather than skipped: "no proposal yet" and "a
// proposal the reader scrolled past" look alike in silence, and they are
// opposite situations at a gate that decides whether to write.
func showFile(stdout io.Writer, path string) error {
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "===== %s =====\n", filepath.Base(path))
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		fmt.Fprintf(stdout, "(nicht vorhanden: %s)\n", path)
		return nil
	}
	text, err := pytext.ReadText(path)
	if err != nil {
		return err
	}
	fmt.Fprint(stdout, text)
	return nil
}

// warnRenamed says so on stderr when the `id` field and the directory name
// have parted ways. `case.toml` is synchronised vault content and gets
// hand-edited, so this is reachable input; the directory wins because it is
// what addresses the case, and it is named rather than silently preferred, or
// the reviewer keeps typing an id nothing answers to.
func warnRenamed(stderr io.Writer, directory string, read maintenance.Case) {
	name := filepath.Base(directory)
	if read.ID == name {
		return
	}
	fmt.Fprintf(stderr, "warning: %s calls itself %s, but the case is addressed as %s; "+
		"the directory name is what counts\n",
		filepath.Join(directory, "case.toml"), pytext.Repr(read.ID), pytext.Repr(name))
}
