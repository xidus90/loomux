# Benchmark & Gap Audit: [ktorio/ktor](https://github.com/ktorio/ktor)

- [← Back to Matrix](../matrix.md)
- **Language:** Kotlin | **Framework:** Ktor | **Tier:** Sehr viel
- **Commit:** `9ff002937e2f992ce7429f486c5459b7c8710a32`
- **Sample file:** `switch-base-branch.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 10.0 ms | 9.1 ms | 12.0 ms | [0] |
| **post-tool-use** | 246.2 ms | 236.5 ms | 227.8 ms | 266.1 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 257.7 ms | 245.5 ms | 237.8 ms | 278.1 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `switch-base-branch.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `update-artifact-dumps.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
