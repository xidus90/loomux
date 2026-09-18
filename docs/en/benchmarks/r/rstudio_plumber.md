# Benchmark & Gap Audit: [rstudio/plumber](https://github.com/rstudio/plumber)

- [← Back to Matrix](../matrix.md)
- **Language:** R | **Framework:** Plumber | **Tier:** Sehr viel
- **Commit:** `393920505f289f914150d57e06d97347556a55dc`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 9.5 ms | 9.5 ms | 9.9 ms | [0] |
| **post-tool-use** | 26.0 ms | 26.0 ms | 23.0 ms | 28.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 37.5 ms | 35.9 ms | 32.5 ms | 37.5 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
