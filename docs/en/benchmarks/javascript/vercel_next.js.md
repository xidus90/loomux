# Benchmark & Gap Audit: [vercel/next.js](https://github.com/vercel/next.js)

- [← Back to Matrix](../matrix.md)
- **Language:** JavaScript | **Framework:** Next.js | **Tier:** Sehr viel
- **Commit:** `9a20fce8f885992fcc44a107adc41390084c2696`
- **Sample file:** `apps/bundle-analyzer/app/layout.tsx`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.8 ms | 11.0 ms | 10.0 ms | 11.0 ms | [0] |
| **post-tool-use** | 1043.8 ms | 1015.3 ms | 993.7 ms | 1058.7 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1056.6 ms | 1025.4 ms | 1004.7 ms | 1069.7 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `cargo-clippy` | lint | `Cargo.toml` | `*none*` | No | ⚠️ Missing in Loomux |
| `cargo-test` | test | `Cargo.toml` | `*none*` | No | ⚠️ Missing in Loomux |
| `cargo-fmt` | format | `Cargo.toml` | `*none*` | No | ⚠️ Missing in Loomux |
| `eslint` | lint | `package.json` | `npx eslint --cache .` | No | ❌ Missing from PATH |
| `prettier` | format | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `tsc` | typecheck | `package.json` | `npx tsc --noEmit` | No | ❌ Missing from PATH |
| `jest` | test | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `tsc` | typecheck | `tsconfig.json` | `npx tsc --noEmit` | No | ❌ Missing from PATH |

- **Coverage Rate:** **37.5 %**
- **Identified Gaps:**
  - ⚠️ cargo-clippy deklariert (Cargo.toml), aber keine Lane in Loomux vorhanden
  - ⚠️ cargo-test deklariert (Cargo.toml), aber keine Lane in Loomux vorhanden
  - ⚠️ cargo-fmt deklariert (Cargo.toml), aber keine Lane in Loomux vorhanden
  - ⚠️ Lane für eslint vorhanden (npx eslint --cache .), aber Werkzeug nicht im PATH
  - ⚠️ prettier deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH
  - ⚠️ jest deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH

## 3. Detected Stacks & Lanes

- **Stacks:** `pnpm, rust, shell, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit, shellcheck **/*.sh, cargo clippy -- -D warnings, cargo fmt --check`
