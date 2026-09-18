# Benchmark & Lücken-Audit: [ssteinbach/zgui_cimgui_implot_sokol](https://github.com/ssteinbach/zgui_cimgui_implot_sokol)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Zig | **Framework:** Zgui | **Tier:** Sehr viel
- **Commit:** `1c6c8fd896cb3ce734098543bed67dfba45f6165`
- **Beispieldatei:** `src/zgui.cpp`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 8.6 ms | 8.0 ms | 9.2 ms | [0] |
| **post-tool-use** | 23.9 ms | 19.0 ms | 18.0 ms | 19.3 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 34.4 ms | 28.0 ms | 26.0 ms | 28.2 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `cpp`
- **Lanes:** `clang-format -i, cmake --build build --parallel`
