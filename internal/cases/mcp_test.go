package cases_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/cases"
)

// writeMCPCase puts one case on disk. The world is always there: a case
// without one is not loadable, which is what one of the tests below measures.
func writeMCPCase(t *testing.T, dir string, files map[string]string, withWorld bool) string {
	t.Helper()
	if withWorld {
		if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
			t.Fatal(err)
		}
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const goodCall = `{"tool":"brain_catalog","arguments":{"scope":"notes"},"channel":"cloud"}`
const goodResult = `{"isError":false,"text":"# brain\n"}`

func TestLoadMCPCaseReadsCallAndResult(t *testing.T) {
	dir := writeMCPCase(t, filepath.Join(t.TempDir(), "brain-catalog", "area"), map[string]string{
		"call":     goodCall,
		"result":   goodResult,
		"notes.md": "what it shows\n",
	}, true)
	c, err := cases.LoadMCPCase(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.Verb != "brain-catalog" || c.Name != "area" {
		t.Fatalf("verb and name: got %q/%q", c.Verb, c.Name)
	}
	if c.Call.Tool != "brain_catalog" || c.Call.Channel != "cloud" {
		t.Fatalf("tool and channel: got %q/%q", c.Call.Tool, c.Call.Channel)
	}
	if string(c.Call.Arguments) != `{"scope":"notes"}` {
		t.Fatalf("arguments: got %s", c.Call.Arguments)
	}
	if c.Result.Text != "# brain\n" || c.Result.IsError {
		t.Fatalf("result: got %+v", c.Result)
	}
	// Text is the default: a case that says nothing compares everything.
	if c.Compare != cases.CompareText {
		t.Fatalf("compare: got %q", c.Compare)
	}
	if c.Notes != "what it shows\n" {
		t.Fatalf("notes: got %q", c.Notes)
	}
}

func TestLoadMCPCaseReadsTheOutcomeGrade(t *testing.T) {
	dir := writeMCPCase(t, filepath.Join(t.TempDir(), "brain-read", "missing"), map[string]string{
		"call":    goodCall,
		"result":  goodResult,
		"compare": "outcome\n",
	}, true)
	c, err := cases.LoadMCPCase(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.Compare != cases.CompareOutcome {
		t.Fatalf("compare: got %q", c.Compare)
	}
}

func TestLoadMCPCaseRefusesWhatItCannotRead(t *testing.T) {
	for _, tc := range []struct {
		name      string
		files     map[string]string
		withWorld bool
		want      string
	}{
		{"no world", map[string]string{"call": goodCall, "result": goodResult}, false, "missing world directory"},
		{"no call", map[string]string{"result": goodResult}, true, "missing call"},
		{"broken call", map[string]string{"call": "{", "result": goodResult}, true, "invalid call"},
		{"nameless call", map[string]string{"call": `{"tool":""}`, "result": goodResult}, true, "empty tool"},
		{"no result", map[string]string{"call": goodCall}, true, "missing result"},
		{"broken result", map[string]string{"call": goodCall, "result": "{"}, true, "invalid result"},
		{"unknown grade", map[string]string{"call": goodCall, "result": goodResult, "compare": "bytes"}, true, "unknown compare"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeMCPCase(t, filepath.Join(t.TempDir(), "verb", "name"), tc.files, tc.withWorld)
			_, err := cases.LoadMCPCase(dir)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestDiscoverMCPCasesSortsAndStopsAtACase(t *testing.T) {
	root := t.TempDir()
	writeMCPCase(t, filepath.Join(root, "b-verb", "second"), map[string]string{"call": goodCall, "result": goodResult}, true)
	writeMCPCase(t, filepath.Join(root, "a-verb", "second"), map[string]string{"call": goodCall, "result": goodResult}, true)
	writeMCPCase(t, filepath.Join(root, "a-verb", "first"), map[string]string{"call": goodCall, "result": goodResult}, true)
	found, err := cases.DiscoverMCPCases(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, c := range found {
		names = append(names, c.Verb+"/"+c.Name)
	}
	want := []string{"a-verb/first", "a-verb/second", "b-verb/second"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("order: got %v", names)
	}
}

func TestDiscoverMCPCasesCarriesUpABrokenCase(t *testing.T) {
	root := t.TempDir()
	writeMCPCase(t, filepath.Join(root, "verb", "name"), map[string]string{"call": "{"}, true)
	if _, err := cases.DiscoverMCPCases(root); err == nil {
		t.Fatal("expected the broken case to stop the walk")
	}
}

func TestDiscoverMCPCasesReportsAnUnreadableRoot(t *testing.T) {
	if _, err := cases.DiscoverMCPCases(filepath.Join(t.TempDir(), "nowhere")); err == nil {
		t.Fatal("expected an error for a root that is not there")
	}
}

// answered is a run function that hands back one fixed outcome and remembers
// what it was asked.
func answered(out cases.MCPOutcome, seen *cases.MCPCall) cases.MCPRunFunc {
	return func(call cases.MCPCall, dir string) (cases.MCPOutcome, error) {
		*seen = call
		seen.Arguments = json.RawMessage(strings.ReplaceAll(string(call.Arguments), dir, "<dir>"))
		return out, nil
	}
}

func TestRunMCPCasePassesWhenTextAndIsErrorMatch(t *testing.T) {
	dir := writeMCPCase(t, filepath.Join(t.TempDir(), "brain-catalog", "area"), map[string]string{
		"call":   goodCall,
		"result": goodResult,
	}, true)
	c, err := cases.LoadMCPCase(dir)
	if err != nil {
		t.Fatal(err)
	}
	var seen cases.MCPCall
	outcome, err := cases.RunMCPCase(c, answered(cases.MCPOutcome{Text: "# brain\n"}, &seen))
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Passed {
		t.Fatalf("expected a pass, got %v", outcome.Mismatches)
	}
	if seen.Tool != "brain_catalog" || seen.Channel != "cloud" {
		t.Fatalf("the call did not arrive: %+v", seen)
	}
}

func TestRunMCPCaseNamesEveryDifference(t *testing.T) {
	dir := writeMCPCase(t, filepath.Join(t.TempDir(), "brain-catalog", "area"), map[string]string{
		"call":   goodCall,
		"result": `{"isError":true,"text":"no such scope","rpcError":"bad request"}`,
	}, true)
	c, err := cases.LoadMCPCase(dir)
	if err != nil {
		t.Fatal(err)
	}
	var seen cases.MCPCall
	outcome, err := cases.RunMCPCase(c, answered(cases.MCPOutcome{Text: "other"}, &seen))
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(outcome.Mismatches, "\n")
	for _, want := range []string{"isError", "text", "protocol error"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected %q among the mismatches, got %q", want, joined)
		}
	}
}

func TestRunMCPCaseOfTheOutcomeGradeIgnoresTheText(t *testing.T) {
	dir := writeMCPCase(t, filepath.Join(t.TempDir(), "brain-read", "refused"), map[string]string{
		"call":    goodCall,
		"result":  `{"isError":true,"text":"'relative'"}`,
		"compare": "outcome",
	}, true)
	c, err := cases.LoadMCPCase(dir)
	if err != nil {
		t.Fatal(err)
	}
	var seen cases.MCPCall
	outcome, err := cases.RunMCPCase(c, answered(cases.MCPOutcome{IsError: true, Text: "a wholly different sentence"}, &seen))
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Passed {
		t.Fatalf("expected a pass, got %v", outcome.Mismatches)
	}
}

func TestRunMCPCaseStagesTheWorldIntoTheArguments(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "brain-read", "whole")
	writeMCPCase(t, dir, map[string]string{
		"call":   `{"tool":"brain_read","arguments":{"relative":"{{WORLD}}/a.md"},"channel":"local"}`,
		"result": `{"isError":false,"text":""}`,
	}, true)
	if err := os.WriteFile(filepath.Join(dir, "world", "a.md"), []byte("in {{WORLD}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := cases.LoadMCPCase(dir)
	if err != nil {
		t.Fatal(err)
	}
	var seen cases.MCPCall
	staged := ""
	if _, err := cases.RunMCPCase(c, func(call cases.MCPCall, world string) (cases.MCPOutcome, error) {
		seen, staged = call, world
		data, readErr := os.ReadFile(filepath.Join(world, "a.md"))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if !strings.Contains(string(data), filepath.ToSlash(world)) {
			t.Fatalf("the staged file kept the token: %q", data)
		}
		return cases.MCPOutcome{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(seen.Arguments), filepath.ToSlash(staged)) {
		t.Fatalf("the arguments kept the token: %s", seen.Arguments)
	}
}

// A text that names the staged world must come back with the token in it, or
// no recording would ever match twice.
func TestRunMCPCaseNormalisesTheAnswer(t *testing.T) {
	dir := writeMCPCase(t, filepath.Join(t.TempDir(), "brain-catalog", "root"), map[string]string{
		"call":   goodCall,
		"result": `{"isError":false,"text":"under {{WORLD}}/repo-a"}`,
	}, true)
	c, err := cases.LoadMCPCase(dir)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := cases.RunMCPCase(c, func(_ cases.MCPCall, world string) (cases.MCPOutcome, error) {
		return cases.MCPOutcome{Text: "under " + filepath.ToSlash(world) + "/repo-a"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Passed {
		t.Fatalf("expected a pass, got %v", outcome.Mismatches)
	}
}

func TestRunMCPCaseCarriesUpAFailedRun(t *testing.T) {
	dir := writeMCPCase(t, filepath.Join(t.TempDir(), "brain-catalog", "root"), map[string]string{
		"call":   goodCall,
		"result": goodResult,
	}, true)
	c, err := cases.LoadMCPCase(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cases.RunMCPCase(c, func(cases.MCPCall, string) (cases.MCPOutcome, error) {
		return cases.MCPOutcome{}, os.ErrClosed
	}); err == nil {
		t.Fatal("expected the run's own error")
	}
}

func TestRunMCPCaseReportsAWorldItCannotStage(t *testing.T) {
	dir := writeMCPCase(t, filepath.Join(t.TempDir(), "brain-catalog", "root"), map[string]string{
		"call":   goodCall,
		"result": goodResult,
	}, true)
	c, err := cases.LoadMCPCase(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(dir, "world")); err != nil {
		t.Fatal(err)
	}
	if _, err := cases.RunMCPCase(c, func(cases.MCPCall, string) (cases.MCPOutcome, error) {
		return cases.MCPOutcome{}, nil
	}); err == nil {
		t.Fatal("expected the staging error")
	}
}

// A directory named `call` is no case: the walk goes past it rather than
// reading a directory as a recording.
func TestDiscoverMCPCasesWalksPastADirectoryNamedCall(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "verb", "name", "call"), 0o755); err != nil {
		t.Fatal(err)
	}
	found, err := cases.DiscoverMCPCases(root)
	if err != nil || len(found) != 0 {
		t.Fatalf("expected no case, got %d and %v", len(found), err)
	}
}

// A file beside the cases is no case and stops nothing.
func TestDiscoverMCPCasesIgnoresALooseFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("the corpus\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeMCPCase(t, filepath.Join(root, "verb", "name"), map[string]string{"call": goodCall, "result": goodResult}, true)
	found, err := cases.DiscoverMCPCases(root)
	if err != nil || len(found) != 1 {
		t.Fatalf("expected one case, got %d and %v", len(found), err)
	}
}

// A call that names no arguments is no call: the protocol carries an object,
// and a tool without arguments is called with an empty one.
func TestLoadMCPCaseRefusesACallWithoutArguments(t *testing.T) {
	dir := writeMCPCase(t, filepath.Join(t.TempDir(), "verb", "name"), map[string]string{
		"call":   `{"tool":"brain_status","channel":"cloud"}`,
		"result": goodResult,
	}, true)
	if _, err := cases.LoadMCPCase(dir); err == nil || !strings.Contains(err.Error(), "missing arguments") {
		t.Fatalf("expected the missing arguments to be refused, got %v", err)
	}
}
