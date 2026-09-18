# Benchmark & Gap Audit: [curl/curl](https://github.com/curl/curl)

- [← Back to Matrix](../matrix.md)
- **Language:** C | **Framework:** curl | **Tier:** Sehr viel
- **Commit:** `7f364029b861d064caa128f4a8cb7b34696f7db3`
- **Sample file:** `CMake/CurlTests.c`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.5 ms | 10.0 ms | 10.5 ms | [0] |
| **post-tool-use** | 52.0 ms | 47.0 ms | 46.2 ms | 50.2 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 64.0 ms | 57.0 ms | 56.7 ms | 60.7 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Yes | ✅ Active |
| `shellcheck` | lint | `appveyor.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `cmake, cpp, shell`
- **Lanes:** `clang-format -i, cmake --build build --parallel, shellcheck **/*.sh`
