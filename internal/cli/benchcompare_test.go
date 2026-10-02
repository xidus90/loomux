package cli

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/dev/benchcompare"
	"github.com/xidus90/loomux/internal/dev/benchhooks"
	"github.com/xidus90/loomux/internal/dev/benchreport"
)

// writeBenchReport writes a report of the given timings and returns its path.
func writeBenchReport(t *testing.T, name string, timings ...benchreport.Timing) string {
	t.Helper()
	data, err := benchreport.Report{Schema: benchreport.Schema, Command: "hooks", Timings: timings}.JSON()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func timing(name string, cold, warm float64) benchreport.Timing {
	return benchreport.Timing{Name: name, ColdMS: cold, WarmMS: []float64{warm, warm}, MedianMS: warm, MinMS: warm, MaxMS: warm}
}

// comparePair is a pair of reports: a is 4 times faster after, b is new, c is gone.
func comparePair(t *testing.T) (string, string) {
	t.Helper()
	before := writeBenchReport(t, "before.json", timing("a", 40, 40), timing("c", 9, 9))
	after := writeBenchReport(t, "after.json", timing("a", 10, 10), timing("b", 5, 5))
	return before, after
}

func TestDevBenchCompareSetsTwoRunsSideBySide(t *testing.T) {
	before, after := comparePair(t)
	code, out, errOut := run("dev", "bench", "compare", "--before", before, "--after", after,
		"--title", "Beispielprojekt 1", "--lang", "de")
	if code != 0 || errOut != "" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	for _, want := range []string{"# Beispielprojekt 1\n", "1 verglichen, 1 schneller", "1 neu, 1 weggefallen",
		"| a |", "4.00×", "## Neu", "| b |", "## Weggefallen", "| c |"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout lacks %q:\n%s", want, out)
		}
	}
}

func TestDevBenchCompareReadsBeforeFirstAndAfterSecond(t *testing.T) {
	before, after := comparePair(t)
	_, out, _ := run("dev", "bench", "compare", "--before", after, "--after", before)
	// Swapped, a is four times slower and c is new.
	if !strings.Contains(out, "0.25×") || !strings.Contains(out, "1 langsamer") {
		t.Fatalf("stdout:\n%s", out)
	}
}

func TestDevBenchCompareSpeaksGermanUnlessToldOtherwise(t *testing.T) {
	before, after := comparePair(t)
	_, out, _ := run("dev", "bench", "compare", "--before", before, "--after", after)
	if !strings.Contains(out, "## Verglichen") {
		t.Fatalf("default language is not German:\n%s", out)
	}
	_, out, _ = run("dev", "bench", "compare", "--before", before, "--after", after, "--lang", "en")
	if !strings.Contains(out, "## Compared") || strings.Contains(out, "Verglichen") {
		t.Fatalf("english:\n%s", out)
	}
}

func TestDevBenchCompareHasADefaultTitle(t *testing.T) {
	before, after := comparePair(t)
	_, out, _ := run("dev", "bench", "compare", "--before", before, "--after", after)
	if !strings.HasPrefix(out, "# loomux dev bench compare\n") {
		t.Fatalf("stdout:\n%s", out)
	}
}

func TestDevBenchCompareNeedsBothReports(t *testing.T) {
	before, after := comparePair(t)
	for name, args := range map[string][]string{
		"before":  {"--after", after},
		"after":   {"--before", before},
		"neither": {},
	} {
		code, out, errOut := run(append([]string{"dev", "bench", "compare"}, args...)...)
		if code != 2 || out != "" || !strings.Contains(errOut, "loomux dev bench compare: --before and --after are required") {
			t.Fatalf("without %s: code %d, out %q, err %q", name, code, out, errOut)
		}
	}
}

func TestDevBenchCompareRefusesBadUsage(t *testing.T) {
	before, after := comparePair(t)
	code, _, errOut := run("dev", "bench", "compare", "--before", before, "--after", after, "--nope")
	if code != 2 || !strings.Contains(errOut, "flag provided but not defined: -nope") {
		t.Fatalf("unknown flag: code %d, err %q", code, errOut)
	}
	code, _, errOut = run("dev", "bench", "compare", "--before", before, "--after", after, "stray")
	if code != 2 || !strings.Contains(errOut, `loomux dev bench compare: unexpected argument "stray"`) {
		t.Fatalf("stray argument: code %d, err %q", code, errOut)
	}
}

// An unknown language is a mistake of the call: it stops before a file is read.
func TestDevBenchCompareRefusesAnUnknownLanguageBeforeReading(t *testing.T) {
	code, _, errOut := run("dev", "bench", "compare", "--before", "gone-1.json", "--after", "gone-2.json", "--lang", "fr")
	if code != 2 || !strings.Contains(errOut, "loomux dev bench compare:") || !strings.Contains(errOut, `unknown language "fr"`) {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevBenchCompareReportsUnreadableReports(t *testing.T) {
	good, _ := comparePair(t)
	notJSON := filepath.Join(t.TempDir(), "x.json")
	if err := os.WriteFile(notJSON, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	gone := filepath.Join(t.TempDir(), "gone.json")
	for name, args := range map[string][]string{
		"before not json": {"--before", notJSON, "--after", good},
		"after not json":  {"--before", good, "--after", notJSON},
		"before missing":  {"--before", gone, "--after", good},
		"after missing":   {"--before", good, "--after", gone},
	} {
		code, out, errOut := run(append([]string{"dev", "bench", "compare"}, args...)...)
		if code != 1 || out != "" || !strings.Contains(errOut, "loomux dev bench compare: ") {
			t.Fatalf("%s: code %d, out %q, err %q", name, code, out, errOut)
		}
		bad := notJSON
		if strings.HasSuffix(name, "missing") {
			bad = gone
		}
		if !strings.Contains(errOut, bad) {
			t.Fatalf("%s: err %q does not name %s", name, errOut, bad)
		}
		// The reason is the one of the read or of the parse, not a later
		// complaint about what an empty or broken file decodes to.
		reason := "unexpected end of JSON input"
		if bad == gone {
			reason = readError(t, gone)
		}
		if !strings.Contains(errOut, reason) {
			t.Fatalf("%s: err %q does not give the reason %q", name, errOut, reason)
		}
	}
}

// readError is the text of the error os.ReadFile gives for path.
func readError(t *testing.T, path string) string {
	t.Helper()
	_, err := os.ReadFile(path)
	if err == nil {
		t.Fatalf("%s can be read", path)
	}
	return err.Error()
}

func TestDevBenchCompareRefusesAReportOfAnotherSchema(t *testing.T) {
	good, _ := comparePair(t)
	other := filepath.Join(t.TempDir(), "other.json")
	if err := os.WriteFile(other, []byte(`{"schema": 99, "timings": []}`), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, args := range map[string][]string{
		"before": {"--before", other, "--after", good},
		"after":  {"--before", good, "--after", other},
	} {
		code, _, errOut := run(append([]string{"dev", "bench", "compare"}, args...)...)
		if code != 1 || !strings.Contains(errOut, "schema 99") || !strings.Contains(errOut, other) {
			t.Fatalf("%s: code %d, err %q", name, code, errOut)
		}
	}
	// An older schema is another one too.
	older := filepath.Join(t.TempDir(), "older.json")
	if err := os.WriteFile(older, []byte(`{"schema": 0, "timings": []}`), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := run("dev", "bench", "compare", "--before", older, "--after", good)
	if code != 1 || !strings.Contains(errOut, "schema 0") {
		t.Fatalf("older: code %d, err %q", code, errOut)
	}
}

func TestDevBenchCompareRefusesACaseNamedTwice(t *testing.T) {
	twice := writeBenchReport(t, "twice.json", timing("a", 1, 1), timing("a", 2, 2))
	good, _ := comparePair(t)
	code, out, errOut := run("dev", "bench", "compare", "--before", twice, "--after", good)
	if code != 1 || out != "" || !strings.Contains(errOut, "loomux dev bench compare:") || !strings.Contains(errOut, `case "a" occurs twice`) {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestDevBenchCompareWritesBothFilesWithOut(t *testing.T) {
	stamp := pinBenchClock(t)
	before, after := comparePair(t)
	dir := t.TempDir()
	code, out, errOut := run("dev", "bench", "compare", "--before", before, "--after", after, "--title", "T", "--out", dir)
	if code != 0 || errOut != "" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	base := filepath.Join(dir, "bench-"+stamp+"-compare")
	text, err := os.ReadFile(base + ".md")
	if err != nil {
		t.Fatal(err)
	}
	if string(text) != out || !strings.HasPrefix(out, "# T\n") {
		t.Fatalf("stdout %q, file %q", out, text)
	}
	data, err := os.ReadFile(base + ".json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(data), "}\n") || !strings.Contains(string(data), "\n  \"Compared\"") {
		t.Fatalf("json is not indented with a final newline:\n%s", data)
	}
	var result benchcompare.Result
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Compared) != 1 || result.Compared[0].Name != "a" || len(result.Added) != 1 || len(result.Dropped) != 1 {
		t.Fatalf("result %+v", result)
	}
}

func TestDevBenchCompareDoesNotOverwriteAnEarlierRun(t *testing.T) {
	stamp := pinBenchClock(t)
	before, after := comparePair(t)
	dir := t.TempDir()
	if code, _, errOut := run("dev", "bench", "compare", "--before", before, "--after", after, "--out", dir); code != 0 {
		t.Fatalf("first run: code %d, err %q", code, errOut)
	}
	base := filepath.Join(dir, "bench-"+stamp+"-compare")
	mdBefore, _ := os.ReadFile(base + ".md")
	jsBefore, _ := os.ReadFile(base + ".json")
	code, out, errOut := run("dev", "bench", "compare", "--before", after, "--after", before, "--out", dir)
	if code != 1 || out != "" || !strings.Contains(errOut, "already exists") {
		t.Fatalf("second run: code %d, out %q, err %q", code, out, errOut)
	}
	mdAfter, _ := os.ReadFile(base + ".md")
	jsAfter, _ := os.ReadFile(base + ".json")
	if string(mdBefore) != string(mdAfter) || string(jsBefore) != string(jsAfter) {
		t.Fatal("the second run changed the first run's files")
	}
}

// The targets are judged before the reports are read.
func TestDevBenchCompareChecksTheTargetsBeforeReading(t *testing.T) {
	stamp := pinBenchClock(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bench-"+stamp+"-compare.json"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := run("dev", "bench", "compare", "--before", "gone-1.json", "--after", "gone-2.json", "--out", dir)
	if code != 1 || !strings.Contains(errOut, "already exists") {
		t.Fatalf("taken target: code %d, err %q", code, errOut)
	}
	code, _, errOut = run("dev", "bench", "compare", "--before", "gone-1.json", "--after", "gone-2.json", "--out", filepath.Join(dir, "none"))
	if code != 1 || !strings.Contains(errOut, "no directory at") {
		t.Fatalf("missing directory: code %d, err %q", code, errOut)
	}
}

func TestDevBenchCompareReportsAFailedWrite(t *testing.T) {
	before, after := comparePair(t)
	orig := benchWriteBoth
	benchWriteBoth = func(string, string, []byte, []byte) error { return errors.New("disk full") }
	t.Cleanup(func() { benchWriteBoth = orig })
	code, _, errOut := run("dev", "bench", "compare", "--before", before, "--after", after, "--out", t.TempDir())
	if code != 1 || !strings.Contains(errOut, "loomux dev bench compare: disk full") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

const benchOldSettings = `{
  "hooks": {
    "SessionStart": [{"hooks": [{"type": "command", "command": "uv run ultraloom hook session-start --root \"${CLAUDE_PROJECT_DIR}\""}]}],
    "PreToolUse": [
      {"matcher": "Write|Edit|Bash", "hooks": [{"type": "command", "command": "ulguard --root \"${CLAUDE_PROJECT_DIR}\""}]},
      {"matcher": "", "hooks": [{"type": "command", "command": "brain guard"}]}
    ],
    "Stop": [{"hooks": [{"type": "command", "command": "brain wiki-gate --root \"${CLAUDE_PROJECT_DIR}\""}]}]
  }
}`

// casesWorld is a settings file, a root and an existing output directory.
type casesWorld struct {
	settings, root, file, out string
}

func newCasesWorld(t *testing.T) casesWorld {
	t.Helper()
	dir := t.TempDir()
	w := casesWorld{
		settings: filepath.Join(dir, "settings.json"),
		root:     filepath.Join(dir, "project"),
		out:      filepath.Join(dir, "out"),
	}
	w.file = filepath.Join(w.root, "README.md")
	if err := os.WriteFile(w.settings, []byte(benchOldSettings), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(w.out, 0o755); err != nil {
		t.Fatal(err)
	}
	return w
}

// args is the call of the command over the world, with more flags behind it.
func (w casesWorld) args(more ...string) []string {
	return append([]string{"dev", "bench", "cases", "--settings", w.settings, "--root", w.root, "--file", w.file, "--out", w.out}, more...)
}

func (w casesWorld) extrasOnly(more ...string) []string {
	return append([]string{"dev", "bench", "cases", "--root", w.root, "--file", w.file, "--out", w.out}, more...)
}

func writeExtras(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "extras.json")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readCaseFile(t *testing.T, dir string) []benchhooks.Case {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []benchhooks.Case
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	return cases
}

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestDevBenchCasesWritesTheCaseFileAndItsPayloads(t *testing.T) {
	w := newCasesWorld(t)
	code, out, errOut := run(w.args()...)
	if code != 0 || out != "" || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	cases := readCaseFile(t, w.out)
	var names []string
	for _, c := range cases {
		names = append(names, c.Name)
	}
	want := []string{"SessionStart", "PreToolUse (Edit on README.md)", "Stop"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names %v, want %v", names, want)
	}
	if got := dirNames(t, w.out); !reflect.DeepEqual(got, []string{"cases.json", "payload-PreToolUse.json", "payload-SessionStart.json", "payload-Stop.json"}) {
		t.Fatalf("files %v", got)
	}
	slashOut := filepath.ToSlash(w.out)
	if cases[0].Stdin != slashOut+"/payload-SessionStart.json" || cases[0].Dir != w.root {
		t.Fatalf("case %+v", cases[0])
	}
	// The payload file is the one the case reads, and it is JSON.
	data, err := os.ReadFile(filepath.Join(w.out, "payload-PreToolUse.json"))
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil || payload["hook_event_name"] != "PreToolUse" {
		t.Fatalf("payload %s (%v)", data, err)
	}
	// The case file is indented and ends in a newline.
	text, _ := os.ReadFile(filepath.Join(w.out, "cases.json"))
	if !strings.Contains(string(text), "\n  {") || !strings.HasSuffix(string(text), "]\n") {
		t.Fatalf("cases.json:\n%s", text)
	}
}

func TestDevBenchCasesCleansTheOutputDirectory(t *testing.T) {
	w := newCasesWorld(t)
	w.out = w.out + string(filepath.Separator) + ".." + string(filepath.Separator) + "out" + string(filepath.Separator)
	if code, _, errOut := run(w.args()...); code != 0 {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	clean := filepath.ToSlash(filepath.Clean(w.out))
	got := readCaseFile(t, w.out)[0].Stdin
	if got != clean+"/payload-SessionStart.json" || strings.Contains(got, "//") || strings.Contains(got, "..") || strings.Contains(got, `\`) {
		t.Fatalf("stdin %q", got)
	}
}

func TestDevBenchCasesAppendsTheExtraCases(t *testing.T) {
	w := newCasesWorld(t)
	extras := writeExtras(t, `[
  {"name": "extra", "dir": "{{ROOT}}", "stdin": "{{OUT}}/payload-x.json", "mode": "single",
   "steps": [{"argv": ["tool", "{{ROOT}}/a", "{{OUT}}", "{{ROOT}}"]}]}
]`)
	code, _, errOut := run(w.args("--extras", extras)...)
	if code != 0 {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	cases := readCaseFile(t, w.out)
	if len(cases) != 4 || cases[3].Name != "extra" {
		t.Fatalf("cases %+v", cases)
	}
	slashOut := filepath.ToSlash(w.out)
	e := cases[3]
	if e.Dir != w.root || e.Stdin != slashOut+"/payload-x.json" ||
		!reflect.DeepEqual(e.Steps[0].Argv, []string{"tool", w.root + "/a", slashOut, w.root}) {
		t.Fatalf("extra %+v", e)
	}
}

// A path with a backslash or a quote must not break the file it is put in.
func TestDevBenchCasesSetsPathsIntoTheExtrasSafely(t *testing.T) {
	w := newCasesWorld(t)
	w.root = `C:\Users\me\my "proj"`
	extras := writeExtras(t, `[{"name": "x", "dir": "{{ROOT}}", "stdin": "", "mode": "single", "steps": [{"argv": ["{{ROOT}}"]}]}]`)
	code, _, errOut := run(w.extrasOnly("--extras", extras)...)
	if code != 0 {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	got := readCaseFile(t, w.out)[0]
	if got.Dir != w.root || got.Steps[0].Argv[0] != w.root {
		t.Fatalf("case %+v, want root %q", got, w.root)
	}
}

func TestDevBenchCasesWithoutSettingsHasOnlyTheExtras(t *testing.T) {
	w := newCasesWorld(t)
	extras := writeExtras(t, `[{"name": "only", "dir": "{{ROOT}}", "stdin": "", "mode": "single", "steps": [{"argv": ["t"]}]}]`)
	code, _, errOut := run(w.extrasOnly("--extras", extras)...)
	if code != 0 || errOut != "" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if got := dirNames(t, w.out); !reflect.DeepEqual(got, []string{"cases.json"}) {
		t.Fatalf("files %v", got)
	}
	if cases := readCaseFile(t, w.out); len(cases) != 1 || cases[0].Name != "only" {
		t.Fatalf("cases %+v", cases)
	}
}

// No case at all is an empty list, not null.
func TestDevBenchCasesWritesAnEmptyListForAnEmptyExtras(t *testing.T) {
	w := newCasesWorld(t)
	code, _, errOut := run(w.extrasOnly("--extras", writeExtras(t, "[]"))...)
	if code != 0 {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if data, _ := os.ReadFile(filepath.Join(w.out, "cases.json")); string(data) != "[]\n" {
		t.Fatalf("cases.json holds %q", data)
	}
}

func TestDevBenchCasesNeedsSettingsOrExtras(t *testing.T) {
	w := newCasesWorld(t)
	code, _, errOut := run(w.extrasOnly()...)
	if code != 2 || !strings.Contains(errOut, "loomux dev bench cases: --settings or --extras is required") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if got := dirNames(t, w.out); len(got) != 0 {
		t.Fatalf("files %v", got)
	}
}

func TestDevBenchCasesNeedsRootFileAndOut(t *testing.T) {
	w := newCasesWorld(t)
	for flag, args := range map[string][]string{
		"--root": {"--settings", w.settings, "--file", w.file, "--out", w.out},
		"--file": {"--settings", w.settings, "--root", w.root, "--out", w.out},
		"--out":  {"--settings", w.settings, "--root", w.root, "--file", w.file},
	} {
		code, _, errOut := run(append([]string{"dev", "bench", "cases"}, args...)...)
		if code != 2 || !strings.Contains(errOut, "loomux dev bench cases: --root, --file and --out are required") {
			t.Fatalf("without %s: code %d, err %q", flag, code, errOut)
		}
	}
}

func TestDevBenchCasesRefusesBadUsage(t *testing.T) {
	w := newCasesWorld(t)
	code, _, errOut := run(w.args("--nope")...)
	if code != 2 || !strings.Contains(errOut, "flag provided but not defined: -nope") {
		t.Fatalf("unknown flag: code %d, err %q", code, errOut)
	}
	code, _, errOut = run(w.args("stray")...)
	if code != 2 || !strings.Contains(errOut, `loomux dev bench cases: unexpected argument "stray"`) {
		t.Fatalf("stray argument: code %d, err %q", code, errOut)
	}
}

func TestDevBenchCasesRefusesADuplicateNameAndWritesNothing(t *testing.T) {
	w := newCasesWorld(t)
	dup := writeExtras(t, `[{"name": "Stop", "dir": ".", "stdin": "", "mode": "single", "steps": [{"argv": ["t"]}]}]`)
	code, _, errOut := run(w.args("--extras", dup)...)
	if code != 1 || !strings.Contains(errOut, "loomux dev bench cases:") || !strings.Contains(errOut, `"Stop"`) {
		t.Fatalf("settings/extras: code %d, err %q", code, errOut)
	}
	twice := writeExtras(t, `[
  {"name": "x", "dir": ".", "stdin": "", "mode": "single", "steps": [{"argv": ["t"]}]},
  {"name": "x", "dir": ".", "stdin": "", "mode": "single", "steps": [{"argv": ["t"]}]}]`)
	code, _, errOut = run(w.extrasOnly("--extras", twice)...)
	if code != 1 || !strings.Contains(errOut, `"x"`) {
		t.Fatalf("extras/extras: code %d, err %q", code, errOut)
	}
	if got := dirNames(t, w.out); len(got) != 0 {
		t.Fatalf("files left behind: %v", got)
	}
}

func TestDevBenchCasesNeedsAnExistingOutputDirectory(t *testing.T) {
	w := newCasesWorld(t)
	w.out = filepath.Join(w.out, "none")
	code, _, errOut := run(w.args()...)
	if code != 1 || !strings.Contains(errOut, "loomux dev bench cases:") || !strings.Contains(errOut, "no directory at") {
		t.Fatalf("missing: code %d, err %q", code, errOut)
	}
	w.out = w.settings // a file, not a directory
	code, _, errOut = run(w.args()...)
	if code != 1 || !strings.Contains(errOut, "no directory at") {
		t.Fatalf("file: code %d, err %q", code, errOut)
	}
}

func TestDevBenchCasesReportsUnreadableInputs(t *testing.T) {
	w := newCasesWorld(t)
	notJSON := writeExtras(t, "{")
	gone := filepath.Join(t.TempDir(), "gone.json")
	badSettings := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(badSettings, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	noHook := filepath.Join(t.TempDir(), "empty.json")
	if err := os.WriteFile(noHook, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	// named is the path each failure has to name, if any.
	for name, c := range map[string]struct {
		args  []string
		named string
	}{
		"settings missing": {[]string{"--settings", gone}, gone},
		"settings bad":     {[]string{"--settings", badSettings}, "settings: not valid JSON"},
		"no hook applies":  {[]string{"--settings", noHook}, "no hook of any event applies"},
		"extras missing":   {[]string{"--settings", w.settings, "--extras", gone}, readError(t, gone)},
		"extras bad":       {[]string{"--settings", w.settings, "--extras", notJSON}, notJSON},
	} {
		code, _, errOut := run(append([]string{"dev", "bench", "cases", "--root", w.root, "--file", w.file, "--out", w.out}, c.args...)...)
		if code != 1 || !strings.Contains(errOut, "loomux dev bench cases: ") || !strings.Contains(errOut, c.named) {
			t.Fatalf("%s: code %d, err %q", name, code, errOut)
		}
		if got := dirNames(t, w.out); len(got) != 0 {
			t.Fatalf("%s: files left behind: %v", name, got)
		}
	}
}

func TestDevBenchCasesReadsTheEnvironmentForAHook(t *testing.T) {
	w := newCasesWorld(t)
	t.Setenv("LOOMUX_BENCH_TEST_TOOL", "/opt/tool")
	settings := `{"hooks": {"Stop": [{"hooks": [{"command": "${LOOMUX_BENCH_TEST_TOOL}/run stop"}]}]}}`
	if err := os.WriteFile(w.settings, []byte(settings), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := run(w.args()...); code != 0 {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	data, err := os.ReadFile(filepath.Join(w.out, "cases.json"))
	if err != nil || !strings.Contains(string(data), `"/opt/tool/run"`) {
		t.Fatalf("cases.json %s, %v", data, err)
	}
}

// All the files of a run are written together or not at all: a name taken
// by any of them, the last included, stops the run before the first write.
func TestDevBenchCasesWritesNothingWhenAFileExists(t *testing.T) {
	for _, taken := range []string{"cases.json", "payload-SessionStart.json", "payload-Stop.json"} {
		w := newCasesWorld(t)
		if err := os.WriteFile(filepath.Join(w.out, taken), []byte("mine"), 0o644); err != nil {
			t.Fatal(err)
		}
		code, _, errOut := run(w.args()...)
		if code != 1 || !strings.Contains(errOut, "already exists") || !strings.Contains(errOut, taken) {
			t.Fatalf("%s: code %d, err %q", taken, code, errOut)
		}
		if got := dirNames(t, w.out); !reflect.DeepEqual(got, []string{taken}) {
			t.Fatalf("%s: files %v", taken, got)
		}
		if data, _ := os.ReadFile(filepath.Join(w.out, taken)); string(data) != "mine" {
			t.Fatalf("%s was overwritten with %q", taken, data)
		}
	}
}

// A write that fails halfway takes back what the run had written.
func TestDevBenchCasesTakesBackWhatItWroteWhenAWriteFails(t *testing.T) {
	for failAt := 1; failAt <= 4; failAt++ {
		w := newCasesWorld(t)
		calls := 0
		orig := benchWriteNew
		benchWriteNew = func(path string, data []byte) error {
			calls++
			if calls == failAt {
				return errors.New("disk full")
			}
			return orig(path, data)
		}
		code, _, errOut := run(w.args()...)
		benchWriteNew = orig
		if code != 1 || !strings.Contains(errOut, "loomux dev bench cases: disk full") {
			t.Fatalf("fail at %d: code %d, err %q", failAt, code, errOut)
		}
		if got := dirNames(t, w.out); len(got) != 0 {
			t.Fatalf("fail at %d: files left behind: %v", failAt, got)
		}
		if calls != failAt {
			t.Fatalf("fail at %d: %d writes, the run went on after the failure", failAt, calls)
		}
	}
}

func TestBenchWriteNewNeverOverwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := benchWriteNew(path, []byte("one")); err != nil {
		t.Fatal(err)
	}
	if err := benchWriteNew(path, []byte("two")); err == nil {
		t.Fatal("wrote over an existing file")
	}
	if data, _ := os.ReadFile(path); string(data) != "one" {
		t.Fatalf("file holds %q", data)
	}
}

// brokenWriter is a file whose write, close or both fail; it counts the
// closes, since a failed write must not leave the file open.
type brokenWriter struct {
	write, close error
	got          []byte
	closed       int
}

func (b *brokenWriter) Write(p []byte) (int, error) {
	b.got = append(b.got, p...)
	return len(p), b.write
}

func (b *brokenWriter) Close() error {
	b.closed++
	return b.close
}

func TestWriteAndCloseReportsTheWriteAndTheClose(t *testing.T) {
	full, late := errors.New("disk full"), errors.New("close failed")
	for name, c := range map[string]struct {
		w    *brokenWriter
		want []error
	}{
		"both succeed":    {&brokenWriter{}, nil},
		"the write fails": {&brokenWriter{write: full}, []error{full}},
		"the close fails": {&brokenWriter{close: late}, []error{late}},
		"both fail":       {&brokenWriter{write: full, close: late}, []error{full, late}},
	} {
		err := writeAndClose(c.w, []byte("data"))
		if (err == nil) != (len(c.want) == 0) {
			t.Errorf("%s: err %v", name, err)
		}
		for _, want := range c.want {
			if !errors.Is(err, want) {
				t.Errorf("%s: err %v lacks %v", name, err, want)
			}
		}
		if string(c.w.got) != "data" || c.w.closed != 1 {
			t.Errorf("%s: wrote %q, closed %d times", name, c.w.got, c.w.closed)
		}
	}
}

// failWriteAndClose makes the write inside writeNewFile fail after half of
// the data reached the file.
func failWriteAndClose(t *testing.T) {
	t.Helper()
	orig := writeAndCloseFile
	writeAndCloseFile = func(w io.WriteCloser, data []byte) error {
		return errors.Join(orig(w, data[:len(data)/2]), errors.New("disk full"))
	}
	t.Cleanup(func() { writeAndCloseFile = orig })
}

// Half a file under the final name would pass for the whole one, and would
// keep the next run from writing it.
func TestWriteNewFileLeavesNoHalfWrittenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	failWriteAndClose(t)
	if err := writeNewFile(path, []byte("whole")); err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("err %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("the half-written file is still there (%v)", err)
	}
}

// The file a failed write leaves is not among the ones the run knows it
// wrote, so the writer itself has to take it back.
func TestDevBenchCasesLeavesNothingWhenAWriteBreaksOff(t *testing.T) {
	w := newCasesWorld(t)
	failWriteAndClose(t)
	code, _, errOut := run(w.args()...)
	if code != 1 || !strings.Contains(errOut, "disk full") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if got := dirNames(t, w.out); len(got) != 0 {
		t.Fatalf("files left behind: %v", got)
	}
}

// The measuring run opens a case's stdin from its own working directory, so
// the case file names it from the root whatever --out was given as.
func TestDevBenchCasesNamesThePayloadsByAnAbsolutePath(t *testing.T) {
	w := newCasesWorld(t)
	abs := filepath.ToSlash(w.out)
	t.Chdir(filepath.Dir(w.out))
	w.out = "out"
	if code, _, errOut := run(w.args()...); code != 0 {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if got := readCaseFile(t, w.out)[0].Stdin; got != abs+"/payload-SessionStart.json" {
		t.Fatalf("stdin %q, want it under %s", got, abs)
	}
	// {{OUT}} of the extras is the same directory.
	w = newCasesWorld(t)
	abs = filepath.ToSlash(w.out)
	t.Chdir(filepath.Dir(w.out))
	w.out = "." + string(filepath.Separator) + "out"
	extras := writeExtras(t, `[{"name": "x", "dir": "{{ROOT}}", "stdin": "{{OUT}}/in.json", "mode": "single", "steps": [{"argv": ["a"]}]}]`)
	if code, _, errOut := run(w.extrasOnly("--extras", extras)...); code != 0 {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if got := readCaseFile(t, w.out)[0].Stdin; got != abs+"/in.json" {
		t.Fatalf("extras stdin %q, want it under %s", got, abs)
	}
}

// Without a working directory to resolve it against, --out stays as given,
// cleaned as before.
func TestDevBenchCasesKeepsOutAsGivenWhenItCannotBeResolved(t *testing.T) {
	w := newCasesWorld(t)
	world := filepath.Dir(w.out)
	t.Chdir(filepath.Dir(world))
	orig := benchAbs
	benchAbs = func(string) (string, error) { return "", errors.New("no working directory") }
	t.Cleanup(func() { benchAbs = orig })
	sep := string(filepath.Separator)
	w.out = filepath.Base(world) + sep + "out" + sep + ".." + sep + "out" + sep
	if code, _, errOut := run(w.args()...); code != 0 {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if got := readCaseFile(t, w.out)[0].Stdin; got != filepath.Base(world)+"/out/payload-SessionStart.json" {
		t.Fatalf("stdin %q", got)
	}
}

// An output directory of the odd kind (a backslash and a quote are legal in
// a name on POSIX) must not break the extras any more than the root does.
func TestBenchReadExtrasSetsBothValuesInSafely(t *testing.T) {
	extras := writeExtras(t, `[{"name": "x", "dir": "{{ROOT}}", "stdin": "{{OUT}}/p.json", "mode": "", "steps": []}]`)
	root, dir := `C:\r "1"`, `/o\u "2"`
	cases, err := benchReadExtras(extras, root, dir)
	if err != nil || len(cases) != 1 || cases[0].Dir != root || cases[0].Stdin != dir+"/p.json" {
		t.Fatalf("cases %+v, err %v", cases, err)
	}
}

func TestBenchJSONStringIsTheInsideOfAJSONString(t *testing.T) {
	for _, s := range []string{`C:\a\b`, `say "hi"`, "tab\there", "é"} {
		var back string
		if err := json.Unmarshal([]byte(`"`+benchJSONString(s)+`"`), &back); err != nil || back != s {
			t.Fatalf("%q came back as %q (%v)", s, back, err)
		}
	}
}
