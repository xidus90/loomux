package benchcorpus

import (
	"time"
)

// Options controls repository inspection and benchmarking.
type Options struct {
	TargetDir        string
	CorpusFile       string
	Tier             string
	Languages        int
	WarmRuns         int
	CacheDir         string
	OutFile          string
	JSONOutFile      string
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

// ComponentTiming records execution time for one pipeline step.
type ComponentTiming struct {
	Name       string        `json:"name"`
	Applicable bool          `json:"applicable"`
	Elapsed    time.Duration `json:"elapsed"`
	ExitCode   int           `json:"exit_code,omitempty"`
	TimedOut   bool          `json:"timed_out,omitempty"`
}

// TimingRun represents one measurement pass (cold or warm).
type TimingRun struct {
	Total      time.Duration     `json:"total"`
	Components []ComponentTiming `json:"components"`
	Baseline   []ComponentTiming `json:"baseline,omitempty"`
}

// RepoAudit holds complete benchmark and gap-audit results for a repository.
type RepoAudit struct {
	RepoURL        string        `json:"repo_url"`
	Dir            string        `json:"dir"`
	Language       string        `json:"language"`
	Framework      string        `json:"framework,omitempty"`
	Tier           string        `json:"tier"`
	CommitSHA      string        `json:"commit_sha,omitempty"`
	SampleFile     string        `json:"sample_file,omitempty"`
	DetectedStacks []string      `json:"detected_stacks"`
	ExecutedLanes  []string      `json:"executed_lanes"`
	Audit          []CheckAudit  `json:"audit"`
	MissingGaps    []string      `json:"missing_gaps"`
	CoverageRate   float64       `json:"coverage_rate"`
	Cold           TimingRun     `json:"cold"`
	Warm           []TimingRun   `json:"warm"`
	WarmMedian     time.Duration `json:"warm_median"`
	WarmMin        time.Duration `json:"warm_min"`
	WarmMax        time.Duration `json:"warm_max"`
	HookWarmMedian time.Duration `json:"hook_warm_median,omitempty"`
	ClaudeWarmMed  time.Duration `json:"claude_warm_median,omitempty"`
	Speedup        float64       `json:"speedup,omitempty"`
	BaselineError  string        `json:"baseline_error,omitempty"`
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
