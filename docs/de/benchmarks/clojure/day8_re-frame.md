# Benchmark & Lücken-Audit: [day8/re-frame](https://github.com/day8/re-frame)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Clojure | **Framework:** Reagent | **Tier:** Sehr viel
- **Commit:** `1a1bf1df6570b17a148ebce70d500ec2da393cdc`
- **Beispieldatei:** `docs/theme/404.html`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 11.2 ms | 9.0 ms | 13.0 ms | [0] |
| **post-tool-use** | 1039.7 ms | 1024.4 ms | 965.2 ms | 1039.6 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 1051.7 ms | 1037.5 ms | 974.2 ms | 1050.8 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `html`
- **Lanes:** `npx htmlhint "**/*.html"`
