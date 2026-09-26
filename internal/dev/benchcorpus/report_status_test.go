package benchcorpus

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/dev/benchreport"
)

// statusAudit carries mixed exit codes, one timeout and a failing baseline.
func statusAudit() *RepoAudit {
	pre := comp("pre-tool-use", 10, 8)
	pre.ExitCodes = []int{2, 0}
	post := comp("post-tool-use", 20, 17)
	post.ExitCodes, post.TimedOut = []int{0, -1}, 1
	base := comp(BaselineTiming("claude PreToolUse"), 50, 40)
	base.ExitCodes = []int{1, 1}
	return &RepoAudit{
		Dir:            "/repo",
		SampleFile:     "cmd/main.go",
		Timings:        timings(benchreport.Summarize("", 30, []float64{25}), pre, post, notApplicable("graph build"), base),
		HookWarmMedian: 25,
		ClaudeWarmMed:  40,
		Speedup:        1.6,
	}
}

func TestStatusColumns(t *testing.T) {
	want := []string{
		"| **pre-tool-use** | 10.0 ms | 8.0 ms | 8.0 ms | 8.0 ms | [0, 2] |",
		"| **post-tool-use** | 20.0 ms | 17.0 ms | 17.0 ms | 17.0 ms | timeout |",
		"| **graph build** | n/a | n/a | n/a | n/a |",
		"| 30.0 ms | 25.0 ms | 25.0 ms | 25.0 ms | timeout |",
		"40.0 ms vs. loomux hooks 25.0 ms (Speedup: 1.6x, Status: [1])",
		"`cmd/main.go`",
	}

	var report bytes.Buffer
	if err := FormatMarkdown(&BenchmarkReport{WarmRuns: 1, Repos: []*RepoAudit{statusAudit()}}, &report); err != nil {
		t.Fatal(err)
	}
	outputs := map[string]string{"report": report.String()}
	for _, lang := range []string{"en", "de"} {
		var buf bytes.Buffer
		if err := FormatDetailMarkdown(statusAudit(), lang, &buf); err != nil {
			t.Fatal(err)
		}
		outputs[lang] = buf.String()
	}
	for name, out := range outputs {
		for _, w := range want {
			if !strings.Contains(out, w) {
				t.Errorf("%s output lacks %q:\n%s", name, w, out)
			}
		}
		if strings.Contains(out, "[0] |") {
			t.Errorf("%s output still carries a literal [0] status:\n%s", name, out)
		}
	}
	if !strings.Contains(outputs["en"], "**Sample file:**") || !strings.Contains(outputs["de"], "**Beispieldatei:**") || !strings.Contains(outputs["report"], "**Beispieldatei:**") {
		t.Errorf("sample file label missing")
	}
}

func TestStatusWithoutSampleOrBaseline(t *testing.T) {
	audit := statusAudit()
	audit.SampleFile = ""
	audit.Timings = audit.Timings[:len(audit.Timings)-1]
	for _, lang := range []string{"en", "de"} {
		var buf bytes.Buffer
		if err := FormatDetailMarkdown(audit, lang, &buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if strings.Contains(out, "Sample file") || strings.Contains(out, "Beispieldatei") || strings.Contains(out, "Status: [") {
			t.Errorf("%s: unexpected sample or baseline status:\n%s", lang, out)
		}
		if !strings.Contains(out, "(Speedup: 1.6x)") {
			t.Errorf("%s: baseline line lost:\n%s", lang, out)
		}
	}
}

func TestTimingWithoutOutcomeFieldsLoadsAsApplicable(t *testing.T) {
	stored := `{"repos":[{"dir":".","timings":[{"name":"pre-tool-use","cold_ms":1,"warm_ms":[1],"median_ms":1,"min_ms":1,"max_ms":1}]}]}`
	var report BenchmarkReport
	if err := json.Unmarshal([]byte(stored), &report); err != nil {
		t.Fatalf("stored report no longer loads: %v", err)
	}
	pre, ok := report.Repos[0].Timing("pre-tool-use")
	if !ok || pre.Applicable != nil || pre.TimedOut != 0 || pre.ExitCodes != nil || report.Repos[0].SampleFile != "" {
		t.Errorf("stored report decoded with unexpected fields: %+v", report.Repos[0])
	}
}
