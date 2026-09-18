# Benchmark & Gap Audit: [Overtorment/NoobHub](https://github.com/Overtorment/NoobHub)

- [← Back to Matrix](../matrix.md)
- **Language:** Lua | **Framework:** Gideros | **Tier:** Sehr viel
- **Commit:** `bea904c9aecd31b4c7f3b2b09bac2533291d8011`
- **Sample file:** `run-tests.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 8.5 ms | 8.5 ms | 9.0 ms | [0] |
| **post-tool-use** | 63.9 ms | 61.0 ms | 58.5 ms | 67.5 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 74.4 ms | 70.0 ms | 67.0 ms | 76.1 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `run-tests.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
