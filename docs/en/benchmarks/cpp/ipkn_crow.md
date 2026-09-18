# Benchmark & Gap Audit: [ipkn/crow](https://github.com/ipkn/crow)

- [← Back to Matrix](../matrix.md)
- **Language:** C++ | **Framework:** Crow | **Tier:** Sehr viel
- **Commit:** `2b43d3cd6a9a9cdbc99dfef9b86ff3f3027f3d1f`
- **Sample file:** `examples/example.cpp`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 15.5 ms | 11.0 ms | 10.5 ms | 11.0 ms | [0] |
| **post-tool-use** | 43.0 ms | 31.5 ms | 28.0 ms | 34.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 58.5 ms | 42.5 ms | 38.5 ms | 45.0 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `cmake, cpp, html`
- **Lanes:** `clang-format -i, cmake --build build --parallel, npx htmlhint "**/*.html"`
