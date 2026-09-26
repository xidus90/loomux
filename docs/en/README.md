# Loomux Documentation

Welcome to the Loomux documentation suite. Loomux is an all-in-one developer operating system for AI coding agents, providing deterministic code graph retrieval, persistent project memory, and zero-overhead write barriers in a single Go binary.

---

## Documentation Navigation

| Guide | Description |
|---|---|
| 🚀 **[Getting Started](getting-started.md)** | Installation, 3-minute quickstart, and agent harness wiring (Claude Code, Antigravity, Cursor). |
| 🏛️ **[Architecture & Concepts](architecture.md)** | Deep dive into Andrej Karpathy's LLM OS, Google Knowledge Items (KI), Graft AST GraphRank, and the Write Barrier Kernel. |
| ⚙️ **[Configuration Reference](configuration.md)** | Complete reference for `.loomux/config.toml` (`[verify]`, `[policy]`, `[worktree]`, `[graph]`, `[skills]`, `[privacy]`). |
| 📖 **[CLI Reference Manual](cli-reference.md)** | Comprehensive UNIX-style manual for all commands, flags, stdin JSON payloads, and exit codes. |
| 🪝 **[Hook Lifecycle & Integration](hooks.md)** | Technical specification of the 4-phase hook lifecycle, host payload formats, and decoupled SSE event streaming. |
| 🗺️ **[Migration Plan](migration.md)** | Every migration stage and every capability carried over or built during the fusion: origin, status, dependencies and priority. What comes after it is on the [roadmap](../../README.md#roadmap). |
| ⏱️ **[Performance Benchmarks](benchmarks.md)** | Measured baseline performance against predecessor binaries and strict execution budgets. |

---

## Core Philosophy

1. **Deterministic Retrieval over Stochastic Guessing**: Keywords propose candidates; the structural AST call graph concentrates mass on the components that truly matter.
2. **Curated Ground Truth over Raw Dumps**: Second Brain Knowledge Items provide verified architectural intent and decision records.
3. **Strict Memory Protection**: Sub-35ms pre-tool guards ensure coding agents never touch sensitive secrets or execute unapproved commands.
