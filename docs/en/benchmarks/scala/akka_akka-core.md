# Benchmark & Gap Audit: [akka/akka-core](https://github.com/akka/akka-core)

- [← Back to Matrix](../matrix.md)
- **Language:** Scala | **Framework:** Akka | **Tier:** Sehr viel
- **Commit:** `e2441c7ae1b0e500ac30a121863cc7e54d919a79`
- **Sample file:** `akka-remote/src/test/resources/ssl/gen-artery-nodes.example.com.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 9.6 ms | 9.5 ms | 12.0 ms | [0] |
| **post-tool-use** | 127.5 ms | 132.5 ms | 126.3 ms | 133.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 139.5 ms | 142.5 ms | 135.9 ms | 144.5 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
