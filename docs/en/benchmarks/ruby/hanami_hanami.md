# Benchmark & Gap Audit: [hanami/hanami](https://github.com/hanami/hanami)

- [← Back to Matrix](../matrix.md)
- **Language:** Ruby | **Framework:** Hanami | **Tier:** Sehr viel
- **Commit:** `c31f9cf8e51e5c6c5231f45fcaea625202c30ebd`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.4 ms | 8.5 ms | 8.5 ms | 8.6 ms | [0] |
| **post-tool-use** | 13.0 ms | 11.5 ms | 11.0 ms | 16.4 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 24.4 ms | 20.0 ms | 19.6 ms | 24.9 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `rspec` | test | `Gemfile` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ rspec deklariert (Gemfile), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
