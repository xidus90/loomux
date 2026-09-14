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

### `loomux brain search "<query>"`
Performs hybrid semantic and keyword search across wiki pages, ADRs, and concepts.

### `loomux brain catalog`
Lists all managed wiki areas, topic hierarchies, and identity documents.

### `loomux brain read <path>`
Reads a typed wiki page, concept definition, or ADR.

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

### `loomux dev swap --dir <bin>`
Atomically replaces the running `loomux.exe` binary with `loomux.new.exe` (solving Windows file-locking constraints).
