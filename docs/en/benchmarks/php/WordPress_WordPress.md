# Benchmark & Gap Audit: [WordPress/WordPress](https://github.com/WordPress/WordPress)

- [← Back to Matrix](../matrix.md)
- **Language:** PHP | **Framework:** WordPress | **Tier:** Sehr viel
- **Commit:** `e2ce85670c84edceb36ed5762e532ca16c0276d9`
- **Sample file:** `readme.html`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.5 ms | 9.6 ms | 9.5 ms | 10.5 ms | [0] |
| **post-tool-use** | 944.1 ms | 924.8 ms | 887.2 ms | 1023.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 957.6 ms | 935.3 ms | 896.8 ms | 1033.0 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `html`
- **Lanes:** `npx htmlhint "**/*.html"`
