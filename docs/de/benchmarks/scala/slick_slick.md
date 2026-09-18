# Benchmark & Lücken-Audit: [slick/slick](https://github.com/slick/slick)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Scala | **Framework:** Slick | **Tier:** Sehr viel
- **Commit:** `53dacd52ded71696ff0237c2452d6511dd6838d4`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.0 ms | 9.0 ms | 8.5 ms | 9.5 ms | [0] |
| **post-tool-use** | 25.5 ms | 25.5 ms | 25.5 ms | 26.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 35.5 ms | 35.0 ms | 34.0 ms | 35.5 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `docker`
- **Lanes:** ``
