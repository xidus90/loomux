# Benchmark & Gap Audit: [jgraph/drawio-desktop](https://github.com/jgraph/drawio-desktop)

- [← Back to Matrix](../matrix.md)
- **Language:** JavaScript | **Framework:** Electron | **Tier:** Sehr viel
- **Commit:** `60ec92a4142a7943eae400f146db38cf0460626d`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 9.0 ms | 8.5 ms | 10.0 ms | [0] |
| **post-tool-use** | 16.5 ms | 15.0 ms | 14.5 ms | 15.9 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 28.0 ms | 23.5 ms | 23.5 ms | 26.0 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `html, shell`
- **Lanes:** `npx htmlhint "**/*.html", shellcheck **/*.sh`
