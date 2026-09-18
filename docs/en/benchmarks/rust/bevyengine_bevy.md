# Benchmark & Gap Audit: [bevyengine/bevy](https://github.com/bevyengine/bevy)

- [← Back to Matrix](../matrix.md)
- **Language:** Rust | **Framework:** Bevy | **Tier:** Sehr viel
- **Commit:** `ebe679428fca885e27057eb0ca16cdcfe2f95d05`
- **Sample file:** `benches/benches/bevy_camera/main.rs`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.0 ms | 9.0 ms | 8.2 ms | 9.6 ms | [0] |
| **post-tool-use** | 39.3 ms | 37.2 ms | 36.6 ms | 42.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 49.3 ms | 46.1 ms | 45.4 ms | 51.0 ms | [0] |

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

- **Stacks:** `html, rust`
- **Lanes:** `npx htmlhint "**/*.html", cargo clippy -- -D warnings, cargo fmt --check`
