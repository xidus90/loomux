# Benchmark & Lücken-Audit: [barry-ran/QtScrcpy](https://github.com/barry-ran/QtScrcpy)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C++ | **Framework:** Qt | **Tier:** Sehr viel
- **Commit:** `e46403fe89ae07f6b8d8253f97e2476ce1681a73`
- **Beispieldatei:** `QtScrcpy/audio/audiooutput.cpp`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.0 ms | 9.5 ms | 10.1 ms | [0] |
| **post-tool-use** | 32.0 ms | 31.0 ms | 28.4 ms | 37.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 44.0 ms | 40.5 ms | 38.4 ms | 47.6 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `clang-format` | format | `.clang-format` | `clang-format -i` | Nein | ❌ Nicht im PATH |
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**
- **Identifizierte Lücken:**
  - ⚠️ Lane für clang-format vorhanden (clang-format -i), aber Werkzeug nicht im PATH

## 3. Erkannte Stacks & Lanes

- **Stacks:** `clang-format, cmake, cpp, shell`
- **Lanes:** `clang-format -i, cmake --build build --parallel, shellcheck **/*.sh`
