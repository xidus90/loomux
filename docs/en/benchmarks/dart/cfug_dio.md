# Benchmark & Gap Audit: [cfug/dio](https://github.com/cfug/dio)

- [← Back to Matrix](../matrix.md)
- **Language:** Dart | **Framework:** Dio | **Tier:** Sehr viel
- **Commit:** `4684e29dabaa2655606f1620a4ed316ee3f8c963`
- **Sample file:** `scripts/prepare_pinning_certs.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 9.6 ms | 9.1 ms | 11.5 ms | [0] |
| **post-tool-use** | 66.6 ms | 71.7 ms | 63.1 ms | 73.5 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 77.6 ms | 82.6 ms | 72.7 ms | 83.2 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
