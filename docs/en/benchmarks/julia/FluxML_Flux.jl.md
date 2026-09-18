# Benchmark & Gap Audit: [FluxML/Flux.jl](https://github.com/FluxML/Flux.jl)

- [← Back to Matrix](../matrix.md)
- **Language:** Julia | **Framework:** Flux | **Tier:** Sehr viel
- **Commit:** `b617b334a07a241027b660b1714dd96ceb331f24`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.9 ms | 9.0 ms | 8.5 ms | 9.5 ms | [0] |
| **post-tool-use** | 24.6 ms | 18.5 ms | 18.5 ms | 18.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 37.5 ms | 27.5 ms | 27.0 ms | 28.0 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
