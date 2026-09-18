# Benchmark & Gap Audit: [nuxt/create-nuxt-app](https://github.com/nuxt/create-nuxt-app)

- [← Back to Matrix](../matrix.md)
- **Language:** JavaScript | **Framework:** Nuxt | **Tier:** Sehr viel
- **Commit:** `2f04969c2f30b16b85a93bea52e489109c0a7210`
- **Sample file:** `packages/cna-template/template/frameworks/vuetify/assets/variables.scss`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.0 ms | 10.0 ms | 10.5 ms | [0] |
| **post-tool-use** | 1199.4 ms | 1093.7 ms | 1058.5 ms | 1229.6 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1211.4 ms | 1104.2 ms | 1068.5 ms | 1239.6 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `jest` | test | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ eslint deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ jest deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `css, stylelint`
- **Lanes:** `npx stylelint "**/*.{css,scss}"`
