# Benchmark & Gap Audit: [jgm/pandoc](https://github.com/jgm/pandoc)

- [← Back to Matrix](../matrix.md)
- **Language:** Haskell | **Framework:** Pandoc | **Tier:** Sehr viel
- **Commit:** `b07038cd12f2d174ce5c0f2a5d94f0bbf56cc4d9`
- **Sample file:** `data/dzslides/template.html`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 15.5 ms | 10.5 ms | 10.0 ms | 23.0 ms | [0] |
| **post-tool-use** | 1336.1 ms | 1027.6 ms | 1018.1 ms | 1046.7 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1351.6 ms | 1038.1 ms | 1028.1 ms | 1069.7 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `css, html, shell`
- **Lanes:** `npx stylelint "**/*.{css,scss}", npx htmlhint "**/*.html", shellcheck **/*.sh`
