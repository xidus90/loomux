# Benchmark & Lücken-Audit: [ocornut/imgui](https://github.com/ocornut/imgui)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C++ | **Framework:** Dear ImGui | **Tier:** Sehr viel
- **Commit:** `420f1793417e39560ca39a4209c55a9f204fde13`
- **Beispieldatei:** `backends/imgui_impl_allegro5.cpp`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 78.5 ms | 68.0 ms | 40.0 ms | 104.5 ms | [0] |
| **post-tool-use** | 144.0 ms | 91.0 ms | 90.5 ms | 126.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 222.5 ms | 159.0 ms | 130.5 ms | 230.5 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `cpp`
- **Lanes:** `clang-format -i, cmake --build build --parallel`
