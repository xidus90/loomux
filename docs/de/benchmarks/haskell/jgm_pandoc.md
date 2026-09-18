# Benchmark & Lücken-Audit: [jgm/pandoc](https://github.com/jgm/pandoc)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Haskell | **Framework:** Pandoc | **Tier:** Sehr viel
- **Commit:** `b07038cd12f2d174ce5c0f2a5d94f0bbf56cc4d9`
- **Beispieldatei:** `data/dzslides/template.html`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 15.5 ms | 10.5 ms | 10.0 ms | 23.0 ms | [0] |
| **post-tool-use** | 1336.1 ms | 1027.6 ms | 1018.1 ms | 1046.7 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 1351.6 ms | 1038.1 ms | 1028.1 ms | 1069.7 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `css, html, shell`
- **Lanes:** `npx stylelint "**/*.{css,scss}", npx htmlhint "**/*.html", shellcheck **/*.sh`
