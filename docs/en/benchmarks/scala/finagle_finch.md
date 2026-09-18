# Benchmark & Gap Audit: [finagle/finch](https://github.com/finagle/finch)

- [← Back to Matrix](../matrix.md)
- **Language:** Scala | **Framework:** Finch | **Tier:** Sehr viel
- **Commit:** `5834b3c0d42f6ce14dc0ef20d550202468820bc1`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 10.0 ms | 9.5 ms | 10.2 ms | [0] |
| **post-tool-use** | 32.5 ms | 31.5 ms | 28.5 ms | 32.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 44.0 ms | 41.7 ms | 38.5 ms | 42.0 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
