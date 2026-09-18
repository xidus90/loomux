# Benchmark & Gap Audit: [snoyberg/conduit](https://github.com/snoyberg/conduit)

- [← Back to Matrix](../matrix.md)
- **Language:** Haskell | **Framework:** Conduit | **Tier:** Sehr viel
- **Commit:** `6b98f070fea09a3bf0a5d0897a2e27e3aa91c8fe`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 11.0 ms | 9.5 ms | 12.0 ms | [0] |
| **post-tool-use** | 23.0 ms | 18.0 ms | 17.5 ms | 22.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 34.5 ms | 28.5 ms | 27.5 ms | 34.5 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
