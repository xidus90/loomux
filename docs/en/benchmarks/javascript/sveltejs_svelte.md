# Benchmark & Gap Audit: [sveltejs/svelte](https://github.com/sveltejs/svelte)

- [← Back to Matrix](../matrix.md)
- **Language:** JavaScript | **Framework:** Svelte | **Tier:** Sehr viel
- **Commit:** `6eb720a1b7cafca3ebe0ab5c76674272cd3044f9`
- **Sample file:** `benchmarking/analyze-compiler-profile.js`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 9.5 ms | 9.5 ms | 10.5 ms | [0] |
| **post-tool-use** | 944.9 ms | 992.9 ms | 991.3 ms | 1004.5 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 955.9 ms | 1002.4 ms | 1000.8 ms | 1015.0 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `npx eslint --cache .` | No | ❌ Missing from PATH |
| `prettier` | format | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `vitest` | test | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **33.3 %**
- **Identified Gaps:**
  - ⚠️ Lane für eslint vorhanden (npx eslint --cache .), aber Werkzeug nicht im PATH
  - ⚠️ prettier deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ vitest deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `pnpm, svelte, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit, npx svelte-check`
