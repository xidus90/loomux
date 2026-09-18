# Benchmark & Lücken-Audit: [FFmpeg/FFmpeg](https://github.com/FFmpeg/FFmpeg)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C | **Framework:** FFmpeg | **Tier:** Sehr viel
- **Commit:** `6a46de95366a50b9320bf753e07338ecd78eb866`
- **Beispieldatei:** `compat/aix/math.h`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 14.4 ms | 10.5 ms | 9.7 ms | 11.0 ms | [0] |
| **post-tool-use** | 74.7 ms | 65.3 ms | 65.2 ms | 65.6 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 89.1 ms | 75.7 ms | 75.2 ms | 76.4 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `cpp, css, html, shell`
- **Lanes:** `clang-format -i, cmake --build build --parallel, npx stylelint "**/*.{css,scss}", npx htmlhint "**/*.html", shellcheck **/*.sh`
