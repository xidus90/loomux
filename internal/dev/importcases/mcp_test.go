package importcases_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/dev/importcases"
)

// renaming is the map of stage 1b-2: the five tools of the reference under
// their loomux names.
func renaming() importcases.Mapping {
	return importcases.Mapping{Tools: []importcases.Rule{
		{From: "search", To: "brain_search"},
		{From: "catalog", To: "brain_catalog"},
	}}
}

// mcpRecording writes one recorded MCP case under from.
func mcpRecording(t *testing.T, from, verb, name, tool string) string {
	t.Helper()
	dir := filepath.Join(from, verb, name)
	if err := os.MkdirAll(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"call":     `{"tool":"` + tool + `","arguments":{"scope":"notes"},"channel":"cloud"}`,
		"result":   `{"isError":false,"text":"the answer"}`,
		"notes.md": "what it shows\n",
	}
	for n, content := range files {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestImportMCPRenamesTheTool(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	mcpRecording(t, from, "brain-catalog", "root", "catalog")
	if err := importcases.ImportMCP(from, to, renaming()); err != nil {
		t.Fatal(err)
	}
	c, err := cases.LoadMCPCase(filepath.Join(to, "brain-catalog", "root"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Call.Tool != "brain_catalog" {
		t.Fatalf("tool: %q", c.Call.Tool)
	}
	// Everything else travels unchanged.
	if c.Call.Channel != "cloud" || c.Result.Text != "the answer" || c.Notes != "what it shows\n" {
		t.Fatalf("case: %+v", c)
	}
	var arguments map[string]any
	if err := json.Unmarshal(c.Call.Arguments, &arguments); err != nil || arguments["scope"] != "notes" {
		t.Fatalf("arguments: %s %v", c.Call.Arguments, err)
	}
}

func TestImportMCPRefusesAToolWithoutARule(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	mcpRecording(t, from, "brain-read", "whole", "read")
	err := importcases.ImportMCP(from, to, renaming())
	if err == nil || !strings.Contains(err.Error(), "no rule for the tool") {
		t.Fatalf("expected the missing rule to be named, got %v", err)
	}
}

// The world of a recording is the old tools' world; the import folds its
// manifests the way the command-line import does.
func TestImportMCPTranslatesTheWorld(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	dir := mcpRecording(t, from, "brain-catalog", "root", "catalog")
	if err := os.WriteFile(filepath.Join(dir, "world", ".brain.toml"),
		[]byte("[area]\nscope = \"notes\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := importcases.ImportMCP(from, to, renaming()); err != nil {
		t.Fatal(err)
	}
	folded := filepath.Join(to, "brain-catalog", "root", "world", ".loomux", "config.toml")
	if _, err := os.Stat(folded); err != nil {
		t.Fatalf("the world was not translated: %v", err)
	}
}

func TestImportMCPPrunesWhatNoRecordingBacks(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	mcpRecording(t, from, "brain-catalog", "root", "catalog")
	mcpRecording(t, to, "brain-catalog", "gone", "catalog")
	if err := importcases.ImportMCP(from, to, renaming()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(to, "brain-catalog", "gone")); !os.IsNotExist(err) {
		t.Fatalf("the unbacked case survived: %v", err)
	}
}

func TestImportMCPReportsWhatItCannotRead(t *testing.T) {
	to := t.TempDir()
	if err := importcases.ImportMCP(filepath.Join(t.TempDir(), "nowhere"), to, renaming()); err == nil {
		t.Fatal("expected the missing source to be reported")
	}
	from := t.TempDir()
	dir := mcpRecording(t, from, "brain-catalog", "root", "catalog")
	if err := os.WriteFile(filepath.Join(dir, "world", ".brain.toml"), []byte("{not toml"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := importcases.ImportMCP(from, to, renaming()); err == nil {
		t.Fatal("expected the broken manifest to stop the import")
	}
}

func TestImportMCPReportsATargetItCannotWrite(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	mcpRecording(t, from, "brain-catalog", "root", "catalog")
	// A file where the case directory has to go.
	if err := os.MkdirAll(filepath.Join(to, "brain-catalog"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(to, "brain-catalog", "root"), []byte("in the way\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := importcases.ImportMCP(from, to, renaming()); err == nil {
		t.Fatal("expected the blocked target to be reported")
	}
}

// A target the import cannot read is not quietly emptied: what is there may be
// somebody's corpus, and a discovery that fails says nothing about which case
// is unbacked.
func TestImportMCPReportsATargetItCannotRead(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	mcpRecording(t, from, "brain-catalog", "root", "catalog")
	broken := filepath.Join(to, "brain-catalog", "broken")
	if err := os.MkdirAll(filepath.Join(broken, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(broken, "call"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := importcases.ImportMCP(from, to, renaming())
	if err == nil || !strings.Contains(err.Error(), "reading the corpus") {
		t.Fatalf("expected the unreadable target to be named, got %v", err)
	}
}

// A case the target already holds and the recordings still back stays where it
// is.
func TestImportMCPKeepsWhatARecordingBacks(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	mcpRecording(t, from, "brain-catalog", "root", "catalog")
	mcpRecording(t, to, "brain-catalog", "root", "brain_catalog")
	if err := importcases.ImportMCP(from, to, renaming()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(to, "brain-catalog", "root", "call")); err != nil {
		t.Fatalf("the backed case was removed: %v", err)
	}
}

// A `call` the target holds as a directory: the copy leaves it alone, and the
// one writer of that file cannot write over it.
func TestImportMCPReportsACallItCannotWrite(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	mcpRecording(t, from, "brain-catalog", "root", "catalog")
	if err := os.MkdirAll(filepath.Join(to, "brain-catalog", "root", "call"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := importcases.ImportMCP(from, to, renaming()); err == nil {
		t.Fatal("expected the blocked call file to be reported")
	}
}
