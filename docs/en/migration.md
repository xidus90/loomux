# Migration plan

This file is kept current with every change: a pull request that starts,
finishes, adds or drops migration work updates it here and in
[`docs/de/migration.md`](../de/migration.md) — the status of the stage, what is
left, its dependencies and its priority, and the status of each capability it
touches. A decision (a new stage, a removal, a
changed order) goes into the [fusion spec](../.superpowers/specs/2026-09-14-loomux-fusion-design.md) first,
and this plan follows it.

## Stages

Loomux is executing a staged fusion plan. A second track — the code graph —
runs **alongside** it rather than after it, because neither of its finished
stages pulls in a dependency. *Priority* orders the open rows by three rules in turn:
what its dependencies already allow, what loomux uses on itself, then size. The
[fusion spec](../.superpowers/specs/2026-09-14-loomux-fusion-design.md) sets it under "Reihenfolge der offenen Stufen".

| Stage | Status | Origin | What it delivered | Depends on | Priority |
|---|---|---|---|---|---|
| **1a** | ✅ | ultraloom + ultra-brain | The pilot: repo scaffolding, gates, the unified guard, the post-edit lanes. loomux uses itself | — | — |
| **1b-1** | ✅ | ultra-brain | The brain read commands — `search`, `status`, `catalog`, `read`, `neighbors` — at parity with the Python reference | — | — |
| **1b-2** | ✅ | ultra-brain | `serve` with MCP over Streamable HTTP and the stdio bridge, held to the reference by a recorded case corpus. Upkeep is Stage 3 | — | — |
| **1b-3** | ✅ | ultraloom + ultra-brain | The wiki and the documentation moved in | — | — |
| **2a** | ✅ | ultraloom | The check chain: `[verify]` with presets per stack, `loomux check <profile>`, `check gocover`, post-edit on `[verify]`, process trees killed whole. loomux gates its own commits with `check precommit` | — | — |
| **2b** | ✅ | ultraloom | commit-msg with `--language`, `--calibrate` and `[commit]` | — | — |
| **2c** | ✅ | ultraloom | The stop gate (`loomux hook stop`, profile `stop`, a content fingerprint, 3 blocks in a row, the marker `.loomux/no-verify`) and `subagent-start`/`subagent-stop`, whose findings the stop gate delivers; the wiki bundle as the lane `lint/wiki` in `loomux check` and the gate. Antigravity measured and host adapters implemented for Claude Code and Antigravity. This repository runs the gate through its tracked `.claude/settings.json` | 2a ✅ | — |
| **3** | open | ultra-brain | Brain upkeep: `reconcile`, `reindex` and `embed`, `apply`/`approve`, `merge-events`, the wiki types; `brain check` with the OKF, house and federation rules, and a lint over every area | 1b-1 ✅, 1b-2 ✅ (upkeep runs in `serve`) | 3 |
| **4** | open | ultraloom + ultra-brain | Conversion and fetching, the local model, `loomux migrate`; `loomux init` with the skills, the generated `AGENTS.md`, the git hooks, the binary on the `PATH` and `--detect-only`; the host switch-over | 2b ✅ (`migrate` maps the commit language onto `[commit]`), 2c ✅ (`init` wires `stop` and `subagent-*`), 3 (the brain skills call `brain check`); the host switch-over also needs a remote for `brain-knowledge` | 5 |
| **G1** | ✅ | new | Ranking and blast radius as libraries, held to the reference by ported test vectors | — | — |
| **G2a** | ✅ | new | The extractor, the resolver, the store, the freshness probe, and `graph build` / `graph check` | — | — |
| **G2b** | ✅ | new | The query: the lexical seed, the ask sidecar, `loomux graph ask` | — | — |
| **G3** | ✅ | new | `graph_find_code` and `graph_check_freshness` on the 1b-2 gateway; a query never builds a first graph | — | — |
| **G4** | open | new | The rest of the `graph` palette (`callers`, `blast`, `grep`, `skeleton`, `map`) and the blast monitor in post-edit | G3 ✅ | 4 |
| **G5** | open | new | Multi-language extraction via `wazero` | G4 | 8 |
| **Flow** | open | ultraloom | Follow-up project: the ulflow runtime, journal, resume and replay, `verify_until_green` as a data flow | ulflow M1 (branch `feature/agent-harness`, not merged) | 6 |
| **W1 – W5** | open | ultra-brain + new | Web OS: shell (W1), the brain web app (W2), graph visualizer (W3), skill suites and review (W4), flow editor and Kanban (W5) | W1 on 1b-2 ✅; W2 on W1; W3 on W1 and G4; W4 on G4 and 4; W5 on W1 and Flow | 7 |

What the two source repositories can do and no stage had taken on yet is listed
in the [fusion spec](../.superpowers/specs/2026-09-14-loomux-fusion-design.md) under "Nachgetragen", each item with a
proposed stage or a proposed removal; the assignment holds once it is signed off.

Each stage ends green and is handed over on its own, with its own plan and — once
it is done — its own parity file recording every ruling it made.

## Capabilities

Where each capability stands:

| Pillar / Capability | Origin | Description | Status |
|---|---|---|---|
| **1. Hooks & Guard** | | | |
| Unified Pre-Tool Guard | ultraloom + ultra-brain | Single-pass validation of write barriers, path protections, and forbidden commands (<35ms budget; 32–34ms measured on predecessor; a write in a linked worktree measured 34.6 ms warm (2026-09-16)). Linked git worktrees of a registered workspace are writable without a registry entry of their own. Registry and area declarations are read through the same checks as the brain commands; a broken entry refuses every write. | ✅ **Implemented** (Stage 1a) |
| Post-Tool Check Lanes | ultraloom | The `edit` profile of `[verify]` on the file that was just edited, for its stack and in its area — `go vet` and `gofmt` of the one file, the in-process wiki lint, ruff/mypy, eslint/tsc, stylelint and the rest, from the same presets `loomux check` runs. Commands start as argv, without a shell. A failing lane exits 2; lanes skipped for a missing tool or a spent budget (`--budget`, default 50 s) are named back to the model. | ✅ **Implemented** (Stage 1a; lanes from `[verify]` since 2a) |
| Post-Tool Blast Monitor | new | Dirty-file hashing and dependent caller warning on edit. The wiring graph it needs is written now (`loomux graph build`), but nothing reads it from the edit hook yet: no hash and no warning today. | 📋 **Specified** (Stage G4) |
| Session Start | ultraloom | Records the commit a session starts on and warns when the binary in the project is older than `go.mod`, `go.sum` or a `.go` file under `cmd/` or `internal/`. Announces only; never blocks a turn. | ✅ **Implemented** (Stage 1a) |
| Subagent Drift & Stop Gate | ultraloom | `loomux hook stop` runs the `stop` profile (by default `precommit`'s four kinds, within a 270 s budget) at every turn end with new content, holds the turn with exit 2 on a red lane and gives up after 3 blocks in a row; a turn end with nothing new costs 169.5 ms on this repository. `subagent-start` and `subagent-stop` snapshot `origin`, the local branches and `HEAD`, and park what moved for the main agent's stop gate. Host adapters implemented for Claude Code and Antigravity. | ✅ **Implemented** (Stage 2c) |
| Check Commands | ultraloom | `loomux check commit-msg` (the language of every line, the Conventional Commits header, `--language`, `--calibrate` and `[commit]`), `check gofmt` (formatting, with the exit code `gofmt -l` does not give) and `check gocover` (100% per function against a profile, or a total with `--floor`). `dev covergate` is gone. | ✅ **Implemented** (Stage 1a; `gocover` 2a; `commit-msg` 2b) |
| Check Chain Table | ultraloom | One `[verify]` table drives `loomux check <profile>` and the post-edit hook: presets per stack that work without any config, one lane per kind, stack and area, `after` edges instead of stages, a verdict per kind, and `--show` to print what runs. A lane can name files it `needs`: the C++ lanes on the build tree wait for `build/CMakeCache.txt` and are `unready` until the build is configured, while an edit still runs `clang-format` on the file. Child process trees are killed whole on a timeout (a Job Object on Windows). | ✅ **Implemented** (Stage 2a) |
| Worktree Mirroring | ultraloom | Isolated subagent git worktrees with symlink/junction mirroring and session tracking. | ✅ **Implemented** (Stage 1a) |
| Zone-Free Start Path | new | Go's local time zone stays off the hook path: the TOML parser builds its local zones on first use (`third_party/toml`), and a gate test fails any package init over 500 allocations. `hook pre-tool-use` 7.5 ms warm against 26.5 ms before (measured 2026-09-17). | ✅ **Implemented** (no stage) |
| Claude Mods Adapter | new | Seat the write barrier in a `tool.check` function hook ([claude-code#91870](https://github.com/anthropics/claude-code/issues/91870)) talking to a long-lived loomux over `$.mcp.call` — removes the spawn, adds an `ask` verdict and a rendered reason. Claude-Code-only; the exec hook stays the portable path. | 💡 **Optional** (no stage) |
| **2. Skills & Best Practices** | | | |
| Curated Language Suites | new | Embedded best-practice rules for Go (zero-alloc, err-handling, no-init), Python, TypeScript, and Rust. | 📋 **Specified** (Stage W4) |
| Graph-Aware Code Review | new | Review skills that leverage `graph_blast` to inspect caller impact and enforce ADR conformance. | 📋 **Specified** (Stage W4) |
| 3-Channel Distribution | new | Configured via `.loomux/config.toml`, synced to host folders, served via MCP prompts, or run via Web UI. | 📋 **Specified** (Stage W4) |
| **3. Code Graph & Loop** | | | |
| Go Native AST Extractor | new | Deterministic symbol & call extraction via `go/parser` and `go/ast` alone — no `go/types`, no build ($0, zero dependencies). Wired behind `loomux graph build`; takes on the order of a tenth of a second on this repository, measured with its command and raw output in `docs/en/benchmarks.md`. | ✅ **Implemented** (Stage G2a) |
| Personalized PageRank | new | Power-iteration random-walk ranking over call and dependency graphs, undirected over five relations, max-normalized with a deterministic tie order. Blended with BM25 lexical candidate scoring in `loomux graph ask` (~48 ms warm retrieval). | ✅ **Implemented** (Stage G2b) |
| Blast Radius Engine | new | Transitive closure and impact analysis (`In`/`Out`, depth limits, smallest depth wins). No command asks it a question yet. | 🧩 **Library** (Stage G1) |
| Symbol-Coupled Grep | new | Regex search grouped by enclosing symbol and ranked by incoming edge degree (`inDegree`). | 📋 **Specified** (Stage G4) |
| Multi-Language AST | new | CGo-free Tree-sitter extraction via WebAssembly (`wazero`) with persistent AOT cache. | 💡 **Planned** (Stage G5) |
| MCP Service & stdio Bridge | ultra-brain | `loomux serve` holds two loopback listeners, one per channel, each with its own token, and answers seven tools over Streamable HTTP — the five `brain_*` tools and, since Stage G3, `graph_find_code` and `graph_check_freshness`; `loomux serve status` and `stop [--force]` control it, and `loomux mcp` is the stdio bridge a host starts, which starts and replaces the service itself. `internal/hooks` links none of it: a gate test reads the import graph. The front is held to the Python reference's own MCP front by a recorded case corpus, which compares the text of each `CallToolResult` and `isError` rather than the envelope two different SDKs negotiate. | ✅ **Implemented** (Stages 1b-2, G3) |
| **4. Second Brain & Wiki** | | | |
| Local Markdown Wiki | ultra-brain | The bundle itself lives in `docs/wiki/` (area `project/loomux`, moved page by page on 2026-09-16 and released line by line). `loomux lint <file>` checks one page's links and frontmatter, `loomux wiki-gate` checks the bundle's freshness and structure, and the lane `lint/wiki` checks its structure in `loomux check lint` and at every turn end. Identity registers and the topic graph are written by the reindex of stage 3, not by the move; the page rules of `brain check` (OKF, house, federation) come with it. | 🚧 **In Migration** (Stage 3) |
| Semantic QMD Index | ultra-brain | Embedding and neural search integration with local caching in `~/.cache/qmd`. | 🚧 **In Migration** (Stage 3) |
| Brain Data Commands | ultra-brain | `loomux brain search`, `catalog`, `read`, `neighbors` and `status` over the one registry, held to the Python reference by a recorded case corpus. A registry or area declaration loomux cannot use refuses the call and names the file, entry and reason. | ✅ **Implemented** (Stage 1b-1) |
| Brain-to-Graph Bridge | new | Code symbols link directly to architectural decisions (ADRs) and design documentation. | 📋 **Specified** (Stage W3) |
| **5. LLM OS & Web Interface** | | | |
| Embedded Web OS Dashboard | ultra-brain | Self-contained React/Vite SPA embedded via `go:embed` on `http://127.0.0.1:<port>` with `embed_stub.go` fallback. | 📋 **Specified** (Stage W1) |
| Interactive Graph Visualizer | new | D3-Force / WebGL interactive graph with edge chips, type filtering, and blast overlays. | 📋 **Specified** (Stage W3) |
| Kanban Board & Loop Tracker | new | Real-time visual tracking of multi-step agent loops, verification lanes, and subagent state. | 📋 **Specified** (Stage W5) |
| Graphical Flow Editor | new | Visual DAG canvas for designing, replaying, and debugging agent verification loops. | 💡 **Future** (Stage W5) |

*Origin: `ultraloom` or `ultra-brain` — migrated from that repository; `new` — built for loomux, not carried over.*

*Legend: ✅ Implemented & Verified in Binary · 🧩 Library implemented, no command wired to it yet · 🚧 In Active Migration / Fusion · 📋 Fully Specified & Ready for Build · 💡 Planned Vision*
