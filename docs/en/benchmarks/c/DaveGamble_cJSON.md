# Benchmark & Gap Audit: [DaveGamble/cJSON](https://github.com/DaveGamble/cJSON)

- [← Back to Matrix](../matrix.md)
- **Language:** C | **Framework:** cJSON | **Tier:** Sehr viel
- **Commit:** `6d9f2443ab071f86e5d9b43025a40929ec41c46c`
- **Sample file:** `cJSON.c`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 8.5 ms | 8.5 ms | 8.6 ms | [0] |
| **post-tool-use** | 25.5 ms | 22.0 ms | 21.4 ms | 23.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 36.0 ms | 30.6 ms | 29.9 ms | 31.5 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `cmake, cpp, shell`
- **Lanes:** `clang-format -i, cmake --build build --parallel, shellcheck **/*.sh`
