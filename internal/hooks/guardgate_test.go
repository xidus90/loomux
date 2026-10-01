package hooks

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/verify"
)

const batteryFile = "testdata/guard-battery-before.tsv"

// inPlaceOfConfig is a line about the manifest, turned into the same line
// about the armed lanes.
func inPlaceOfConfig(line string) string {
	return strings.NewReplacer("config.toml", "armed.toml", "Config.toml", "Armed.toml").Replace(line)
}

// armedShellWrites are the manifest's write spellings aimed at the armed
// lanes, and the ways that reach the file through its folder.
func armedShellWrites() []string {
	var lines []string
	for _, line := range manifestShellWrites() {
		lines = append(lines, inPlaceOfConfig(line))
	}
	return append(lines,
		"git restore -- .loomux/armed.toml",
		"git restore --source=HEAD~1 -- .loomux/armed.toml",
		"git checkout HEAD~1 -- .loomux/armed.toml",
		"sed -i '/lint/d' .loomux/armed.toml",
		"rm -f ./.loomux/armed.toml",
		"Remove-Item -Force .loomux/armed.toml",
		"echo 'armed = []' > .loomux/armed.toml",
	)
}

// armedShellReads read the armed lanes in ways the manifest's reads do not
// name; they pass before the rule and after it.
func armedShellReads() []string {
	return []string{
		"git diff .loomux/armed.toml",
		"git diff HEAD -- .loomux/armed.toml",
		"git log -p -- .loomux/armed.toml",
		"git show HEAD~1:.loomux/armed.toml > old.toml",
		"type .loomux\\armed.toml",
	}
}

// folderRemovals take the file with its folder; they were refused for the
// manifest's sake before the armed lanes existed.
func folderRemovals() []string {
	return []string{
		"rm -rf .loomux",
		"rm -r .loomux/",
		"Remove-Item -Recurse -Force .loomux",
		"git clean -fdx",
		"mv .loomux elsewhere",
	}
}

// knownGaps are spellings that can disarm a lane and pass, before this rule
// and after it: the armed lanes share them with the manifest, and this
// change does not close them. They stand in the battery as recorded passes,
// so that nobody takes them for closed and a later fix flips them on purpose.
//   - git checkout <rev> -- <folder> and git restore -s <rev> <folder> bring
//     back an older file through the folder above it; the guard judges a
//     folder as a write target only for a removal.
//   - git stash, git reset --hard and git switch rewrite the working tree
//     without naming a path.
func knownGaps() []string {
	return []string{
		"git checkout HEAD~1 -- .loomux",
		"git checkout HEAD~1 -- .",
		"git restore -s HEAD~1 .",
		"git stash",
		"git reset --hard",
		"git switch other-branch",
	}
}

// gateLines are calls of the gate group, with what the guard says after the
// change: true for a refusal.
func gateLines() map[string]bool {
	lines := map[string]bool{
		"loomux gate status":                         false,
		"loomux gate status --root .":                false,
		"bin/loomux.exe gate status":                 false,
		"loomux gate":                                false,
		"echo \"loomux gate arm lint/go@.\" > notes": false,
		// A block's closing brace glued to the last word is no part of it.
		"{ loomux gate status}":                          false,
		"1..3 | ForEach-Object { loomux gate status}":    false,
		"loomux gate arm lint/go@.":                      true,
		"loomux gate disarm lint/go@.":                   true,
		"loomux gate disarm --all":                       true,
		"loomux gate --root . arm lint/go@.":             true,
		"loomux gate status; loomux gate arm x":          true,
		"loomux gate status && loomux gate disarm --all": true,
		"cd x && loomux gate arm lint/go@.":              true,
		"{ loomux gate disarm --all}":                    true,
		// A shell's string is read as the configuration rule reads it.
		`sh -c "loomux gate disarm --all"`: true,
		`sh -c "loomux gate status"`:       false,
	}
	for _, row := range loomuxSpellings() {
		lines[inPlaceOfCommand(nil, row, "gate arm lint/go@.", `ga\te arm lint/go@.`, "ga`te arm lint/go@.", `ga^te arm lint/go@.`)] = true
	}
	return lines
}

// battery is every line the guard is asked about, each once, in a fixed order.
func battery() []string {
	lines := slices.Concat(manifestShellWrites(), manifestShellReads(), armedShellWrites(), armedShellReads(), folderRemovals(), knownGaps())
	for _, line := range manifestShellReads() {
		lines = append(lines, inPlaceOfConfig(line))
	}
	for line := range gateLines() {
		lines = append(lines, line)
	}
	slices.Sort(lines)
	return slices.Compact(lines)
}

// refusedBy says whether the guard refuses line from Bash.
func refusedBy(root, line string) bool {
	return len(checkTool(root, "Bash", map[string]any{"command": line}, config.Policy{})) > 0
}

// TestRecordTheGuardBattery writes what the guard says about every line of
// the battery. It runs only when asked, once, at the state before a change to
// the guard; the file it writes is the "before" the differential test reads.
func TestRecordTheGuardBattery(t *testing.T) {
	if os.Getenv("LOOMUX_RECORD_GUARD_BATTERY") != "1" {
		t.Skip("set LOOMUX_RECORD_GUARD_BATTERY=1 to record the guard's verdicts")
	}
	root := t.TempDir()
	var b strings.Builder
	for _, line := range battery() {
		verdict := "pass"
		if refusedBy(root, line) {
			verdict = "refused"
		}
		b.WriteString(verdict + "\t" + strconv.Quote(line) + "\n")
	}
	if err := os.MkdirAll(filepath.Dir(batteryFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(batteryFile, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// readBattery is the recorded verdict of every line, by line.
func readBattery(t *testing.T) map[string]bool {
	t.Helper()
	data, err := os.ReadFile(batteryFile)
	if err != nil {
		t.Fatal(err)
	}
	before := map[string]bool{}
	for _, row := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		verdict, quoted, _ := strings.Cut(row, "\t")
		line, err := strconv.Unquote(quoted)
		if err != nil {
			t.Fatalf("%q: %v", row, err)
		}
		before[line] = verdict == "refused"
	}
	return before
}

// The differential probe: no line the guard refused before passes now, and
// the lines that flipped to a refusal are exactly the writes of the armed
// lanes and the calls that arm or disarm.
func TestTheGuardOpensNothingItRefusedBefore(t *testing.T) {
	before := readBattery(t)
	lines := battery()
	if len(before) != len(lines) {
		t.Fatalf("the battery has %d lines, the record %d: record it again at the state before the change", len(lines), len(before))
	}
	for _, line := range lines {
		if _, recorded := before[line]; !recorded {
			t.Errorf("%q is not in the record: record it again at the state before the change", line)
		}
	}
	root := t.TempDir()
	flips := map[string]bool{}
	for _, line := range armedShellWrites() {
		flips[line] = true
	}
	for line, refused := range gateLines() {
		if refused {
			flips[line] = true
		}
	}
	for line, was := range before {
		now := refusedBy(root, line)
		switch {
		case was && !now:
			t.Errorf("%q was refused and passes now", line)
		case !was && now && !flips[line]:
			t.Errorf("%q passed and is refused now, and is no write of the armed lanes", line)
		case !was && !now && flips[line]:
			t.Errorf("%q still passes", line)
		}
	}
}

// The gaps the rule names are gaps: each passed before and passes now. One
// that is refused here was closed, and then it leaves knownGaps, the comment
// of armedReason and the docs together.
func TestTheNamedGapsAreStillOpen(t *testing.T) {
	before := readBattery(t)
	root := t.TempDir()
	for _, line := range knownGaps() {
		was, recorded := before[line]
		if !recorded || was || refusedBy(root, line) {
			t.Errorf("%q: recorded %v, refused before %v, refused now %v", line, recorded, was, refusedBy(root, line))
		}
	}
}

// The path rule is a second copy of verify.ArmedFile, a verbatim glob: it
// keeps the file the gate reads, in either separator, and follows it should
// the file move.
func TestTheArmedRuleKeepsTheFileTheGateReads(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{verify.ArmedFile, filepath.FromSlash(verify.ArmedFile), filepath.Join(root, filepath.FromSlash(verify.ArmedFile))} {
		if got := checkTool(root, "Write", map[string]any{"file_path": path}, config.Policy{}); !slices.Equal(got, []string{armedReason}) {
			t.Errorf("%s: reasons %q, want the armed lanes' reason", path, got)
		}
	}
}

// Both refusals say the one way an agent arms a lane: a green pre-commit run
// with --arm, which a commit makes.
func TestTheRefusalsNameTheWayAnAgentArms(t *testing.T) {
	const way = "an agent arms a lane only through a green `loomux check precommit --arm`"
	for name, reason := range map[string]string{"armedReason": armedReason, "gateReason": gateReason} {
		if !strings.Contains(reason, way) {
			t.Errorf("%s = %q, want it to say %q", name, reason, way)
		}
	}
}

func TestTheArmedLanesAreKeptFromEveryWritingTool(t *testing.T) {
	root := t.TempDir()
	for _, tool := range []string{"Write", "Edit"} {
		for _, path := range []string{".loomux/armed.toml", ".LOOMUX/Armed.toml", filepath.Join(root, ".loomux", "armed.toml"), "../other/.loomux/armed.toml"} {
			if got := checkTool(root, tool, map[string]any{"file_path": path}, config.Policy{}); !slices.Equal(got, []string{armedReason}) {
				t.Errorf("%s %s: reasons %q", tool, path, got)
			}
		}
	}
	for _, path := range []string{".loomux/armed.toml.bak", "docs/armed.toml", ".loomux/armedx.toml"} {
		if got := checkTool(root, "Write", map[string]any{"file_path": path}, config.Policy{}); len(got) != 0 {
			t.Errorf("%s: reasons %q, want none", path, got)
		}
	}
}

func TestTheShellPathCheckKeepsTheArmedLanes(t *testing.T) {
	root := t.TempDir()
	for _, line := range armedShellWrites() {
		if got := shellReasons(t, root, line, config.Policy{}); !slices.Equal(got, []string{armedReason}) {
			t.Errorf("%q: reasons %q, want the armed lanes'", line, got)
		}
	}
	for _, line := range manifestShellReads() {
		if got := shellReasons(t, root, inPlaceOfConfig(line), config.Policy{}); len(got) != 0 {
			t.Errorf("%q: reasons %q, want none", inPlaceOfConfig(line), got)
		}
	}
	for _, line := range armedShellReads() {
		if got := shellReasons(t, root, line, config.Policy{}); len(got) != 0 {
			t.Errorf("%q: reasons %q, want none", line, got)
		}
	}
	// A removal of the folder takes both files with it and names both.
	if got := shellReasons(t, root, "rm -rf .loomux", config.Policy{}); !slices.Contains(got, armedReason) || !slices.Contains(got, manifestReason) {
		t.Errorf("rm -rf .loomux: reasons %q", got)
	}
	// Without its --, git restore and git checkout take a word for a path only
	// where one stands (checkedOut): the file has to be there to be named.
	for _, line := range []string{"git restore .loomux/armed.toml", "git checkout .loomux/armed.toml"} {
		if got := shellReasons(t, root, line, config.Policy{}); len(got) != 0 {
			t.Errorf("%q without the file: reasons %q", line, got)
		}
	}
	mkfile(t, root, ".loomux/armed.toml")
	for _, line := range []string{"git restore .loomux/armed.toml", "git checkout .loomux/armed.toml", "git restore --worktree --staged .loomux/armed.toml"} {
		if got := shellReasons(t, root, line, config.Policy{}); !slices.Equal(got, []string{armedReason}) {
			t.Errorf("%q: reasons %q", line, got)
		}
	}
	// The index alone is not the file.
	if got := shellReasons(t, root, "git restore --staged .loomux/armed.toml", config.Policy{}); len(got) != 0 {
		t.Errorf("restore --staged: reasons %q", got)
	}
}

func TestAnAgentMayNotArmOrDisarm(t *testing.T) {
	for line, refused := range gateLines() {
		if got := armsOrDisarms(line, false); got != refused && !strings.HasPrefix(strings.ToLower(line), "start") && !strings.HasPrefix(strings.ToLower(line), "saps") {
			t.Errorf("%q: armsOrDisarms = %v, want %v", line, got, refused)
		}
	}
	// Another loomux command with arm or disarm among its words is no gate.
	for _, line := range []string{"loomux check arm", "loomux flow run disarm", "loomux status arm"} {
		if armsOrDisarms(line, false) {
			t.Errorf("%q: armsOrDisarms = true", line)
		}
	}
	// In strict mode a renamed binary is loomux by what it is told.
	if !armsOrDisarms("doc.exe gate disarm --all", true) || armsOrDisarms("doc.exe gate disarm --all", false) {
		t.Error("a renamed binary: strict must refuse, the default mode must not")
	}
	// So is a wrapper named by a path, and a renamed binary behind a wrapper.
	for _, line := range []string{"./nohup gate disarm --all", "winpty doc.exe gate arm x"} {
		if !armsOrDisarms(line, true) || armsOrDisarms(line, false) {
			t.Errorf("%q: strict must refuse, the default mode must not", line)
		}
	}
	// A known program keeps its own gate subcommand in strict mode as well.
	if armsOrDisarms("git gate arm x", true) {
		t.Error("git gate arm: refused in strict mode")
	}
}

// The refusal names the way a human does it, and status stays open.
func TestCheckToolNamesTheGateReason(t *testing.T) {
	for _, tool := range []string{"Bash", "PowerShell"} {
		got := checkTool(t.TempDir(), tool, map[string]any{"command": "loomux gate arm lint/go@."}, config.Policy{})
		if !slices.Equal(got, []string{gateReason}) {
			t.Errorf("[%s] reasons %q", tool, got)
		}
		if got := checkTool(t.TempDir(), tool, map[string]any{"command": "loomux gate status"}, config.Policy{}); len(got) != 0 {
			t.Errorf("[%s] gate status: reasons %q", tool, got)
		}
	}
	// Strict mode reaches a renamed binary through checkTool, too.
	if got := checkTool(t.TempDir(), "Bash", map[string]any{"command": "doc.exe gate disarm --all"}, config.Policy{Strict: true}); !slices.Contains(got, gateReason) {
		t.Errorf("strict, renamed binary: reasons %q", got)
	}
	// A Start-Process of loomux keeps the reasons it had: the words cannot see
	// its arguments, and the call is refused without this rule.
	line := "Start-Process loomux -ArgumentList 'gate','arm','lint/go@.'"
	got := checkTool(t.TempDir(), "PowerShell", map[string]any{"command": line}, config.Policy{})
	if len(got) == 0 || slices.Contains(got, gateReason) {
		t.Errorf("Start-Process: reasons %q", got)
	}
}
