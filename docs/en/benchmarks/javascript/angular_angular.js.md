# Benchmark & Gap Audit: [angular/angular.js](https://github.com/angular/angular.js)

- [← Back to Matrix](../matrix.md)
- **Language:** JavaScript | **Framework:** Angular | **Tier:** Sehr viel
- **Commit:** `d8f77817eb5c98dec5317bc3756d1ea1812bcfbe`
- **Sample file:** `css/angular-scenario.css`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 15.0 ms | 10.0 ms | 10.0 ms | 10.0 ms | [0] |
| **post-tool-use** | 1255.8 ms | 1174.5 ms | 1172.7 ms | 1175.5 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1270.8 ms | 1184.5 ms | 1182.7 ms | 1185.5 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ eslint deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `css, shell`
- **Lanes:** `npx stylelint "**/*.{css,scss}", shellcheck **/*.sh`
