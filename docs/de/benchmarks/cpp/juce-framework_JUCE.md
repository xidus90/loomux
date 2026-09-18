# Benchmark & Lücken-Audit: [juce-framework/JUCE](https://github.com/juce-framework/JUCE)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C++ | **Framework:** JUCE | **Tier:** Sehr viel
- **Commit:** `72782788ce18c2d4d760b28e0921d6ffc6431102`
- **Beispieldatei:** `examples/Assets/ADSRComponent.h`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 14.0 ms | 12.5 ms | 9.5 ms | 14.0 ms | [0] |
| **post-tool-use** | 46.0 ms | 31.5 ms | 26.0 ms | 32.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 60.0 ms | 45.0 ms | 35.5 ms | 45.5 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `clang-tidy` | lint | `.clang-tidy` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **50.0 %**
- **Identifizierte Lücken:**
  - ⚠️ clang-tidy deklariert (.clang-tidy), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `clang-tidy, cmake, cpp`
- **Lanes:** `clang-format -i, cmake --build build --parallel`
