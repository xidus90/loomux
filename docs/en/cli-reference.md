# Loomux CLI Reference Manual

This manual documents the syntax, flags, standard I/O contracts, and exit codes for all Loomux command-line interfaces.

---

## 1. Global Conventions

### Exit Codes
Loomux uses strict exit code semantics aligned with AI coding agent harnesses:

| Exit Code | Meaning | Behavior in Harness (Claude / Antigravity) |
|---|---|---|
| **`0`** | **Success / Allowed** | Tool execution proceeds; turn passes. |
| **`1`** | **Announcement / Warning** | Read by harnesses as "informational" or "carry-on". |
| **`2`** | **Hard Refusal / Denied** | The write barrier or policy blocked the action. Tool execution is aborted. |

### Global Flags & Environment
- `--root <path>`: Explicit project root directory. If omitted, Loomux walks upwards from the current working directory until it locates `.loomux/config.toml`.
- `LOOMUX_STATE_DIR`: Overrides the global state directory (defaults to `%LOCALAPPDATA%\loomux` on Windows or `~/.local/state/loomux` on POSIX).
- `LOOMUX_LEGACY_BRAIN_DIR`: ultra-brain's state directory, read as the fallback for brain artefacts and never written (see section 7).

---

## 2. Policy & Verification (`loomux check`)

`loomux check` takes a **request** first and its flags after it. The request is
a profile (`edit`, `precommit`, `stop` or one of `[verify.profiles]`), `all`, a comma
list of kinds (`lint,types`), or one of the three built-in checks below. What
each kind runs per stack is set by
[`[verify]`](configuration.md#verify-check-chains--quality-gates) and the
presets.

### `loomux check <request> [--root <path>] [--show] [-v]`
Runs the lanes of the requested kinds for every active stack and area, and
judges them.

- **Flags** (after the request; `check --show` alone is a usage error):
  - `--root <path>` — project root; found upwards from the working directory
    when empty, the working directory when nothing is found. A project without
    `.loomux/config.toml` is checked by the presets alone.
  - `--show` — run nothing; print the effective lanes as `[verify]` tables.
  - `-v` — print the output of green lanes too.
- **Order**: a lane starts as soon as the lane it waits for (`after`) is done,
  with at most `max_parallel` processes at once. The report comes at the end,
  never interleaved: kinds in request order, within them stacks in byte order,
  then areas. The one exception is `lint/wiki`, the lint over the wiki bundle
  that runs in this process wherever `lint` is requested and the project has a
  wiki: it comes last. It checks the bundle's structure only; the drift rule
  stays with `loomux wiki-gate`.
- **Output** (all on `stdout`): one line per lane,
  `<kind>/<stack>[@<area>]: <state> [<origin>]`, followed by the duration for
  a lane that started, `by <lane>` for a blocked one, or the reason for one
  that never started. `<origin>` is `preset`, `preset, variant <signal>`,
  `config` or `in-process`. A red lane prints its output below its line; a
  green one only with `-v`. A lane with several commands prints one block per
  command, headed `$ <argv>`, with `(failed)` on a red one. A kind that had
  nothing to check ends the report with ``nothing to check for `<kind>` ``.
  ```text
  lint/go: ok [preset] 0.2s
  types/go: not-applicable [preset] no command
  test/go: ok [preset] 0.5s
  coverage/go: failed [preset] 0.1s
  not covered: w.go:3 A 66.7%
  ```
- **Coverage files**: measuring lanes write to `.loomux/state/cover/`, which
  `check` and the post-edit hook create themselves. A green
  run deletes its own files, a red one keeps them; other runs' files go once
  they are 24 hours old.
- **Exit Codes**: `0` (no lane red, and every requested kind had a lane that
  ran or is `not-applicable` somewhere), `1` (a lane is red, a kind had nothing
  to check, or `[verify]` or the request cannot be loaded; a load error is one
  line on `stderr`), `2` (malformed call: no request, a flag before the
  request, an unknown flag, a second request).

### `loomux check <request> --show`
Prints what the request would run, as TOML, and runs nothing.

```text
# max_parallel = 16, timeout = 600s

# go: areas ., tests found
[verify.go.lint]
commands = ["go vet ./...", "{loomux} check gofmt ."]  # preset
on_file = ["go vet ./...", "{loomux} check gofmt {file}"]  # preset
threaded = true  # preset

# [verify.go.types] not defined
```

- The first line holds the run's limits; each active stack gets a comment
  with its areas and its test state (`tests found`, `no tests found`, or
  `tests not detected` for a stack without a test signal).
- Every key carries the origin of its **lane**: a table that changes one key
  of a preset marks the whole lane `config`.
- There is no budget line: `loomux check` has no budget. The post-edit budget
  is the hook flag `--budget`.
- The output loads as it stands and can be copied into
  `.loomux/config.toml`. A lane with nothing to run is a comment,
  `# [verify.<stack>.<kind>] not defined`; a lane switched off is printed as
  `<kind> = false` under `[verify.<stack>]`, ahead of the stack's other
  tables. Pasted back, both stay as they were.

### `loomux check gocover --profile <path> [--floor <n>] [--dir <dir>]`
Judges a Go coverage profile of the module in `--dir` through
`go tool cover -func`. It is what the Go preset's `coverage` lane runs.

- **Flags**: `--profile` (required; relative to `--dir`), `--floor <n>` (a
  minimum total in percent instead of the per-function gate), `--dir`
  (directory holding `go.mod`; default `.`). The module path comes from
  `go.mod`.
- **Without `--floor`**: every function at 100 %, unless
  `//coverage:exempt <reason>` is the last comment line directly above its
  `func`. Each other function is one line on `stdout`:
  `not covered: <file>:<line> <func> <percent>%`. A profile without functions
  fails.
- **With `--floor`**: prints `coverage <total>%`, or fails with
  `coverage <total>% is below the floor of <n>%` on `stderr`.
- **Exit Codes**: `0` (passed), `1` (a function or the total below the gate, no
  `go.mod`, an unreadable profile), `2` (no `--profile`, an unknown flag).

### `loomux check commit-msg [flags] [<file>]`
Validates a git commit message file against language and formatting rules, or measures thresholds against the git history.

- **Arguments**:
  - `<file>`: Path to the commit message file (e.g., `.git/COMMIT_EDITMSG`). Cannot be passed together with `--calibrate`.
- **Flags**:
  - `--root <path>`: Path to project root (defaults to walking up to find `.loomux/config.toml` or `.git`).
  - `--calibrate <N>`: Measure refusal rates across the last `N` commits instead of checking a single file.
  - `--language <en|de>`: The language to calibrate against (defaults to `[commit].language` or `"en"`). Cannot be passed when checking a file.
- **Behavior**:
  - **Vocabulary & Language**: Scans all lines for foreign-language stop words (Variant B). In English mode (`en`), a word carrying an umlaut counts as a hit, and 82 German developer words (`fehler`, `datei`, `behebe`, `aktualisiere` …) are a fourth word source beside the German, English and Romance ones.
  - **Non-Latin Scripts**: Detects runs of non-Latin script characters (CJK ideographs, Hiragana, Katakana, Cyrillic, etc.) and treats each run as a foreign-language hit.
  - **Exemptions & Spans**: Skips text in multi-line backtick code blocks (``` `...` ```), single-line quotes (`"..."`; an apostrophe is not a delimiter), git comment lines (`#`), scissors lines (`# ------------------------ >8 ------------------------`) and trailing diffs, and git trailers (`Signed-off-by:`, `Co-authored-by:`, etc.) in the body. Also exempts file paths, name particles (`van`, `von`), hyphenated/underscored identifiers, and every line a `[[commit.allow]]` pattern matches, which is skipped whole.
  - **Conventional Commits**: When `[commit].conventional = true` (default), validates that the subject line conforms to `<type>[(<scope>)][!]: <description>`.
- **Exit Codes**:
  - `0`: Valid commit message, or calibration completed successfully.
  - `1`: Commit message refused (reason and offending lines on `stderr`), or configuration/git error.
  - `2`: Command line usage error (invalid flags or arguments).

### `loomux check gofmt [paths...]`
Inspects Go source files for formatting compliance without modifying them.

- **Arguments**: Optional directory or file paths (defaults to working directory).
- **Exit Codes**: `0` (Formatted), `1` (Unformatted files listed on `stdout`).

---

## 3. Agent Harness Hooks (`loomux hook`)

Hook entry points are called synchronously by coding agents on tool invocations.

```bash
loomux hook <event> --host <claude|antigravity|codex> [--root <path>]
```

### `loomux hook pre-tool-use`
Evaluates the project policy and global write barrier before an agent executes a tool call.

- **Standard Input**: JSON payload emitted by the agent harness:
  ```json
  {
    "tool_name": "Write",
    "tool_input": {
      "file_path": "C:/Projects/repo/.env"
    }
  }
  ```
- **Execution Budget**: `<35ms` cold start floor.
- **Standard Output / Error**:
  - On Refusal: JSON refusal envelope on `stdout`, human-readable reason on `stderr`.
- **Exit Codes**:
  - `0`: Permitted.
  - `2`: Refused (policy violation or write outside registered workspace).

### `loomux hook post-tool-use`
Fires after an agent has edited a file.

- **Standard Input**: Tool name and input payload; the edited path comes from `file_path`, else `notebook_path`.
- **Flags**: `--host <h>` (required), `--root <r>`, `--budget <duration>` — how long the lanes may take in all (Go duration, default `50s`, below the host's hook timeout of 60 s). Each command gets the smaller of its own `timeout` and what is left of the budget.
- **Behavior**: Runs the lanes of the `edit` profile (by default `lint` and `types`) for the edited file's stack, in the area that holds the file, as [`[verify]`](configuration.md#verify-check-chains--quality-gates) and the presets lay them out, with `on_file` where a lane has it; see [Hooks](hooks.md#5-the-post-edit-lanes-by-stack).
- **Skipped lanes**: a lane whose tool is not on the `PATH`, a Godot project not yet imported, and every lane the budget did not reach are skipped, not failed. They are named on `stdout` as `{"hookSpecificOutput":{"additionalContext":"loomux hook post-tool-use: lane skipped, the edit budget ran out: lint/go","hookEventName":"PostToolUse"}}`.
- **Exit Codes**: `0` (all lanes passed, skipped, or nothing to run), `1` (malformed call, such as a missing `--host`, or a `[verify]` that cannot be loaded), `2` (a lane failed, timed out or is blocked; its output on `stderr`).

### `loomux hook session-start`
Records the commit the session starts on.

- **Flags**: `--host <h>` (required; only `claude` has an adapter), `--root <r>`.
- **Behavior**:
  - Writes `HEAD` as `base` into `.loomux/state/hooks/<session_id>.json`.
  - Warns in `hookSpecificOutput.additionalContext` when the binary inside the project is older than its Go sources.
  - Makes no worktree junctions; that is `loomux worktree link`. See [Hooks](hooks.md#8-session-hooks).
- **Exit Codes**: `0` (Success), `1` (missing or unknown host, no adapter for the host, unreadable payload, failed write).

### `loomux hook stop`
The gate at the end of a turn: delivers what subagents left, then runs the `stop` profile over what changed since the last green run.

- **Flags**: `--host <h>` (required; only `claude` has an adapter), `--root <r>`, `--budget <duration>` — how long the lanes may take in all (Go duration, default `270s`, below the 300 s its settings entry grants). Each command gets the smaller of its own `timeout` and what is left of the budget.
- **Standard Input**: the host's `Stop` payload; only `session_id` is read.
- **Behavior**: in this order — the subagents' findings to `stderr`, the block counter (after 3 blocks in a row it gives up for one turn and leaves the findings on disk for the next), the marker `.loomux/no-verify` (it skips the chain, not the findings), the content fingerprint (nothing new since the last green run or the base: no tool starts), then the kinds of the `stop` profile (by default `lint`, `types`, `test`, `coverage`) in the check scope, plus `lint/wiki` where `lint` is asked for and there is a wiki. A green run moves `base` to `HEAD` and remembers the tree. See [Hooks](hooks.md#stop).
- **Standard Error**: delivered findings as `subagent <agent_id>: <line>`, then only the red lanes, in the format of `loomux check`.
- **Exit Codes**: `0` (the turn ends: green, nothing new, the marker, or the counter gave up), `2` (the turn is held: a red lane, a git failure, or findings delivered — with findings even an exit 1 becomes 2), `1` (the gate could not judge: an unreadable payload or one without `session_id`, the budget ran out, a requested kind had nothing that ran, `[verify]` cannot be loaded, the plan fails, the coverage directory cannot be prepared (`verify.PrepareCover`), or a malformed call; the turn ends).

### `loomux hook subagent-start`
Writes down where `origin`, the local branches and `HEAD` stand before a subagent runs.

- **Flags**: `--host <h>` (required; only `claude` has an adapter), `--root <r>`.
- **Standard Input**: the host's `SubagentStart` payload; `session_id` and `agent_id` are read.
- **Behavior**: `git ls-remote origin` (10 s deadline, `GIT_TERMINAL_PROMPT=0`), the local branches and `HEAD` go into `.loomux/state/hooks/<session_id>/agents/<agent_id>.json`; a remote that does not answer is recorded as `unavailable`. A finding still parked in that file is kept. See [Hooks](hooks.md#subagent-start-and-subagent-stop).
- **Exit Codes**: `0` (written), `1` (missing or unknown host, no `session_id` or `agent_id`, unreadable payload, failed write). Never 2.

### `loomux hook subagent-stop`
Compares against the snapshot and parks what moved for the main agent's `stop`.

- **Flags**: `--host <h>` (required; only `claude` has an adapter), `--root <r>`.
- **Standard Input**: the host's `SubagentStop` payload; `session_id` and `agent_id` are read.
- **Behavior**: one line per ref of `origin` or local branch that is new, gone or moved, and one `new commit <oneline>` per commit `HEAD` and the moved branches gained, appended to the agent's file; nothing to report removes the file. Without a file or a snapshot it is silent. It writes nothing for the model: its own output would reach the subagent, not the main agent.
- **Exit Codes**: `0` (compared, or nothing to compare), `1` (missing or unknown host, no `session_id` or `agent_id`, unreadable payload, failed write). Never 2.

---

## 4. Diagnostic Doctor (`loomux status`)

### `loomux status [--root <path>]`
Inspects the active repository setup, verification lanes, detected harnesses, and hook registration.

```bash
loomux status
```
- **Aliases**: `loomux explain`, `loomux doctor`.
- **Output Details**:
  - Project root path and declared areas.
  - Active harnesses (`.claude/`, `.agents/`, `.cursor/`).
  - Write barrier status and policy rule count.
  - The lanes the post-edit hook runs per active stack: the `edit` profile as `[verify]` and the presets lay it out, each with its origin, and which of their tools are missing from the `PATH`.
  - The `Stop` entry to wire (`loomux hook stop`, profile `stop`, the wiki bundle as `lint/wiki`), and for each of the six events `PreToolUse`, `PostToolUse`, `SessionStart`, `Stop`, `SubagentStart` and `SubagentStop` whether `.claude/settings.json` calls its `loomux hook` (`[OK]`) or not (`[INFO]`), plus legacy hooks it replaces.
- **Exit Codes**: `0` (Ready), `1` (Configuration error).

---

## 5. Worktree Mirroring (`loomux worktree`)

Puts the directories named in `[worktree] mirror` into linked git worktrees as Windows junctions, and takes them out again. Junctions exist only on Windows. The full decision path is in [Hooks](hooks.md#9-worktree-mirroring).

### `loomux worktree link [--root <path>]`
In a linked worktree, makes a junction into the main checkout for every configured path that is missing there; then, wherever it runs, sweeps our junctions out of directories under `.worktrees/` and `.claude/worktrees/` that git no longer holds.

### `loomux worktree unlink [--root <path>]`
Reads `session_id` from the payload on `stdin`, removes this session's file under `.loomux/state/hooks/`, and removes the junctions only when no other session file younger than 24 hours remains.

### `loomux worktree remove <worktree-path>`
Refuses the main checkout and any directory git holds no worktree at, removes the junctions, runs `git worktree remove --force`, checks that the directory is gone, and prints `removed <path>`.

- **Exit Codes**: `0` (in order, or nothing to do), `1` (a fault, named on `stderr`), `2` (no subcommand, or an unknown one).

---

## 6. Code Graph Engine (`loomux graph`)

> [!NOTE]
> **`build`, `check`, `ask`, `callers`, `skeleton`, `grep`, `map` and `stats` are wired; `blast` and `viz` remain specified.** Stage G1 built the packages the graph relies on — `internal/code/model`, `internal/code/pagerank` and `internal/code/blast` — stage G2a added the extractor, the wiring writer, the freshness probe and `graph build` and `check`, stage G2b added the lexicon, lexical scoring, Personalized PageRank blend and `graph ask`, stage G3 put `ask` and `check` behind the MCP tools `graph_find_code` and `graph_check_freshness` (§8), and stage G4a delivered the navigation suite (`callers`, `skeleton`, `grep`, `map`, `stats`) and their four MCP tools. Stage G4b adds git-diff blast radius analysis and the post-edit blast monitor hook.

### `loomux graph build [--root <path>]`
Reads and hashes every Go source file `internal/code/sourceset` finds under the root, extracts and resolves them into the deterministic AST graph, and writes it to `.loomux/state/graph/wiring.json`. It also writes the freshness record (`.loomux/state/graph/cache/fingerprint.json`) a later probe reads; a failure to write that record is announced on `stderr` but does not fail the build, since the graph on disk is already correct.

- **Flags**: `--root <path>` — project root; the working directory when empty.
- **Output**: one line naming files, nodes and edges by relation, then a line with unresolved import targets, files without a symbol, and the time taken. Illustrative shape, not a value to expect — every part of it, including the timing, moves with this repository's own code, and the last three commits each shipped a number here that the next commit falsified: `N files, N nodes, N edges (N contains, N calls, N imports)` / `N unresolved import targets, N files without a symbol, Nms`. Measured figures with their commands and raw output are in `docs/en/benchmarks.md`.
- **Exit codes**: `0` on success; `1` if the root cannot be resolved, a file cannot be read or parsed, module resolution fails, or the graph cannot be written; `2` for a usage error.
- **Cost**: `build` never reads the freshness record — it reads and hashes every file, every time, cold or warm alike. It is the command that produces the state a probe compares against, so a stale byte in it would be a stale answer, not a saved read. See `docs/en/benchmarks.md` for measured figures.

### `loomux graph ask "<query>" [flags]`
Retrieves code symbols ranked by BM25-style lexical matching blended with **Personalized PageRank** over the AST call graph.

- **Flags**:
  - `--root <path>` — project root; the working directory when empty.
  - `--limit <n>` — maximum number of hits to report (default `8`).
  - `--in <prefix>` — filter candidate nodes by path prefix before scoring and PageRank walk; recomputes document frequency over the remainder.
  - `--source` — inline the source code span for each hit (capped at 80 lines unless `--full`). Without `--source`, only locations and signatures are reported.
  - `--full` — when `--source` is enabled, inlines the full source code span without the 80-line cap.
  - `--json` — emit machine-readable JSON matching the `ask.Answer` struct (`hits`, `query`, `note`, `stats`).
  - `--no-refresh` — skip the freshness probe and automatic background rebuild on drift.
- **The two things this otherwise gets asked twice**:
  - Without `--source`, no code is shown — only location (path, line span), symbol id, signature, and composite score along with its lexical and graph components.
  - By default, `ask` probes the graph for freshness before answering. If the working tree has drifted, the freshness record is missing, or the ask sidecar is missing or carries another index version, it rebuilds the graph and sidecar under a cross-process lock before answering, logging rebuild progress to `stderr`. The sidecar is part of the probe because the freshness record knows about source files only: without that check, a deleted `ask-index.json` would leave every later question ranking on names and paths until some source file happened to change. To query the existing graph without rebuilding, pass `--no-refresh` — which skips the sidecar check as well, so the answer may fall back to names and paths, and says so on `stderr`.
  - Without a graph, exits 1 and points at `loomux graph build`; a query never builds a first graph.
- **Output**: Ranked list of hits in the format:
  `N. <id>  <path>:<span-or-line>  (<score> lex <lexical> graph <graph>)`
  followed by signature and, if `--source` is requested, the inlined code block prefixed with `|`. If no symbols match the query, outputs an empty answer note and exits 0.
- **Exit codes**: `0` on success (including when no symbols match the query); `1` on failure (no graph yet, unreadable graph, rebuild failure, syntax error in file when rebuilding); `2` on usage error (missing query, negative limit).

### `loomux graph callers <symbol> [--direction in|out] [-d <depth>] [--in <prefix>] [--json]`
Traces who calls, imports, or references a symbol (`--direction in`, default), or what this symbol calls (`--direction out`).

- **Flags**:
  - `--direction <in|out>`: Trace callers into this symbol (`in`) or callees out of this symbol (`out`).
  - `-d <depth>`: Transitive depth (default `1`; `-d all` or `-d full` for full transitive closure).
  - `--in <prefix>`: Filter symbols by repository path prefix before resolving.
  - Each direct hit (depth 1) carries the first line in the caller's span that names the callee. For `--direction out` that line lies in the start symbol's file and is printed with its path.
  - `--json`: Output machine-readable JSON (`query.CallersAnswer`).
- **Exit codes**: `0` on success; `1` if graph is missing, unreadable, or symbol not found; `2` on usage error.

### `loomux graph skeleton <file> [--json]`
Exports definition signatures, types, and line spans for a file from the graph without function bodies (~10x token reduction).

- **Flags**:
  - `--json`: Output machine-readable JSON (`skeleton.FileSkeleton`).
- **Exit codes**: `0` on success; `1` if graph is missing or file not found in graph; `2` on usage error.

### `loomux graph grep <pattern> [-i] [--fixed] [--in <prefix>] [--max-hits <n>] [--json]`
Regex search across indexed files, grouped by enclosing symbol and ranked by incoming edge degree (`inDegree`).

- **Flags**:
  - `-i`: Case-insensitive regex matching.
  - `--fixed`: Treat pattern as a literal string (no regex syntax).
  - `--in <prefix>`: Narrow search to files under path prefix.
  - `--max-hits <n>`: Maximum number of matched lines to return (default `300`); further matches are only counted.
  - `--json`: Output machine-readable JSON (`grep.Result`).
- **Exit codes**: `0` on success (even with 0 hits); `1` on missing/unreadable graph or invalid regex; `2` on usage error.

### `loomux graph map [--max-dirs <n>] [--hubs-per-dir <n>] [--hotspots <n>] [--json]`
Displays token-budgeted directory clusters, local hubs, and global codebase hotspots ranked by in-degree coupling.

- **Flags**:
  - `--max-dirs <n>`: Maximum number of directory clusters to display (default `16`).
  - `--hubs-per-dir <n>`: Maximum hubs listed per directory (default `3`).
  - `--hotspots <n>`: Maximum repository-wide hotspots (default `12`).
  - `--json`: Output machine-readable JSON (`repomap.RepoMap`).
- **Exit codes**: `0` on success; `1` on missing or unreadable graph; `2` on usage error.

### `loomux graph stats [--json]`
Prints structural codebase metrics from `.loomux/state/graph/wiring.json`: total node count, edge count grouped by relation, indexed file count, language distribution, and file size.

- **Flags**:
  - `--json`: Output machine-readable JSON (`query.StatsAnswer`).
- **Exit codes**: `0` on success; `1` on missing or unreadable graph; `2` on usage error.

### `loomux graph check [--root <path>] [--json]`
Re-extracts the whole tree and diffs it, node by node, against the graph written on disk.

- **Flags**: `--root <path>` — project root; the working directory when empty. `--json` — write the drift as JSON (`checkResult`: `ok`, `missing`, `foreign`, `added`, `removed`, `changed`) instead of the human report.
- **The one thing this otherwise gets asked twice**: `check` does not read the freshness record. That sidecar answers "should a query bother rebuilding"; `check` answers "does the graph still describe the code", and the only honest way to answer that is to extract again and compare body hashes. A `touch` that changes a file's mtime but not its bytes is therefore not a finding here, same as it is not one for the probe — but for a different reason: the probe never gets past its stat comparison, `check` gets all the way to a hash and finds it unchanged.
- **Output**: `NO GRAPH` when nothing has been built yet; `FOREIGN GRAPH` when the graph on disk names an extractor version other than this binary's; `OK` when nothing has drifted; otherwise `DRIFT` with one line per added, removed or changed node id.
- **Exit codes**: `0` — fresh (`OK`); `1` — no graph yet, a foreign graph, drift found, or a fault while re-extracting; `2` — usage error.

### `loomux graph blast [dir]`
Computes the downstream blast radius of a git diff against the working tree or merge base.

- **Flags**:
  - `--base <ref>`: Diff against git reference (e.g., `origin/main`).
  - `--format markdown`: Formats output as a GitHub PR comment.
  - `--export-viz <dir>`: Exports a standalone interactive HTML blast visualization.

### `loomux graph viz [dir]`
Starts the local D3-Force / WebGL interactive graph visualizer.
- **Flags**: `--port <p>`, `--no-open`.

---

## 7. Second Brain & Wiki (`loomux brain`)

The five data commands read the areas of the one registry (`registry.toml` in `LOOMUX_STATE_DIR` or its platform default) and answer as ultra-brain's `brain-mcp` does; a recorded case corpus (`testdata/cases/1b-1`) holds them to it. A read-only area's artefacts (`index.md`, `graph.json`, `_identities.tsv`) and the reconcile stamp are read from loomux's state directory first, where stage 3a writes them, and from ultra-brain's state directory as long as nothing lies in the new place: `LOOMUX_LEGACY_BRAIN_DIR`, defaulting to `%LOCALAPPDATA%\brain` on Windows and to `$XDG_STATE_HOME/brain` or `~/.local/state/brain` on POSIX. The whole area directory decides, never a single file; `loomux migrate` (stage 4) moves the rest. Until stage 4, an area directory whose `.loomux/config.toml` is missing or has no `[area]` table is read through `.ultra-brain/config.toml` or `.brain.toml`.

- **Channel**: every command takes `--channel local|cloud` (default `local`). An area with `[privacy] mode = "local_only"` does not exist on `cloud`; `[privacy] never` globs apply on every channel.
- **Usage errors** (exit `2`): the usage line, then `loomux brain <command>: error: <reason>` for a missing argument, an invalid choice or `-n` below 1, and `loomux brain: error: <reason>` when the command is missing or unknown or arguments are left over.
- **Runtime errors** (exit `1`): `error: <reason>` on `stderr` and nothing on `stdout` — an unknown scope, a refusal, a missing section, a broken `graph.json` or identity register, a missing or unreadable manifest of any registered area, a missing or broken registry, a search engine that cannot be reached.
- **Advice**: the messages still name `brain reindex`, `brain reconcile` and `brain embed`, the commands of ultra-brain, because the recorded cases of 1b-1 and 1b-2 hold that wording; they move to `loomux reindex`, `loomux reconcile` and `loomux embed` with the switch-over.

### `loomux brain search <query> [--scope <scope>] [--profile fast|full|keyword] [-n <n>] [--channel local|cloud]`
Searches the visible areas (`--scope all` by default) through the qmd MCP daemon at `http://localhost:8765/mcp`.

- **Profiles**: `fast` (default) vector search without reranking or query expansion; `keyword` BM25 keyword search; `full` the hybrid chain with expansion and reranking. `-n` (default `5`) must be at least 1.
- **Daemon**: when nothing answers there, loomux starts `qmd mcp --http --daemon --port 8765` detached, writes `note: starting the search engine; the first call after a start pays a model load (measured 5.7 s). Later calls are warm.` on `stderr` and waits up to 60 s for it. The backbone defaults to CUDA; a `QMD_LLAMA_GPU` or `QMD_FORCE_CPU` the user set stays untouched.
- **Output**: per hit `brain://<scope>/<path>:<line>  <score>%  <title>`, each snippet line indented by four spaces, then an empty line; exactly `no matches` without hits.
- **Notes**: after the hits, `note: <finding>` lines on `stderr`, in this order — the engine answered empty twice, a hit is missing from its area's identity register, how many hits were withheld, the reconcile stamp is 24 hours old or older.
- **Exit codes**: `0` with hits or `no matches`; `1` for a runtime error, including an engine that does not answer — never an empty answer instead; `2` for a usage error.

### `loomux brain catalog [--scope <scope>] [--channel local|cloud]`
Prints the catalog of the visible areas, or of one area.

- **Output**: with `--scope all` (default) `# brain`, an empty line and `* [<scope>](brain://<scope>/)` per visible area, sorted by scope; with a named scope, that area's `index.md`, strict UTF-8 with line ends folded to `\n`.
- **Exit codes**: `0`; `1` for an unknown scope, a missing `index.md` or another runtime error; `2` for a usage error.

### `loomux brain read <path> --scope <scope> [--section <title>] [--channel local|cloud]`
Prints one file of an area, or one section of it.

- **Reading**: strict UTF-8 with line ends folded to `\n`. `--section` prints from the heading with that title to the next heading of the same or a higher level.
- **Refusals** (exit `1`): `<scope>/<path> leaves the area`; `<scope>/<path> is excluded by [privacy] never`; `<scope>/<path> is the review centre; refused on the cloud channel`; `no section titled '<title>'`.
- **Exit codes**: `0`; `1` for a refusal or another runtime error; `2` for a usage error.

### `loomux brain neighbors <path> --scope <scope> [--channel local|cloud]`
Prints the links into and out of one page, read from the area's `graph.json`.

- **Output**: `incoming: <a>, <b>` and `outgoing: <c>`, each list sorted, `-` for a direction without links.
- **Exit codes**: `0`; `1` for a refusal, an area without `graph.json` (``<scope>: never indexed; run `brain reindex` ``) or another runtime error; `2` for a usage error.

### `loomux brain status [--channel local|cloud]`
Prints what to know before trusting an answer, one line per finding.

- **Lines, in this order**: always the last reconciliation (``last reconcile: never; run `brain reconcile` ``, ``last reconcile: <iso>; older than 24 h, run `brain reconcile` `` or `last reconcile: <iso>`); per visible area in registry order, include globs the search engine does not see, a path that does not exist, an area never indexed, fewer than half of its links resolved, and indexed documents the search engine does not know; across all visible areas, the same content under several paths; once, documents indexed but not yet searchable.
- **Search engine**: two lines ask the qmd CLI (`qmd ls <collection>`, `qmd status`); when it does not answer, the line says so and the command goes on.
- **Exit codes**: `0`; `1` for a runtime error (registry, manifest, stamp, `graph.json`, identity register); `2` for a usage error.

### `loomux brain lint`
Validates wikilinks (`[[Page]]`), orphaned documents, broken cross-references, and frontmatter taxonomy.

### Upkeep: `loomux reindex`, `loomux embed`, `loomux reconcile`, `loomux area add`

Four commands of ultra-brain's `brain` CLI, top-level commands of loomux since stage 3a; a recorded case corpus (`testdata/cases/3a`) holds them to the Python reference. They write only to loomux's state directory and read the legacy one as the fallback described above.

- **Environment**: `LOOMUX_STATE_DIR` holds the registry, the artefacts of read-only areas, `maintenance/` and `qmd-collections.json`; `LOOMUX_LEGACY_BRAIN_DIR` is the fallback and is never written. qmd's `index.yml` is found through `XDG_CONFIG_HOME`, else `~/.config`.
- **No `--state-dir`**: the reference accepts it on all four; loomux refuses it like any unknown flag (exit `2`). The state comes from the environment, the one state model of every loomux command.
- **Positional arguments** (exit `2`): none of the four takes one. A word left after the flags is refused with `<command>: unrecognized arguments: <words>` before the environment or qmd is looked at.
- **Messages** of the reconcile pass are German, word for word the reference's.

#### `loomux reindex [--registry <path>]`
Runs a reconcile pass over the registered areas, then rebuilds each area's directory catalogs (`index.md`), link graph (`graph.json`) and identity register (`_identities.tsv`) and enters the areas as collections into qmd's `index.yml`. A writable area keeps its artefacts in its own tree; a read-only area's are written to `<state>/areas/<scope>/` through a staging directory and swapped in whole.

- **`--registry`**: a `registry.toml`, or the directory holding one; default `registry.toml` in the state directory.
- **Catch-up**: the reconcile pass runs first, so that a changed source becomes a case before the index run advances its hash. Cases it opens are listed on `stderr` and the run **goes on**; a vault without a review centre gets a warning and is indexed; any other failure of the pass stops the command before anything is indexed.
- **Output**: `indexed the areas of <path>` on `stdout`; on `stderr` the collections updated or dropped, and each collection refused because qmd already holds one of that name that brain did not create.
- **Exit codes**: `0` on success, and also when there is no registry in the state directory (`no areas registered in <path>; nothing to index` on `stdout`); `1` for a registry named with `--registry` that does not exist, a registry that does not read, a failed catch-up, a failed index run or a refused collection; `2` for a usage error.

#### `loomux embed [--registry <path>]`
Asks qmd to generate the vectors the index run leaves pending, for every registered area.

- **qmd first**: without `qmd` on `PATH` it prints `loomux embed: qmd is not on PATH; install it with: npm install -g @tobilu/qmd` and exits `1` before it reads the registry.
- **Output**: `embedded <n> area(s)` on `stderr`.
- **Exit codes**: `0` on success, and also when there is no registry in the state directory (`no areas registered in <path>; nothing to embed` on `stdout`); `1` for missing qmd, a named registry that does not exist, a registry that does not read, or an engine that refuses; `2` for a usage error.

#### `loomux reconcile`
Measures every source of the registered areas against its identity register and, for each wiki page derived from a changed source, opens a case (`case.toml` and a package with the diff) in the review centre, the one directory an area declares under `[layout] review`. A merge recorded in `maintenance/merge-events.tsv` opens a case with the merge's evidence, after the source cases.

- **Output** on `stdout`: `<n> Quellen geprüft, <m> davon gehasht`, one line per case (directory, area, target, state and `manuell` for a case that asks for a manual decision, tab-separated, indented by two spaces), then `<k> Fälle`.
- **A case is not a failure**: open cases leave the exit code at `0`.
- **Stamp**: the pass writes `maintenance/last-run.txt` in UTC, which `brain status` and `brain search` read.
- **Exit codes**: `0` for a pass that ran to the end; `1` when a case file cannot be read (`unreadable case: <entry>` on `stderr`), and for a registry that does not read, a vault that declares no review centre or two, or another failure (`error: <reason>`); `2` for a usage error.

#### `loomux area add [--path P] [--scope S] [--wiki W] [--sources S] [--merge-branch B] [--privacy M] [--no-reindex] [-y|--yes]`
Registers a repository as an area and prepares it: the registry entry (written under a lock); `.loomux/config.toml` with `[area]`, `[layout]`, `[index]`, `[privacy]` and `[maintenance]` when the repository has none; the routing rule, appended once to `AGENTS.md`; the wiki bundle frame; then `loomux reindex` with its catch-up.

- **Defaults**: `--path` the working directory; `--scope` `project/<directory name>`; `--sources` `docs` when there is a `docs/` directory, else `.`; `--wiki` `<repo>/docs/wiki` or `<repo>/wiki` (must be absolute); `--merge-branch` the branch git names, `master` without one; `--privacy` `manual_cloud`, one of `automatic_cloud`, `local_only`, `manual_cloud`.
- **Registry first**: a scope already registered is refused before the repository is touched.
- **A kept configuration**: an existing `.loomux/config.toml` is kept byte for byte, with a warning when it declares no `[area]` or another scope. One the declaration reader refuses ends the command with nothing registered.
- **Differences from `brain init`**: no `.mcp.json` and no agent hooks (`loomux init`, stage 4); the index run really happens unless `--no-reindex` is given; the branch is written as `[maintenance] branch`, not `merge_branch`; `--privacy` is checked; the first area of a machine needs no registry file prepared by hand. `-y`/`--yes` is accepted and changes nothing.
- **Exit codes**: `0`, or the exit code of the index run; `1` for a path that is not a directory, an invalid scope, a relative `--wiki`, a refused registry entry, an unreadable file or a failed write; `2` for a usage error, a missing or unknown subcommand (with the usage line) or an unknown `--privacy`.

---

## 8. MCP Service & stdio Bridge (`loomux serve` / `loomux mcp`)

The service answers eleven tools over Streamable HTTP — the five `brain_*` tools
and the six `graph_*` tools of Stages G3 and G4a; the bridge is what an MCP host starts
and all it does is pass calls on. Both were built in Stage 1b-2. The Web OS of
Stage W1 is not here yet, and neither are git-diff blast radius analysis (Stage G4b)
or the upstream proxies.

**The channel is the address, not a field of the request.** `serve` binds two
loopback listeners, one for `local` and one for `cloud`, each with its own
random token. A caller holding the cloud token cannot reach the local view at
all. Everything the service writes — `serve.json`, `serve.lock` and
`logs/serve.log` — lives under the state directory (`LOOMUX_STATE_DIR`, by
default `%LOCALAPPDATA%\loomux`), never under a hard-wired path.

### `loomux serve [--foreground]`
Starts the long-lived localhost MCP service. Without `--foreground` the start is
detached: the service is spawned beside the caller, asked to break away from the
caller's job object, and the command returns at once naming the log file. With
`--foreground` the service runs in this process, which is the only way a human
sees a failed start — and it is also what the detached child runs.

- **One instance**: `serve.lock` plus a liveness check. A second `loomux serve`
  says so instead of starting a second service.
- **Breakaway**: if the request to leave the host's job object is refused, the
  start is retried without it and the command says `note: the breakaway was
  refused, so this service dies with its host`.
- **Exit codes**: `0` started; `1` already running, or the spawn or the run
  failed; `2` an unrecognized argument or an unknown subcommand.

### `loomux serve status`
Asks the local listener whether it answers — a state file is a hint, a listener
that answers is the truth — and then prints what `serve.json` says: PID, both
endpoint URLs, the executable, its size and build time, whether the lock is held
and whether the breakaway held. The tokens stay out of the report. A service
that is not there is a state and not a failure: the report says so and the exit
code is `0`.

- **Exit codes**: `0` reported; `1` the state could not be read; `2` an
  unrecognized argument.

### `loomux serve stop [--force]`
Ends the service through its own endpoint. `--force` kills it by the PID in
`serve.json` when the endpoint no longer answers. Success is silent.

- **Exit codes**: `0` stopped, and also when nothing was running; `1` the stop
  failed; `2` an unrecognized argument.

### `loomux mcp [--channel local|cloud]`
The stdio bridge an MCP host starts. It offers the eleven tools itself — the
descriptions are static, so a cold service never sits inside the host's
handshake — and forwards every `tools/call` to the service over the channel's
address, name to name and arguments to arguments.

- **Default channel**: `local`, exactly as `loomux brain` falls back. The narrow
  channel is the only safe default.
- **It starts the service itself**: in the background, beside the handshake. A
  recorded build older than the bridge's own is stopped and replaced (newer
  wins); an equal or newer one whose lock nobody holds is started again.
- **One retry**: a call whose session died — a commit rebuilt the binary and
  another bridge replaced the service — is renegotiated once. Two retries would
  hide a real outage.
- **Nothing is ever written to stdout**: stdout is the host's MCP pipe. Refusals
  and failures go to stderr.
- **Exit codes**: `0` the host hung up, or Ctrl+C; `1` the bridge failed; `2` an
  unrecognized argument or an invalid `--channel`.

### The eleven tools

| Tool | Arguments |
|---|---|
| `brain_search` | `query` (required), `scope` → `all`, `profile` ∈ {`fast`, `full`, `keyword`} → `fast`, `n` → 10 |
| `brain_catalog` | `scope` → `all` |
| `brain_read` | `scope` and `relative` (both required), `section` |
| `brain_neighbors` | `scope` and `relative` (both required) |
| `brain_status` | none |
| `graph_find_code` | `scope` and `query` (both required), `limit` → 5, `full`, `in` |
| `graph_check_freshness` | `scope` (required) |
| `graph_file_api` | `scope` and `file_path` (both required) |
| `graph_trace_calls` | `scope` and `symbol` (both required), `direction` ∈ {`in`, `out`} → `in`, `depth` → 1 |
| `graph_find_all` | `scope` and `pattern` (both required), `ignore_case`, `fixed` |
| `graph_repo_map` | `scope` (required), `max_dirs` → 16 |

`n` is 10 here and 5 on the command line; that is parity with the Python
reference, which does the same, not an inconsistency. `limit` is 5 here and 8
for `loomux graph ask`, both Graft's values.

The six `graph_*` tools operate over the repository of one registered area:

- **`graph_find_code`** always inlines the source at each hit; `full` takes
  the whole span instead of the capped excerpt, and `in` narrows to a path
  prefix before scoring. A refresh note stands before the answer. Without a
  graph the call is an error pointing at `loomux graph build` — a query never
  builds a first graph.
- **`graph_check_freshness`** never refreshes, so it reports on the graph as it
  was found. Drift and a missing graph are text, not errors. `isError` marks
  a refused call — a missing, unknown or hidden scope, or for
  `graph_find_code` a missing query — and a real read failure.
- **`graph_file_api`** exports definitions and types for `file_path` from the
  cached AST graph without function bodies.
- **`graph_trace_calls`** traces incoming callers (`direction: in`) or outgoing
  callees (`direction: out`) of `symbol`, either direct (`depth: 1`) or
  transitive (`depth: "all"`).
- **`graph_find_all`** performs symbol-coupled regex search across indexed files,
  grouped by enclosing symbol and ranked by incoming edge degree (`inDegree`).
- **`graph_repo_map`** generates a token-budgeted structural overview of directory
  clusters, hubs, and hotspots.

**Visibility:** an area whose manifest sets `[privacy] mode = "local_only"` does not exist on
the cloud channel (`unknown scope`, as for `brain_*`), and on both channels
paths under the manifest's `[privacy] never` globs are dropped before scoring,
from file listings, and from the drift report — which names how many it left out on the local
channel and not on the cloud one. On the cloud channel:
- `graph_file_api` on a file under `never` globs reports not found (`isError`).
- `graph_trace_calls` suppresses callers and callees located in files under `never` globs, incrementing `hidden`.
- `graph_find_all` uses an injected reader that refuses files under `never` globs (`os.ErrPermission`), reporting them in `unreadable_files` without returning any matching lines.
- `graph_repo_map` strips directories and nodes under `never` globs before constructing the map.
- No refresh note goes out either, neither before the answer nor as progress: a note counts files
under the `never` globs too. Every read or query error becomes the fixed text
"the graph could not be read on this channel; ask on the local channel for
details" there, because an error message can name a hidden file or a local
path. A missing graph is the exception and keeps its own text. An internal
error reads "internal error; ask on the local channel for details" there,
without the value it carried.

### The `.mcp.json` of a host

Until `loomux init` writes it (Stage 4), a human does:

```json
{ "mcpServers": { "loomux": { "command": "loomux", "args": ["mcp", "--channel", "local"] } } }
```

---

## 9. Developer Quality Gates (`loomux dev`)

### `loomux dev mutants <package>... [--only <name>] [--family a1|a2|a3|a4] [--workers <n>]`
Mutates the Go decisions of each package and reports which mutants its test suite does not notice — a port of ultra-brain's `tools/go_mutants.py`.

- **Families**: `a1` the whole `if` condition as `true` and as `false`; `a2` each operand of a top-level `&&` or `||` on its own; `a3` every comparison operator flipped (`==`/`!=`, each ordering against its neighbour), not inside comments or strings; `a4` the condition negated. `for` conditions are never mutated.
- **Mechanism**: each mutant reaches `go test -overlay <json> -count=1 -failfast -timeout 60s ./<package>/` through an overlay in a temporary directory; the working tree is never written. Each overlay directory is removed after its run, also after an error or Ctrl+C. `--workers` runs that many at once (default: half the processors, at least 1). `--only` keeps files whose name contains the text.
- **Report**: one line per mutant — `killed`, `SURVIVED` or `no mutant` (does not compile, or changes nothing) — then the sums and the survivors. A run that hits the time limit counts as killed.
- **Exit codes**: `0` after a complete round, survivors included; `2` for a usage error, a package without source files, or a suite that is not green before the first mutant; `1` when a run cannot be started or the round is interrupted with Ctrl+C.

### `loomux dev swap-binary --dir <bin>`
Atomically replaces the running `loomux.exe` binary with `loomux.new.exe` (solving Windows file-locking constraints). The one it replaces is kept as `loomux.old.exe`, or as the first free `loomux.old.<n>.exe` beside it when a process started from an earlier swap -- a `loomux serve` or a bridge -- still holds that name; every slot whose process has ended is removed on the next swap, so at most 16 generations are kept. Two cases still fail, and both leave the binaries where they were: all 16 slots held at once, and a `loomux.exe` that something holds so that it cannot be renamed at all -- a running `loomux.exe` is not that holder, since Windows keeps a running image renamable.

### `loomux dev bench [--dir <dir>] [--corpus <file>] [--languages <n>] [--tier <tier>] [--warm <n>] [--cache-dir <dir>] [--out <file>] [--json-out <file>] [--component-timeout <d>] [--save] [--report-dir <dir>]`
Runs comprehensive latency benchmarks and normalized gap audits on a single repository or against the open-source matrix corpus (1 cold + N warm runs, median/min/max).

- **How hooks are measured**: each hook receives a Claude Code payload for an edit of a sample file in the repository's primary language, so `post-tool-use` runs its real lanes. The Status column lists the exit codes seen across all runs.
- **Single repository mode** (default): measures `pre-tool-use`, `post-tool-use`, and `graph build` (applicable on Go projects), compares against baseline Claude hooks if defined, and audits test/linter coverage gaps against native configuration.
- **Corpus mode** (`--corpus <path>`): clones and benchmarks top-N open-source projects across cataloged languages and frameworks, reporting aggregate matrix latency and tool gaps.
- **Flags**:
  - `--dir <path>`: Target project directory (default: `.`).
  - `--corpus <path>`: Path to open-source matrix Markdown document.
  - `--languages <n>`: Number of languages from corpus to benchmark (default: `5`).
  - `--tier <tier>`: Star category tier filter (default: `"Sehr viel"`).
  - `--warm <n>`: Number of warm measurement runs for median calculation (default: `3`).
  - `--component-timeout <d>`: Deadline for each measured command; a command past it is killed and reported as `timeout` (default: `60s`).
  - `--cache-dir <dir>`: Directory for cached cloned repositories (default: `.cache/benchcorpus`).
  - `--out <path>`: Write Markdown report to file (default: stdout).
  - `--json-out <path>`: Write detailed machine-readable JSON report to file.
  - `--save`: Automatically save benchmark reports into language subdirectories (`docs/{en,de}/benchmarks/<language>/<repo_slug>.md`) and update the central matrix (`docs/{en,de}/benchmarks/matrix.md`).
  - `--report-dir <dir>`: Documentation root directory for saved reports (default: `docs`).


