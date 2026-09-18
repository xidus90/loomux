# Benchmark & Gap Audit: [diesel-rs/diesel](https://github.com/diesel-rs/diesel)

- [← Back to Matrix](../matrix.md)
- **Language:** Rust | **Framework:** Diesel | **Tier:** Sehr viel
- **Commit:** `bcf28baa8a2be82aa46b79aa111062363e23e6d5`
- **Sample file:** `diesel/src/associations/belongs_to.rs`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 10.0 ms | 9.5 ms | 10.2 ms | [0] |
| **post-tool-use** | 50.5 ms | 47.4 ms | 46.9 ms | 57.6 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 61.5 ms | 57.4 ms | 56.4 ms | 67.8 ms | [0] |

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

- **Stacks:** `docker, rust`
- **Lanes:** `cargo clippy -- -D warnings, cargo fmt --check`
