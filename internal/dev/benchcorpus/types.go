package benchcorpus

import (
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/dev/benchreport"
)

// Options controls repository inspection and benchmarking.
type Options struct {
	TargetDir        string
	CorpusFile       string
	Tier             string
	Languages        int
	WarmRuns         int
	CacheDir         string
	Timeout          time.Duration
	ComponentTimeout time.Duration
}

// ToolSignal represents a detected native tool in a project.
type ToolSignal struct {
	Tool       string `json:"tool"`
	Category   string `json:"category"`
	ConfigFile string `json:"config_file"`
}

// ProjectSignals holds detected capabilities, tools, and hooks for a project.
type ProjectSignals struct {
	HasClaude   bool         `json:"has_claude"`
	HasGitHooks bool         `json:"has_git_hooks"`
	ClaudeHooks []ClaudeHook `json:"claude_hooks,omitempty"`
	NativeTools []ToolSignal `json:"native_tools,omitempty"`
	Languages   []string     `json:"languages,omitempty"`
}

// CheckAudit compares native tools against Loomux lanes.
type CheckAudit struct {
	Tool     string `json:"tool"`
	Category string `json:"category"`
	Native   string `json:"native"`
	Lane     string `json:"lane"`
	OnPath   bool   `json:"on_path"`
}

// RepoAudit holds complete benchmark and gap-audit results for a repository.
type RepoAudit struct {
	RepoURL        string               `json:"repo_url"`
	Dir            string               `json:"dir"`
	Language       string               `json:"language"`
	Framework      string               `json:"framework,omitempty"`
	Tier           string               `json:"tier"`
	CommitSHA      string               `json:"commit_sha,omitempty"`
	SampleFile     string               `json:"sample_file,omitempty"`
	DetectedStacks []string             `json:"detected_stacks"`
	ExecutedLanes  []string             `json:"executed_lanes"`
	Audit          []CheckAudit         `json:"audit"`
	MissingGaps    []string             `json:"missing_gaps"`
	CoverageRate   float64              `json:"coverage_rate"`
	Timings        []benchreport.Timing `json:"timings"`
	HookWarmMedian float64              `json:"hook_warm_median_ms,omitempty"`
	ClaudeWarmMed  float64              `json:"claude_warm_median_ms,omitempty"`
	Speedup        float64              `json:"speedup,omitempty"`
	BaselineError  string               `json:"baseline_error,omitempty"`
}

// TotalTiming names the timing of a whole pass. A row's timings come in this
// order: the total first, then one per component, then one per baseline hook
// under BaselineTiming.
const TotalTiming = "total"

// baselinePrefix sets the Claude hooks a row compares against apart from
// what loomux measured.
const baselinePrefix = "baseline:"

// BaselineTiming names the timing of the Claude hook component.
func BaselineTiming(component string) string { return baselinePrefix + component }

// Timing finds the timing called name.
func (a *RepoAudit) Timing(name string) (benchreport.Timing, bool) {
	for _, t := range a.Timings {
		if t.Name == name {
			return t, true
		}
	}
	return benchreport.Timing{}, false
}

// Components are the steps loomux measured, without the total and the baseline.
func (a *RepoAudit) Components() []benchreport.Timing {
	var parts []benchreport.Timing
	for _, t := range a.Timings {
		if t.Name != TotalTiming && !strings.HasPrefix(t.Name, baselinePrefix) {
			parts = append(parts, t)
		}
	}
	return parts
}

// SkippedRepo names a corpus repository that could not be cloned or benchmarked.
type SkippedRepo struct {
	RepoURL   string `json:"repo_url"`
	Language  string `json:"language"`
	Framework string `json:"framework,omitempty"`
	Reason    string `json:"reason"`
}

// BenchmarkReport summarizes the full benchmark run.
type BenchmarkReport struct {
	Timestamp string        `json:"timestamp"`
	Mode      string        `json:"mode"`
	WarmRuns  int           `json:"warm_runs"`
	Repos     []*RepoAudit  `json:"repos"`
	Skipped   []SkippedRepo `json:"skipped,omitempty"`
}

// ClaudeHook is one command hook from a project's Claude Code settings.
type ClaudeHook struct {
	Event   string `json:"event"`
	Command string `json:"command"`
}
