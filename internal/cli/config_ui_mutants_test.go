package cli

import (
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config/schema"
	"github.com/xidus90/loomux/internal/tui"
)

// tableDialog returns the text of the last table dialog shown for id, from
// the key's description to the question, and the entry it was built from.
func tableDialog(t *testing.T, target configTarget, text, id, out string) (string, schema.Entry) {
	t.Helper()
	entries, err := target.entries(text)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Key.ID() != id {
			continue
		}
		start := strings.LastIndex(out, e.Key.Doc+"\n\n")
		question := id + " is a table; edit it by hand. Back? [y/n] "
		end := strings.LastIndex(out, question)
		if start < 0 || end < start {
			t.Fatalf("no dialog for %s in\n%s", id, out)
		}
		return out[start : end+len(question)], e
	}
	t.Fatalf("no entry %s", id)
	return "", schema.Entry{}
}

// Without an open proposal the title names none, not "0 proposals open".
func TestConfigUITitleNamesNoProposalWhenNoneIsOpen(t *testing.T) {
	_, target := projectTarget(t, "")
	// Wide enough that the title, temp path and all, is not cut.
	term := tui.Script(1000, 60, tui.Keys("q")...)
	if err := configUI(term, target); err != nil {
		t.Fatal(err)
	}
	out := term.Output()
	if first, _, _ := strings.Cut(out, "\n"); first != "loomux config — "+target.path || strings.Contains(out, "open — loomux config proposals") {
		t.Fatal(out)
	}
}

// A plain table's dialog shows its value, not a count of blocks it has none of.
func TestConfigUITableDialogShowsTheValueNotACount(t *testing.T) {
	_, target := projectTarget(t, "")
	term := tui.Script(100, 60, keysTo(t, target, "", "verify.profiles", "enter", "n", "q")...)
	if err := configUI(term, target); err != nil {
		t.Fatal(err)
	}
	dialog, e := tableDialog(t, target, "", "verify.profiles", term.Output())
	if e.Key.Kind != schema.Table {
		t.Fatalf("verify.profiles is no longer a plain table: %v", e.Key.Kind)
	}
	want := e.Key.Doc + "\n\n" + shownValue(e) + "\nverify.profiles is a table; edit it by hand. Back? [y/n] "
	if dialog != want || strings.Contains(dialog, "entries") {
		t.Fatalf("dialog:\n%q\nwant:\n%q", dialog, want)
	}
}

// An empty list of tables has no blocks to list after its count.
func TestConfigUIEmptyListOfTablesShowsOnlyItsCount(t *testing.T) {
	_, target := projectTarget(t, "")
	term := tui.Script(100, 60, keysTo(t, target, "", "commit.allow", "enter", "n", "q")...)
	if err := configUI(term, target); err != nil {
		t.Fatal(err)
	}
	dialog, e := tableDialog(t, target, "", "commit.allow", term.Output())
	if e.Key.Kind != schema.TableList || e.Count != 0 {
		t.Fatalf("commit.allow: kind %v, count %d", e.Key.Kind, e.Count)
	}
	want := e.Key.Doc + "\n\n0 entries\ncommit.allow is a table; edit it by hand. Back? [y/n] "
	if dialog != want {
		t.Fatalf("dialog:\n%q\nwant:\n%q", dialog, want)
	}
}
