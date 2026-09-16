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
Fires immediately after an agent completes a file edit or shell command.

- **Standard Input**: Tool name and input payload.
- **Behavior**:
  - Computes dirty file hash.
  - Calculates the immediate blast radius of modified symbols.
  - Emits inline warnings if critical callers were touched.
- **Exit Codes**: Always `0` (never blocks completion).

### `loomux hook session-start`
Announces session initialization and synchronizes the harness environment.

- **Flags**: `--host <h>` (required), `--root <r>`.
- **Behavior**:
  - Validates repository cleanliness and working tree freshness.
  - Sets up isolated worktree junction mirrors if in a subagent session.
- **Exit Codes**: `0` (Success), `1` (Missing host or invalid flag).

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

Manages fast subagent git worktrees with junction-mirrored dependencies.

### `loomux worktree link [--root <path>]`
Mirrors configured directories (`node_modules`, `.cache`) from the main checkout into the active worktree via NTFS junctions (Windows) or symlinks (POSIX).

### `loomux worktree unlink [--root <path>]`
Safely removes junction mirrors when a subagent session completes without touching real assets in the main repository.

### `loomux worktree remove <worktree-path>`
Atomically sweeps and removes an isolated worktree directory.

- **Exit Codes**: `0` (Clean), `1` (Target cannot be inspected or git refused).

---

## 6. Code Graph Engine (`loomux graph`)

### `loomux graph build [dir]`
Parses source files into the deterministic AST code graph and writes `.loomux/state/graph/wiring.json`.

- **Flags**:
  - `--deep`: Enrich symbols with LLM crux summaries (cached).
  - `--extensions <exts>`: Restrict parsed extensions (e.g., `.go .ts`).

### `loomux graph ask "<query>" [dir]`
Retrieves code symbols ranked by **Personalized PageRank** over the AST call graph.

- **Output**: Ranked symbols with file, lines, and inlined crux spans ($0 token read cost).
- **Flags**: `--json` (machine-readable output).

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

### `loomux graph check [dir]`
Checks whether the code graph has drifted from the live working tree.
- **Exit Codes**: `0` (Fresh), `1` (Stale / Drift detected).

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

## 8. Services & MCP Gateway (`loomux serve` / `loomux mcp`)

### `loomux serve [--port <port>]`
Launches the long-running localhost HTTP service hosting the Web OS and root MCP gateway.

- **Token Bootstrap**: Emits an authenticated one-time URL (`http://127.0.0.1:<port>/?token=<hex>`). The first hit sets a secure session cookie.
- **SSE Stream**: Streams real-time audit logs from `.loomux/state/journal/events.jsonl` on `/api/events`.

### `loomux mcp`
Stdio transport bridge connecting Claude Code, Cursor, and Antigravity directly to Loomux tools:
- `graph_find_code`
- `graph_file_api`
- `graph_trace_calls`
- `graph_find_all`
- `graph_repo_map`
- `graph_check_freshness`
- `brain_search`
- `brain_catalog`

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

### `loomux dev swap --dir <bin>`
Atomically replaces the running `loomux.exe` binary with `loomux.new.exe` (solving Windows file-locking constraints).
