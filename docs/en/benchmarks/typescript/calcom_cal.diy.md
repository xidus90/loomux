# Benchmark & Gap Audit: [calcom/cal.diy](https://github.com/calcom/cal.diy)

- [← Back to Matrix](../matrix.md)
- **Language:** TypeScript | **Framework:** tRPC | **Tier:** Sehr viel
- **Commit:** `6bc45298226f96ff79e0c070c8b2ce39727e8477`
- **Sample file:** `__checks__/calcom-dashboard.check.js`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 11.0 ms | 10.5 ms | 12.0 ms | [0] |
| **post-tool-use** | 1315.1 ms | 1162.0 ms | 1112.4 ms | 1194.8 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1326.6 ms | 1173.9 ms | 1123.4 ms | 1205.3 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `tsc` | typecheck | `package.json` | `npx tsc --noEmit` | No | ❌ Missing from PATH |
| `vitest` | test | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `jest` | test | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **33.3 %**
- **Identified Gaps:**
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH
  - ⚠️ vitest deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ jest deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `biome, css, docker, shell, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit, npx stylelint "**/*.{css,scss}", shellcheck **/*.sh`
