package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/dev/recordcase"
)

func TestDevRecordMCPCaseRefusesAnIncompleteInvocation(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"unknown flag", []string{"--bogus"}, "flag provided but not defined"},
		{"nothing", nil, "--argv, --tool, --world and --out are required"},
		{"unsplittable argv", []string{"--argv", "'unclosed", "--tool", "catalog",
			"--world", ".", "--out", "."}, "--argv: unclosed quote"},
		{"environment without a value", []string{"--env", "NOVALUE"}, "is not KEY=VALUE"},
		{"unknown grade", []string{"--argv", "x", "--tool", "catalog", "--world", ".",
			"--out", ".", "--compare", "bytes"}, `--compare must be "text" or "outcome"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, _, errOut := run(append([]string{"dev", "record-mcp-case"}, tc.args...)...)
			if code != 2 || !strings.Contains(errOut, tc.want) {
				t.Fatalf("code %d, err %q", code, errOut)
			}
		})
	}
}

func TestDevRecordMCPCaseReportsAFailedRecording(t *testing.T) {
	code, _, errOut := run("dev", "record-mcp-case",
		"--argv", filepath.Join(t.TempDir(), "gone.exe"), "--tool", "catalog",
		"--world", t.TempDir(), "--out", filepath.Join(t.TempDir(), "v", "n"))
	if code != 1 || !strings.Contains(errOut, "loomux dev record-mcp-case:") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// The whole command with a reference that answers: the recorder is stood in
// for, because a real one is a Python process this suite may not start.
func TestDevRecordMCPCaseWritesTheCase(t *testing.T) {
	var seen recordcase.MCPSpec
	original := recordMCPCase
	recordMCPCase = func(s recordcase.MCPSpec) error { seen = s; return nil }
	t.Cleanup(func() { recordMCPCase = original })

	out := filepath.Join(t.TempDir(), "brain-catalog", "root")
	code, _, errOut := run("dev", "record-mcp-case",
		"--argv", "uv run brain-mcp", "--tool", "catalog", "--arguments", `{"scope":"notes"}`,
		"--channel", "cloud", "--env", "PYTHONUTF8=1", "--path-prepend", "bin",
		"--world", "worlds/vault", "--out", out, "--notes", "what it shows", "--compare", "outcome")
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if strings.Join(seen.Argv, " ") != "uv run brain-mcp" {
		t.Fatalf("argv: %v", seen.Argv)
	}
	if seen.Tool != "catalog" || seen.Arguments != `{"scope":"notes"}` || seen.Channel != "cloud" {
		t.Fatalf("call: %+v", seen)
	}
	if seen.World != "worlds/vault" || seen.Out != out || seen.Notes != "what it shows" {
		t.Fatalf("places: %+v", seen)
	}
	if seen.Compare != "outcome" || seen.PathPrepend != "bin" || strings.Join(seen.Env, " ") != "PYTHONUTF8=1" {
		t.Fatalf("rest: %+v", seen)
	}
}

// The MCP corpus goes through the same command with --mcp: one map file
// format, two discoveries.
func TestDevImportCasesImportsTheMCPCorpus(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	dir := filepath.Join(from, "brain-catalog", "root")
	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"call":   `{"tool":"catalog","arguments":{},"channel":"cloud"}`,
		"result": `{"isError":false,"text":"the answer"}`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mapFile := filepath.Join(t.TempDir(), "map.toml")
	if err := os.WriteFile(mapFile, []byte("[[tool]]\nfrom = \"catalog\"\nto = \"brain_catalog\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := run("dev", "import-cases", "--mcp", "--map", mapFile, "--from", from, "--to", to)
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	data, err := os.ReadFile(filepath.Join(to, "brain-catalog", "root", "call"))
	if err != nil || !strings.Contains(string(data), "brain_catalog") {
		t.Fatalf("call: %s %v", data, err)
	}
}
