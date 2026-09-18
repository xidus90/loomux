# Benchmark & Lücken-Audit: [scotty-web/scotty](https://github.com/scotty-web/scotty)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Haskell | **Framework:** Scotty | **Tier:** Sehr viel
- **Commit:** `b5f4dfbdbd6c94f3d21492ac7925e699c3b54e87`
- **Beispieldatei:** `examples/404.html`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 9.5 ms | 9.0 ms | 10.0 ms | [0] |
| **post-tool-use** | 904.9 ms | 930.9 ms | 918.4 ms | 1038.9 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 915.9 ms | 940.9 ms | 927.4 ms | 1048.4 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `html`
- **Lanes:** `npx htmlhint "**/*.html"`
