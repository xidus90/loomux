package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/xidus90/loomux/internal/config/edit"
	"github.com/xidus90/loomux/internal/config/schema"
	"github.com/xidus90/loomux/internal/tui"
)

// defaultChoice is the pick that takes a key back to its default.
const defaultChoice = "(default)"

// openTerminal is the seam between the command and a real console.
var openTerminal = openConsole

// openConsole puts the process's own console into raw mode.
//
//coverage:exempt a test process has no console of its own, and tests must not take the one go test runs in
func openConsole() (tui.Terminal, func() error, error) {
	if !tui.IsTerminal(os.Stdin) || !tui.IsTerminal(os.Stdout) {
		return nil, nil, errors.New("not a terminal")
	}
	return tui.Open(os.Stdin, os.Stdout)
}

func runConfigUI(target configTarget, stderr io.Writer) (code int) {
	term, restore, err := openTerminal()
	if err != nil {
		fmt.Fprintln(stderr, "loomux config: the interactive form needs a terminal; use `loomux config list`, `get`, `set` or `unset`")
		return 2
	}
	// Deferred so a panic in the form still hands the shell back a cooked
	// console; a restore that fails leaves it raw, which the human must hear.
	defer func() {
		if err := restore(); err != nil {
			fmt.Fprintf(stderr, "loomux config: restoring the terminal: %v\n", err)
			code = 1
		}
	}()
	err = configUI(term, target)
	// A console that closes under the form ends it like q does.
	if err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintf(stderr, "loomux config: %v\n", err)
		return 1
	}
	return 0
}

// configUI is the list loop: choose a row, change it, see the diff, confirm.
// Every change is written on its own, the way `config set` writes it, and the
// file is read again before the next choice.
func configUI(term tui.Terminal, target configTarget) error {
	for {
		text, err := target.read()
		if err != nil {
			return err
		}
		entries, err := target.entries(text)
		if err != nil {
			return err
		}
		rows := make([]tui.Row, len(entries))
		for i, e := range entries {
			value := shownValue(e)
			if e.Key.Kind == schema.TableList {
				value = entryCount(e.Count)
			}
			rows[i] = tui.Row{Group: string(e.Key.Module), Label: e.Key.ID(), Value: value, Note: string(e.Origin)}
		}
		title := "loomux config — " + target.path
		if n := target.openProposals(); n > 0 {
			title += " — " + proposalHint(n)
		}
		chosen, err := tui.List(term, title, rows)
		if err != nil || chosen < 0 {
			return err
		}
		if err := changeOne(term, target, text, entries[chosen]); err != nil {
			return err
		}
	}
}

func changeOne(term tui.Terminal, target configTarget, text string, e schema.Entry) error {
	if e.Key.Kind == schema.Table || e.Key.Kind == schema.TableList {
		body := shownValue(e)
		// The row only counts the blocks; the dialog is where they are read.
		if e.Key.Kind == schema.TableList {
			body = entryCount(e.Count)
			if e.Count > 0 {
				body += ":\n" + e.Value
			}
		}
		_, err := tui.Confirm(term, e.Key.Doc+"\n\n"+body, e.Key.ID()+" is a table; edit it by hand. Back?")
		return err
	}
	// A bool typed free-hand has one right spelling in two; picking it
	// leaves nothing to mistype.
	choices := slices.Clone(e.Key.Choices)
	if e.Key.Kind == schema.Bool {
		choices = []string{"true", "false"}
	}
	// A picked key has no free text to clear, so going back to the default
	// is a choice of its own; a typed key goes back with `config unset`.
	if len(choices) > 0 && e.Key.Default != "" {
		choices = append(choices, defaultChoice)
	}
	var next string
	_, ok, err := tui.Input(term, e.Key.ID()+" — "+e.Key.Doc, e.Input, choices, func(s string) error {
		var err error
		if s == defaultChoice {
			next, err = proposeUnset(target, text, e.Key.ID())
		} else {
			next, err = proposeChange(target, text, e.Key.ID(), s)
		}
		return err
	})
	if err != nil || !ok {
		return err
	}
	diff := edit.Diff(text, next)
	if diff == "" {
		return nil
	}
	yes, err := tui.Confirm(term, diff, "write these changes?")
	if err != nil || !yes {
		return err
	}
	return writeConfig(target, text, next)
}
