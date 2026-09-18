# Benchmark & Gap Audit: [flame-engine/flame](https://github.com/flame-engine/flame)

- [← Back to Matrix](../matrix.md)
- **Language:** Dart | **Framework:** Flame | **Tier:** Sehr viel
- **Commit:** `04358b21d2233a2f84f6b03ebe2d91fd528adb77`
- **Sample file:** `examples/games/padracing/scripts/merge_files.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.5 ms | 9.5 ms | 9.2 ms | 9.5 ms | [0] |
| **post-tool-use** | 88.9 ms | 73.1 ms | 69.5 ms | 77.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 102.5 ms | 82.6 ms | 78.6 ms | 86.5 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
