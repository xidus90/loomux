# loomux documentation

One Go binary that guards AI coding agents: hooks and a write barrier, a check chain, a code graph and a project wiki.

## Get started

1. [Installing loomux](install.md) — download, verify, the machine-wide binary, updates; Linux and macOS.
2. [First-time setup](getting-started.md) — `loomux init`, reviewing the configuration, wiring your agent.
3. [Configuration](configuration.md) — `.loomux/config.toml`, starting with your first day.

## Reference

| Guide | What it covers |
|---|---|
| [CLI reference](cli-reference.md) | Comprehensive UNIX-style manual for all commands, flags, stdin JSON payloads, and exit codes. |
| [Configuration](configuration.md) | Complete reference for `.loomux/config.toml` (`[modules]`, `[verify]`, `[policy]`, `[commit]`, `[worktree]`, `[area]`, `[index]`, `[privacy]`, `[wiki]`, `[agent]`, `[flow]`). |
| [Hooks](hooks.md) | Technical specification of the 4-phase hook lifecycle, host payload formats, and decoupled SSE event streaming. |

## Concepts

| Guide | What it covers |
|---|---|
| [Architecture](architecture.md) | Deep dive into Andrej Karpathy's LLM OS, Google Knowledge Items (KI), AST GraphRank, and the Write Barrier Kernel. |
| [Flows](flows.md) | Flows as data: the folder format, roles and models, the catalog and overrides, contributing a flow, and why a gate is a human's. |

## Project

| Guide | What it covers |
|---|---|
| [Releases and pull requests](releases.md) | How a merged pull request becomes a release, beta releases, opening a pull request, and setting up the release machinery. |
| [Benchmarks](benchmarks.md) | Chronological performance measurements with date, baseline and change, cold and warm. |
| [Benchmark matrix](benchmarks/matrix.md) | Open-source matrix across top languages with detailed reports per language and repository. |
| [Open-source matrix](open-source-matrix.md) | A catalogue of open-source GitHub projects by language, framework and star category. |
| [Supported languages and versions](support.md) | Which languages loomux checks, what it runs for each and which versions that has been verified against. |
| [GDScript frameworks](stacks/gdscript.md) | Which third-party frameworks loomux drives for GDScript, how it recognises each and what each carries. |

## Core philosophy

1. **Deterministic Retrieval over Stochastic Guessing**: Keywords propose candidates; the structural AST call graph concentrates mass on the components that truly matter.
2. **Curated Ground Truth over Raw Dumps**: Second Brain Knowledge Items provide verified architectural intent and decision records.
3. **Strict Memory Protection**: The guard decides within a budget of 35 ms. It refuses writes to secrets such as `.env`, and its command rules in `[policy.commands]` refuse a command line that a regular expression matches as written. Both hold within the limits listed in [Hooks](hooks.md#7-the-decision-path-of-pre-tool-use).
4. **Start-Time Discipline**: No package-level variable or `init()` function may parse embedded data or perform network I/O.
5. **Strict Isolation**: Hook paths execute in-process and never depend on a running `serve` daemon.
