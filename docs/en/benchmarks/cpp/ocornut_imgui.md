# Benchmark & Gap Audit: [ocornut/imgui](https://github.com/ocornut/imgui)

- [← Back to Matrix](../matrix.md)
- **Language:** C++ | **Framework:** Dear ImGui | **Tier:** Sehr viel
- **Commit:** `420f1793417e39560ca39a4209c55a9f204fde13`
- **Sample file:** `backends/imgui_impl_allegro5.cpp`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 78.5 ms | 68.0 ms | 40.0 ms | 104.5 ms | [0] |
| **post-tool-use** | 144.0 ms | 91.0 ms | 90.5 ms | 126.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 222.5 ms | 159.0 ms | 130.5 ms | 230.5 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `cpp`
- **Lanes:** `clang-format -i, cmake --build build --parallel`
