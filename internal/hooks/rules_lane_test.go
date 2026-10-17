package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/verify"
)

// The rules kind keeps a copy without a graph: its check reads the index it
// is handed.
func TestStopIndexForRules(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"blocks":0}`)
	if !wantsStopIndex(root, []string{"lint", "rules"}) {
		t.Fatal("rules wants a copy")
	}
	idx, err := OpenStopIndex(root, []string{"lint", "rules"})
	if err != nil || idx == nil {
		t.Fatalf("not kept for rules: %v %v", idx, err)
	}
	defer idx.Close()
	if env := idx.Env(root); len(env) != 1 || !strings.HasPrefix(env[0], "GIT_INDEX_FILE=") {
		t.Fatalf("%v", env)
	}
}

// With a [rules] table an edit of a source or of the config plans the rules
// check at the root; the config is a TOML file, which no other lane reads.
func TestPostEditRunsTheRulesCheck(t *testing.T) {
	root := goProject(t)
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[rules]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for file, want := range map[string]bool{"a.go": true, ".loomux/config.toml": true, "notes.md": false, "other.toml": false} {
		code, _, se, seen := postEdit(t, root, filePayload(t, filepath.Join(root, file)), passing)
		if code != ExitOK {
			t.Fatalf("%s: %d %q", file, code, se)
		}
		if got := strings.Contains(strings.Join(seen, "\n"), "loomux check rules"); got != want {
			t.Errorf("%s: rules ran %v: %v", file, got, seen)
		}
	}
	// Outside the project the config path names nothing of its own.
	other := t.TempDir()
	code, _, _, seen := postEdit(t, root, filePayload(t, filepath.Join(other, ".loomux", "config.toml")), passing)
	if code != ExitOK || len(seen) != 0 {
		t.Fatalf("%d %v", code, seen)
	}
}

// A red rules check after an edit warns and lets the edit stand.
func TestPostEditOnlyWarnsOnARedRulesCheck(t *testing.T) {
	root := goProject(t)
	writeWorldFile(t, root, ".loomux/config.toml", "[rules]\n")
	red := func(s child.Spec) child.Result {
		if strings.Contains(strings.Join(s.Argv, " "), "check rules") {
			return child.Result{Code: 1, Stdout: "a.go:3 core uses ui\n"}
		}
		return child.Result{}
	}
	code, _, se, _ := postEdit(t, root, filePayload(t, filepath.Join(root, "a.go")), red)
	if code != ExitOK || !strings.Contains(se, "rules/go: failed (warn only)") || !strings.Contains(se, "core uses ui") {
		t.Fatalf("%d %q", code, se)
	}
}

func TestPostEditWithoutARulesTableRunsNoRulesCheck(t *testing.T) {
	root := goProject(t)
	_, _, _, seen := postEdit(t, root, filePayload(t, filepath.Join(root, "a.go")), passing)
	if strings.Contains(strings.Join(seen, "\n"), "check rules") {
		t.Fatalf("%v", seen)
	}
}

func rulesJob(t *testing.T, jobs []verify.Job) verify.Job {
	t.Helper()
	for _, j := range jobs {
		if j.Kind == "rules" {
			return j
		}
	}
	t.Fatalf("no rules job in %v", jobs)
	return verify.Job{}
}

// Before the first commit there is no copy of the index to read, and the real
// one is partial: the rules lane says so instead of judging it.
func TestStopRulesLaneWithoutAHeadSitsOut(t *testing.T) {
	jobs := planned(t)
	root := gitWorld(t, unbornWorld, `{"blocks":0}`)
	writeWorldFile(t, root, ".loomux/config.toml", "[rules]\n[verify.go]\ncoverage = false\n")
	if code, se := runStop(t, root, s1, greenTools()); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	if j := rulesJob(t, *jobs); j.Pre != verify.StateNotApplicable || j.Note != "no HEAD yet" {
		t.Fatalf("%+v", j)
	}
}

func TestStopIndexRulesReadyWithoutACopy(t *testing.T) {
	if ok, note := (*StopIndex)(nil).RulesReady("."); ok || note != "no HEAD yet" {
		t.Fatalf("%v %q", ok, note)
	}
	if ok, note := (&StopIndex{}).RulesReady("."); !ok || note != "" {
		t.Fatalf("%v %q", ok, note)
	}
}

// A rules-only project has no lane that judges against HEAD, so a green tree
// stays green under a new one.
func TestStopKeepsAGreenTreeUnderANewHeadWithRulesOnly(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeWorldFile(t, root, ".loomux/config.toml", "[rules]\n[verify.go]\ncoverage = false\n")
	writeWorldFile(t, root, "b_test.go", "package a\n")
	if code, se := runStop(t, root, s1, greenTools()); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	gitOut(t, root, nil, "add", "b_test.go")
	gitOut(t, root, nil, "commit", "-q", "-m", "the test alone")
	jobs := planned(t)
	if code, se := runStop(t, root, s1, greenTools()); code != ExitOK || *jobs != nil {
		t.Fatalf("a green tree ran again: %d %q %v", code, se, *jobs)
	}
}
