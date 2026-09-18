# Benchmark & Gap Audit: [yesodweb/yesod](https://github.com/yesodweb/yesod)

- [← Back to Matrix](../matrix.md)
- **Language:** Haskell | **Framework:** Yesod | **Tier:** Sehr viel
- **Commit:** `4d3fcc460fabc0b1f85ee4cc705cc79c5753ccfc`
- **Sample file:** `yesod-bin/refreshing.html`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.0 ms | 9.5 ms | 9.5 ms | 10.5 ms | [0] |
| **post-tool-use** | 955.0 ms | 939.2 ms | 924.9 ms | 978.7 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 965.0 ms | 949.7 ms | 934.4 ms | 988.2 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `html, shell`
- **Lanes:** `npx htmlhint "**/*.html", shellcheck **/*.sh`
