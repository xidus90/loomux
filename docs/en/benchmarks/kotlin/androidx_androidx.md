# Benchmark & Gap Audit: [androidx/androidx](https://github.com/androidx/androidx)

- [← Back to Matrix](../matrix.md)
- **Language:** Kotlin | **Framework:** Android Jetpack | **Tier:** Sehr viel
- **Commit:** `994f2ad5cf175f7cebd39303bbee355d19eeb929`
- **Sample file:** `appfunctions/local_tests.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 9.5 ms | 9.2 ms | 10.0 ms | [0] |
| **post-tool-use** | 231.3 ms | 249.2 ms | 225.7 ms | 256.8 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 243.3 ms | 259.2 ms | 235.2 ms | 266.0 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `cleanBuild.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
