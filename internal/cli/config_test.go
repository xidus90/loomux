package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/tui"
)

func configRoot(t *testing.T, text string) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), text)
	return root
}

func runConfig(t *testing.T, stdin string, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(append([]string{"config"}, args...), strings.NewReader(stdin), &out, &errOut)
	return code, out.String(), errOut.String()
}

func readConfig(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".loomux", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestConfigListShowsValueAndOrigin(t *testing.T) {
	root := configRoot(t, "[commit]\nlanguage = \"de\"\n")
	code, out, _ := runConfig(t, "", "list", "--root", root)
	if code != 0 {
		t.Fatal(code)
	}
	for _, want := range []string{"commit.language", `"de"`, "set", "commit.threshold", "default"} {
		if !strings.Contains(out, want) {
			t.Errorf("list lacks %q:\n%s", want, out)
		}
	}
}

// A [[…]] list is counted, not printed: its blocks run over many lines.
func TestConfigListCountsTableListEntries(t *testing.T) {
	root := configRoot(t, "[[commit.allow]]\nregex = \"x\"\nreason = \"y\"\n")
	_, out, _ := runConfig(t, "", "list", "--root", root)
	if !strings.Contains(out, "1 entry") || strings.Contains(out, "1 entries") {
		t.Fatalf("list:\n%s", out)
	}
}

// A preset row has no value of its own; the default it mirrors stands in.
func TestConfigListShowsTheDefaultOfAPresetRow(t *testing.T) {
	root := configRoot(t, "")
	_, out, _ := runConfig(t, "", "list", "--root", root)
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "verify.profiles") {
			if !strings.Contains(line, "edit = [") || !strings.Contains(line, "preset") {
				t.Fatalf("row: %q", line)
			}
			return
		}
	}
	t.Fatalf("no verify.profiles row:\n%s", out)
}

func TestConfigListJSON(t *testing.T) {
	root := configRoot(t, "")
	code, out, _ := runConfig(t, "", "list", "--json", "--root", root)
	var rows []map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &rows) != nil || len(rows) == 0 {
		t.Fatalf("code %d, out %s", code, out)
	}
	for _, row := range rows {
		if row["key"] == "verify.profiles" && (row["origin"] != "preset" || !strings.Contains(row["value"].(string), "edit = [")) {
			t.Fatalf("preset row: %v", row)
		}
	}
}

func TestConfigGet(t *testing.T) {
	root := configRoot(t, "[commit]\nlanguage = \"de\"\n")
	if code, out, _ := runConfig(t, "", "get", "commit.language", "--root", root); code != 0 || out != "\"de\"\n" {
		t.Fatalf("%d %q", code, out)
	}
	if code, out, _ := runConfig(t, "", "get", "commit.threshold", "--root", root); code != 0 || out != "2\n" {
		t.Fatalf("%d %q", code, out)
	}
	if code, _, errOut := runConfig(t, "", "get", "commit.nope", "--root", root); code != 1 || !strings.Contains(errOut, "unknown") {
		t.Fatalf("%d %q", code, errOut)
	}
	if code, out, _ := runConfig(t, "", "get", "area.scope", "--root", root); code != 0 || out != "\n" {
		t.Fatalf("unset: %d %q", code, out)
	}
}

// A file that is there but cannot be read, or is no TOML, is an error for
// every form rather than an empty configuration.
func TestConfigReportsAnUnreadableFile(t *testing.T) {
	dirRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dirRoot, ".loomux", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	badRoot := configRoot(t, "[commit\n")
	for _, args := range [][]string{
		{"list", "--root", dirRoot},
		{"get", "commit.language", "--root", dirRoot},
		{"set", "commit.language", "de", "--yes", "--root", dirRoot},
		{"list", "--root", badRoot},
		{"get", "commit.language", "--root", badRoot},
		{"set", "commit.language", "de", "--yes", "--root", badRoot},
	} {
		if code, _, errOut := runConfig(t, "", args...); code != 1 || errOut == "" {
			t.Errorf("%v: %d %q", args, code, errOut)
		}
	}
}

func TestConfigSetAsksAndWrites(t *testing.T) {
	root := configRoot(t, "# mine\n[commit]\nlanguage = \"en\"  # keep\n")
	code, _, errOut := runConfig(t, "y\n", "set", "commit.language", "de", "--root", root)
	if code != 0 {
		t.Fatalf("%d %s", code, errOut)
	}
	if !strings.Contains(errOut, "+ language = \"de\"  # keep") {
		t.Errorf("no diff shown:\n%s", errOut)
	}
	if got := readConfig(t, root); got != "# mine\n[commit]\nlanguage = \"de\"  # keep\n" {
		t.Fatalf("file:\n%s", got)
	}
}

func TestConfigSetTakesYesAsAConfirmation(t *testing.T) {
	for _, answer := range []string{"yes\n", " YES \n", "Y\n"} {
		root := configRoot(t, "[commit]\nlanguage = \"en\"\n")
		code, _, errOut := runConfig(t, answer, "set", "commit.language", "de", "--root", root)
		if got := readConfig(t, root); code != 0 || got != "[commit]\nlanguage = \"de\"\n" {
			t.Errorf("%q: %d %s\n%s", answer, code, errOut, got)
		}
	}
	for _, answer := range []string{"no\n", "\n", "yess\n", ""} {
		root := configRoot(t, "[commit]\nlanguage = \"en\"\n")
		runConfig(t, answer, "set", "commit.language", "de", "--root", root)
		if got := readConfig(t, root); got != "[commit]\nlanguage = \"en\"\n" {
			t.Errorf("%q wrote:\n%s", answer, got)
		}
	}
}

func TestConfigSetWritesNothingWhenDeclined(t *testing.T) {
	root := configRoot(t, "[commit]\nlanguage = \"en\"\n")
	code, _, errOut := runConfig(t, "n\n", "set", "commit.language", "de", "--root", root)
	if got := readConfig(t, root); code != 0 || got != "[commit]\nlanguage = \"en\"\n" || !strings.Contains(errOut, "nothing written") {
		t.Fatalf("%d %q %s", code, errOut, got)
	}
}

func TestConfigSetOfTheSameValueWritesNothing(t *testing.T) {
	root := configRoot(t, "[commit]\nthreshold = 3\n")
	code, _, errOut := runConfig(t, "", "set", "commit.threshold", "3", "--yes", "--root", root)
	if code != 0 || !strings.Contains(errOut, "already so") {
		t.Fatalf("%d %s", code, errOut)
	}
}

func TestConfigSetToTheDefaultRemovesTheLine(t *testing.T) {
	root := configRoot(t, "[area]\nscope = \"s\"\n\n[commit]\nlanguage = \"de\"\n")
	runConfig(t, "", "set", "commit.language", "en", "--yes", "--root", root)
	if got := readConfig(t, root); got != "[area]\nscope = \"s\"\n" {
		t.Fatalf("file:\n%s", got)
	}
}

// A default typed in another spelling is still the default: +600 must not
// pin verify.timeout into a file that does not name it.
func TestConfigSetOfTheDefaultInAnotherSpellingWritesNothing(t *testing.T) {
	root := configRoot(t, "[commit]\nthreshold = 4\n")
	code, _, errOut := runConfig(t, "", "set", "verify.timeout", "+600", "--yes", "--root", root)
	if got := readConfig(t, root); code != 0 || got != "[commit]\nthreshold = 4\n" || !strings.Contains(errOut, "already so") {
		t.Fatalf("%d %s\nfile:\n%s", code, errOut, got)
	}
}

func TestConfigUnsetRemovesTheLineAndAnEmptiedSection(t *testing.T) {
	root := configRoot(t, "[commit]\nthreshold = 5\n")
	if code, _, errOut := runConfig(t, "", "unset", "commit.threshold", "--yes", "--root", root); code != 0 || !strings.Contains(errOut, "- threshold = 5") {
		t.Fatalf("%d %s", code, errOut)
	}
	if got := readConfig(t, root); got != "" {
		t.Fatalf("file:\n%s", got)
	}
}

// The word default is a value like any other: a string key would store it,
// a number key refuses it, and it never stands for "remove the line".
func TestConfigSetOfTheWordDefaultIsAValue(t *testing.T) {
	root := configRoot(t, "[commit]\nthreshold = 5\n")
	code, _, errOut := runConfig(t, "", "set", "commit.threshold", "default", "--yes", "--root", root)
	if code != 1 || !strings.Contains(errOut, "whole number") {
		t.Fatalf("%d %s", code, errOut)
	}
	if got := readConfig(t, root); got != "[commit]\nthreshold = 5\n" {
		t.Fatalf("file:\n%s", got)
	}
}

func TestConfigUnsetAsksAndHonoursNo(t *testing.T) {
	root := configRoot(t, "[commit]\nthreshold = 5\n")
	code, _, errOut := runConfig(t, "n\n", "unset", "commit.threshold", "--root", root)
	if code != 0 || !strings.Contains(errOut, "declined") || readConfig(t, root) != "[commit]\nthreshold = 5\n" {
		t.Fatalf("%d %s", code, errOut)
	}
}

func TestConfigUnsetOfAnAbsentKeyWritesNothing(t *testing.T) {
	root := configRoot(t, "[commit]\nthreshold = 5\n")
	code, _, errOut := runConfig(t, "", "unset", "commit.language", "--yes", "--root", root)
	if code != 0 || !strings.Contains(errOut, "already so") {
		t.Fatalf("%d %s", code, errOut)
	}
}

func TestConfigUnsetRefusesUnknownKeysAndTables(t *testing.T) {
	root := configRoot(t, "")
	for _, id := range []string{"commit.nope", "verify.profiles", "commit.allow"} {
		if code, _, errOut := runConfig(t, "", "unset", id, "--yes", "--root", root); code != 1 || errOut == "" {
			t.Errorf("%s: %d %q", id, code, errOut)
		}
	}
}

// Taking a brain key out needs no [area]: the line is inert without one, and
// removing it is the tidy-up the refusal of set would otherwise block.
func TestConfigUnsetTakesABrainKeyWithoutAnArea(t *testing.T) {
	root := configRoot(t, "[layout]\nwiki = \"w\"\n")
	if code, _, errOut := runConfig(t, "", "unset", "layout.wiki", "--yes", "--root", root); code != 0 {
		t.Fatalf("%d %s", code, errOut)
	}
	if got := readConfig(t, root); got != "" {
		t.Fatalf("file:\n%s", got)
	}
}

// The text after the removal goes through the readers like any set, and
// through the same refusal of a shape the editor cannot place; taking out
// the value a reader refuses is what repairs the file.
func TestConfigUnsetRefusesWhatAReaderOrTheEditorRefuses(t *testing.T) {
	const bad = "[commit]\nlanguage = \"fr\"\nthreshold = 5\n"
	root := configRoot(t, bad)
	code, _, errOut := runConfig(t, "", "unset", "commit.threshold", "--yes", "--root", root)
	if code != 1 || !strings.Contains(errOut, "language") || readConfig(t, root) != bad {
		t.Fatalf("%d %s", code, errOut)
	}
	if code, _, errOut := runConfig(t, "", "unset", "commit.language", "--yes", "--root", root); code != 0 {
		t.Fatalf("repair: %d %s", code, errOut)
	}
	const twice = "[commit]\nthreshold = 1\n[commit]\nlanguage = \"en\"\n"
	root = configRoot(t, twice)
	if code, _, errOut := runConfig(t, "", "unset", "commit.threshold", "--yes", "--root", root); code != 1 || !strings.Contains(errOut, "guessing") {
		t.Fatalf("twice: %d %s", code, errOut)
	}
}

func TestConfigUnsetReportsAFileItCannotRead(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := runConfig(t, "", "unset", "commit.threshold", "--yes", "--root", root); code != 1 || errOut == "" {
		t.Fatalf("%d %q", code, errOut)
	}
}

// A file saved with a byte order mark is edited like any other, and keeps it.
func TestConfigSetKeepsAByteOrderMark(t *testing.T) {
	bom := string(rune(0xFEFF))
	root := configRoot(t, bom+"[area]\nscope = \"s\"\n")
	if code, _, errOut := runConfig(t, "", "set", "layout.wiki", "w", "--yes", "--root", root); code != 0 {
		t.Fatalf("%d %s", code, errOut)
	}
	if got := readConfig(t, root); got != bom+"[area]\nscope = \"s\"\n\n[layout]\nwiki = \"w\"\n" {
		t.Fatalf("file:\n%q", got)
	}
}

// A project found upwards from the working directory is the target when no
// --root is given; the file is created where it is missing.
func TestConfigSetFindsTheProjectAndCreatesTheSection(t *testing.T) {
	root := configRoot(t, "")
	t.Chdir(root)
	if code, _, errOut := runConfig(t, "", "set", "commit.threshold", "4", "--yes"); code != 0 {
		t.Fatalf("%d %s", code, errOut)
	}
	if got := readConfig(t, root); got != "[commit]\nthreshold = 4\n" {
		t.Fatalf("file:\n%s", got)
	}
}

func TestConfigOutsideAProjectFails(t *testing.T) {
	t.Chdir(t.TempDir())
	if code, _, errOut := runConfig(t, "", "list"); code != 1 || errOut == "" {
		t.Fatalf("%d %q", code, errOut)
	}
}

func TestConfigSetReportsAFailedWrite(t *testing.T) {
	// .loomux is a file, so the directory for config.toml cannot be made.
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".loomux"), "")
	if code, _, errOut := runConfig(t, "", "set", "commit.threshold", "4", "--yes", "--root", root); code != 1 || errOut == "" {
		t.Fatalf("%d %q", code, errOut)
	}
}

// changingReader rewrites path when the answer is read, the way a second
// terminal would between the diff and the confirmation.
type changingReader struct {
	t          *testing.T
	path, text string
	done       bool
}

func (r *changingReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	writeFile(r.t, r.path, r.text)
	return copy(p, "y\n"), nil
}

func TestConfigSetRefusesAFileChangedBeforeTheAnswer(t *testing.T) {
	root := configRoot(t, "[commit]\nthreshold = 2\n")
	path := filepath.Join(root, ".loomux", "config.toml")
	var out, errOut bytes.Buffer
	in := &changingReader{t: t, path: path, text: "[commit]\nthreshold = 5\n"}
	code := Run([]string{"config", "set", "commit.threshold", "3", "--root", root}, in, &out, &errOut)
	if code != 1 || !strings.Contains(errOut.String(), "changed since it was read") {
		t.Fatalf("code %d, stderr %q; want a refusal", code, errOut.String())
	}
	if got := readConfig(t, root); got != "[commit]\nthreshold = 5\n" {
		t.Fatalf("file = %q, want the other writer's text", got)
	}
}

func TestWriteConfigRefusesATextItWasNotMadeFrom(t *testing.T) {
	_, target := projectTarget(t, "[commit]\nthreshold = 5\n")
	err := writeConfig(target, "[commit]\nthreshold = 2\n", "[commit]\nthreshold = 3\n")
	if err == nil || !strings.Contains(err.Error(), "changed since it was read; nothing written") {
		t.Fatalf("err = %v, want a refusal", err)
	}
	// A file that went missing since it was read changed too.
	os.Remove(target.path)
	if err := writeConfig(target, "[commit]\nthreshold = 5\n", "x"); err == nil {
		t.Fatal("a vanished file was written over")
	}
	if _, err := os.Stat(target.path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stat = %v, want the file still missing", err)
	}
}

func TestWriteConfigReportsAFileItCannotRead(t *testing.T) {
	_, target := projectTarget(t, "")
	os.Remove(target.path)
	os.Mkdir(target.path, 0o755)
	if err := writeConfig(target, "", "x"); err == nil {
		t.Fatal("no error for a configuration that is a directory")
	}
}

func TestConfigSetRefusesWhatAReaderRefuses(t *testing.T) {
	root := configRoot(t, "")
	code, _, errOut := runConfig(t, "", "set", "commit.language", "fr", "--yes", "--root", root)
	if code != 1 || !strings.Contains(errOut, "language") {
		t.Fatalf("%d %s", code, errOut)
	}
}

func TestConfigSetRefusesAMalformedValue(t *testing.T) {
	root := configRoot(t, "")
	code, _, errOut := runConfig(t, "", "set", "commit.threshold", "abc", "--yes", "--root", root)
	if code != 1 || !strings.Contains(errOut, "whole number") {
		t.Fatalf("%d %s", code, errOut)
	}
}

func TestConfigSetRefusesATable(t *testing.T) {
	root := configRoot(t, "")
	// A table and a list of tables alike, each by its name.
	for _, id := range []string{"verify.profiles", "commit.allow"} {
		code, _, errOut := runConfig(t, "", "set", id, "x", "--yes", "--root", root)
		if code != 1 || !strings.Contains(errOut, id+" is a table") {
			t.Fatalf("%s: %d %s", id, code, errOut)
		}
	}
}

func TestConfigSetRefusesAnAmbiguousFile(t *testing.T) {
	root := configRoot(t, "commit.language = \"en\"\n")
	code, _, errOut := runConfig(t, "", "set", "commit.language", "de", "--yes", "--root", root)
	if code != 1 || !strings.Contains(errOut, "by hand") {
		t.Fatalf("%d %s", code, errOut)
	}
}

// Without [area] no reader looks at the brain's keys, so a bad value there
// would pass unchecked; area.scope is the one that opens the section.
func TestConfigSetRefusesABrainKeyWithoutAnArea(t *testing.T) {
	root := configRoot(t, "")
	code, _, errOut := runConfig(t, "", "set", "privacy.mode", "local_only", "--yes", "--root", root)
	if code != 1 || !strings.Contains(errOut, "area.scope") || readConfig(t, root) != "" {
		t.Fatalf("%d %s", code, errOut)
	}
	if code, _, errOut := runConfig(t, "", "set", "area.scope", "s", "--yes", "--root", root); code != 0 {
		t.Fatalf("area.scope: %d %s", code, errOut)
	}
	if code, _, errOut := runConfig(t, "", "set", "privacy.mode", "local_only", "--yes", "--root", root); code != 0 {
		t.Fatalf("after area.scope: %d %s", code, errOut)
	}
	if got := readConfig(t, root); got != "[area]\nscope = \"s\"\n\n[privacy]\nmode = \"local_only\"\n" {
		t.Fatalf("file:\n%s", got)
	}
}

func TestConfigSetOnAFileThatIsNoTOMLNamesTheParse(t *testing.T) {
	root := configRoot(t, "[area\n")
	code, _, errOut := runConfig(t, "", "set", "privacy.mode", "local_only", "--yes", "--root", root)
	if code != 1 || strings.Contains(errOut, "area.scope") || !strings.Contains(errOut, "TOML") {
		t.Fatalf("%d %s", code, errOut)
	}
}

func TestConfigGlobalEditsTheModelBlock(t *testing.T) {
	state := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", state)
	if code, out, _ := runConfig(t, "", "list", "--global"); code != 0 || !strings.Contains(out, "model.endpoint") || !strings.Contains(out, "http://127.0.0.1:11434") {
		t.Fatalf("%d %q", code, out)
	}
	// No [area] is asked for: the global file has none.
	if code, _, errOut := runConfig(t, "", "set", "model.enabled", "true", "--yes", "--global"); code != 0 {
		t.Fatalf("%d %s", code, errOut)
	}
	if code, _, _ := runConfig(t, "", "set", "model.temperature", "0.3", "--yes", "--global"); code != 0 {
		t.Fatal(code)
	}
	data, _ := os.ReadFile(filepath.Join(state, "config.toml"))
	if !strings.Contains(string(data), "enabled = true") || !strings.Contains(string(data), "temperature = 0.3") {
		t.Fatalf("%q", data)
	}
	// The reader in operation judges the value.
	if code, _, errOut := runConfig(t, "", "set", "model.temperature", "3", "--yes", "--global"); code != 1 || !strings.Contains(errOut, "between 0 and 2") {
		t.Fatalf("%d %s", code, errOut)
	}
	// A project key is still no global key.
	if code, _, _ := runConfig(t, "", "set", "commit.threshold", "4", "--yes", "--global"); code != 1 {
		t.Fatal(code)
	}
}

// The global file is guarded like the client it feeds: an address off the
// loopback is refused before it is written, not only when a pass reads it.
func TestConfigGlobalRefusesAnEndpointOffTheLoopback(t *testing.T) {
	state := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", state)
	path := filepath.Join(state, "config.toml")
	if err := os.WriteFile(path, []byte("[model]\nenabled = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := runConfig(t, "", "set", "model.endpoint", "http://192.0.2.1:11434", "--yes", "--global")
	if code != 1 || !strings.Contains(errOut, "loopback") {
		t.Fatalf("%d %s", code, errOut)
	}
	if data, _ := os.ReadFile(path); string(data) != "[model]\nenabled = true\n" {
		t.Fatalf("the file changed: %q", data)
	}
	if code, _, errOut := runConfig(t, "", "set", "model.endpoint", "http://localhost:11434", "--yes", "--global"); code != 0 {
		t.Fatalf("%d %s", code, errOut)
	}
	if data, _ := os.ReadFile(path); !strings.Contains(string(data), `endpoint = "http://localhost:11434"`) {
		t.Fatalf("%q", data)
	}
	if code, _, errOut := runConfig(t, "", "unset", "model.endpoint", "--yes", "--global"); code != 0 {
		t.Fatalf("%d %s", code, errOut)
	}
}

// A global file that is no TOML is named by its own path, not the project's.
func TestConfigGlobalNamesItsOwnFileInAParseError(t *testing.T) {
	state := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", state)
	if err := os.WriteFile(filepath.Join(state, "config.toml"), []byte("[model\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := runConfig(t, "", "list", "--global")
	if code != 1 || !strings.Contains(errOut, filepath.Join(state, "config.toml")) || strings.Contains(errOut, ".loomux/config.toml") {
		t.Fatalf("%d %s", code, errOut)
	}
}

func TestConfigUsage(t *testing.T) {
	// Outside any project: a wrong call is judged before the file is sought.
	t.Chdir(t.TempDir())
	for _, args := range [][]string{{"nope"}, {"get"}, {"set", "x"}, {"unset"}, {"unset", "a", "b"}, {"list", "--global", "--root", "x"}, {"list", "--bogus"}} {
		if code, _, _ := runConfig(t, "", args...); code != 2 {
			t.Errorf("%v: code %d, want 2", args, code)
		}
	}
}

// A long value would widen the value column for every row; the list cuts
// it, and get and --json still give it whole.
func TestConfigListShortensALongValue(t *testing.T) {
	root := configRoot(t, "")
	_, out, _ := runConfig(t, "", "list", "--root", root)
	_, whole, _ := runConfig(t, "", "get", "verify.profiles", "--root", root)
	whole = strings.TrimSuffix(whole, "\n")
	if len([]rune(whole)) <= listWidth {
		t.Fatalf("verify.profiles is no longer long: %q", whole)
	}
	cut := string([]rune(whole)[:listWidth-1]) + "…"
	if !strings.Contains(out, cut+"  ") || strings.Contains(out, whole) {
		t.Errorf("list does not cut the value to %q:\n%s", cut, out)
	}
	_, asJSON, _ := runConfig(t, "", "list", "--json", "--root", root)
	if !strings.Contains(asJSON, "precommit") || !strings.Contains(asJSON, "stop") {
		t.Errorf("--json cut the value:\n%s", asJSON)
	}
	if got := shorten("short"); got != "short" {
		t.Errorf("shorten(short) = %q", got)
	}
}

// Flags may come before the subcommand, the way --root is typed first when
// a human reaches for the project before the verb.
func TestConfigTakesFlagsBeforeTheSubcommand(t *testing.T) {
	root := configRoot(t, "[commit]\nlanguage = \"de\"\n")
	if code, out, _ := runConfig(t, "", "--root", root, "get", "commit.language"); code != 0 || out != "\"de\"\n" {
		t.Fatalf("get: %d %q", code, out)
	}
	if code, _, errOut := runConfig(t, "", "--root", root, "set", "commit.threshold", "7", "--yes"); code != 0 {
		t.Fatalf("set: %d %s", code, errOut)
	}
	if got := readConfig(t, root); got != "[commit]\nlanguage = \"de\"\nthreshold = 7\n" {
		t.Fatalf("file:\n%s", got)
	}
}

// A flag the subcommand has no use for is a wrong call, not one to ignore:
// `get --yes` reads as if it could write.
func TestConfigRefusesAFlagTheSubcommandDoesNotTake(t *testing.T) {
	root := configRoot(t, "[commit]\nlanguage = \"de\"\n")
	for _, args := range [][]string{
		{"get", "commit.language", "--yes"},
		{"get", "commit.language", "--json"},
		{"list", "--yes"},
		{"list", "--propose"},
		{"proposals", "--yes"},
		{"reject", "x", "--yes"},
		{"apply", "x", "--json"},
		{"unset", "commit.language", "--json"},
		{"--yes"},
	} {
		if code, _, _ := runConfig(t, "", append(args, "--root", root)...); code != 2 {
			t.Errorf("%v: code %d, want 2", args, code)
		}
	}
	if got := readConfig(t, root); got != "[commit]\nlanguage = \"de\"\n" {
		t.Fatalf("file changed:\n%s", got)
	}
}

func TestConfigWithoutATerminalNamesTheOtherForms(t *testing.T) {
	// Chdir into a project so the target resolves. The seam stands in for
	// the console: a human running go test in one would otherwise have it
	// put into raw mode.
	restore := openTerminal
	openTerminal = func() (tui.Terminal, func() error, error) { return nil, nil, errors.New("not a terminal") }
	t.Cleanup(func() { openTerminal = restore })
	t.Chdir(configRoot(t, ""))
	code, _, errOut := runConfig(t, "")
	if code != 2 || !strings.Contains(errOut, "config list") {
		t.Fatalf("%d %s", code, errOut)
	}
}
