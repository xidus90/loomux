# Benchmark & Gap Audit: [yewstack/yew](https://github.com/yewstack/yew)

- [← Back to Matrix](../matrix.md)
- **Language:** Rust | **Framework:** Yew | **Tier:** Sehr viel
- **Commit:** `bfa6c19af971084f9547495388bde9aceeb81aaa`
- **Sample file:** `examples/actix_ssr_router/src/bin/ssr_router_hydrate.rs`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.0 ms | 8.0 ms | 8.0 ms | 9.0 ms | [0] |
| **post-tool-use** | 27.5 ms | 25.0 ms | 25.0 ms | 25.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 37.5 ms | 33.0 ms | 33.0 ms | 34.5 ms | [0] |

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

- **Stacks:** `css, html, rust, shell, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit, npx stylelint "**/*.{css,scss}", npx htmlhint "**/*.html", shellcheck **/*.sh, cargo clippy -- -D warnings, cargo fmt --check`
