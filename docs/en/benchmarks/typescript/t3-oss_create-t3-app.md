# Benchmark & Gap Audit: [t3-oss/create-t3-app](https://github.com/t3-oss/create-t3-app)

- [← Back to Matrix](../matrix.md)
- **Language:** TypeScript | **Framework:** Next.js | **Tier:** Sehr viel
- **Commit:** `4709861f7e67a15564c0460c13e7b4b6cfcae40d`
- **Sample file:** `cli/eslint.config.js`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.5 ms | 10.5 ms | 10.0 ms | 10.5 ms | [0] |
| **post-tool-use** | 975.0 ms | 953.0 ms | 925.0 ms | 962.1 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 987.5 ms | 963.5 ms | 935.5 ms | 972.1 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `npx eslint --cache .` | No | ❌ Missing from PATH |
| `prettier` | format | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `tsc` | typecheck | `tsconfig.json` | `npx tsc --noEmit` | No | ❌ Missing from PATH |

- **Coverage Rate:** **66.7 %**
- **Identified Gaps:**
  - ⚠️ Lane für eslint vorhanden (npx eslint --cache .), aber Werkzeug nicht im PATH
  - ⚠️ prettier deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH

## 3. Detected Stacks & Lanes

- **Stacks:** `astro, pnpm, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit`
