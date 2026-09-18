# Benchmark & Lücken-Audit: [JuliaPluto/PlutoUI.jl](https://github.com/JuliaPluto/PlutoUI.jl)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Julia | **Framework:** Pluto.jl | **Tier:** Sehr viel
- **Commit:** `110178bdda33c620335bb32056824e310260260b`
- **Beispieldatei:** `assets/clock.css`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 9.5 ms | 9.1 ms | 10.0 ms | [0] |
| **post-tool-use** | 1239.1 ms | 1106.4 ms | 1087.0 ms | 1122.5 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 1250.6 ms | 1115.9 ms | 1097.0 ms | 1131.6 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `css`
- **Lanes:** `npx stylelint "**/*.{css,scss}"`
