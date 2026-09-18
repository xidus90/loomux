# Benchmark & Gap Audit: [fmtlib/fmt](https://github.com/fmtlib/fmt)

- [← Back to Matrix](../matrix.md)
- **Language:** C++ | **Framework:** fmt | **Tier:** Sehr viel
- **Commit:** `fd0a9b6620c8f44fac2122adb1cc29664aa96325`
- **Sample file:** `include/fmt/args.h`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 9.0 ms | 8.5 ms | 9.0 ms | [0] |
| **post-tool-use** | 28.0 ms | 26.5 ms | 25.0 ms | 27.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 39.5 ms | 35.0 ms | 34.0 ms | 36.5 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `clang-format` | format | `.clang-format` | `clang-format -i` | No | ❌ Missing from PATH |
| `clang-tidy` | lint | `.clang-tidy` | `*none*` | No | ⚠️ Missing in Loomux |
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Yes | ✅ Active |

- **Coverage Rate:** **66.7 %**
- **Identified Gaps:**
  - ⚠️ Lane für clang-format vorhanden (clang-format -i), aber Werkzeug nicht im PATH
  - ⚠️ clang-tidy deklariert (.clang-tidy), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `clang-format, clang-tidy, cmake, cpp, css`
- **Lanes:** `clang-format -i, cmake --build build --parallel, npx stylelint "**/*.{css,scss}"`
