# Benchmark & Gap Audit: [opencv/opencv](https://github.com/opencv/opencv)

- [← Back to Matrix](../matrix.md)
- **Language:** C++ | **Framework:** OpenCV | **Tier:** Sehr viel
- **Commit:** `fb96a94a037c8d6bbb60e971e865dd89a052c031`
- **Sample file:** `3rdparty/clapack/include/cblas.h`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.5 ms | 9.5 ms | 9.5 ms | 11.0 ms | [0] |
| **post-tool-use** | 47.0 ms | 52.5 ms | 48.7 ms | 56.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 60.5 ms | 63.5 ms | 58.2 ms | 65.5 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `cmake, cpp, css, html, python`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, clang-format -i, cmake --build build --parallel, npx stylelint "**/*.{css,scss}", npx htmlhint "**/*.html"`
