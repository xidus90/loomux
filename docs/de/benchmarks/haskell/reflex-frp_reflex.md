# Benchmark & Lücken-Audit: [reflex-frp/reflex](https://github.com/reflex-frp/reflex)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Haskell | **Framework:** Reflex | **Tier:** Sehr viel
- **Commit:** `c89e20e50475283eeea2d2b484a7551b5758ab83`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.0 ms | 9.5 ms | 13.0 ms | [0] |
| **post-tool-use** | 21.5 ms | 22.0 ms | 19.5 ms | 25.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 33.5 ms | 31.5 ms | 29.5 ms | 38.5 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
