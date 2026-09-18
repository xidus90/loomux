package benchcorpus

import (
	"bytes"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestFindMatchingLaneBoundary(t *testing.T) {
	if got := findMatchingLane("go-test", []string{"cargo test"}); got != "" {
		t.Errorf("cargo test must not satisfy go-test, got %q", got)
	}
	if got := findMatchingLane("go-test", []string{"set -e; go test ./..."}); got != "set -e; go test ./..." {
		t.Errorf("expected chained go test to match, got %q", got)
	}
	if got := findMatchingLane("go-test", []string{"go testify"}); got != "" {
		t.Errorf("go testify must not satisfy go-test, got %q", got)
	}
	if got := findMatchingLane("go-test", []string{"(go test)"}); got != "" {
		t.Errorf("a trailing parenthesis is no command end, got %q", got)
	}
	if got := findMatchingLane("go-vet", []string{"x|go vet"}); got != "x|go vet" {
		t.Errorf("expected piped go vet to match, got %q", got)
	}
}

func TestParseMatrixTierExact(t *testing.T) {
	data := []byte("## Go\n| Viel | [a/b](https://github.com/a/b) |\n| **Sehr viel** | [c/d](https://github.com/c/d) |\n")
	entries := parseMatrix(data, 0, "viel")
	if len(entries) != 1 || entries[0].RepoURL != "https://github.com/a/b" {
		t.Errorf("expected only the Viel entry, got %+v", entries)
	}
}

func TestExitStatus(t *testing.T) {
	cases := []struct {
		name string
		runs []ComponentTiming
		want string
	}{
		{"single", []ComponentTiming{{ExitCode: 0}, {ExitCode: 0}}, "[0]"},
		{"distinct sorted", []ComponentTiming{{ExitCode: 2}, {ExitCode: 0}, {ExitCode: 2}}, "[0, 2]"},
		{"timeout wins", []ComponentTiming{{ExitCode: 0}, {TimedOut: true, ExitCode: -1}}, "timeout"},
		{"none", nil, "n/a"},
	}
	for _, c := range cases {
		if got := exitStatus(c.runs); got != c.want {
			t.Errorf("%s: exitStatus = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestComponentStatusesSkipsInapplicable(t *testing.T) {
	cold := TimingRun{Components: []ComponentTiming{{Name: "a", Applicable: true, ExitCode: 2}, {Name: "b"}}}
	warm := []TimingRun{{Components: []ComponentTiming{{Name: "a", Applicable: true, ExitCode: 0}}}}
	if got := exitStatus(componentRuns(cold, warm, 0)); got != "[0, 2]" {
		t.Errorf("component status = %q", got)
	}
	if got := exitStatus(allRuns(cold, warm)); got != "[0, 2]" {
		t.Errorf("total status = %q", got)
	}
}

func TestPickSampleFilePrefersPrimaryLanguage(t *testing.T) {
	root := fstest.MapFS{
		"a.sh":      &fstest.MapFile{Data: []byte("echo\n")},
		"b.css":     &fstest.MapFile{Data: []byte("a{}\n")},
		"docs/n.md": &fstest.MapFile{Data: []byte("# n\n")},
		"z/main.go": &fstest.MapFile{Data: []byte("package main\n")},
		"README.md": &fstest.MapFile{Data: []byte("# r\n")},
	}
	cases := []struct {
		stacks []string
		want   string
	}{
		{[]string{"shell", "css", "go"}, "z/main.go"},
		{[]string{"css", "shell"}, "a.sh"},
		{[]string{"wiki"}, "README.md"},
		{[]string{"rust"}, "README.md"},
	}
	for _, c := range cases {
		if got := pickSampleFile(root, c.stacks); got != c.want {
			t.Errorf("pickSampleFile(%v) = %q, want %q", c.stacks, got, c.want)
		}
	}
}

func TestSpeedupComparesHooksOnly(t *testing.T) {
	mockFS := fstest.MapFS{
		"main.go":               &fstest.MapFile{Data: []byte("package main\n")},
		"go.mod":                &fstest.MapFile{Data: []byte("module foo\n")},
		".claude/settings.json": &fstest.MapFile{Data: []byte(`{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"x"}]}]}}`)},
	}
	openFS := func(string) (fs.FS, error) { return mockFS, nil }
	runner := func(string, []string, []byte, time.Duration) (string, int, bool, error) { return "", 0, false, nil }
	audit, err := BenchmarkRepo("/r", Options{WarmRuns: 1}, runner, fixedClock(), openFS, noLookPath)
	if err != nil {
		t.Fatal(err)
	}
	// Every call takes one 10 ms clock tick: two hooks, graph build, one baseline hook.
	if audit.HookWarmMedian != 20*time.Millisecond || audit.WarmMedian != 30*time.Millisecond {
		t.Errorf("hook median %v, total median %v", audit.HookWarmMedian, audit.WarmMedian)
	}
	if audit.Speedup != 0.5 {
		t.Errorf("speedup = %v, want 0.5", audit.Speedup)
	}
}

func TestBenchmarkRepoResolvesRelativeRoot(t *testing.T) {
	mockFS := fstest.MapFS{"go.mod": &fstest.MapFile{Data: []byte("module foo\n")}, "main.go": &fstest.MapFile{Data: []byte("package main\n")}}
	openFS := func(string) (fs.FS, error) { return mockFS, nil }
	want, _ := filepath.Abs(filepath.Join(".cache", "repo"))
	runner := func(dir string, argv []string, _ []byte, _ time.Duration) (string, int, bool, error) {
		if dir != want || argv[len(argv)-1] != want {
			t.Errorf("runner dir %q, argv %q; want root %q", dir, argv, want)
		}
		return "", 0, false, nil
	}
	audit, err := BenchmarkRepo(".cache/repo", Options{WarmRuns: 1}, runner, fixedClock(), openFS, noLookPath)
	if err != nil {
		t.Fatal(err)
	}
	if audit.Dir != ".cache/repo" {
		t.Errorf("audit dir = %q", audit.Dir)
	}
}

func TestBaselineUnavailableLine(t *testing.T) {
	audit := &RepoAudit{
		Dir:           "/r",
		BaselineError: `exec: "sh": not found`,
		Cold:          TimingRun{Components: []ComponentTiming{{Name: "graph build"}}},
	}
	want := map[string]string{
		"en": "- **Baseline Claude Hook:** unavailable (exec: \"sh\": not found)",
		"de": "- **Baseline Claude Hook:** nicht verfügbar (exec: \"sh\": not found)",
	}
	for lang, line := range want {
		var buf bytes.Buffer
		if err := FormatDetailMarkdown(audit, lang, &buf); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), line) {
			t.Errorf("%s lacks %q:\n%s", lang, line, buf.String())
		}
	}
	var buf bytes.Buffer
	if err := FormatMarkdown(&BenchmarkReport{Repos: []*RepoAudit{audit}}, &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), want["de"]) {
		t.Errorf("report lacks the unavailable baseline:\n%s", buf.String())
	}
}

func TestBaselineFailingInAWarmPassDropsEarlierBaselines(t *testing.T) {
	mockFS := fstest.MapFS{
		"go.mod":                &fstest.MapFile{Data: []byte("module foo")},
		"main.go":               &fstest.MapFile{Data: []byte("package main\n")},
		".claude/settings.json": &fstest.MapFile{Data: []byte(`{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"x"}]}]}}`)},
	}
	openFS := func(string) (fs.FS, error) { return mockFS, nil }
	shCalls := 0
	runner := func(_ string, argv []string, _ []byte, _ time.Duration) (string, int, bool, error) {
		if argv[0] == "sh" {
			shCalls++
			if shCalls == 3 {
				return "", -1, false, errors.New("sh vanished")
			}
		}
		return "", 0, false, nil
	}
	audit, err := BenchmarkRepo("/r", Options{WarmRuns: 3}, runner, fixedClock(), openFS, noLookPath)
	if err != nil {
		t.Fatal(err)
	}
	if audit.BaselineError != "sh vanished" || audit.ClaudeWarmMed != 0 {
		t.Errorf("expected unavailable baseline, got %q %v", audit.BaselineError, audit.ClaudeWarmMed)
	}
	for _, run := range append([]TimingRun{audit.Cold}, audit.Warm...) {
		if run.Baseline != nil {
			t.Errorf("baseline timings left behind: %+v", run.Baseline)
		}
	}
	if shCalls != 3 {
		t.Errorf("baseline kept running after it failed: %d calls", shCalls)
	}
}
