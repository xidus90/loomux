# Benchmark & Lücken-Audit: [fmtlib/fmt](https://github.com/fmtlib/fmt)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C++ | **Framework:** fmt | **Tier:** Sehr viel
- **Commit:** `fd0a9b6620c8f44fac2122adb1cc29664aa96325`
- **Beispieldatei:** `include/fmt/args.h`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 9.0 ms | 8.5 ms | 9.0 ms | [0] |
| **post-tool-use** | 28.0 ms | 26.5 ms | 25.0 ms | 27.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 39.5 ms | 35.0 ms | 34.0 ms | 36.5 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `clang-format` | format | `.clang-format` | `clang-format -i` | Nein | ❌ Nicht im PATH |
| `clang-tidy` | lint | `.clang-tidy` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **66.7 %**
- **Identifizierte Lücken:**
  - ⚠️ Lane für clang-format vorhanden (clang-format -i), aber Werkzeug nicht im PATH
  - ⚠️ clang-tidy deklariert (.clang-tidy), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `clang-format, clang-tidy, cmake, cpp, css`
- **Lanes:** `clang-format -i, cmake --build build --parallel, npx stylelint "**/*.{css,scss}"`
