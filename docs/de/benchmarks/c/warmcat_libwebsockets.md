# Benchmark & Lücken-Audit: [warmcat/libwebsockets](https://github.com/warmcat/libwebsockets)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C | **Framework:** Libwebsockets | **Tier:** Sehr viel
- **Commit:** `9a798e5f07e65bacad69a1caee9c4e5fcf22ce16`
- **Beispieldatei:** `contrib/assets/bluecat.jpg.h`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.5 ms | 10.5 ms | 11.0 ms | [0] |
| **post-tool-use** | 73.1 ms | 86.5 ms | 86.2 ms | 92.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 85.1 ms | 97.2 ms | 97.0 ms | 103.0 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `cmake, cpp, css, html, shell`
- **Lanes:** `clang-format -i, cmake --build build --parallel, npx stylelint "**/*.{css,scss}", npx htmlhint "**/*.html", shellcheck **/*.sh`
