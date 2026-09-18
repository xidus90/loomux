# Benchmark & Gap Audit: [uwiger/unsplit](https://github.com/uwiger/unsplit)

- [← Back to Matrix](../matrix.md)
- **Language:** Erlang | **Framework:** Mnesia | **Tier:** Sehr viel
- **Commit:** `43febfcdb56c5ad5d7a3cfa0c260d9fc25026909`
- **Sample file:** `doc/stylesheet.css`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 9.5 ms | 9.5 ms | 10.6 ms | [0] |
| **post-tool-use** | 1075.2 ms | 1115.5 ms | 1078.8 ms | 1135.3 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1086.2 ms | 1126.1 ms | 1088.3 ms | 1144.9 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `css`
- **Lanes:** `npx stylelint "**/*.{css,scss}"`
