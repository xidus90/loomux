# Benchmark & Gap Audit: [kyantech/Palmr](https://github.com/kyantech/Palmr)

- [← Back to Matrix](../matrix.md)
- **Language:** TypeScript | **Framework:** Fastify | **Tier:** Sehr viel
- **Commit:** `6d785dc2b2d13ea41e74ffab6688571f34580b3c`
- **Sample file:** `apps/server/reset-password.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.2 ms | 9.7 ms | 12.5 ms | [0] |
| **post-tool-use** | 88.3 ms | 77.6 ms | 77.5 ms | 85.9 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 100.3 ms | 87.7 ms | 87.2 ms | 98.4 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
