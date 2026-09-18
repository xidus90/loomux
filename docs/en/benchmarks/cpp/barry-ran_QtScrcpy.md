# Benchmark & Gap Audit: [barry-ran/QtScrcpy](https://github.com/barry-ran/QtScrcpy)

- [← Back to Matrix](../matrix.md)
- **Language:** C++ | **Framework:** Qt | **Tier:** Sehr viel
- **Commit:** `e46403fe89ae07f6b8d8253f97e2476ce1681a73`
- **Sample file:** `QtScrcpy/audio/audiooutput.cpp`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.0 ms | 9.5 ms | 10.1 ms | [0] |
| **post-tool-use** | 32.0 ms | 31.0 ms | 28.4 ms | 37.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 44.0 ms | 40.5 ms | 38.4 ms | 47.6 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `clang-format` | format | `.clang-format` | `clang-format -i` | No | ❌ Missing from PATH |
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**
- **Identified Gaps:**
  - ⚠️ Lane für clang-format vorhanden (clang-format -i), aber Werkzeug nicht im PATH

## 3. Detected Stacks & Lanes

- **Stacks:** `clang-format, cmake, cpp, shell`
- **Lanes:** `clang-format -i, cmake --build build --parallel, shellcheck **/*.sh`
