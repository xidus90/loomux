# Benchmark & Gap Audit: [robatipoor/rustfulapi](https://github.com/robatipoor/rustfulapi)

- [← Back to Matrix](../matrix.md)
- **Language:** Rust | **Framework:** SeaORM | **Tier:** Sehr viel
- **Commit:** `dd9a9cdcab3826c65367e9a496fe4e6f583e2435`
- **Sample file:** `src/bin/app.rs`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.7 ms | 9.0 ms | 8.5 ms | 9.5 ms | [0] |
| **post-tool-use** | 29.5 ms | 28.5 ms | 27.0 ms | 32.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 40.2 ms | 38.0 ms | 35.5 ms | 41.5 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `cargo-clippy` | lint | `Cargo.toml` | `*none*` | No | ⚠️ Missing in Loomux |
| `cargo-test` | test | `Cargo.toml` | `*none*` | No | ⚠️ Missing in Loomux |
| `cargo-fmt` | format | `Cargo.toml` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ cargo-clippy deklariert (Cargo.toml), aber keine Lane in Loomux vorhanden
  - ⚠️ cargo-test deklariert (Cargo.toml), aber keine Lane in Loomux vorhanden
  - ⚠️ cargo-fmt deklariert (Cargo.toml), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `rust, shell`
- **Lanes:** `shellcheck **/*.sh, cargo clippy -- -D warnings, cargo fmt --check`
