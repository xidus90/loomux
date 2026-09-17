# loomux

🌐 **[English](README.md)** | **[Deutsch](README.de.md)** · 📖 **[Docs (EN)](docs/en/)** | **[Handbuch (DE)](docs/de/)**

**The unified autonomous developer runtime in a single Go binary: Hooks, Skills, Code Graph, Second Brain & LLM OS.**

Loomux gives AI coding agents (Claude Code, Antigravity, Cursor, Codex) deep codebase understanding, deterministic graph retrieval, impenetrable write barriers, and automated verification loops — with zero external runtime dependencies.

- **Zero Python. Zero Node.js.** A single, self-contained Go binary (`loomux.exe` / `loomux`).
- **Sub-35ms cold start.** Lightweight execution that fits strictly within agent tool-call budgets.
- **100% test coverage per function.** Rigorous quality with mutation testing and zero unchecked code paths.
- **Windows-first, POSIX-native.** Native OS primitives (Job Objects, Junctions) with full Linux/macOS support.

> **What’s in a name?**  
> **Loomux** brings together the **Loom** (the loom that weaves together agent loops, execution threads, code graphs, and team knowledge into a single coherent fabric) and **-ux** (inspired by the Unix philosophy of small, robust, composable OS tooling and a frictionless Developer/Agent Experience).

---

## The 5 Pillars of Loomux

```mermaid
flowchart TD
    subgraph Core["loomux (Single Go Binary)"]
        P1["1. Hooks & Guard<br/><b>Policy & Write Barrier</b>"]
        P2["2. Skills & Review<br/><b>Best Practice Suites</b>"]
        P3["3. Graph & Loop<br/><b>AST, PageRank, Blast Radius</b>"]
        P4["4. Second Brain<br/><b>Wiki, ADRs, Semantic QMD</b>"]
        P5["5. LLM OS & UI<br/><b>Embedded Web Dashboard</b>"]
    end

    Agents["Coding Agents<br/>(Claude Code / Antigravity / Cursor)"] <--> |Hooks| P1
    Agents <--> |MCP / Prompts| P2
    Agents <--> |MCP Tools| P3
    Agents <--> |MCP Tools| P4
    Human["Developer / Team"] <--> |Browser / localhost| P5
```

---

## Core Workflows

### 1. The Autonomous Guard & Verification Loop

Every agent interaction is guarded and monitored in real time without blocking developer momentum.

```mermaid
sequenceDiagram
    autonumber
    actor Agent as Coding Agent
    participant Hook as loomux hook
    participant Policy as Policy & Barrier
    participant Graph as Code Graph (State)
    participant Verify as Check Chain

    Agent->>Hook: PreToolUse (Tool Call, stdin)
    Hook->>Policy: Validate path, command & write barrier (<35ms)
    alt Disallowed
        Policy-->>Agent: Exit 2 (Forbidden with exact line explanation)
    else Allowed
        Hook-->>Agent: Exit 0 (Proceed)
    end

    Agent->>Agent: Executes file edit / command

    Agent->>Hook: PostToolUse (stdin)
    Hook->>Graph: Fingerprint modified file & calculate Blast Radius (target <5ms, G4)
    Hook-->>Agent: Inline dependent callers & blast warnings

    Agent->>Hook: Stop (Turn Completion, stage 1b — no stop event wired yet)
    Hook->>Verify: Run Check Chain (configured lanes, tests, coverage gate — stage 1b)
    Verify-->>Agent: Pass (Exit 0) or Halt with feedback (Exit 1/2)
```

### 2. Deterministic Code Graph Retrieval ("GraphRank")

Most coding agents re-explore codebases from scratch every session, burning tokens and tool calls. Loomux builds a local, deterministic AST code graph once and answers queries from it using **Personalized PageRank**.

> **State (stage G1).** The ranking and the blast radius are Go packages — `internal/code/pagerank` and `internal/code/blast` — held to the reference by ported test vectors. Nothing writes the wiring graph yet and no command reads it: extractor, lexical seed, freshness check and the `loomux graph` commands are stage G2. The retrieval time is therefore still unmeasured; the figure lands in `docs/en/benchmarks.md` when G2 can rank a real repository.

```mermaid
flowchart LR
    Q["Query / Task"] --> Lex["Lexical Match<br/>(Tokens / Symbols)"]
    Lex --> |Seeds| PR["Personalized PageRank<br/>(Power-Iteration, alpha=0.25)"]
    Graph[".loomux/state/graph/<br/>AST Wiring Graph"] --> PR
    PR --> Ranked["Ranked Symbols<br/>(Structural Hubs top)"]
    Ranked --> Crux["Crux Inliner<br/>(5-10 lines key logic, $0)"]
    Crux --> Context["Injected Agent Context<br/>(Full answer, no file reads)"]
```

> **"Lexical proposes, graph disposes"**: Keywords find candidate symbols; the structural call graph concentrates mass on the components that actually matter, filtering out dead or isolated hits.

### 3. Nested MCP Composite Gateway

`loomux serve` acts as a modular **MCP Gateway Router** over Streamable HTTP, exposing cleanly namespaced capabilities to your coding agents:

```mermaid
flowchart TD
    Host["Agent Host (Claude / Antigravity / Cursor)"] <--> |stdio| Bridge["loomux mcp"]
    Bridge <--> |localhost HTTP| Root["loomux serve (Root MCP Gateway)"]
    
    subgraph Namespaces["Sub-Server Modules"]
        Root <--> Brain["brain_*<br/>(search, catalog, read, neighbors, status)"]
        Root <--> Graph["graph_*<br/>(find_code, trace_calls, file_api, find_all, repo_map)"]
        Root <--> Upstreams["Upstream Proxies<br/>(LSP servers, qmd mcp)"]
    end
```

---

## Feature & Status Matrix

Loomux is currently executing its staged fusion plan (Stage 1a pilot and Stage 1b-1 data commands complete, the wiki bundle moved in, the Stage G1 graph libraries standing without a command on them; subsequent stages in active development):

| Pillar / Capability | Description | Status |
|---|---|---|
| **1. Hooks & Guard** | | |
| Unified Pre-Tool Guard | Single-pass validation of write barriers, path protections, and forbidden commands (<35ms budget; 32–34ms measured on predecessor; a write in a linked worktree measured 34.6 ms warm (2026-09-16)). Linked git worktrees of a registered workspace are writable without a registry entry of their own. Registry and area declarations are read through the same checks as the brain commands; a broken entry refuses every write. | ✅ **Implemented** (Stage 1a) |
| Post-Tool Check Lanes | Lanes on the file that was just edited run side by side, chosen from the stacks detection finds in the tree — `go vet`, the in-process wiki lint, ruff/mypy, eslint/tsc, stylelint and the rest. A failing lane exits 2; lanes that had to be dropped are named back to the model. | ✅ **Implemented** (Stage 1a) |
| Post-Tool Blast Monitor | Dirty-file hashing and dependent caller warning on edit. Needs the wiring graph, which nothing writes yet: no hash and no warning today. | 📋 **Specified** (Stage G4) |
| Session Start | Records the commit a session starts on and warns when the binary in the project is older than `go.mod`, `go.sum` or a `.go` file under `cmd/` or `internal/`. Announces only; never blocks a turn. | ✅ **Implemented** (Stage 1a) |
| Subagent Drift & Stop Gate | Subagent drift detection and the execution counter of the stop gate. `loomux hook` knows three events — `pre-tool-use`, `post-tool-use`, `session-start`; no `stop` or `subagent-*` event is wired. | 🚧 **In Migration** (Stage 1b) |
| Check Commands | `loomux check commit-msg` (language and structure of a message), `check gofmt` (formatting, with the exit code `gofmt -l` does not give) and `dev covergate` (100% per function against a profile). | ✅ **Implemented** (Stage 1a) |
| Check Chain Table | One configured table driving every lane. `[check] lanes` is parsed from the manifest and `loomux status` names the tools a lane would need, but nothing executes the table; `config.example.toml` still calls the section `[verify]`. | 🚧 **In Migration** (Stage 1b) |
| Worktree Mirroring | Isolated subagent git worktrees with symlink/junction mirroring and session tracking. | ✅ **Implemented** (Stage 1a) |
| Zone-Free Start Path | Go's local time zone stays off the hook path: the TOML parser builds its local zones on first use (`third_party/toml`), and a gate test fails any package init over 500 allocations. `hook pre-tool-use` 7.5 ms warm against 26.5 ms before (measured 2026-09-17). | ✅ **Implemented** (no stage) |
| Claude Mods Adapter | Seat the write barrier in a `tool.check` function hook ([claude-code#91870](https://github.com/anthropics/claude-code/issues/91870)) talking to a long-lived loomux over `$.mcp.call` — removes the spawn, adds an `ask` verdict and a rendered reason. Claude-Code-only; the exec hook stays the portable path. | 💡 **Optional** (no stage) |
| **2. Skills & Best Practices** | | |
| Curated Language Suites | Embedded best-practice rules for Go (zero-alloc, err-handling, no-init), Python, TypeScript, and Rust. | 📋 **Specified** (Stage W4) |
| Graph-Aware Code Review | Review skills that leverage `graph_blast` to inspect caller impact and enforce ADR conformance. | 📋 **Specified** (Stage W4) |
| 3-Channel Distribution | Configured via `.loomux/config.toml`, synced to host folders, served via MCP prompts, or run via Web UI. | 📋 **Specified** (Stage W4) |
| **3. Code Graph & Loop** | | |
| Go Native AST Extractor | Deterministic symbol & call extraction via `go/parser` and `go/types` ($0, zero dependencies). | 📋 **Specified** (Stage G2) |
| Personalized PageRank | Power-iteration random-walk ranking over call and dependency graphs, undirected over five relations, max-normalized with a deterministic tie order. | 🧩 **Library** (Stage G1) |
| Blast Radius Engine | Transitive closure and impact analysis (`In`/`Out`, depth limits, smallest depth wins). | 🧩 **Library** (Stage G1) |
| Symbol-Coupled Grep | Regex search grouped by enclosing symbol and ranked by incoming edge degree (`inDegree`). | 📋 **Specified** (Stage G4) |
| Multi-Language AST | CGo-free Tree-sitter extraction via WebAssembly (`wazero`) with persistent AOT cache. | 💡 **Planned** (Stage G5) |
| **4. Second Brain & Wiki** | | |
| Local Markdown Wiki | The bundle itself lives in `docs/wiki/` (area `project/loomux`, moved page by page on 2026-09-16 and released line by line). `loomux lint <file>` checks one page's links and frontmatter, `loomux wiki-gate` checks the bundle's freshness and structure. Identity registers and the topic graph are written by the reindex of stage 3, not by the move. | 🚧 **In Migration** (Stage 2) |
| Semantic QMD Index | Embedding and neural search integration with local caching in `~/.cache/qmd`. | 🚧 **In Migration** (Stage 3) |
| Brain Data Commands | `loomux brain search`, `catalog`, `read`, `neighbors` and `status` over the one registry, held to the Python reference by a recorded case corpus. A registry or area declaration loomux cannot use refuses the call and names the file, entry and reason. | ✅ **Implemented** (Stage 1b-1) |
| Brain-to-Graph Bridge | Code symbols link directly to architectural decisions (ADRs) and design documentation. | 📋 **Specified** (Stage W3) |
| **5. LLM OS & Web Interface** | | |
| Embedded Web OS Dashboard | Self-contained React/Vite SPA embedded via `go:embed` on `http://127.0.0.1:<port>` with `embed_stub.go` fallback. | 📋 **Specified** (Stage W1) |
| Interactive Graph Visualizer | D3-Force / WebGL interactive graph with edge chips, type filtering, and blast overlays. | 📋 **Specified** (Stage W3) |
| Kanban Board & Loop Tracker | Real-time visual tracking of multi-step agent loops, verification lanes, and subagent state. | 📋 **Specified** (Stage W5) |
| Graphical Flow Editor | Visual DAG canvas for designing, replaying, and debugging agent verification loops. | 💡 **Future** (Stage W5) |

*Legend: ✅ Implemented & Verified in Binary · 🧩 Library implemented, no command wired to it yet · 🚧 In Active Migration / Fusion · 📋 Fully Specified & Ready for Build · 💡 Planned Vision*

---

## CLI Reference

Commands active after Stages 1a and 1b-1 vs. specified for subsequent fusion and graph stages:

### Active Commands (Stages 1a and 1b-1)
```bash
loomux check commit-msg <file>      # validate commit message against language & structure rules
loomux check gofmt [paths...]       # inspect Go file formatting without modifying files
loomux hook pre-tool-use            # run policy and global write barrier against stdin payload
loomux hook post-tool-use           # run the detected check lanes against the file just edited
loomux hook session-start           # record the session's base commit and warn about a stale binary
loomux status|doctor|explain        # inspect hook setup, verification lanes, and active harnesses (three names, one code path)
loomux worktree link|unlink|remove  # manage isolated worktree mirrors and junction paths
loomux dev covergate                # enforce 100% test coverage per function
loomux dev swap-binary              # atomically swap running binary with new compilation
loomux lint <file>                  # lint one markdown wiki page's links and frontmatter
loomux wiki-gate                    # gate wiki freshness and structural constraints
loomux brain search "<query>"       # search the visible areas through the qmd daemon (--profile fast|full|keyword)
loomux brain catalog [--scope S]    # the root catalog of the visible areas, or one area's index.md
loomux brain read <path> --scope S  # one file of an area, or one section of it (--section)
loomux brain neighbors <path> --scope S  # incoming and outgoing links of one page
loomux brain status                 # what to know before trusting an answer
```

### Specified Commands (Code Graph — Stages G2–G5)

Stage G1 wired no command: it built the libraries these commands will call.
```bash
loomux graph build [dir]            # build/rebuild .loomux/state/graph/wiring.json
loomux graph ask "<query>"          # retrieve code symbols ranked by Personalized PageRank
loomux graph callers <symbol>       # list direct callers, callees (--direction out), or full closure (-d all)
loomux graph blast [dir]            # compute blast radius of a git diff against working tree or merge base
loomux graph grep "<regex>"         # regex search grouped by enclosing symbol and ranked by coupling
loomux graph skeleton <file>        # export definition signatures and line spans (~10x token reduction)
loomux graph map                    # print token-budgeted directory clusters, hubs, and hotspots
loomux graph check                  # verify graph freshness against the working tree (exit 1 on drift)
loomux graph viz                    # launch the interactive graph viewer in your browser
```

### Specified Commands (Second Brain & Services — Stages 2–3 & W1–W5)
```bash
loomux brain reconcile             # synchronize state changes, identities, and index collections
loomux serve                        # start long-running localhost HTTP MCP service and Web OS
loomux mcp                          # stdio bridge for Claude Code, Cursor, and Antigravity
loomux init                         # wire hooks, settings, and skills into detected coding agents
```

### Developer & Worktree Tools
```bash
loomux dev covergate --profile <p>  # verify strict 100% test coverage threshold
loomux dev bench-hooks <case>       # benchmark hook execution latency against the <35ms baseline
loomux dev mutants <pkg>            # run mutation test suites across critical decision packages
loomux dev record-case --out <dir>  # record one run of a reference binary as a case
loomux dev import-cases --map <f>   # translate a directory of recorded cases into loomux cases
```

---

## Architectural Decisions: What We Built, Replaced, and Left Out

| Component / Idea | Origin / Inspiration | Decision in Loomux | Rationale |
|---|---|---|---|
| **Single Go Binary** | Architecture | ✅ **Core Mandate** | Zero Python, zero Node.js. 7.5 ms warm hook, single executable deployment, 100% test coverage. |
| **AST Code Graph & PageRank** | `trailhq/Graft` | ✅ **Adopted Natively** | $0 deterministic code graph. Personalized PageRank concentrates mass on structural hubs instead of naive keyword dumps. |
| **Blast Radius & Crux Inlining** | `trailhq/Graft` | ✅ **Adopted Natively** | Impact calculation on edit (target <5ms, unmeasured); inlines 5–10 critical logic lines instead of full file reads. |
| **Symbol-Coupled Grep** | `trailhq/Graft` | ✅ **Adopted Natively** | Regex hits grouped by enclosing symbol and ranked by incoming call edges (`inDegree`). |
| **Local Second Brain & Wiki** | Architecture | ✅ **Core Mandate** | Markdown wiki, ADRs, and identity registers stored in-repo. Code symbols directly link to architectural decisions. |
| **Node.js & C++ Toolchain** | `trailhq/Graft` | ❌ **Rejected** | Graft requires Node.js >=20, `node-gyp`, and MSVC C++ builds. Loomux remains 100% pure Go with zero external compilers. |
| **Cloud Brain Synchronization** | `trailhq/Graft` | ❌ **Rejected** | Graft syncs symbol hashes to commercial cloud APIs. Loomux keeps all knowledge, rules, and graphs 100% local and offline. |
| **Telemetry & Usage Tracking** | `trailhq/Graft` | ❌ **Rejected** | Graft includes remote telemetry pings. Loomux has zero telemetry and never calls home. |

---

## Architecture Principles

1. **Deterministic by Default**: Code graph construction, blast radius traversal, and write barriers never invoke external LLM APIs by default. They run locally, deterministically, and cost $0.
2. **Start-Time Discipline**: `loomux` measures its start floor (5.5 ms warm, 2026-09-17) continuously. No package-level variable or `init()` function may parse embedded data or perform network I/O.
3. **Strict Isolation**: Hook paths execute in-process and never depend on a running `serve` daemon.
4. **Agent-Safe Configuration**: `.loomux/config.toml` declares trust barriers and policies; it is human-maintained and write-protected from agent edits. Runtime state lives in `.loomux/state/` (git-ignored).

---

## Documentation Suite

Exhaustive guides and technical manuals are organized under [`docs/en/`](docs/en/):

| Guide | Description |
|---|---|
| 🚀 **[Getting Started](docs/en/getting-started.md)** | Installation, 3-minute quickstart, and agent harness wiring (Claude Code, Antigravity, Cursor). |
| 🏛️ **[Architecture & Concepts](docs/en/architecture.md)** | Deep dive into Andrej Karpathy's LLM OS, Google Knowledge Items (KI), Graft AST GraphRank, and the Write Barrier Kernel. |
| ⚙️ **[Configuration Reference](docs/en/configuration.md)** | Complete reference for `.loomux/config.toml` (`[verify]`, `[policy]`, `[worktree]`, `[graph]`, `[skills]`, `[privacy]`). |
| 📖 **[CLI Reference Manual](docs/en/cli-reference.md)** | Comprehensive UNIX-style manual for all commands, flags, stdin JSON payloads, and exit codes. |
| 🪝 **[Hook Lifecycle & Integration](docs/en/hooks.md)** | Technical specification of the 4-phase hook lifecycle, host payload formats, and decoupled SSE event streaming. |
| ⏱️ **[Performance Benchmarks](docs/en/benchmarks.md)** | Measured baseline performance against predecessor binaries and strict execution budgets. |

---

## Specifications & Internal Working Papers

Design documents and internal working papers are located under `docs/.superpowers/specs/`:
- [Fusion Design: Ultraloom & Ultra-Brain](docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md)
- [Code Graph Subsystem Specification](docs/.superpowers/specs/2026-09-14-loomux-code-graph-design.md)
- [Web OS & Skill System Specification](docs/.superpowers/specs/2026-09-14-loomux-web-os-design.md)

---

## Licence

PolyForm Noncommercial License 1.0 (`LICENSE.md`).
