# Benchmark & Lücken-Audit: [snoyberg/conduit](https://github.com/snoyberg/conduit)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Haskell | **Framework:** Conduit | **Tier:** Sehr viel
- **Commit:** `6b98f070fea09a3bf0a5d0897a2e27e3aa91c8fe`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 11.0 ms | 9.5 ms | 12.0 ms | [0] |
| **post-tool-use** | 23.0 ms | 18.0 ms | 17.5 ms | 22.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 34.5 ms | 28.5 ms | 27.5 ms | 34.5 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
