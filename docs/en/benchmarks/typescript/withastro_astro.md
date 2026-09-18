# Benchmark & Gap Audit: [withastro/astro](https://github.com/withastro/astro)

- [← Back to Matrix](../matrix.md)
- **Language:** TypeScript | **Framework:** Astro | **Tier:** Sehr viel
- **Commit:** `db2eaf17ce84a5f75c5eab30f4ae15af32de1a13`
- **Sample file:** `benchmark/bench/_template.js`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 10.7 ms | 9.5 ms | 15.0 ms | [0] |
| **post-tool-use** | 1036.2 ms | 962.2 ms | 948.1 ms | 982.7 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1047.7 ms | 971.7 ms | 958.8 ms | 997.7 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `npx eslint --cache .` | No | ❌ Missing from PATH |
| `prettier` | format | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `tsc` | typecheck | `package.json` | `npx tsc --noEmit` | No | ❌ Missing from PATH |
| `vitest` | test | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `tsc` | typecheck | `tsconfig.json` | `npx tsc --noEmit` | No | ❌ Missing from PATH |

- **Coverage Rate:** **60.0 %**
- **Identified Gaps:**
  - ⚠️ Lane für eslint vorhanden (npx eslint --cache .), aber Werkzeug nicht im PATH
  - ⚠️ prettier deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH
  - ⚠️ vitest deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH

## 3. Detected Stacks & Lanes

- **Stacks:** `pnpm, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit`
