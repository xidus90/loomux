# Benchmark & Gap Audit: [clash-verge-rev/clash-verge-rev](https://github.com/clash-verge-rev/clash-verge-rev)

- [← Back to Matrix](../matrix.md)
- **Language:** Rust | **Framework:** Tauri | **Tier:** Sehr viel
- **Commit:** `c599f5547ec98cf998626a30c0003b41a5bd5579`
- **Sample file:** `crates/clash-verge-draft/bench/benche_me.rs`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 14.5 ms | 9.6 ms | 9.0 ms | 10.2 ms | [0] |
| **post-tool-use** | 33.0 ms | 27.8 ms | 26.9 ms | 28.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 47.5 ms | 37.4 ms | 35.9 ms | 38.7 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `cargo-clippy` | lint | `Cargo.toml` | `*none*` | No | ⚠️ Missing in Loomux |
| `cargo-test` | test | `Cargo.toml` | `*none*` | No | ⚠️ Missing in Loomux |
| `cargo-fmt` | format | `Cargo.toml` | `*none*` | No | ⚠️ Missing in Loomux |
| `eslint` | lint | `package.json` | `npx eslint --cache .` | No | ❌ Missing from PATH |
| `tsc` | typecheck | `package.json` | `npx tsc --noEmit` | No | ❌ Missing from PATH |
| `vitest` | test | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `tsc` | typecheck | `tsconfig.json` | `npx tsc --noEmit` | No | ❌ Missing from PATH |

- **Coverage Rate:** **42.9 %**
- **Identified Gaps:**
  - ⚠️ cargo-clippy deklariert (Cargo.toml), aber keine Lane in Loomux vorhanden
  - ⚠️ cargo-test deklariert (Cargo.toml), aber keine Lane in Loomux vorhanden
  - ⚠️ cargo-fmt deklariert (Cargo.toml), aber keine Lane in Loomux vorhanden
  - ⚠️ Lane für eslint vorhanden (npx eslint --cache .), aber Werkzeug nicht im PATH
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH
  - ⚠️ vitest deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH

## 3. Detected Stacks & Lanes

- **Stacks:** `biome, html, pnpm, rust, shell, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit, npx htmlhint "**/*.html", shellcheck **/*.sh, cargo clippy -- -D warnings, cargo fmt --check`
