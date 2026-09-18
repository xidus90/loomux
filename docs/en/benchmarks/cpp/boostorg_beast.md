# Benchmark & Gap Audit: [boostorg/beast](https://github.com/boostorg/beast)

- [← Back to Matrix](../matrix.md)
- **Language:** C++ | **Framework:** Boost | **Tier:** Sehr viel
- **Commit:** `66e232db4baf4d7a0bc0ab153d414829b9f28d6b`
- **Sample file:** `example/advanced/server/advanced_server.cpp`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 20.5 ms | 9.5 ms | 9.5 ms | 9.7 ms | [0] |
| **post-tool-use** | 44.3 ms | 32.5 ms | 31.5 ms | 33.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 64.8 ms | 42.2 ms | 41.0 ms | 43.0 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `cmake, cpp, html, shell`
- **Lanes:** `clang-format -i, cmake --build build --parallel, npx htmlhint "**/*.html", shellcheck **/*.sh`
