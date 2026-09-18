# Benchmark & Gap Audit: [reflex-frp/reflex](https://github.com/reflex-frp/reflex)

- [← Back to Matrix](../matrix.md)
- **Language:** Haskell | **Framework:** Reflex | **Tier:** Sehr viel
- **Commit:** `c89e20e50475283eeea2d2b484a7551b5758ab83`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.0 ms | 9.5 ms | 13.0 ms | [0] |
| **post-tool-use** | 21.5 ms | 22.0 ms | 19.5 ms | 25.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 33.5 ms | 31.5 ms | 29.5 ms | 38.5 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
