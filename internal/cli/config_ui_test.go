package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config/schema"
	"github.com/xidus90/loomux/internal/tui"
)

// keysTo moves the list cursor onto id and then plays rest. The index comes
// from entries, so a key added to the schema does not break the script.
func keysTo(t *testing.T, target configTarget, text, id string, rest ...string) []tui.Key {
	t.Helper()
	entries, err := target.entries(text)
	if err != nil {
		t.Fatal(err)
	}
	index := -1
	for i, e := range entries {
		if e.Key.ID() == id {
			index = i
		}
	}
	if index < 0 {
		t.Fatalf("no entry %s", id)
	}
	keys := []tui.Key{}
	for range index {
		keys = append(keys, tui.Key{Name: "down"})
	}
	return append(keys, tui.Keys(rest...)...)
}

func projectTarget(t *testing.T, text string) (string, configTarget) {
	t.Helper()
	root := configRoot(t, text)
	target, err := resolveConfigTarget(root, false)
	if err != nil {
		t.Fatal(err)
	}
	return root, target
}

func TestConfigUIChangesOneValue(t *testing.T) {
	text := "[commit]\nlanguage = \"en\"\n"
	root, target := projectTarget(t, text)
	term := tui.Script(100, 60, keysTo(t, target, text, "commit.language", "enter", "tab", "enter", "y", "q")...)
	if err := configUI(term, target); err != nil {
		t.Fatal(err)
	}
	if got := readConfig(t, root); got != "[commit]\nlanguage = \"de\"\n" {
		t.Fatalf("file:\n%s", got)
	}
}

func TestConfigUIOffersTrueAndFalseForABool(t *testing.T) {
	root, target := projectTarget(t, "")
	term := tui.Script(100, 60, keysTo(t, target, "", "modules.graph", "enter", "tab", "enter", "y", "q")...)
	if err := configUI(term, target); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(term.Output(), "[true false (default)]") {
		t.Fatal(term.Output())
	}
	if got := readConfig(t, root); got != "[modules]\ngraph = false\n" {
		t.Fatalf("file:\n%s", got)
	}
}

// The (default) choice takes the key's line out, the way `config unset`
// does; it is never written as a value.
func TestConfigUIDefaultChoiceUnsetsTheKey(t *testing.T) {
	text := "[commit]\nlanguage = \"de\"\n"
	root, target := projectTarget(t, text)
	// From "de" one tab reaches the choice after the enum's values.
	term := tui.Script(100, 60, keysTo(t, target, text, "commit.language", "enter", "tab", "enter", "y", "q")...)
	if err := configUI(term, target); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(term.Output(), ": (default)  (tab: [en de (default)])") {
		t.Fatalf("the (default) choice was not the one picked:\n%s", term.Output())
	}
	if got := readConfig(t, root); got != "" {
		t.Fatalf("file:\n%s", got)
	}
}

// A key without a default has nothing to fall back to, so no such choice.
func TestConfigUIOffersNoDefaultChoiceWithoutADefault(t *testing.T) {
	text := "[area]\nscope = \"s\"\n"
	_, target := projectTarget(t, text)
	term := tui.Script(100, 60, keysTo(t, target, text, "model.enabled", "enter", "esc", "q")...)
	if err := configUI(term, target); err != nil {
		t.Fatal(err)
	}
	if out := term.Output(); !strings.Contains(out, "[true false]") || strings.Contains(out, "(default)") {
		t.Fatal(out)
	}
}

func TestConfigUIShowsARefusalAndWritesNothing(t *testing.T) {
	root, target := projectTarget(t, "")
	term := tui.Script(100, 60, keysTo(t, target, "", "verify.timeout",
		"enter", "backspace", "backspace", "backspace", "backspace", "backspace", "soon", "enter", "esc", "q")...)
	if err := configUI(term, target); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(term.Output(), `"soon" is not a whole number`) {
		t.Fatal("the refusal must be shown in place")
	}
	if got := readConfig(t, root); got != "" {
		t.Fatalf("file:\n%s", got)
	}
}

func TestConfigUIWritesNothingWhenDeclinedOrUnchanged(t *testing.T) {
	text := "[commit]\nlanguage = \"en\"\n"
	root, target := projectTarget(t, text)
	// Once declined at the diff, once confirmed without a change.
	keys := keysTo(t, target, text, "commit.language", "enter", "tab", "enter", "n")
	keys = append(keys, keysTo(t, target, text, "commit.conventional", "enter", "enter", "q")...)
	term := tui.Script(100, 60, keys...)
	if err := configUI(term, target); err != nil {
		t.Fatal(err)
	}
	if got := readConfig(t, root); got != text {
		t.Fatalf("file:\n%s", got)
	}
}

func TestConfigUITablesAreShownNotEdited(t *testing.T) {
	_, target := projectTarget(t, "")
	term := tui.Script(100, 60, keysTo(t, target, "", "verify.profiles", "enter", "n", "q")...)
	if err := configUI(term, target); err != nil {
		t.Fatal(err)
	}
	out := term.Output()
	if !strings.Contains(out, "edit it by hand") {
		t.Fatal(out)
	}
	// A preset row shows the default the presets mirror, as `config list` does.
	if !strings.Contains(out, `precommit = ["lint"`) {
		t.Fatal(out)
	}
}

// A family row has no line of its own to write; like a table row, it opens a
// dialog that names the command setting one member, with the name to fill in
// where the star stands, and the form goes on.
func TestConfigUIShowsAFamilyRowsHint(t *testing.T) {
	root, target := projectTarget(t, "")
	for id, command := range map[string]string{
		"agent.roles.*":           "agent.roles.<name>",
		"agent.models.*.provider": "agent.models.<name>.provider",
	} {
		term := tui.Script(100, 60, keysTo(t, target, "", id, "enter", "n", "q")...)
		if err := configUI(term, target); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		want := id + " stands for one key per name; set one with `loomux config set " + command + " <value>`"
		if out := term.Output(); !strings.Contains(out, want) {
			t.Errorf("%s: no %q in\n%s", id, want, out)
		}
	}
	if got := readConfig(t, root); got != "" {
		t.Fatalf("file:\n%s", got)
	}
}

func TestConfigUIShowsAListOfTablesByItsCount(t *testing.T) {
	text := "[commit]\nthreshold = 3\n\n[[commit.allow]]\nregex = 'x'\nreason = \"y\"\n"
	_, target := projectTarget(t, text)
	term := tui.Script(100, 60, keysTo(t, target, text, "commit.allow", "enter", "n", "q")...)
	if err := configUI(term, target); err != nil {
		t.Fatal(err)
	}
	out := term.Output()
	// The dialog shows what the row only counts.
	if !strings.Contains(out, "edit it by hand") || !strings.Contains(out, "1 entry:") || !strings.Contains(out, "regex:x") {
		t.Fatal(out)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "  commit.allow ") && (!strings.Contains(line, "1 entry") || strings.Contains(line, "entries")) {
			t.Fatalf("a list of tables shows its count: %q", line)
		}
		if strings.HasPrefix(line, "  commit.threshold ") && (strings.Contains(line, "entr") || !strings.Contains(line, " 3 ")) {
			t.Fatalf("any other key shows its value: %q", line)
		}
	}
}

func TestConfigUIMakesSeveralChangesInOneSession(t *testing.T) {
	root, target := projectTarget(t, "")
	keys := keysTo(t, target, "", "modules.graph", "enter", "tab", "enter", "y")
	keys = append(keys, keysTo(t, target, "[modules]\ngraph = false\n", "modules.hooks", "enter", "tab", "enter", "y", "q")...)
	if err := configUI(tui.Script(100, 60, keys...), target); err != nil {
		t.Fatal(err)
	}
	if got := readConfig(t, root); got != "[modules]\ngraph = false\nhooks = false\n" {
		t.Fatalf("file:\n%s", got)
	}
}

func TestConfigUIReturnsWhatStopsIt(t *testing.T) {
	// Keys run out in the list: the terminal's end is the answer.
	_, target := projectTarget(t, "")
	if err := configUI(tui.Script(100, 60), target); err == nil {
		t.Fatal("want the end of the keys")
	}
	// Keys run out inside the form.
	if err := configUI(tui.Script(100, 60, tui.Keys("enter")...), target); err == nil {
		t.Fatal("want the end of the keys")
	}
	_, broken := projectTarget(t, "[commit\n")
	if err := configUI(tui.Script(100, 60, tui.Keys("q")...), broken); err == nil {
		t.Fatal("want the TOML error")
	}
	dir := configTarget{path: t.TempDir(), keys: schema.Keys()}
	if err := configUI(tui.Script(100, 60, tui.Keys("q")...), dir); err == nil {
		t.Fatal("want the read error")
	}
}

func TestConfigUICreatesTheFile(t *testing.T) {
	root := t.TempDir()
	target, err := resolveConfigTarget(root, false)
	if err != nil {
		t.Fatal(err)
	}
	keys := keysTo(t, target, "", "commit.language", "enter", "tab", "enter", "y", "q")
	if err := configUI(tui.Script(100, 60, keys...), target); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".loomux", "config.toml"))
	if err != nil || string(data) != "[commit]\nlanguage = \"de\"\n" {
		t.Fatalf("%v\n%s", err, data)
	}
}

func TestConfigInteractiveOpensTheTerminal(t *testing.T) {
	root := configRoot(t, "")
	restore := openTerminal
	t.Cleanup(func() { openTerminal = restore })
	script := func(keys ...tui.Key) func() (tui.Terminal, func() error, error) {
		return func() (tui.Terminal, func() error, error) {
			return tui.Script(80, 20, keys...), func() error { return nil }, nil
		}
	}
	openTerminal = script(tui.Keys("q")...)
	if code, _, errOut := runConfig(t, "", "--root", root); code != 0 {
		t.Fatalf("%d %s", code, errOut)
	}
	// A console that closes is an end, not a failure.
	openTerminal = script()
	if code, _, errOut := runConfig(t, "", "--root", root); code != 0 {
		t.Fatalf("%d %s", code, errOut)
	}
	openTerminal = func() (tui.Terminal, func() error, error) { return nil, nil, errors.New("no console") }
	if code, _, errOut := runConfig(t, "", "--root", root); code != 2 || !strings.Contains(errOut, "`get`, `set` or `unset`") {
		t.Fatalf("%d %s", code, errOut)
	}
	// A console left in raw mode is a failure the human has to hear about.
	openTerminal = func() (tui.Terminal, func() error, error) {
		return tui.Script(80, 20, tui.Keys("q")...), func() error { return errors.New("mode lost") }, nil
	}
	if code, _, errOut := runConfig(t, "", "--root", root); code != 1 || !strings.Contains(errOut, "restoring the terminal: mode lost") {
		t.Fatalf("%d %s", code, errOut)
	}
	broken := configRoot(t, "[commit\n")
	openTerminal = script(tui.Keys("q")...)
	if code, _, errOut := runConfig(t, "", "--root", broken); code != 1 || !strings.Contains(errOut, "not valid TOML") {
		t.Fatalf("%d %s", code, errOut)
	}
}
