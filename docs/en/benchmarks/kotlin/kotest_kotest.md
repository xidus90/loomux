# Benchmark & Gap Audit: [kotest/kotest](https://github.com/kotest/kotest)

- [← Back to Matrix](../matrix.md)
- **Language:** Kotlin | **Framework:** Kotest | **Tier:** Sehr viel
- **Commit:** `8fabcdb687d17d4f6432d4d7b2fb340dc97c34e4`
- **Sample file:** `documentation/cut-docs.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.2 ms | 9.5 ms | 8.6 ms | 9.7 ms | [0] |
| **post-tool-use** | 90.6 ms | 87.0 ms | 86.9 ms | 94.4 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 101.8 ms | 96.6 ms | 96.5 ms | 103.0 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
