# Benchmark & Gap Audit: [tokio-rs/tokio](https://github.com/tokio-rs/tokio)

- [← Back to Matrix](../matrix.md)
- **Language:** Rust | **Framework:** Tokio | **Tier:** Sehr viel
- **Commit:** `cf782c5b917b7ea21b6f97f07104ebf16d35f5f8`
- **Sample file:** `benches/copy.rs`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.0 ms | 8.5 ms | 11.6 ms | [0] |
| **post-tool-use** | 39.2 ms | 35.5 ms | 32.9 ms | 36.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 51.2 ms | 44.0 ms | 42.9 ms | 48.1 ms | [0] |

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
