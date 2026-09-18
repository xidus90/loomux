# Benchmark & Lücken-Audit: [ipkn/crow](https://github.com/ipkn/crow)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C++ | **Framework:** Crow | **Tier:** Sehr viel
- **Commit:** `2b43d3cd6a9a9cdbc99dfef9b86ff3f3027f3d1f`
- **Beispieldatei:** `examples/example.cpp`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 15.5 ms | 11.0 ms | 10.5 ms | 11.0 ms | [0] |
| **post-tool-use** | 43.0 ms | 31.5 ms | 28.0 ms | 34.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 58.5 ms | 42.5 ms | 38.5 ms | 45.0 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `cmake, cpp, html`
- **Lanes:** `clang-format -i, cmake --build build --parallel, npx htmlhint "**/*.html"`
