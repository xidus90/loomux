# Benchmark & Gap Audit: [raysan5/raylib](https://github.com/raysan5/raylib)

- [← Back to Matrix](../matrix.md)
- **Language:** C | **Framework:** Raylib | **Tier:** Sehr viel
- **Commit:** `2525c16f4bbb5f1753939fc0eebe0883a4f52be8`
- **Sample file:** `examples/audio/audio_amp_envelope.c`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 15.0 ms | 9.5 ms | 8.5 ms | 9.5 ms | [0] |
| **post-tool-use** | 33.5 ms | 26.5 ms | 26.5 ms | 27.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 48.5 ms | 36.0 ms | 35.0 ms | 37.0 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `cmake, cpp, html`
- **Lanes:** `clang-format -i, cmake --build build --parallel, npx htmlhint "**/*.html"`
