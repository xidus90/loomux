# Benchmark & Gap Audit: [tokio-rs/axum](https://github.com/tokio-rs/axum)

- [← Back to Matrix](../matrix.md)
- **Language:** Rust | **Framework:** Axum | **Tier:** Sehr viel
- **Commit:** `3f46d25e9274d485a99905d3734d37828112bc31`
- **Sample file:** `axum/benches/benches.rs`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 9.0 ms | 9.0 ms | 9.5 ms | [0] |
| **post-tool-use** | 32.0 ms | 27.0 ms | 26.5 ms | 31.4 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 43.5 ms | 36.5 ms | 35.5 ms | 40.4 ms | [0] |

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

- **Stacks:** `rust`
- **Lanes:** `cargo clippy -- -D warnings, cargo fmt --check`
