# Benchmark & Gap Audit: [sinatra/sinatra](https://github.com/sinatra/sinatra)

- [← Back to Matrix](../matrix.md)
- **Language:** Ruby | **Framework:** Sinatra | **Tier:** Sehr viel
- **Commit:** `cb22afd7902b566b6eaba6c4ea89739494a65d12`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.2 ms | 9.0 ms | 8.1 ms | 9.6 ms | [0] |
| **post-tool-use** | 18.5 ms | 16.0 ms | 15.4 ms | 16.9 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 30.8 ms | 25.6 ms | 23.5 ms | 25.9 ms | [0] |

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
