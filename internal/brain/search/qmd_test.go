package search_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/search"
)

func TestQmdPort_SearchSuccess(t *testing.T) {
	var capturedArgs []string
	jsonOut := `[
		{
			"file": "qmd://knowledge/topics/search.md",
			"line": 15,
			"title": "Search Topic",
			"snippet": "some matched text",
			"score": 0.92,
			"docid": "01DOC_ABC"
		}
	]`

	port := &search.QmdPort{
		Executable: "qmd",
		Runner: func(args []string) ([]byte, []byte, int, error) {
			capturedArgs = args
			return []byte(jsonOut), nil, 0, nil
		},
	}

	hits, err := port.Search("my query", []string{"knowledge"}, search.ProfileFull, 10)
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}

	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	hit := hits[0]
	if hit.Collection != "knowledge" {
		t.Errorf("expected collection knowledge, got %q", hit.Collection)
	}
	if hit.Relative != "topics/search.md" {
		t.Errorf("expected relative topics/search.md, got %q", hit.Relative)
	}
	if hit.Line != 15 {
		t.Errorf("expected line 15, got %d", hit.Line)
	}
	if hit.Title != "Search Topic" {
		t.Errorf("expected title 'Search Topic', got %q", hit.Title)
	}
	if hit.Snippet != "some matched text" {
		t.Errorf("expected snippet 'some matched text', got %q", hit.Snippet)
	}
	if hit.Score != 0.92 {
		t.Errorf("expected score 0.92, got %f", hit.Score)
	}
	if hit.ContentKey != "01DOC_ABC" {
		t.Errorf("expected docid '01DOC_ABC', got %q", hit.ContentKey)
	}

	expectedArgs := []string{"qmd", "query", "my query", "--json", "-n", "10", "-c", "knowledge"}
	if len(capturedArgs) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %d: %v", len(expectedArgs), len(capturedArgs), capturedArgs)
	}
	for i := range expectedArgs {
		if capturedArgs[i] != expectedArgs[i] {
			t.Errorf("arg %d mismatch: expected %q, got %q", i, expectedArgs[i], capturedArgs[i])
		}
	}
}

func TestQmdPort_ProfileSubcommands(t *testing.T) {
	profiles := []struct {
		profile search.Profile
		subcmd  string
	}{
		{search.ProfileKeyword, "search"},
		{search.ProfileFast, "vsearch"},
		{search.ProfileFull, "query"},
		{search.Profile("other"), "query"},
	}

	for _, tc := range profiles {
		t.Run(string(tc.profile), func(t *testing.T) {
			var capturedSubcmd string
			port := &search.QmdPort{
				Executable: "qmd",
				Runner: func(args []string) ([]byte, []byte, int, error) {
					if len(args) > 1 {
						capturedSubcmd = args[1]
					}
					return []byte("[]"), nil, 0, nil
				},
			}
			hits, err := port.Search("test", nil, tc.profile, 5)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(hits) != 0 {
				t.Fatalf("expected 0 hits, got %d", len(hits))
			}
			if capturedSubcmd != tc.subcmd {
				t.Errorf("profile %q: expected subcmd %q, got %q", tc.profile, tc.subcmd, capturedSubcmd)
			}
		})
	}
}

func TestQmdPort_RunnerErrors(t *testing.T) {
	t.Run("runner failure", func(t *testing.T) {
		port := &search.QmdPort{
			Executable: "qmd",
			Runner: func(args []string) ([]byte, []byte, int, error) {
				return nil, nil, 1, errors.New("cannot execute binary")
			},
		}
		_, err := port.Search("test", nil, search.ProfileFast, 5)
		if err == nil || !strings.Contains(err.Error(), "cannot execute binary") {
			t.Fatalf("expected runner execution error, got: %v", err)
		}
	})

	t.Run("nonzero exit code", func(t *testing.T) {
		port := &search.QmdPort{
			Executable: "qmd",
			Runner: func(args []string) ([]byte, []byte, int, error) {
				return nil, []byte("fatal: index corrupted"), 2, nil
			},
		}
		_, err := port.Search("test", nil, search.ProfileFast, 5)
		if err == nil || !strings.Contains(err.Error(), "qmd exited with 2: fatal: index corrupted") {
			t.Fatalf("expected nonzero exit error, got: %v", err)
		}
	})
}

func TestQmdPort_MalformedJSONOutput(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		errSubstr string
	}{
		{"invalid json", "not valid json", "could not read the search output"},
		{"json not array", `{"error": "bad"}`, "expected a list of hits"},
		{"hit missing file", `[{"line": 1, "score": 1.0, "docid": "k"}]`, "missing 'file'"},
		{"hit invalid file prefix", `[{"file": "http://x", "line": 1, "score": 1.0, "docid": "k"}]`, "expected a qmd:// location"},
		{"hit missing docid", `[{"file": "qmd://c/p", "line": 1, "score": 1.0}]`, "missing 'docid'"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			port := &search.QmdPort{
				Executable: "qmd",
				Runner: func(args []string) ([]byte, []byte, int, error) {
					return []byte(tc.output), nil, 0, nil
				},
			}
			_, err := port.Search("query", nil, search.ProfileKeyword, 5)
			if err == nil || !strings.Contains(err.Error(), tc.errSubstr) {
				t.Fatalf("expected error containing %q, got: %v", tc.errSubstr, err)
			}
		})
	}
}

func TestDefaultRunner(t *testing.T) {
	t.Run("empty argv", func(t *testing.T) {
		_, _, exitCode, err := search.DefaultRunner(nil)
		if err == nil || exitCode != 1 {
			t.Fatalf("expected error for empty argv, got exit %d, err %v", exitCode, err)
		}
	})

	t.Run("successful command", func(t *testing.T) {
		stdout, _, exitCode, err := search.DefaultRunner([]string{"go", "version"})
		if err != nil || exitCode != 0 {
			t.Fatalf("expected success, got exit %d, err %v", exitCode, err)
		}
		if !strings.Contains(string(stdout), "go version") {
			t.Errorf("expected go version in output, got %q", string(stdout))
		}
	})

	t.Run("command exit error", func(t *testing.T) {
		_, _, exitCode, err := search.DefaultRunner([]string{"go", "invalid-subcommand-xyz"})
		if err != nil {
			t.Fatalf("expected exit code rather than exec error, got %v", err)
		}
		if exitCode == 0 {
			t.Errorf("expected non-zero exit code for invalid subcommand, got 0")
		}
	})
}

func TestQmdPort_DefaultExecutableAndRunner(t *testing.T) {
	port := &search.QmdPort{}
	var capturedArgs []string
	port.Runner = func(args []string) ([]byte, []byte, int, error) {
		capturedArgs = args
		return []byte("[]"), nil, 0, nil
	}

	hits, err := port.Search("test", nil, search.ProfileFull, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hits) != 0 {
		t.Errorf("expected empty hits, got %d", len(hits))
	}
	if len(capturedArgs) == 0 || capturedArgs[0] != "qmd" {
		t.Errorf("expected default executable 'qmd', got %v", capturedArgs)
	}

	// Test nil Runner triggers DefaultRunner
	nilRunnerPort := &search.QmdPort{Executable: "nonexistent-binary-qmd-xyz"}
	_, err = nilRunnerPort.Search("test", nil, search.ProfileFast, 1)
	if err == nil {
		t.Errorf("expected error running nonexistent binary with DefaultRunner, got nil")
	}
}

func TestQmdPort_Indexed(t *testing.T) {
	output := `
qmd collection ls:
1234  2026-08-20  qmd://test-col/docs/alpha.md
5678  2026-08-20  qmd://other-col/docs/beta.md
  some header without uri
9012  2026-08-20  qmd://test-col/gamma.md
`
	var capturedArgs []string
	port := &search.QmdPort{
		Executable: "qmd",
		Runner: func(args []string) ([]byte, []byte, int, error) {
			capturedArgs = args
			return []byte(output), nil, 0, nil
		},
	}

	files, err := port.Indexed("test-col")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 || files[0] != "docs/alpha.md" || files[1] != "gamma.md" {
		t.Errorf("unexpected indexed files: %v", files)
	}
	if len(capturedArgs) < 3 || capturedArgs[1] != "ls" || capturedArgs[2] != "test-col" {
		t.Errorf("unexpected args: %v", capturedArgs)
	}
}

func TestQmdPort_Refresh(t *testing.T) {
	var capturedArgs []string
	port := &search.QmdPort{
		Executable: "qmd",
		Runner: func(args []string) ([]byte, []byte, int, error) {
			capturedArgs = args
			return nil, nil, 0, nil
		},
	}

	err := port.Refresh([]string{"col1", "col2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(capturedArgs) != 2 || capturedArgs[0] != "qmd" || capturedArgs[1] != "update" {
		t.Errorf("expected [qmd update], got %v", capturedArgs)
	}
}

func TestQmdPort_NotYetSearchable_Found(t *testing.T) {
	output := `
Documents:
  Indexed:  100
  Pending:  24 need embedding (run 'qmd embed')
`
	var capturedArgs []string
	port := &search.QmdPort{
		Executable: "qmd",
		Runner: func(args []string) ([]byte, []byte, int, error) {
			capturedArgs = args
			return []byte(output), nil, 0, nil
		},
	}

	count, err := port.NotYetSearchable()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 24 {
		t.Errorf("expected 24 pending, got %d", count)
	}
	if len(capturedArgs) != 2 || capturedArgs[1] != "status" {
		t.Errorf("expected [qmd status], got %v", capturedArgs)
	}
}

func TestQmdPort_NotYetSearchable_Absent(t *testing.T) {
	output := `
Documents:
  Indexed:  100
`
	port := &search.QmdPort{
		Executable: "qmd",
		Runner: func(args []string) ([]byte, []byte, int, error) {
			return []byte(output), nil, 0, nil
		},
	}

	count, err := port.NotYetSearchable()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 pending, got %d", count)
	}
}

func TestQmdPort_Embed(t *testing.T) {
	var capturedArgs []string
	port := &search.QmdPort{
		Executable: "qmd",
		Runner: func(args []string) ([]byte, []byte, int, error) {
			capturedArgs = args
			return nil, nil, 0, nil
		},
	}

	err := port.Embed([]string{"col1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(capturedArgs) != 2 || capturedArgs[0] != "qmd" || capturedArgs[1] != "embed" {
		t.Errorf("expected [qmd embed], got %v", capturedArgs)
	}
}

func TestQmdPort_InvokeErrors(t *testing.T) {
	portFailed := &search.QmdPort{
		Runner: func(args []string) ([]byte, []byte, int, error) {
			return nil, []byte("command failed"), 1, nil
		},
	}

	if _, err := portFailed.Indexed("c"); err == nil {
		t.Error("expected error from failed Indexed, got nil")
	}
	if err := portFailed.Refresh(nil); err == nil {
		t.Error("expected error from failed Refresh, got nil")
	}
	if _, err := portFailed.NotYetSearchable(); err == nil {
		t.Error("expected error from failed NotYetSearchable, got nil")
	}
	if err := portFailed.Embed(nil); err == nil {
		t.Error("expected error from failed Embed, got nil")
	}

	portExecErr := &search.QmdPort{
		Executable: "qmd",
		Runner: func(args []string) ([]byte, []byte, int, error) {
			return nil, nil, 1, errors.New("cannot start")
		},
	}
	if err := portExecErr.Refresh(nil); err == nil {
		t.Error("expected error from execution error, got nil")
	}
}

// stdoutOf is a runner that answers every call with out and exit code 0.
func stdoutOf(out string) search.RunnerFunc {
	return func(argv []string) ([]byte, []byte, int, error) {
		return []byte(out), nil, 0, nil
	}
}

func TestQmdPortPutsTheIndexBehindTheProgram(t *testing.T) {
	var seen [][]string
	port := &search.QmdPort{Executable: "qmd", Index: "loomux-bench-x", Runner: func(argv []string) ([]byte, []byte, int, error) {
		seen = append(seen, argv)
		return []byte("[]"), nil, 0, nil
	}}
	_, _ = port.Search("q", []string{"c"}, search.ProfileKeyword, 5)
	_, _ = port.Indexed("c")
	_ = port.Refresh(nil)
	_ = port.Embed(nil)
	_, _ = port.NotYetSearchable()
	if len(seen) != 5 {
		t.Fatalf("calls = %v", seen)
	}
	for _, argv := range seen {
		if len(argv) < 4 || argv[0] != "qmd" || argv[1] != "--index" || argv[2] != "loomux-bench-x" {
			t.Fatalf("argv = %v", argv)
		}
	}
}

func TestQmdPortCutsTheIndexFromTheFileURI(t *testing.T) {
	out := `[{"file":"qmd://colla/baustatik-04.md?index=loomux-bench-x","docid":"#1"}]`
	port := &search.QmdPort{Index: "loomux-bench-x", Runner: stdoutOf(out)}
	hits, err := port.Search("q", []string{"colla"}, search.ProfileKeyword, 5)
	if err != nil || hits[0].Collection != "colla" || hits[0].Relative != "baustatik-04.md" {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
}

func TestQmdPortWithoutIndexKeepsAQuestionMarkInTheName(t *testing.T) {
	// No index configured: the path is taken as written, as before.
	out := `[{"file":"qmd://c/a?b.md","docid":"#1"}]`
	hits, _ := (&search.QmdPort{Runner: stdoutOf(out)}).Search("q", nil, search.ProfileKeyword, 5)
	if hits[0].Relative != "a?b.md" {
		t.Fatalf("relative = %q", hits[0].Relative)
	}
}

func TestQmdPortWithoutIndexKeepsAnEmptyIndexMarkerInTheName(t *testing.T) {
	// Only a named index is cut from the URI. Without one there is nothing
	// qmd appended, so a name that happens to end in "?index=" (legal on
	// POSIX) stays whole.
	out := `[{"file":"qmd://c/note?index=","docid":"#1"}]`
	hits, _ := (&search.QmdPort{Runner: stdoutOf(out)}).Search("q", nil, search.ProfileKeyword, 5)
	if hits[0].Relative != "note?index=" {
		t.Fatalf("relative = %q", hits[0].Relative)
	}
}
