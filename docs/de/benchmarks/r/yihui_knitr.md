# Benchmark & Lücken-Audit: [yihui/knitr](https://github.com/yihui/knitr)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** R | **Framework:** Knitr | **Tier:** Sehr viel
- **Commit:** `660520ef99e08650377d4d7cc12f298293c6ff0a`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 11.0 ms | 10.1 ms | 12.0 ms | [0] |
| **post-tool-use** | 21.5 ms | 20.0 ms | 19.5 ms | 20.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 33.5 ms | 31.0 ms | 29.6 ms | 32.5 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
