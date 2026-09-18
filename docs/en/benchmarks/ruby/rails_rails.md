# Benchmark & Gap Audit: [rails/rails](https://github.com/rails/rails)

- [← Back to Matrix](../matrix.md)
- **Language:** Ruby | **Framework:** Ruby on Rails | **Tier:** Sehr viel
- **Commit:** `74d13d9845857db065051a42461e9d54f6ab4e39`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.0 ms | 9.5 ms | 9.0 ms | 9.5 ms | [0] |
| **post-tool-use** | 38.1 ms | 35.6 ms | 32.8 ms | 36.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 48.1 ms | 44.6 ms | 42.3 ms | 45.5 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `rubocop` | lint | `Gemfile` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ rubocop deklariert (Gemfile), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
