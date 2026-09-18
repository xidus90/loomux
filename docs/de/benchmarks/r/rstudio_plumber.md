# Benchmark & Lücken-Audit: [rstudio/plumber](https://github.com/rstudio/plumber)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** R | **Framework:** Plumber | **Tier:** Sehr viel
- **Commit:** `393920505f289f914150d57e06d97347556a55dc`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 9.5 ms | 9.5 ms | 9.9 ms | [0] |
| **post-tool-use** | 26.0 ms | 26.0 ms | 23.0 ms | 28.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 37.5 ms | 35.9 ms | 32.5 ms | 37.5 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
