# Benchmark & Gap Audit: [ChicagoBoss/ChicagoBoss](https://github.com/ChicagoBoss/ChicagoBoss)

- [← Back to Matrix](../matrix.md)
- **Language:** Erlang | **Framework:** ChicagoBoss | **Tier:** Sehr viel
- **Commit:** `c7b9119b090cf5f4439e8b2b034a4541e746ab0c`
- **Sample file:** `doc-src/api-controller.html`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.0 ms | 10.0 ms | 9.5 ms | 10.0 ms | [0] |
| **post-tool-use** | 934.0 ms | 952.5 ms | 939.9 ms | 979.1 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 944.0 ms | 962.0 ms | 949.9 ms | 989.1 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `css, html, shell`
- **Lanes:** `npx stylelint "**/*.{css,scss}", npx htmlhint "**/*.html", shellcheck **/*.sh`
