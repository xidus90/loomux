# Benchmark & Gap Audit: [zio/zio](https://github.com/zio/zio)

- [← Back to Matrix](../matrix.md)
- **Language:** Scala | **Framework:** ZIO | **Tier:** Sehr viel
- **Commit:** `7d62484d1a8bd93c016ed665ee5924a7acf3849c`
- **Sample file:** `website/patch-guides.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 9.2 ms | 9.0 ms | 9.5 ms | [0] |
| **post-tool-use** | 105.1 ms | 109.1 ms | 103.1 ms | 113.4 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 116.6 ms | 118.6 ms | 112.1 ms | 122.6 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
