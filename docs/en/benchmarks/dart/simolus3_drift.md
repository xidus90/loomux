# Benchmark & Gap Audit: [simolus3/drift](https://github.com/simolus3/drift)

- [← Back to Matrix](../matrix.md)
- **Language:** Dart | **Framework:** Drift | **Tier:** Sehr viel
- **Commit:** `9676564c67ebd0b8a462f38fee94d3b2757957f3`
- **Sample file:** `docs/tool/build.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 9.0 ms | 8.5 ms | 10.0 ms | [0] |
| **post-tool-use** | 79.1 ms | 81.2 ms | 80.1 ms | 85.2 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 90.1 ms | 90.1 ms | 89.8 ms | 94.2 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
