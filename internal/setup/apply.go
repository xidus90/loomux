package setup

import (
	"fmt"
	"maps"
	"path/filepath"
	"time"

	"github.com/xidus90/loomux/internal/lock"
	"github.com/xidus90/loomux/internal/setup/hostfile"
	"github.com/xidus90/loomux/internal/setup/write"
)

// backupDir keeps the first text of every project file init rewrote.
const backupDir = ".loomux/state/backup"

// Runner runs one action of a plan.
type Runner func(a Action) error

// Report is what a run of Apply did.
type Report struct {
	Written, Skipped, Failed []string // paths and action ids
	Refused                  []string // changes the human declined
	// Notes are failures that stop nothing and leave nothing half done: a
	// pull of the local model that did not finish.
	Notes []string
}

// applier carries a run of Apply: the files it wrote and the actions it ran
// go into installed.toml, the report to the caller.
type applier struct {
	root   string
	files  []string
	ran    []string
	report Report
}

// Apply writes the approved changes and runs the actions in this order:
// binary, area-add, files, hooks-path, merge-hook, model-pull, graph-build,
// answers, installed. A failed model-pull is a note and the run goes on,
// as the next plan asks Ollama again. area-add goes before the files because it writes the
// declaration only into a configuration that is not there yet; a change
// with a Redo is then made again over what it left. A change whose file
// calls a binary that is not there (Change.Binary) is dropped and reported
// as failed, and so are the git-hooks actions without the binary the
// entries call and merge-hook without the installed binary under
// LOCALAPPDATA, whichever binary the entries call.
//
// Any other failure stops the run and leaves installed.toml unwritten, so
// the next plan shows what is still open; version is the version of the
// running loomux, recorded there.
func Apply(root string, p Plan, c Choice, approve func(Change) bool, run Runner, binaryThere func() bool, version string, now time.Time) (Report, error) {
	a := &applier{root: root}
	var first, rest []Action
	for _, act := range p.Actions {
		switch act.ID {
		case "binary-install", "binary-build":
		case "area-add":
			first = append(first, act)
			continue
		default:
			rest = append(rest, act)
			continue
		}
		// A failed install may still find an earlier binary in place, so
		// binaryThere, not the error, decides what follows.
		if err := run(act); err != nil {
			a.report.Failed = append(a.report.Failed, act.ID)
			continue
		}
		a.ran = append(a.ran, act.ID)
	}
	present := binaryThere()
	canonical := isFile(BinaryPath(root, hostfile.Canonical))
	// missing says whether the binary a file calls is absent; the file
	// would call nothing.
	missing := func(binary string) bool {
		switch binary {
		case "":
			return false
		case hostfile.Canonical:
			return !canonical
		}
		return !present
	}
	// An action names no binary of its own: merge-hook's hook calls the
	// installed one in every project, and hooks-path goes with the git
	// hooks, which call the binary the entries call.
	dropped := func(part string) bool {
		switch part {
		case "merge-hook":
			return !canonical
		case "git-hooks":
			return !present
		}
		return false
	}
	if err := a.actions(first, run, dropped); err != nil {
		return a.done(), err
	}
	for _, ch := range p.Changes {
		switch {
		case missing(ch.Binary):
			a.report.Failed = append(a.report.Failed, ch.Path)
		case !approve(ch):
			a.report.Refused = append(a.report.Refused, ch.Path)
		default:
			if err := a.change(ch); err != nil {
				a.report.Failed = append(a.report.Failed, ch.Path)
				return a.done(), err
			}
		}
	}
	if err := a.actions(rest, run, dropped); err != nil {
		return a.done(), err
	}
	hostNames := make([]string, len(c.Hosts))
	for i, h := range c.Hosts {
		hostNames[i] = string(h)
	}
	state := map[string]any{answersPath: Answers{Hosts: hostNames, Parts: maps.Clone(c.Parts)}}
	order := []string{answersPath}
	// A run that changed nothing leaves the record of the last one that did,
	// and one in which a step failed leaves none, so the next plan shows
	// what is still open.
	if len(a.files)+len(a.ran) > 0 && len(a.report.Failed) == 0 {
		state[installedPath] = installed{Version: version, At: now.UTC().Truncate(time.Second), Files: a.files, Actions: a.ran}
		order = append(order, installedPath)
	}
	for _, rel := range order {
		if err := writeState(root, rel, state[rel]); err != nil {
			return a.done(), err
		}
	}
	return a.done(), nil
}

// done is the report with what was written and run.
func (a *applier) done() Report {
	a.report.Written = append(append([]string(nil), a.files...), a.ran...)
	return a.report
}

// actions runs acts in order and stops at the first that fails; one whose
// part is dropped is reported as failed and not run.
func (a *applier) actions(acts []Action, run Runner, dropped func(string) bool) error {
	for _, act := range acts {
		if dropped(act.Part) {
			a.report.Failed = append(a.report.Failed, act.ID)
			continue
		}
		err := run(act)
		switch {
		case err != nil && act.ID == "model-pull":
			a.report.Notes = append(a.report.Notes, act.ID+": "+err.Error()+"; run it by hand: "+act.Describe)
			continue
		case err != nil:
			a.report.Failed = append(a.report.Failed, act.ID)
			return fmt.Errorf("%s: %w", act.ID, err)
		}
		a.ran = append(a.ran, act.ID)
	}
	return nil
}

// redo makes a change again over the text that stands now, when that is no
// longer the text it was planned over.
func (a *applier) redo(ch Change) (Change, error) {
	if ch.Redo == nil {
		return ch, nil
	}
	data, exists, err := readOptional(a.root, ch.Path)
	if err != nil || (exists == ch.Exists && string(data) == ch.Before) {
		return ch, err
	}
	after, err := ch.Redo(string(data))
	if err != nil {
		return ch, err
	}
	ch.Before, ch.After, ch.Exists = string(data), after, exists
	return ch, nil
}

// change writes one approved change: a new file only where none stands, an
// existing one after its first text is kept in backupDir. The configuration
// has no backup; it is replaced whole, and Build or Redo has validated the
// new text. A file this run made before -- area add's -- has no backup
// either: there was no text before init.
func (a *applier) change(ch Change) error {
	planned := ch.Exists
	if ch.Redo == nil && ch.Exists {
		if err := a.unchanged(ch); err != nil {
			return err
		}
	}
	ch, err := a.redo(ch)
	if err != nil || ch.Empty() {
		return err
	}
	if !ch.Exists {
		written, err := createFile(a.root, ch.Path, ch.After)
		if err != nil {
			return err
		}
		if written {
			a.files = append(a.files, ch.Path)
		} else {
			a.report.Skipped = append(a.report.Skipped, ch.Path)
		}
		return nil
	}
	if planned && ch.Path != configPath {
		// An exclusive create: a backup from an earlier run holds the text
		// before init first touched the file, and stays.
		if _, err := createFile(a.root, backupDir+"/"+ch.Path+".bak", ch.Before); err != nil {
			return fmt.Errorf("backing up %s: %w", ch.Path, err)
		}
	}
	err = write.CheckParents(a.root, ch.Path)
	if err == nil {
		err = lock.ReplaceText(filepath.Join(a.root, filepath.FromSlash(ch.Path)), ch.After)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", ch.Path, err)
	}
	a.files = append(a.files, ch.Path)
	return nil
}

// unchanged says whether the file of a change nothing can redo still holds
// the text the change was planned over. Someone else may have written it
// between the plan and the approval; After would take that back without a
// word, and the backup would keep the older text instead of theirs.
func (a *applier) unchanged(ch Change) error {
	data, exists, err := readOptional(a.root, ch.Path)
	if err != nil {
		return err
	}
	if !exists || string(data) != ch.Before {
		return fmt.Errorf("%s changed since the plan was made; nothing written", ch.Path)
	}
	return nil
}

// createFile writes name under root unless something stands there, and
// says whether it did.
func createFile(root, name, text string) (bool, error) {
	plan, err := write.Prepare(root, map[string]string{name: text})
	var written []string
	if err == nil {
		written, err = write.Commit(root, plan)
	}
	return len(written) == 1, err
}
