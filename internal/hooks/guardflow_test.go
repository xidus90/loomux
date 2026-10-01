package hooks

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/flows"
	"github.com/xidus90/loomux/internal/config"
)

func TestAnAgentMayNotAnswerAGate(t *testing.T) {
	for _, line := range []string{
		`loomux flow resume 0001 --answer yes`,
		`loomux flow resume 0001 --answer=yes`,
		`loomux flow resume 0001 -answer "no: thin"`,
		`loomux flow resume --answer yes 0001`,
		`bin/loomux.exe flow resume 0001 --answer yes`,
		`echo ok && loomux flow resume 0001 --answer yes`,
		`"C:/Users/x/AppData/Local/loomux/bin/loomux.exe" flow resume 0001 --answer yes`,
		`& "C:\Program Files\loomux\loomux.exe" flow resume 0001 --answer yes`,
		`.\bin\loomux.exe flow resume 0001 -answer=yes`,
		`go run ./cmd/loomux flow resume 0001 --answer yes`,
		// Behind the prefixes a shell or wrapper puts before the program.
		`sudo loomux flow resume 0001 --answer yes`,
		`winpty loomux flow resume 0001 --answer yes`,
		`stdbuf -o 0 loomux flow resume 0001 --answer yes`,
		`nohup.exe loomux flow resume 0001 --answer yes`,
		`unbuffer -ignore HUP loomux flow resume 0001 --answer yes`,
		`LOOMUX_STATE_DIR=x loomux flow resume 0001 --answer yes`,
		`try { loomux flow resume 0001 --answer yes } catch {}`,
		`1..3 | ForEach-Object { loomux flow resume 0001 --answer yes }`,
		`loomux flow resume 0001 --answer "no; thin"`,
		"loomux flow res`ume 0001 --answer yes",
	} {
		if !answersAGate(line, false) {
			t.Errorf("%q passes", line)
		}
	}
	for _, line := range []string{
		`loomux flow resume 0001`,
		`loomux flow run example`,
		`loomux flow show 0001`,
		`loomux flow list`,
		`loomux flow replay 0001`,
		`loomux flow`,
		`loomux`,
		`loomux flow resume 0001 answer`,
		`loomux flow run example --answer yes`,
		// The command only as text for echo: the program is echo, the way
		// writesConfiguration lets `echo loomux config set` pass.
		`echo "loomux flow resume 0001 --answer yes"`,
		// Quoted for a program the guard does not know: only the reading that
		// splits quoted strings would find a call there.
		`pgrep -f "loomux flow resume 0001 --answer yes"`,
	} {
		if answersAGate(line, false) {
			t.Errorf("%q is refused", line)
		}
	}
}

// Every spelling of a loomux call that hides a configuration write from
// nothing hides an answer from nothing either: the rows are
// writesConfiguration's, with the gate's command in place of theirs.
func TestAnAgentMayNotAnswerAGateInAnySpelling(t *testing.T) {
	for _, row := range loomuxSpellings() {
		if line := asGateAnswer(t, row); !answersAGate(line, false) {
			t.Errorf("%q passes (from %q)", line, row)
		}
	}
}

// asGateAnswer puts flow resume with an answer where a spelling row has its
// command, the first init, config, area add or merge-hook install or remove
// that stands as a word. An escape inside the command stays an escape inside
// the new one, since the escape is what such a row spells. A row that names
// no command must be a Start-Process spelling, refused whatever it runs; the
// answer goes after it. Any other such row is a new spelling this helper
// cannot place the answer in, and fails the test rather than pass untested.
func asGateAnswer(t *testing.T, row string) string {
	t.Helper()
	tick := "`"
	command := regexp.MustCompile(`(^|[\s"'{(])(con\\fig|con` + tick + `fig|con\^fig|merge-hook (?:install|remove)|area add|config|init)($|[\s"'` + tick + `;})])`)
	answer := "flow resume 0001 --answer yes"
	at := command.FindStringSubmatchIndex(row)
	if at == nil {
		if head := strings.ToLower(strings.Fields(row)[0]); head != "start-process" && head != "start" && head != "saps" {
			t.Fatalf("%q names no command to put the answer in place of, and is no Start-Process spelling", row)
		}
		return row + " " + answer
	}
	switch row[at[4]:at[5]] {
	case `con\fig`:
		answer = `flow res\ume 0001 --answer yes`
	case "con" + tick + "fig":
		answer = "flow res" + tick + "ume 0001 --answer yes"
	case `con^fig`:
		answer = `flow res^ume 0001 --answer yes`
	}
	return row[:at[4]] + answer + row[at[5]:]
}

// The refusal names the way a human answers: the reason is its own, not the
// configuration's.
func TestCheckToolNamesTheGateAnswerReason(t *testing.T) {
	const want = "a flow's gate asks a human; the answer is theirs. Ask the user to answer it with `flow resume <run> --answer \"…\"` themselves"
	for _, tool := range []string{"Bash", "PowerShell"} {
		got := checkTool(t.TempDir(), tool, map[string]any{"command": "loomux flow resume 0001 --answer yes"}, config.Policy{})
		if len(got) != 1 || got[0] != want {
			t.Errorf("[%s] reasons %q, want %q", tool, got, want)
		}
	}
	if got := checkTool(t.TempDir(), "Bash", map[string]any{"command": "loomux flow resume 0001"}, config.Policy{}); len(got) != 0 {
		t.Fatalf("a resume without an answer: reasons %q", got)
	}
	// Start-Process hides its arguments from the words, so the gate reason
	// stands beside the configuration's rather than depending on it.
	got := checkTool(t.TempDir(), "PowerShell", map[string]any{"command": "Start-Process loomux -ArgumentList 'flow','resume','0001','--answer','yes'"}, config.Policy{})
	if !slices.Contains(got, want) {
		t.Fatalf("Start-Process: reasons %q, want %q among them", got, want)
	}
}

const runFilesWant = "a flow's journal and marker are written by loomux, not by the party the gates ask"

// A journal or marker a writing tool touched would carry an answer no human
// gave, so the run files are refused like the stop gate's own state, in any
// case the file system folds together.
func TestAWritingToolMayNotTouchARunFile(t *testing.T) {
	root := t.TempDir()
	for _, tool := range []string{"Write", "Edit"} {
		for _, path := range []string{
			".loomux/state/runs/0001.jsonl",
			".loomux/state/runs/0001.flow",
			filepath.Join(root, ".loomux", "state", "runs", "0001.jsonl"),
			".LOOMUX/State/Runs/0001.jsonl",
		} {
			got := checkTool(root, tool, map[string]any{"file_path": path}, config.Policy{})
			if len(got) != 1 || got[0] != runFilesWant {
				t.Errorf("[%s] %s: reasons %q, want the run files refusal", tool, path, got)
			}
		}
	}
	if got := checkTool(root, "Write", map[string]any{"file_path": ".loomux/state/runsx/0001.jsonl"}, config.Policy{}); len(got) != 0 {
		t.Fatalf("a folder that only begins like runs: reasons %q", got)
	}
}

// bundledWant is the refusal for a flow folder a human keeps.
func bundledWant(name string) string {
	return "a bundled flow's gates and instructions are a human's to change; give your flow a name of its own, or ask the user to hide or overlay `" + name + "`"
}

// catalog makes names the bundled flows for one test, so that a reason
// naming every protected flow does not change when the catalog grows.
func catalog(t *testing.T, names ...string) {
	t.Helper()
	bundledFlows = func() []string { return slices.Clone(names) }
	t.Cleanup(func() { bundledFlows = flows.Names })
}

// A catalog name and a name [flow] overrides lets hide a bundled flow are a
// human's folders, in either case; a flow of its own an agent may write.
func TestAWritingToolMayNotTouchABundledFlowFolder(t *testing.T) {
	root := project(t)
	manifest(t, root, "[flow]\noverrides = [\"review\"]\n")
	for _, row := range []struct{ path, name string }{
		{".loomux/flows/example/flow.toml", "example"},
		{".loomux/flows/Example/flow.toml", "example"},
		{".loomux/flows/example", "example"},
		{filepath.Join(root, ".loomux", "flows", "example", "instructions", "draft.md"), "example"},
		{".loomux/flows/review/questions/q.md", "review"},
		{".LOOMUX/Flows/REVIEW/questions/q.md", "review"},
		{"docs/.loomux/flows/example/flow.toml", "example"},
		{"../sibling/.loomux/flows/review/flow.toml", "review"},
		// Every .loomux/flows in the path counts, the outer and the inner.
		{".loomux/flows/example/.loomux/flows/mine/x.md", "example"},
		{"docs/.loomux/flows/x/.loomux/flows/example/f.md", "example"},
		{".loomux/flows/example/.loomux/flows/Example/f.md", "example"},
	} {
		got := checkTool(root, "Write", map[string]any{"file_path": row.path}, config.Policy{})
		if len(got) != 1 || got[0] != bundledWant(row.name) {
			t.Errorf("%s: reasons %q, want %q", row.path, got, bundledWant(row.name))
		}
	}
	both := ".loomux/flows/example/.loomux/flows/review/x.md"
	if got := checkTool(root, "Write", map[string]any{"file_path": both}, config.Policy{}); !slices.Equal(got, []string{bundledWant("example"), bundledWant("review")}) {
		t.Errorf("%s: reasons %q, want example and review", both, got)
	}
	for _, path := range []string{
		".loomux/flows/mine/flow.toml",
		".loomux/flows/example2/flow.toml",
		".loomux/flows",
	} {
		if got := checkTool(root, "Edit", map[string]any{"file_path": path}, config.Policy{}); len(got) != 0 {
			t.Errorf("%s: reasons %q, want none", path, got)
		}
	}
}

// A [flow] the guard cannot read might name any folder, so every write under
// .loomux/flows is refused, and one elsewhere is not judged by it.
func TestAFlowFolderUnderAnUnreadableFlowTableIsRefused(t *testing.T) {
	root := project(t)
	manifest(t, root, "[flow]\noverrides = 5\n")
	const prefix = "loomux cannot read [flow] of .loomux/config.toml, so it refuses writes under .loomux/flows: "
	got := checkTool(root, "Write", map[string]any{"file_path": ".loomux/flows/mine/flow.toml"}, config.Policy{})
	if len(got) != 1 || !strings.HasPrefix(got[0], prefix) {
		t.Errorf("write: reasons %q", got)
	}
	for _, line := range []string{"rm -r .loomux/flows/mine", "rm -r .loomux/flows"} {
		got = checkTool(root, "Bash", map[string]any{"command": line}, config.Policy{})
		if len(got) != 1 || !strings.HasPrefix(got[0], prefix) {
			t.Errorf("shell %q: reasons %q", line, got)
		}
	}
	if got := checkTool(root, "Write", map[string]any{"file_path": "src/main.go"}, config.Policy{}); len(got) != 0 {
		t.Errorf("a write elsewhere: reasons %q", got)
	}
	if got := checkTool(root, "Bash", map[string]any{"command": "cat .loomux/flows/mine/flow.toml"}, config.Policy{}); len(got) != 0 {
		t.Errorf("a read: reasons %q", got)
	}
}

// With no flow to keep, no folder under .loomux/flows is kept.
func TestNoProtectedFlowMeansNoShellRule(t *testing.T) {
	catalog(t)
	root := project(t)
	for _, line := range []string{"rm -r .loomux/flows/example", "rm -r .loomux/flows"} {
		if got := checkTool(root, "Bash", map[string]any{"command": line}, config.Policy{}); len(got) != 0 {
			t.Fatalf("%q: reasons %q", line, got)
		}
	}
}

// Overriding a bundled flow is how a project lets its overlay run, so a
// catalog name in [flow] overrides is the normal case and is named once.
func TestAFlowNamedTwiceIsNamedOnce(t *testing.T) {
	catalog(t, "example")
	root := project(t)
	manifest(t, root, "[flow]\noverrides = [\"example\"]\n")
	for _, input := range []map[string]any{
		{"command": "rm -r .loomux/flows/example"},
		{"file_path": ".loomux/flows/example/flow.toml"},
	} {
		tool := "Bash"
		if _, path := input["file_path"]; path {
			tool = "Write"
		}
		if got := checkTool(root, tool, input, config.Policy{}); len(got) != 1 || got[0] != bundledWant("example") {
			t.Errorf("[%s] reasons %q, want %q", tool, got, bundledWant("example"))
		}
	}
}

// A flow name is lower case, so an override spelled in capitals is no
// second spelling of a catalog name but a [flow] that will not read, and
// every write under .loomux/flows is refused.
func TestAnOverrideInCapitalsRefusesEveryFlowFolder(t *testing.T) {
	catalog(t, "example")
	root := project(t)
	manifest(t, root, "[flow]\noverrides = [\"Example\"]\n")
	const prefix = "loomux cannot read [flow] of .loomux/config.toml, so it refuses writes under .loomux/flows: "
	for _, input := range []map[string]any{
		{"command": "rm -r .loomux/flows/mine"},
		{"file_path": ".loomux/flows/mine/flow.toml"},
	} {
		tool := "Bash"
		if _, path := input["file_path"]; path {
			tool = "Write"
		}
		if got := checkTool(root, tool, input, config.Policy{}); len(got) != 1 || !strings.HasPrefix(got[0], prefix) {
			t.Errorf("[%s] reasons %q, want the unreadable [flow] refusal", tool, got)
		}
	}
}
