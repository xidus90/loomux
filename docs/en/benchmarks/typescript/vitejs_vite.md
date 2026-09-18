# Benchmark & Gap Audit: [vitejs/vite](https://github.com/vitejs/vite)

- [← Back to Matrix](../matrix.md)
- **Language:** TypeScript | **Framework:** Vite | **Tier:** Sehr viel
- **Commit:** `e9078f865cdff6bed77cd729214a7e2868f126b5`
- **Sample file:** `docs/_data/acknowledgements.data.ts`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.5 ms | 10.5 ms | 13.2 ms | [0] |
| **post-tool-use** | 1039.2 ms | 962.9 ms | 953.0 ms | 974.4 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1051.2 ms | 973.4 ms | 966.2 ms | 984.9 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `npx eslint --cache .` | No | ❌ Missing from PATH |
| `tsc` | typecheck | `package.json` | `npx tsc --noEmit` | No | ❌ Missing from PATH |
| `vitest` | test | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **66.7 %**
- **Identified Gaps:**
  - ⚠️ Lane für eslint vorhanden (npx eslint --cache .), aber Werkzeug nicht im PATH
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH
  - ⚠️ vitest deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `pnpm, shell, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit, shellcheck **/*.sh`
