# Benchmark & Gap Audit: [day8/re-frame](https://github.com/day8/re-frame)

- [← Back to Matrix](../matrix.md)
- **Language:** Clojure | **Framework:** Reagent | **Tier:** Sehr viel
- **Commit:** `1a1bf1df6570b17a148ebce70d500ec2da393cdc`
- **Sample file:** `docs/theme/404.html`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 11.2 ms | 9.0 ms | 13.0 ms | [0] |
| **post-tool-use** | 1039.7 ms | 1024.4 ms | 965.2 ms | 1039.6 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1051.7 ms | 1037.5 ms | 974.2 ms | 1050.8 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `html`
- **Lanes:** `npx htmlhint "**/*.html"`
