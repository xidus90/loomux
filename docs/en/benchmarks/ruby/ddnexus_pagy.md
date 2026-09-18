# Benchmark & Gap Audit: [ddnexus/pagy](https://github.com/ddnexus/pagy)

- [← Back to Matrix](../matrix.md)
- **Language:** Ruby | **Framework:** Padrino | **Tier:** Sehr viel
- **Commit:** `e85430530be38b76d4329c2fea7d870dee60e94f`
- **Sample file:** `assets/nav.html`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 10.1 ms | 9.0 ms | 10.5 ms | [0] |
| **post-tool-use** | 1110.9 ms | 940.5 ms | 919.8 ms | 942.9 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1121.9 ms | 951.0 ms | 929.9 ms | 951.9 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `rubocop` | lint | `Gemfile` | `*none*` | No | ⚠️ Missing in Loomux |
| `eslint` | lint | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ rubocop deklariert (Gemfile), aber keine Lane in Loomux vorhanden
  - ⚠️ eslint deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `html`
- **Lanes:** `npx htmlhint "**/*.html"`
