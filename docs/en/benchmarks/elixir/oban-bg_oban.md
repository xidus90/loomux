# Benchmark & Gap Audit: [oban-bg/oban](https://github.com/oban-bg/oban)

- [← Back to Matrix](../matrix.md)
- **Language:** Elixir | **Framework:** Oban | **Tier:** Sehr viel
- **Commit:** `55982090280690703c91e6834f5daecb69a67d21`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.0 ms | 9.7 ms | 9.0 ms | 12.2 ms | [0] |
| **post-tool-use** | 17.3 ms | 16.0 ms | 15.8 ms | 16.3 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 27.3 ms | 25.7 ms | 25.4 ms | 27.9 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `docker`
- **Lanes:** ``
