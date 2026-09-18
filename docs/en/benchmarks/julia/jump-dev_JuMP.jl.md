# Benchmark & Gap Audit: [jump-dev/JuMP.jl](https://github.com/jump-dev/JuMP.jl)

- [← Back to Matrix](../matrix.md)
- **Language:** Julia | **Framework:** JuMP.jl | **Tier:** Sehr viel
- **Commit:** `760f535f7ac25563ec36a88ff9df85a334279de4`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 9.5 ms | 8.5 ms | 8.0 ms | 8.5 ms | [0] |
| **post-tool-use** | 16.0 ms | 14.5 ms | 14.0 ms | 15.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 25.5 ms | 23.0 ms | 22.5 ms | 23.0 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
