# Benchmark & Lücken-Audit: [WordPress/WordPress](https://github.com/WordPress/WordPress)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** PHP | **Framework:** WordPress | **Tier:** Sehr viel
- **Commit:** `e2ce85670c84edceb36ed5762e532ca16c0276d9`
- **Beispieldatei:** `readme.html`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.5 ms | 9.6 ms | 9.5 ms | 10.5 ms | [0] |
| **post-tool-use** | 944.1 ms | 924.8 ms | 887.2 ms | 1023.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 957.6 ms | 935.3 ms | 896.8 ms | 1033.0 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `html`
- **Lanes:** `npx htmlhint "**/*.html"`
