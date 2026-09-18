package benchcorpus

import (
	"encoding/json"
	"errors"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestBenchmarkRepo(t *testing.T) {
	mockClock := func() func() time.Time {
		currentTime := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
		return func() time.Time {
			currentTime = currentTime.Add(10 * time.Millisecond)
			return currentTime
		}
	}

	mockLookPathSuccess := func(file string) (string, error) {
		return "/bin/" + file, nil
	}

	t.Run("Invalid warm runs returns error", func(t *testing.T) {
		opts := Options{WarmRuns: 0}
		_, err := BenchmarkRepo(".", opts, nil, nil, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "warm_runs") {
			t.Fatalf("expected error for warm_runs < 1, got %v", err)
		}
	})

	t.Run("OpenFS failure returns error", func(t *testing.T) {
		opts := Options{WarmRuns: 2}
		openFSError := func(string) (fs.FS, error) {
			return nil, errors.New("open failed")
		}
		_, err := BenchmarkRepo(".", opts, nil, nil, openFSError, nil)
		if err == nil || !strings.Contains(err.Error(), "open failed") {
			t.Fatalf("expected openFS error, got %v", err)
		}
	})

	t.Run("Go project with all components applicable", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"go.mod":       &fstest.MapFile{Data: []byte("module example.com/test\n\ngo 1.25\n")},
			"main.go":      &fstest.MapFile{Data: []byte("package main\n")},
			"main_test.go": &fstest.MapFile{Data: []byte("package main\n")},
		}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }

		var commandsRun []string
		runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
			commandsRun = append(commandsRun, strings.Join(argv, " "))
			return "ok", 0, false, nil
		}

		opts := Options{WarmRuns: 3}
		audit, err := BenchmarkRepo("/repo/go", opts, runner, mockClock(), openFS, mockLookPathSuccess)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if audit.Dir != "/repo/go" {
			t.Errorf("expected dir /repo/go, got %s", audit.Dir)
		}
		if len(audit.Warm) != 3 {
			t.Fatalf("expected 3 warm runs, got %d", len(audit.Warm))
		}
		if audit.WarmMedian == 0 || audit.WarmMin == 0 || audit.WarmMax == 0 {
			t.Errorf("expected non-zero warm stats: median=%v min=%v max=%v", audit.WarmMedian, audit.WarmMin, audit.WarmMax)
		}

		// Check component applicability
		var graphFound bool
		for _, comp := range audit.Cold.Components {
			if comp.Name == "graph build" {
				graphFound = true
				if !comp.Applicable {
					t.Errorf("expected graph build to be applicable for Go project")
				}
			}
		}
		if !graphFound {
			t.Errorf("expected graph build component in cold run")
		}
	})

	t.Run("Non-Go project with graph build non-applicable", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"package.json": &fstest.MapFile{Data: []byte(`{"name":"foo","scripts":{"test":"vitest"}}`)},
		}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }

		runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
			return "ok", 0, false, nil
		}

		opts := Options{WarmRuns: 2}
		audit, err := BenchmarkRepo("/repo/node", opts, runner, mockClock(), openFS, mockLookPathSuccess)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, comp := range audit.Cold.Components {
			if comp.Name == "graph build" {
				if comp.Applicable {
					t.Errorf("expected graph build to NOT be applicable for non-Go project")
				}
				if comp.Elapsed != 0 {
					t.Errorf("expected graph build elapsed to be 0 when non-applicable, got %v", comp.Elapsed)
				}
			}
		}
	})

	t.Run("Baseline Claude hook calculation and speedup", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"main.go": &fstest.MapFile{Data: []byte("package main\n")},
			"go.mod":  &fstest.MapFile{Data: []byte("module foo\n")},
			".claude/settings.json": &fstest.MapFile{
				Data: []byte(`{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"python -m unittest"}]}]}}`),
			},
		}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }

		runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
			return "ok", 0, false, nil
		}

		opts := Options{WarmRuns: 2}
		audit, err := BenchmarkRepo("/repo/claude", opts, runner, mockClock(), openFS, mockLookPathSuccess)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if audit.ClaudeWarmMed == 0 {
			t.Errorf("expected non-zero ClaudeWarmMed")
		}
		if audit.Speedup <= 0 {
			t.Errorf("expected positive speedup, got %f", audit.Speedup)
		}
	})

	t.Run("Runner returns error during pre-tool-use", func(t *testing.T) {
		mockFS := fstest.MapFS{"go.mod": &fstest.MapFile{Data: []byte("module foo\n")}, "main.go": &fstest.MapFile{Data: []byte("package main\n")}}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }

		runnerErr := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
			if slices.Contains(argv, "pre-tool-use") {
				return "", 1, false, errors.New("pre-tool-use crashed")
			}
			return "ok", 0, false, nil
		}

		opts := Options{WarmRuns: 1}
		_, err := BenchmarkRepo("/repo/err", opts, runnerErr, mockClock(), openFS, mockLookPathSuccess)
		if err == nil || !strings.Contains(err.Error(), "pre-tool-use crashed") {
			t.Fatalf("expected pre-tool-use error, got %v", err)
		}
	})

	t.Run("Runner returns error during post-tool-use", func(t *testing.T) {
		mockFS := fstest.MapFS{"go.mod": &fstest.MapFile{Data: []byte("module foo\n")}, "main.go": &fstest.MapFile{Data: []byte("package main\n")}}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }

		runnerErr := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
			if slices.Contains(argv, "post-tool-use") {
				return "", 1, false, errors.New("post-tool-use crashed")
			}
			return "ok", 0, false, nil
		}

		opts := Options{WarmRuns: 1}
		_, err := BenchmarkRepo("/repo/err", opts, runnerErr, mockClock(), openFS, mockLookPathSuccess)
		if err == nil || !strings.Contains(err.Error(), "post-tool-use crashed") {
			t.Fatalf("expected post-tool-use error, got %v", err)
		}
	})

	t.Run("Runner returns error during graph build", func(t *testing.T) {
		mockFS := fstest.MapFS{"go.mod": &fstest.MapFile{Data: []byte("module foo\n")}, "main.go": &fstest.MapFile{Data: []byte("package main\n")}}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }

		runnerErr := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
			if slices.Contains(argv, "graph") {
				return "", 1, false, errors.New("graph build crashed")
			}
			return "ok", 0, false, nil
		}

		opts := Options{WarmRuns: 1}
		_, err := BenchmarkRepo("/repo/err", opts, runnerErr, mockClock(), openFS, mockLookPathSuccess)
		if err == nil || !strings.Contains(err.Error(), "graph build crashed") {
			t.Fatalf("expected graph build error, got %v", err)
		}
	})

	t.Run("Runner returns error during claude hook", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"main.go": &fstest.MapFile{Data: []byte("package main\n")},
			"go.mod":  &fstest.MapFile{Data: []byte("module foo\n")},
			".claude/settings.json": &fstest.MapFile{
				Data: []byte(`{"hooks":{"PostToolUse":[{"hooks":[{"type":"command","command":"sh fail.sh"}]}]}}`),
			},
		}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }

		runnerErr := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
			if strings.Contains(strings.Join(argv, " "), "sh fail.sh") {
				return "", 1, false, errors.New("claude hook crashed")
			}
			return "ok", 0, false, nil
		}

		opts := Options{WarmRuns: 1}
		audit, err := BenchmarkRepo("/repo/err", opts, runnerErr, mockClock(), openFS, mockLookPathSuccess)
		if err != nil {
			t.Fatalf("a broken baseline must not fail the repository: %v", err)
		}
		if audit.BaselineError != "claude hook crashed" || audit.ClaudeWarmMed != 0 || audit.Speedup != 0 || audit.Cold.Baseline != nil || audit.Warm[0].Baseline != nil {
			t.Errorf("expected an unavailable baseline, got %+v", audit)
		}
		if audit.WarmMedian == 0 {
			t.Errorf("loomux measurement lost")
		}
	})

	t.Run("Project with pre-commit gate matches test commands", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"go.mod": &fstest.MapFile{Data: []byte("module foo\n\ngo 1.25\n")},
			".githooks/pre-commit": &fstest.MapFile{
				Data: []byte("#!/bin/sh\nset -e\ngo test ./... -count=1\n"),
			},
		}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }

		runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
			return "ok", 0, false, nil
		}

		opts := Options{WarmRuns: 1}
		audit, err := BenchmarkRepo("/repo/gate", opts, runner, mockClock(), openFS, mockLookPathSuccess)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var goTestFound bool
		for _, check := range audit.Audit {
			if check.Tool == "go-test" {
				goTestFound = true
				if check.Lane != "go test ./... -count=1" {
					t.Errorf("expected lane 'go test ./... -count=1', got '%s'", check.Lane)
				}
			}
		}
		if !goTestFound {
			t.Errorf("expected go-test in audit")
		}
		if audit.CoverageRate != 100.0 {
			t.Errorf("expected 100.0%% coverage rate, got %f", audit.CoverageRate)
		}
	})
}

func TestCalculateMedian(t *testing.T) {
	if m := calculateMedian(nil); m != 0 {
		t.Errorf("expected 0 for nil, got %v", m)
	}
	if m := calculateMedian([]time.Duration{10 * time.Millisecond}); m != 10*time.Millisecond {
		t.Errorf("expected 10ms for single, got %v", m)
	}
	if m := calculateMedian([]time.Duration{10 * time.Millisecond, 20 * time.Millisecond}); m != 15*time.Millisecond {
		t.Errorf("expected 15ms for even, got %v", m)
	}
}

func TestBenchmarkCorpus(t *testing.T) {
	matrixMD := `# Open Source Matrix

## Python

### Python + Django

| Kategorie | Repository | Sterne | LLM-Nutzung | Beschreibung |
|---|---|---|---|---|
| **Sehr viel (≥100k / Top)** | [django/django](https://github.com/django/django) | ⭐ 91.134 | ` + "`Nein`" + ` | The Web framework |
| **Viel (10k–100k)** | [encode/django-rest-framework](https://github.com/encode/django-rest-framework) | ⭐ 30.188 | ` + "`Nein`" + ` | Web APIs |

### Python + Flask

| Kategorie | Repository | Sterne | LLM-Nutzung | Beschreibung |
|---|---|---|---|---|
| **Sehr viel (≥100k / Top)** | [pallets/flask](https://github.com/pallets/flask) | ⭐ 74.749 | ` + "`Nein`" + ` | Micro framework |
| **Sehr viel (≥100k / Top)** | [django/django](https://github.com/django/django) | ⭐ 91.134 | ` + "`Nein`" + ` | Duplicate entry test |

## JavaScript

### JavaScript + React

| Kategorie | Repository | Sterne | LLM-Nutzung | Beschreibung |
|---|---|---|---|---|
| **Sehr viel (≥100k / Top)** | [facebook/react](https://github.com/facebook/react) | ⭐ 220.000 | ` + "`Nein`" + ` | UI library |
`

	mockCloner := func(repoURL, targetDir string) (string, error) {
		return "abc1234", nil
	}

	mockBenchRepo := func(dir string, opts Options) (*RepoAudit, error) {
		return &RepoAudit{
			Dir:          dir,
			WarmMedian:   50 * time.Millisecond,
			CoverageRate: 100.0,
		}, nil
	}

	t.Run("Corpus run with languages limit and tier filter", func(t *testing.T) {
		opts := Options{
			Languages: 1, // Only Python
			Tier:      "Sehr viel",
			WarmRuns:  2,
			CacheDir:  "/tmp/cache",
		}

		report, err := BenchmarkCorpus([]byte(matrixMD), opts, mockCloner, mockBenchRepo)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if report.Mode != "corpus" {
			t.Errorf("expected mode corpus, got %s", report.Mode)
		}
		if report.WarmRuns != 2 {
			t.Errorf("expected warm runs 2, got %d", report.WarmRuns)
		}
		// Expect django/django and pallets/flask (deduplicated django/django)
		if len(report.Repos) != 2 {
			t.Fatalf("expected 2 unique repos for Python Sehr viel, got %d", len(report.Repos))
		}
		if report.Repos[0].RepoURL != "https://github.com/django/django" {
			t.Errorf("expected django/django, got %s", report.Repos[0].RepoURL)
		}
		if report.Repos[0].CommitSHA != "abc1234" {
			t.Errorf("expected commit sha abc1234, got %s", report.Repos[0].CommitSHA)
		}
		if report.Repos[0].Language != "Python" {
			t.Errorf("expected language Python, got %s", report.Repos[0].Language)
		}
	})

	t.Run("Cloner failure skips the repository and continues", func(t *testing.T) {
		opts := Options{
			Languages: 1,
			Tier:      "Sehr viel",
			WarmRuns:  1,
		}
		clonerFail := func(repoURL, targetDir string) (string, error) {
			if strings.Contains(repoURL, "django") {
				return "", errors.New("git clone: exit status 128: Cloning into 'x'...\nerror: invalid path 'a|b.md'\nfatal: unable to checkout working tree")
			}
			return "abc1234", nil
		}

		report, err := BenchmarkCorpus([]byte(matrixMD), opts, clonerFail, mockBenchRepo)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(report.Repos) != 1 || report.Repos[0].RepoURL != "https://github.com/pallets/flask" {
			t.Fatalf("expected only pallets/flask to be benchmarked, got %+v", report.Repos)
		}
		want := SkippedRepo{
			RepoURL:   "https://github.com/django/django",
			Language:  "Python",
			Framework: "Django",
			Reason:    "error: invalid path 'a|b.md'; fatal: unable to checkout working tree",
		}
		if len(report.Skipped) != 1 || report.Skipped[0] != want {
			t.Fatalf("expected %+v skipped, got %+v", want, report.Skipped)
		}
	})

	t.Run("Every repository failing returns an error", func(t *testing.T) {
		opts := Options{
			Languages: 1,
			Tier:      "Sehr viel",
			WarmRuns:  1,
		}
		benchFail := func(dir string, opts Options) (*RepoAudit, error) {
			return nil, errors.New("bench failed")
		}

		_, err := BenchmarkCorpus([]byte(matrixMD), opts, mockCloner, benchFail)
		if err == nil || !strings.Contains(err.Error(), "none of 2 repositories") || !strings.Contains(err.Error(), "bench failed") {
			t.Fatalf("expected an all-failed error naming the cause, got %v", err)
		}
	})

	t.Run("Corpus run with all languages and no tier filter", func(t *testing.T) {
		opts := Options{
			Languages: 0,  // All languages
			Tier:      "", // All tiers
			WarmRuns:  1,
		}
		report, err := BenchmarkCorpus([]byte(matrixMD), opts, mockCloner, mockBenchRepo)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Expect django/django, encode/django-rest-framework, pallets/flask, facebook/react
		if len(report.Repos) != 4 {
			t.Fatalf("expected 4 repos across all languages and tiers, got %d", len(report.Repos))
		}
	})
}

func fixedClock() func() time.Time {
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	return func() time.Time {
		now = now.Add(10 * time.Millisecond)
		return now
	}
}

func noLookPath(string) (string, error) { return "", errors.New("absent") }

type hookPayloadDoc struct {
	SessionID     string `json:"session_id"`
	HookEventName string `json:"hook_event_name"`
	ToolName      string `json:"tool_name"`
	ToolInput     struct {
		FilePath  string `json:"file_path"`
		OldString string `json:"old_string"`
		NewString string `json:"new_string"`
	} `json:"tool_input"`
	Cwd string `json:"cwd"`
}

func TestBenchmarkRepoHookPayload(t *testing.T) {
	mockFS := fstest.MapFS{
		"go.mod":            &fstest.MapFile{Data: []byte("module foo\n\ngo 1.25\n")},
		".hidden/a.go":      &fstest.MapFile{Data: []byte("package a\n")},
		"build/b.go":        &fstest.MapFile{Data: []byte("package b\n")},
		"cmd/tool/main.go":  &fstest.MapFile{Data: []byte("package main\n")},
		"vendor/x/c.go":     &fstest.MapFile{Data: []byte("package c\n")},
		"zzz_last/other.go": &fstest.MapFile{Data: []byte("package z\n")},
	}
	openFS := func(string) (fs.FS, error) { return mockFS, nil }

	payloads := map[string][]byte{}
	var timeouts []time.Duration
	runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
		timeouts = append(timeouts, timeout)
		if len(argv) > 2 && argv[1] == "hook" {
			payloads[argv[2]] = stdin
			if argv[2] == "pre-tool-use" {
				return "", 2, false, nil
			}
		}
		return "", 0, false, nil
	}

	opts := Options{WarmRuns: 1, ComponentTimeout: 7 * time.Second}
	audit, err := BenchmarkRepo("/repo/go", opts, runner, fixedClock(), openFS, noLookPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if audit.SampleFile != "cmd/tool/main.go" {
		t.Errorf("sample file = %q, want cmd/tool/main.go", audit.SampleFile)
	}
	for _, timeout := range timeouts {
		if timeout != 7*time.Second {
			t.Errorf("runner got timeout %v, want 7s", timeout)
		}
	}
	wantAbs, _ := filepath.Abs("/repo/go")
	for hook, event := range map[string]string{"pre-tool-use": "PreToolUse", "post-tool-use": "PostToolUse"} {
		var doc hookPayloadDoc
		if err := json.Unmarshal(payloads[hook], &doc); err != nil {
			t.Fatalf("%s payload is no JSON: %v", hook, err)
		}
		if doc.SessionID != "loomux-bench" || doc.HookEventName != event || doc.ToolName != "Edit" || doc.Cwd != wantAbs {
			t.Errorf("%s payload = %+v", hook, doc)
		}
		if doc.ToolInput.FilePath != filepath.Join(wantAbs, "cmd", "tool", "main.go") {
			t.Errorf("%s file_path = %q", hook, doc.ToolInput.FilePath)
		}
	}
	pre := audit.Cold.Components[0]
	if pre.Name != "pre-tool-use" || pre.ExitCode != 2 || pre.TimedOut {
		t.Errorf("pre-tool-use timing = %+v", pre)
	}
}

func TestBenchmarkRepoRecordsTimeout(t *testing.T) {
	mockFS := fstest.MapFS{"go.mod": &fstest.MapFile{Data: []byte("module foo\n")}, "main.go": &fstest.MapFile{Data: []byte("package main\n")}}
	openFS := func(string) (fs.FS, error) { return mockFS, nil }
	runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
		if slices.Contains(argv, "post-tool-use") {
			return "", -1, true, nil
		}
		return "", 0, false, nil
	}
	audit, err := BenchmarkRepo("/repo/go", Options{WarmRuns: 1}, runner, fixedClock(), openFS, noLookPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !audit.Cold.Components[1].TimedOut || !audit.Warm[0].Components[1].TimedOut {
		t.Errorf("expected post-tool-use to be recorded as timed out, got %+v", audit.Cold.Components[1])
	}
}

func TestBenchmarkRepoSampleFallbacks(t *testing.T) {
	var hookCalls int
	runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
		if slices.Contains(argv, "hook") {
			hookCalls++
		}
		return "", 0, false, nil
	}

	t.Run("README when no source file matches", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"go.mod":    &fstest.MapFile{Data: []byte("module foo\n")},
			"README.md": &fstest.MapFile{Data: []byte("# foo\n")},
		}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }
		audit, err := BenchmarkRepo("/repo/doc", Options{WarmRuns: 1}, runner, fixedClock(), openFS, noLookPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if audit.SampleFile != "README.md" || !audit.Cold.Components[0].Applicable {
			t.Errorf("expected README.md sample with applicable hooks, got %q %+v", audit.SampleFile, audit.Cold.Components[0])
		}
	})

	t.Run("no file skips hook components", func(t *testing.T) {
		hookCalls = 0
		mockFS := fstest.MapFS{
			"package.json": &fstest.MapFile{Data: []byte(`{"name":"foo"}`)},
			".claude/settings.json": &fstest.MapFile{
				Data: []byte(`{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"true"}]}]}}`),
			},
		}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }
		audit, err := BenchmarkRepo("/repo/empty", Options{WarmRuns: 1}, runner, fixedClock(), openFS, noLookPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hookCalls != 0 || audit.SampleFile != "" {
			t.Errorf("expected no hook calls and no sample, got %d calls, sample %q", hookCalls, audit.SampleFile)
		}
		if audit.Cold.Components[0].Applicable || audit.Cold.Components[1].Applicable {
			t.Errorf("expected hook components to be n/a, got %+v", audit.Cold.Components)
		}
		if audit.ClaudeWarmMed != 0 {
			t.Errorf("expected no baseline without a sample file, got %v", audit.ClaudeWarmMed)
		}
	})
}

func TestBenchmarkRepoBaselineHooks(t *testing.T) {
	mockFS := fstest.MapFS{
		"main.go": &fstest.MapFile{Data: []byte("package main\n")},
		"go.mod":  &fstest.MapFile{Data: []byte("module foo\n")},
		".claude/settings.json": &fstest.MapFile{Data: []byte(`{"hooks":{
			"PreToolUse":[{"matcher":"Edit","hooks":[{"type":"command","command":"pre-check"}]}],
			"PostToolUse":[{"hooks":[{"type":"command","command":"post-check"}]}],
			"Stop":[{"hooks":[{"type":"command","command":"stop-check"}]}]}}`)},
	}
	openFS := func(string) (fs.FS, error) { return mockFS, nil }

	var shells []string
	events := map[string]string{}
	runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
		if argv[0] == "sh" {
			shells = append(shells, argv[2])
			var doc hookPayloadDoc
			_ = json.Unmarshal(stdin, &doc)
			events[argv[2]] = doc.HookEventName
			return "", 1, false, nil
		}
		return "", 0, false, nil
	}
	audit, err := BenchmarkRepo("/it's", Options{WarmRuns: 1}, runner, fixedClock(), openFS, noLookPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	abs, _ := filepath.Abs("/it's")
	quoted := "'" + strings.ReplaceAll(abs, "'", `'\''`) + "'"
	wantPost := "export CLAUDE_PROJECT_DIR=" + quoted + "; post-check"
	wantPre := "export CLAUDE_PROJECT_DIR=" + quoted + "; pre-check"
	if len(shells) != 4 || shells[0] != wantPost || shells[1] != wantPre {
		t.Fatalf("baseline shells = %q", shells)
	}
	if events[wantPost] != "PostToolUse" || events[wantPre] != "PreToolUse" {
		t.Errorf("baseline payload events = %v", events)
	}
	if len(audit.Cold.Baseline) != 2 || audit.Cold.Baseline[0].ExitCode != 1 || audit.Cold.Baseline[0].Name != "claude PostToolUse" {
		t.Errorf("baseline timings = %+v", audit.Cold.Baseline)
	}
	if audit.ClaudeWarmMed == 0 {
		t.Errorf("expected a baseline median")
	}
}

// readDirFailFS refuses to list one directory, as a permission error would.
type readDirFailFS struct{ fstest.MapFS }

func (f readDirFailFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == "locked" {
		return nil, errors.New("permission denied")
	}
	return f.MapFS.ReadDir(name)
}

func TestPickSampleFileSkipsUnreadableDirs(t *testing.T) {
	root := readDirFailFS{fstest.MapFS{
		"locked/a.go": &fstest.MapFile{Data: []byte("package a\n")},
		"src/b.go":    &fstest.MapFile{Data: []byte("package b\n")},
		"notes.md":    &fstest.MapFile{Data: []byte("# notes\n")},
	}}
	if got := pickSampleFile(root, []string{"go", "wiki"}); got != "src/b.go" {
		t.Errorf("pickSampleFile = %q, want src/b.go", got)
	}
}
