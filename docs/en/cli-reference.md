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

---

## 2. Policy & Verification (`loomux check`)

### `loomux check commit-msg <file>`
Validates a git commit message file against language and formatting rules.

- **Arguments**: `<file>` — Path to `COMMIT_EDITMSG`.
- **Behavior**:
  - Enforces English language for commit title and body.
  - Rejects conversational preamble (e.g., *"Sure, I'll commit that..."*).
  - Validates subject line length.
- **Exit Codes**: `0` (Valid), `1` (Malformed commit message with reason on `stderr`).

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
- **Behavior**: Runs the lanes of the edited file's stack in parallel; see [Hooks](hooks.md#5-the-post-edit-lanes-by-stack).
- **Exit Codes**: `0` (all lanes passed, or nothing to run), `1` (malformed call, such as a missing `--host`), `2` (a lane failed; its output on `stderr`).

### `loomux hook session-start`
Records the commit the session starts on.

- **Flags**: `--host <h>` (required; only `claude` has an adapter), `--root <r>`.
- **Behavior**:
  - Writes `HEAD` as `base` into `.loomux/state/hooks/<session_id>.json`.
  - Warns in `hookSpecificOutput.additionalContext` when the binary inside the project is older than its Go sources.
  - Makes no worktree junctions; that is `loomux worktree link`. See [Hooks](hooks.md#8-session-hooks-what-runs-today-what-comes-with-stage-2).
- **Exit Codes**: `0` (Success), `1` (missing or unknown host, no adapter for the host, unreadable payload, failed write).

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
  - Configured `[verify]` lanes.
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
> **`build`, `check` and `ask` are wired; the rest below is still specified.** Stage G1 built the packages the graph relies on — `internal/code/model`, `internal/code/pagerank` and `internal/code/blast` — stage G2a added the extractor, the wiring writer, the freshness probe and `graph build` and `check`, and stage G2b adds the lexicon, lexical scoring, Personalized PageRank blend and `graph ask`. `callers`, `blast`, `skeleton`, `map` and `viz` remain unwired.

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
  - By default, `ask` probes the graph for freshness before answering. If the working tree has drifted, the graph has not been built yet, or the ask sidecar is missing or carries another index version, it rebuilds the graph and sidecar under a cross-process lock before answering, logging rebuild progress to `stderr`. The sidecar is part of the probe because the freshness record knows about source files only: without that check, a deleted `ask-index.json` would leave every later question ranking on names and paths until some source file happened to change. To query the existing graph without rebuilding, pass `--no-refresh` — which skips the sidecar check as well, so the answer may fall back to names and paths, and says so on `stderr`.
- **Output**: Ranked list of hits in the format:
  `N. <id>  <path>:<span-or-line>  (<score> lex <lexical> graph <graph>)`
  followed by signature and, if `--source` is requested, the inlined code block prefixed with `|`. If no symbols match the query, outputs an empty answer note and exits 0.
- **Exit codes**: `0` on success (including when no symbols match the query); `1` on failure (unreadable graph, rebuild failure, syntax error in file when rebuilding); `2` on usage error (missing query, negative limit).

### `loomux graph callers <symbol> [dir]`
Traces who calls, imports, implements, or extends a symbol.

- **Flags**:
  - `--direction out`: Reverse trace — what this symbol calls.
  - `-d <depth>`: Transitive depth (`-d all` for full closure).

### `loomux graph blast [dir]`
Computes the downstream blast radius of a git diff against the working tree or merge base.

- **Flags**:
  - `--base <ref>`: Diff against git reference (e.g., `origin/main`).
  - `--format markdown`: Formats output as a GitHub PR comment.
  - `--export-viz <dir>`: Exports a standalone interactive HTML blast visualization.

### `loomux graph skeleton <file>`
Exports all function, type, interface, and method signatures without function bodies (~10x token reduction).

### `loomux graph map [dir]`
Displays token-budgeted directory clusters, local hubs, and global codebase hotspots ranked by in-degree coupling.

### `loomux graph check [--root <path>] [--json]`
Re-extracts the whole tree and diffs it, node by node, against the graph written on disk.

- **Flags**: `--root <path>` — project root; the working directory when empty. `--json` — write the drift as JSON (`checkResult`: `ok`, `missing`, `foreign`, `added`, `removed`, `changed`) instead of the human report.
- **The one thing this otherwise gets asked twice**: `check` does not read the freshness record. That sidecar answers "should a query bother rebuilding"; `check` answers "does the graph still describe the code", and the only honest way to answer that is to extract again and compare body hashes. A `touch` that changes a file's mtime but not its bytes is therefore not a finding here, same as it is not one for the probe — but for a different reason: the probe never gets past its stat comparison, `check` gets all the way to a hash and finds it unchanged.
- **Output**: `NO GRAPH` when nothing has been built yet; `FOREIGN GRAPH` when the graph on disk names an extractor version other than this binary's; `OK` when nothing has drifted; otherwise `DRIFT` with one line per added, removed or changed node id.
- **Exit codes**: `0` — fresh (`OK`); `1` — no graph yet, a foreign graph, drift found, or a fault while re-extracting; `2` — usage error.

### `loomux graph viz [dir]`
Starts the local D3-Force / WebGL interactive graph visualizer.
- **Flags**: `--port <p>`, `--no-open`.

---

## 7. Second Brain & Wiki (`loomux brain`)

The five data commands read the areas of the one registry (`registry.toml` in `LOOMUX_STATE_DIR` or its platform default) and answer as ultra-brain's `brain-mcp` does; a recorded case corpus (`testdata/cases/1b-1`) holds them to it. Until stage 3, a read-only area keeps its artefacts (`index.md`, `graph.json`, `_identities.tsv`) and the reconcile stamp in ultra-brain's state directory: `LOOMUX_LEGACY_BRAIN_DIR`, defaulting to `%LOCALAPPDATA%\brain` on Windows and to `$XDG_STATE_HOME/brain` or `~/.local/state/brain` on POSIX. Until stage 4, an area directory whose `.loomux/config.toml` is missing or has no `[area]` table is read through `.ultra-brain/config.toml` or `.brain.toml`.

- **Channel**: every command takes `--channel local|cloud` (default `local`). An area with `[privacy] mode = "local_only"` does not exist on `cloud`; `[privacy] never` globs apply on every channel.
- **Usage errors** (exit `2`): the usage line, then `loomux brain <command>: error: <reason>` for a missing argument, an invalid choice or `-n` below 1, and `loomux brain: error: <reason>` when the command is missing or unknown or arguments are left over.
- **Runtime errors** (exit `1`): `error: <reason>` on `stderr` and nothing on `stdout` — an unknown scope, a refusal, a missing section, a broken `graph.json` or identity register, a missing or unreadable manifest of any registered area, a missing or broken registry, a search engine that cannot be reached.
- **Advice**: the messages name `brain reindex`, `brain reconcile` and `brain embed`, the commands of ultra-brain, until stage 3 rewrites them.

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

### `loomux brain reconcile`
Synchronizes state changes, identity registers, and vector index collections.

---

## 8. MCP Service & stdio Bridge (`loomux serve` / `loomux mcp`)

The service answers the five `brain_*` tools over Streamable HTTP; the bridge is
what an MCP host starts and all it does is pass calls on. Both were built in
Stage 1b-2. The Web OS of Stage W1 is not here yet, and neither are the `graph_*`
tools of Stages G2–G5.

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
The stdio bridge an MCP host starts. It offers the five tools itself — the
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

### The five tools

| Tool | Arguments |
|---|---|
| `brain_search` | `query` (required), `scope` → `all`, `profile` ∈ {`fast`, `full`, `keyword`} → `fast`, `n` → 10 |
| `brain_catalog` | `scope` → `all` |
| `brain_read` | `scope` and `relative` (both required), `section` |
| `brain_neighbors` | `scope` and `relative` (both required) |
| `brain_status` | none |

`n` is 10 here and 5 on the command line; that is parity with the Python
reference, which does the same, not an inconsistency.

### The `.mcp.json` of a host

Until `loomux init` writes it (Stage 4), a human does:

```json
{ "mcpServers": { "loomux": { "command": "loomux", "args": ["mcp", "--channel", "local"] } } }
```

---

## 9. Developer Quality Gates (`loomux dev`)

### `loomux dev covergate --profile <coverage.out>`
Enforces strict 100% test coverage per function.

- **Policy**: Every non-exempt function below 100.0% fails the gate.
- **Exemptions**: Permitted only with `//coverage:exempt <reason>` directly preceding the `func` declaration.

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


