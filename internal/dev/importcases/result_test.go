package importcases

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resultRecording writes one recorded MCP case under from whose result text
// is text, as the recorder indents it.
func resultRecording(t *testing.T, from, text string) {
	t.Helper()
	dir := filepath.Join(from, "brain-status", "advice")
	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"call":   `{"tool":"status","arguments":{},"channel":"cloud"}`,
		"result": "{\n  \"isError\": false,\n  \"text\": \"" + text + "\"\n}\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func statusRule() Mapping {
	return Mapping{
		Tools:  []Rule{{From: "status", To: "brain_status"}},
		Result: []Rule{{From: "run `brain reindex`", To: "run `loomux reindex`"}, {From: "brain embed", To: "loomux embed"}},
	}
}

// Every rule applies, every occurrence, and the bytes around them stay as the
// recorder wrote them -- indentation and the escaped backslash included.
func TestImportMCPRewritesTheResultByEveryRule(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	resultRecording(t, from, `a\\b: run `+"`brain reindex`"+`; run `+"`brain reindex`"+`; run `+"`brain embed`")
	if err := ImportMCP(from, to, statusRule()); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(to, "brain-status", "advice", "result"))
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"isError\": false,\n  \"text\": \"a\\\\b: run `loomux reindex`; run `loomux reindex`; run `loomux embed`\"\n}\n"
	if string(got) != want {
		t.Fatalf("result\n%q\nwant\n%q", got, want)
	}
}

// Without a rule the result is the recording, byte for byte.
func TestImportMCPKeepsAResultNoRuleNames(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	resultRecording(t, from, "run `brain reindex`")
	m := statusRule()
	m.Result = nil
	if err := ImportMCP(from, to, m); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(to, "brain-status", "advice", "result"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "{\n  \"isError\": false,\n  \"text\": \"run `brain reindex`\"\n}\n"; string(got) != want {
		t.Fatalf("result %q, want %q", got, want)
	}
}

func TestImportMCPReportsAResultItCannotRead(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	resultRecording(t, from, "x")
	boom := errors.New("boom")
	saved := readRecorded
	readRecorded = func(string) ([]byte, error) { return nil, boom }
	t.Cleanup(func() { readRecorded = saved })
	if err := ImportMCP(from, to, statusRule()); !errors.Is(err, boom) {
		t.Fatalf("got %v, want the read failure", err)
	}
}

// A `result` the target holds as a directory: the copy leaves it alone, and
// the one writer of that file cannot write over it.
func TestImportMCPReportsAResultItCannotWrite(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	resultRecording(t, from, "x")
	if err := os.MkdirAll(filepath.Join(to, "brain-status", "advice", "result"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := ImportMCP(from, to, statusRule())
	if err == nil || !strings.Contains(err.Error(), filepath.Join("advice", "result")) {
		t.Fatalf("got %v; want the blocked result file named", err)
	}
}
