# Benchmark & Gap Audit: [warpdotdev/warp](https://github.com/warpdotdev/warp)

- [← Back to Matrix](../matrix.md)
- **Language:** Rust | **Framework:** Warp | **Tier:** Sehr viel
- **Commit:** `a0f5eb31a2ba46e46898f41d8c0e256ae7d20dda`
- **Sample file:** `app/build.rs`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.0 ms | 8.6 ms | 8.0 ms | 9.0 ms | [0] |
| **post-tool-use** | 36.0 ms | 32.6 ms | 32.0 ms | 35.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 46.0 ms | 41.0 ms | 40.6 ms | 44.1 ms | [0] |

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
