# Benchmark & Gap Audit: [rstudio/shiny](https://github.com/rstudio/shiny)

- [← Back to Matrix](../matrix.md)
- **Language:** R | **Framework:** Shiny | **Tier:** Sehr viel
- **Commit:** `81844600fc15f1952838546faa6699d0506ce7f9`
- **Sample file:** `inst/www/shared/bootstrap/accessibility/js/bootstrap-accessibility.min.js`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.6 ms | 9.6 ms | 9.5 ms | 10.0 ms | [0] |
| **post-tool-use** | 1076.1 ms | 970.7 ms | 955.2 ms | 1006.4 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1088.7 ms | 980.7 ms | 964.7 ms | 1016.0 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `npx eslint --cache .` | No | ❌ Missing from PATH |
| `prettier` | format | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `tsc` | typecheck | `package.json` | `npx tsc --noEmit` | No | ❌ Missing from PATH |
| `tsc` | typecheck | `tsconfig.json` | `npx tsc --noEmit` | No | ❌ Missing from PATH |

- **Coverage Rate:** **75.0 %**
- **Identified Gaps:**
  - ⚠️ Lane für eslint vorhanden (npx eslint --cache .), aber Werkzeug nicht im PATH
  - ⚠️ prettier deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH

## 3. Detected Stacks & Lanes

- **Stacks:** `typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit`
