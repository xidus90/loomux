# Benchmark & Gap Audit: [dcloudio/uni-app](https://github.com/dcloudio/uni-app)

- [← Back to Matrix](../matrix.md)
- **Language:** JavaScript | **Framework:** Vue | **Tier:** Sehr viel
- **Commit:** `8a3b223b1dfbc1cf1da5e20f4c676a37375db592`
- **Sample file:** `examples/hello-uts/common/uni-uvue.css`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.6 ms | 9.5 ms | 9.5 ms | 10.6 ms | [0] |
| **post-tool-use** | 1222.0 ms | 1257.3 ms | 1183.1 ms | 1300.7 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1234.6 ms | 1266.8 ms | 1193.7 ms | 1310.2 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `css, html`
- **Lanes:** `npx stylelint "**/*.{css,scss}", npx htmlhint "**/*.html"`
