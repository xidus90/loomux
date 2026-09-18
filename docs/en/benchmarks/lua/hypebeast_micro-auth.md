# Benchmark & Gap Audit: [hypebeast/micro-auth](https://github.com/hypebeast/micro-auth)

- [← Back to Matrix](../matrix.md)
- **Language:** Lua | **Framework:** Lapis | **Tier:** Sehr viel
- **Commit:** `bf4f729ab26a0df133f727d18d6a48f169ea6548`
- **Sample file:** `app/entrypoint.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 8.5 ms | 8.5 ms | 9.0 ms | [0] |
| **post-tool-use** | 62.2 ms | 59.5 ms | 59.5 ms | 62.5 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 72.7 ms | 68.5 ms | 68.0 ms | 71.0 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `docker, shell`
- **Lanes:** `shellcheck **/*.sh`
