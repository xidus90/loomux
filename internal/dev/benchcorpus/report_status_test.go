package benchcorpus

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// statusAudit carries mixed exit codes, one timeout and a failing baseline.
func statusAudit() *RepoAudit {
	ms := time.Millisecond
	return &RepoAudit{
		Dir:        "/repo",
		SampleFile: "cmd/main.go",
		Cold: TimingRun{
			Total: 30 * ms,
			Components: []ComponentTiming{
				{Name: "pre-tool-use", Applicable: true, Elapsed: 10 * ms, ExitCode: 2},
				{Name: "post-tool-use", Applicable: true, Elapsed: 20 * ms},
				{Name: "graph build"},
			},
			Baseline: []ComponentTiming{{Name: "claude PreToolUse", Applicable: true, Elapsed: 50 * ms, ExitCode: 1}},
		},
		Warm: []TimingRun{{
			Total: 25 * ms,
			Components: []ComponentTiming{
				{Name: "pre-tool-use", Applicable: true, Elapsed: 8 * ms},
				{Name: "post-tool-use", Applicable: true, Elapsed: 17 * ms, ExitCode: -1, TimedOut: true},
				{Name: "graph build"},
			},
			Baseline: []ComponentTiming{{Name: "claude PreToolUse", Applicable: true, Elapsed: 40 * ms, ExitCode: 1}},
		}},
		WarmMedian:     25 * ms,
		WarmMin:        25 * ms,
		WarmMax:        25 * ms,
		HookWarmMedian: 25 * ms,
		ClaudeWarmMed:  40 * ms,
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
	audit.Cold.Baseline = nil
	audit.Warm[0].Baseline = nil
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

func TestOldReportJSONStillLoads(t *testing.T) {
	old := `{"repos":[{"dir":".","cold":{"total":1,"components":[{"name":"pre-tool-use","applicable":true,"elapsed":1}]},"warm":[],"warm_median":1}]}`
	var report BenchmarkReport
	if err := json.Unmarshal([]byte(old), &report); err != nil {
		t.Fatalf("old report no longer loads: %v", err)
	}
	comp := report.Repos[0].Cold.Components[0]
	if comp.ExitCode != 0 || comp.TimedOut || report.Repos[0].SampleFile != "" {
		t.Errorf("old report decoded with unexpected new fields: %+v", report.Repos[0])
	}
}
