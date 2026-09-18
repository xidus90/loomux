# Benchmark & Gap Audit: [warmcat/libwebsockets](https://github.com/warmcat/libwebsockets)

- [← Back to Matrix](../matrix.md)
- **Language:** C | **Framework:** Libwebsockets | **Tier:** Sehr viel
- **Commit:** `9a798e5f07e65bacad69a1caee9c4e5fcf22ce16`
- **Sample file:** `contrib/assets/bluecat.jpg.h`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.5 ms | 10.5 ms | 11.0 ms | [0] |
| **post-tool-use** | 73.1 ms | 86.5 ms | 86.2 ms | 92.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 85.1 ms | 97.2 ms | 97.0 ms | 103.0 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `cmake, cpp, css, html, shell`
- **Lanes:** `clang-format -i, cmake --build build --parallel, npx stylelint "**/*.{css,scss}", npx htmlhint "**/*.html", shellcheck **/*.sh`
