# Benchmark & Gap Audit: [react/react](https://github.com/react/react)

- [← Back to Matrix](../matrix.md)
- **Language:** JavaScript | **Framework:** React | **Tier:** Sehr viel
- **Commit:** `59aff3e18cb5b3a336c280bbfa57ec37999511b9`
- **Sample file:** `compiler/crates/react_compiler/src/debug_print.rs`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 9.5 ms | 9.4 ms | 9.5 ms | [0] |
| **post-tool-use** | 33.0 ms | 30.0 ms | 28.6 ms | 32.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 44.0 ms | 39.5 ms | 38.0 ms | 41.5 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `prettier` | format | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `jest` | test | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ eslint deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ prettier deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ jest deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `rust, shell`
- **Lanes:** `shellcheck **/*.sh, cargo clippy -- -D warnings, cargo fmt --check`
