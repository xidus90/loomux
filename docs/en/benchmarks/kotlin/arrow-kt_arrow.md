# Benchmark & Gap Audit: [arrow-kt/arrow](https://github.com/arrow-kt/arrow)

- [← Back to Matrix](../matrix.md)
- **Language:** Kotlin | **Framework:** Arrow | **Tier:** Sehr viel
- **Commit:** `f1ce3a45cacf8058c22bb39aa14f60921a5707a8`
- **Sample file:** `test-optics-gradle-plugin.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.7 ms | 9.0 ms | 8.5 ms | 9.0 ms | [0] |
| **post-tool-use** | 68.0 ms | 63.1 ms | 62.6 ms | 64.6 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 79.7 ms | 72.1 ms | 71.1 ms | 73.6 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `test-optics-gradle-plugin.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
