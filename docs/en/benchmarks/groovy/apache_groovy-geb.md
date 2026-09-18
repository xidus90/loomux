# Benchmark & Gap Audit: [apache/groovy-geb](https://github.com/apache/groovy-geb)

- [← Back to Matrix](../matrix.md)
- **Language:** Groovy | **Framework:** Geb | **Tier:** Sehr viel
- **Commit:** `9c73494607825cd8900d51681919737e17c6fb12`
- **Sample file:** `build-in-docker.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 9.0 ms | 8.7 ms | 9.5 ms | [0] |
| **post-tool-use** | 73.1 ms | 72.3 ms | 70.0 ms | 76.4 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 84.1 ms | 81.0 ms | 79.5 ms | 85.4 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `build-in-docker.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
