# Benchmark & Gap Audit: [FFmpeg/FFmpeg](https://github.com/FFmpeg/FFmpeg)

- [← Back to Matrix](../matrix.md)
- **Language:** C | **Framework:** FFmpeg | **Tier:** Sehr viel
- **Commit:** `6a46de95366a50b9320bf753e07338ecd78eb866`
- **Sample file:** `compat/aix/math.h`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 14.4 ms | 10.5 ms | 9.7 ms | 11.0 ms | [0] |
| **post-tool-use** | 74.7 ms | 65.3 ms | 65.2 ms | 65.6 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 89.1 ms | 75.7 ms | 75.2 ms | 76.4 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `cpp, css, html, shell`
- **Lanes:** `clang-format -i, cmake --build build --parallel, npx stylelint "**/*.{css,scss}", npx htmlhint "**/*.html", shellcheck **/*.sh`
