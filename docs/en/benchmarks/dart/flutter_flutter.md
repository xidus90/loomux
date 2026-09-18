# Benchmark & Gap Audit: [flutter/flutter](https://github.com/flutter/flutter)

- [← Back to Matrix](../matrix.md)
- **Language:** Dart | **Framework:** Flutter | **Tier:** Sehr viel
- **Commit:** `2cb5ae40e66710e75a7fe0285dd874db820db2c0`
- **Sample file:** `dev/a11y_assessments/ios/Runner/Runner-Bridging-Header.h`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 8.0 ms | 8.0 ms | 8.5 ms | [0] |
| **post-tool-use** | 31.0 ms | 31.0 ms | 30.5 ms | 34.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 42.0 ms | 39.0 ms | 39.0 ms | 42.5 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `.autoroller-preupload.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `clang-format, cpp, shell`
- **Lanes:** `clang-format -i, cmake --build build --parallel, shellcheck **/*.sh`
