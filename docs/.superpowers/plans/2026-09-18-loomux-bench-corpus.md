# loomux dev bench Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `loomux dev bench`, a comprehensive benchmark and gap-audit tool for single repositories and the open-source matrix corpus (top-N languages, default 5), measuring 1 cold + N warm runs (median/min/max) and reporting coverage gaps.

**Architecture:** A modular Go package under `internal/dev/benchcorpus/` with clean separation between project inspection (`inspect.go`), gap auditing (`gap.go`), timing/benchmarking engine (`benchcorpus.go`), and deterministic reporting (`report.go`), exposed via `loomux dev bench` in `internal/cli/dev.go` using a new exported hook inspection API in `internal/hooks`.

**Tech Stack:** Go 1.25.0, standard library (`os`, `io`, `time`, `flag`, `encoding/json`, `io/fs`), Loomux internal modules (`detect`, `hooks`, `graph`).

**Spec:** `docs/.superpowers/specs/2026-09-18-loomux-bench-corpus-design.md`

## Global Constraints

- Go 100% coverage rule per function: Every function must have 100% statement coverage unless explicitly marked with `//coverage:exempt <reason>`.
- Zero external dependencies: Standard library and internal packages only.
- Strict dependency injection: All clocks (`time.Time`), processes (`exec.Cmd`), and filesystems (`fs.FS`) must be injectable for deterministic, zero-network unit tests.
- Working tree isolation: All changes live in the `.worktrees/open-source-matrix` worktree on branch `open-source-matrix`.
- Multilingual parity: Maintain `README.md`/`README.de.md` and `docs/en/`/`docs/de/` docs alongside changes.

---

### Task 0: Export Hook Lanes Inspection API in `internal/hooks`

**Files:**
- Modify: `internal/hooks/post_edit.go`
- Test: `internal/hooks/post_edit_test.go`

**Interfaces:**
- Produces:
  - `func TargetCommandsForStacks(stacks []string, targetPath string, godotDir string, projectRoot string, wikiDir string) []string`

- [ ] **Step 1: Write failing test for TargetCommandsForStacks**

In `internal/hooks/post_edit_test.go`, add `TestTargetCommandsForStacks` testing that `TargetCommandsForStacks` returns the command strings for python (`ruff check`, `mypy`), typescript (`eslint`, `tsc`), go (`go vet`), etc.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/hooks -run TestTargetCommandsForStacks`
Expected: FAIL (undefined: TargetCommandsForStacks).

- [ ] **Step 3: Implement TargetCommandsForStacks**

In `internal/hooks/post_edit.go`, export `TargetCommandsForStacks` wrapping `getCommandsForStacks` and returning the string command lines.

- [ ] **Step 4: Run tests and verify 100% coverage**

Run: `go test ./internal/hooks -run TestTargetCommandsForStacks -coverprofile=coverage.out`
Expected: PASS with 100% function coverage.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/post_edit.go internal/hooks/post_edit_test.go
git commit -m "feat(hooks): export TargetCommandsForStacks for lane inspection"
```

---

### Task 1: Models & Normalized Project Inspection Engine

**Files:**
- Create: `internal/dev/benchcorpus/types.go`
- Create: `internal/dev/benchcorpus/inspect.go`
- Test: `internal/dev/benchcorpus/inspect_test.go`

**Interfaces:**
- Produces:
  - `type Options struct { TargetDir, CorpusFile, Tier, CacheDir, OutFile, JSONOutFile string; Languages, WarmRuns int; Timeout time.Duration }`
  - `type ToolSignal struct { Tool, Category, ConfigFile string }`
  - `type ProjectSignals struct { HasClaude, HasGitHooks bool; ClaudeHooks []string; NativeTools []ToolSignal; Languages []string }`
  - `func InspectProject(root fs.FS) ProjectSignals`

- [ ] **Step 1: Write failing tests for InspectProject**

Create `internal/dev/benchcorpus/inspect_test.go` covering Python (`pyproject.toml` with ruff/pytest/mypy), JavaScript/TypeScript (`package.json` with scripts: `"test": "vitest"`, `"lint": "eslint"`), Go (`go.mod`, `*_test.go`), Java (`pom.xml`), C# (`*.csproj`), and `.claude/settings.json` detection using `testing/fstest.MapFS`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/dev/benchcorpus -run TestInspectProject`
Expected: FAIL.

- [ ] **Step 3: Implement types and InspectProject**

Implement `types.go` and `inspect.go` scanning `fs.FS` root and 1 level down for native tools, parsing script definitions to normalized tool names (`vitest`, `jest`, `pytest`, `ruff`, `eslint`, etc.).

- [ ] **Step 4: Run tests and verify 100% coverage**

Run: `go test ./internal/dev/benchcorpus -run TestInspectProject -coverprofile=coverage.out`
Expected: PASS with 100% function coverage.

- [ ] **Step 5: Commit**

```bash
git add internal/dev/benchcorpus/types.go internal/dev/benchcorpus/inspect.go internal/dev/benchcorpus/inspect_test.go
git commit -m "feat(benchcorpus): add models and normalized project inspection engine"
```

---

### Task 2: Normalized Gap Audit Engine

**Files:**
- Create: `internal/dev/benchcorpus/gap.go`
- Test: `internal/dev/benchcorpus/gap_test.go`

**Interfaces:**
- Consumes: `ProjectSignals` from Task 1, `detect.Facts` from `internal/detect`.
- Produces:
  - `type CheckAudit struct { Tool, Category, Native, Lane string; OnPath bool }`
  - `func AuditGaps(project ProjectSignals, detectedStacks []string, executedLanes []string, lookPath func(string) (string, error)) ([]CheckAudit, []string, float64)`

- [ ] **Step 1: Write failing tests for AuditGaps**

Create `internal/dev/benchcorpus/gap_test.go` testing:
1. Full coverage scenario (`ruff` and `mypy` both native and in lanes).
2. Missing lane scenario (`pytest` native, but no lane in Loomux).
3. Missing stack scenario (`dotnet-test` in C# project not in detected stacks).
4. Tool missing from PATH (`lookPath` returns error -> `OnPath: false`).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/dev/benchcorpus -run TestAuditGaps`
Expected: FAIL.

- [ ] **Step 3: Implement AuditGaps**

Implement `internal/dev/benchcorpus/gap.go` matching normalized tools to lanes, calculating coverage rate and missing gaps.

- [ ] **Step 4: Run tests and verify 100% coverage**

Run: `go test ./internal/dev/benchcorpus -run TestAuditGaps -coverprofile=coverage.out`
Expected: PASS with 100% function coverage.

- [ ] **Step 5: Commit**

```bash
git add internal/dev/benchcorpus/gap.go internal/dev/benchcorpus/gap_test.go
git commit -m "feat(benchcorpus): add normalized gap audit engine"
```

---

### Task 3: Benchmarking Engine (Cold + Warm Median, min, max)

**Files:**
- Create: `internal/dev/benchcorpus/benchcorpus.go`
- Test: `internal/dev/benchcorpus/benchcorpus_test.go`

**Interfaces:**
- Consumes: `InspectProject` (Task 1), `AuditGaps` (Task 2), `hooks.TargetCommandsForStacks` (Task 0).
- Produces:
  - `type ProcessRunner func(dir string, argv []string) (output string, exitCode int, err error)`
  - `type Cloner func(repoURL, targetDir string) (commitSHA string, err error)`
  - `func BenchmarkRepo(dir string, opts Options, runner ProcessRunner, clock func() time.Time, openFS func(string) (fs.FS, error), lookPath func(string) (string, error)) (*RepoAudit, error)`
  - `func BenchmarkCorpus(matrixData []byte, opts Options, cloner Cloner, benchRepo func(string, Options) (*RepoAudit, error)) (*BenchmarkReport, error)`

- [ ] **Step 1: Write failing tests for BenchmarkRepo and BenchmarkCorpus**

Create `internal/dev/benchcorpus/benchcorpus_test.go` testing:
1. Cold + Warm runs with median, min, max.
2. CodeGraph marked `Applicable: false` on non-Go projects.
3. Speedup and delta calculation when baseline Claude hook exists.
4. Deduplication of repositories and deterministic order in corpus mode.
5. Error handling for invalid flags (`WarmRuns < 1`, invalid dir).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/dev/benchcorpus -run TestBenchmark`
Expected: FAIL.

- [ ] **Step 3: Implement BenchmarkRepo and BenchmarkCorpus**

Implement `internal/dev/benchcorpus/benchcorpus.go` with strict dependency injection, ordered timings slice, median calculations, and gap auditing.

- [ ] **Step 4: Run tests and verify 100% coverage**

Run: `go test ./internal/dev/benchcorpus -coverprofile=coverage.out`
Expected: PASS with 100% function coverage.

- [ ] **Step 5: Commit**

```bash
git add internal/dev/benchcorpus/benchcorpus.go internal/dev/benchcorpus/benchcorpus_test.go
git commit -m "feat(benchcorpus): implement benchmarking engine with median timing and corpus runner"
```

---

### Task 4: Deterministic Report Formatter (Markdown & JSON)

**Files:**
- Create: `internal/dev/benchcorpus/report.go`
- Test: `internal/dev/benchcorpus/report_test.go`

**Interfaces:**
- Consumes: `BenchmarkReport` and `RepoAudit` from Task 3.
- Produces:
  - `func FormatMarkdown(report *BenchmarkReport, w io.Writer) error`
  - `func FormatJSON(report *BenchmarkReport, w io.Writer) error`

- [ ] **Step 1: Write failing tests for FormatMarkdown and FormatJSON**

Create `internal/dev/benchcorpus/report_test.go` testing deterministic Markdown table output (handling `n/a` for non-applicable components, speedup column, check audit table) and JSON serialization.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/dev/benchcorpus -run TestFormat`
Expected: FAIL.

- [ ] **Step 3: Implement FormatMarkdown and FormatJSON**

Implement `internal/dev/benchcorpus/report.go` ensuring exact column alignments, deterministic ordering, and valid JSON.

- [ ] **Step 4: Run tests and verify 100% coverage**

Run: `go test ./internal/dev/benchcorpus -coverprofile=coverage.out`
Expected: PASS with 100% function coverage.

- [ ] **Step 5: Commit**

```bash
git add internal/dev/benchcorpus/report.go internal/dev/benchcorpus/report_test.go
git commit -m "feat(benchcorpus): add deterministic markdown and JSON report formatters"
```

---

### Task 5: CLI Integration (`loomux dev bench`)

**Files:**
- Modify: `internal/cli/dev.go`
- Test: `internal/cli/dev_test.go`

**Interfaces:**
- Produces: Subcommand `dev bench` in `devCommands` table in `internal/cli/dev.go`.

- [ ] **Step 1: Write failing CLI tests in dev_test.go**

Add tests in `internal/cli/dev_test.go` for `loomux dev bench` flag parsing:
1. Default flags (`--dir=.`, `--warm=3`, `--languages=5`, `--tier="Sehr viel"`).
2. Validation errors (`--warm=0`, `--languages=0`, `--dir` not existing).
3. Execution routing for single-repo mode and corpus mode.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run TestDevBench`
Expected: FAIL (dev bench unknown).

- [ ] **Step 3: Implement devBench in dev.go**

Add `devBench` to `devCommands` in `internal/cli/dev.go`, wire up flags and invoke `benchcorpus.BenchmarkRepo` / `benchcorpus.BenchmarkCorpus`.

- [ ] **Step 4: Run all CLI tests and verify 100% coverage**

Run: `go test ./internal/cli -coverprofile=coverage.out`
Expected: PASS with 100% function coverage.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/dev.go internal/cli/dev_test.go
git commit -m "feat(cli): register loomux dev bench command"
```

---

### Task 6: Full Verification, Documentation & Pilot Runs

**Files:**
- Modify: `README.md`, `README.de.md`
- Modify: `docs/en/cli-reference.md`, `docs/de/cli-reference.md`
- Output: `docs/en/benchmarks.md`, `docs/de/benchmarks.md`

- [ ] **Step 1: Run full pre-commit gate**

Run:
```bash
gofmt -l internal/hooks internal/dev/benchcorpus internal/cli
go vet ./...
go test ./... -count=1 -covermode=set -coverpkg=github.com/xidus90/loomux/... -coverprofile=coverage.out
go run ./cmd/loomux dev covergate --profile coverage.out
```
Expected: PASS with 100% statement coverage across all functions.

- [ ] **Step 2: Document dev bench in README and CLI reference**

Update `README.md`, `README.de.md`, `docs/en/cli-reference.md`, and `docs/de/cli-reference.md` with `loomux dev bench` usage and flags.

- [ ] **Step 3: Pilot run on local repository (Single-Repo Mode)**

Run:
```bash
go run ./cmd/loomux dev bench --dir . --warm 3
```
Verify stdout output formatting and gap audit.

- [ ] **Step 4: Final commit**

```bash
git add README.md README.de.md docs/
git commit -m "docs: document loomux dev bench and record pilot benchmark"
```
