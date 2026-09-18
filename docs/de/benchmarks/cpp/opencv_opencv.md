# Benchmark & Lücken-Audit: [opencv/opencv](https://github.com/opencv/opencv)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C++ | **Framework:** OpenCV | **Tier:** Sehr viel
- **Commit:** `fb96a94a037c8d6bbb60e971e865dd89a052c031`
- **Beispieldatei:** `3rdparty/clapack/include/cblas.h`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.5 ms | 9.5 ms | 9.5 ms | 11.0 ms | [0] |
| **post-tool-use** | 47.0 ms | 52.5 ms | 48.7 ms | 56.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 60.5 ms | 63.5 ms | 58.2 ms | 65.5 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `cmake, cpp, css, html, python`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, clang-format -i, cmake --build build --parallel, npx stylelint "**/*.{css,scss}", npx htmlhint "**/*.html"`
