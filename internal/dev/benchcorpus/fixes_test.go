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

	"github.com/xidus90/loomux/internal/dev/benchreport"
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
		runs []benchreport.Timing
		want string
	}{
		{"single", []benchreport.Timing{{ExitCodes: []int{0, 0}}}, "[0]"},
		{"distinct sorted", []benchreport.Timing{{ExitCodes: []int{2, 0, 2}}}, "[0, 2]"},
		{"across timings", []benchreport.Timing{{ExitCodes: []int{2}}, {ExitCodes: []int{0}}}, "[0, 2]"},
		{"timeout wins", []benchreport.Timing{{ExitCodes: []int{0}}, {ExitCodes: []int{-1}, TimedOut: 1}}, "timeout"},
		{"none", nil, "n/a"},
	}
	for _, c := range cases {
		if got := exitStatus(c.runs...); got != c.want {
			t.Errorf("%s: exitStatus = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestTotalStatusSkipsInapplicableAndBaseline(t *testing.T) {
	a := comp("a", 1, 1)
	a.ExitCodes = []int{2, 0}
	base := comp(BaselineTiming("claude PreToolUse"), 1, 1)
	base.ExitCodes = []int{5, 5}
	audit := &RepoAudit{Timings: timings(benchreport.Timing{}, a, notApplicable("b"), base)}
	if got := exitStatus(allRuns(audit)...); got != "[0, 2]" {
		t.Errorf("total status = %q", got)
	}
	if got := exitStatus(baselineRuns(audit)...); got != "[5]" {
		t.Errorf("baseline status = %q", got)
	}
}

func TestAllRunsTakesExactlyTheApplicableComponents(t *testing.T) {
	yes := true
	marked := comp("marked", 1, 1)
	marked.Applicable = &yes
	skipped := notApplicable("skipped")
	skipped.ExitCodes = []int{7}
	audit := &RepoAudit{Timings: timings(benchreport.Timing{}, comp("unmarked", 1, 1), marked, skipped)}
	var names []string
	for _, r := range allRuns(audit) {
		names = append(names, r.Name)
	}
	if strings.Join(names, ",") != "unmarked,marked" {
		t.Errorf("allRuns = %v, want unmarked and marked", names)
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
	if total, _ := audit.Timing(TotalTiming); audit.HookWarmMedian != 20 || total.MedianMS != 30 {
		t.Errorf("hook median %v, total median %v", audit.HookWarmMedian, total.MedianMS)
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
		Timings:       timings(benchreport.Timing{}, notApplicable("graph build")),
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
	if runs := baselineRuns(audit); runs != nil {
		t.Errorf("baseline timings left behind: %+v", runs)
	}
	if shCalls != 3 {
		t.Errorf("baseline kept running after it failed: %d calls", shCalls)
	}
}
