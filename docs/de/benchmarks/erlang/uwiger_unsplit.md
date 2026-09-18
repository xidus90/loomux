# Benchmark & Lücken-Audit: [uwiger/unsplit](https://github.com/uwiger/unsplit)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Erlang | **Framework:** Mnesia | **Tier:** Sehr viel
- **Commit:** `43febfcdb56c5ad5d7a3cfa0c260d9fc25026909`
- **Beispieldatei:** `doc/stylesheet.css`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 9.5 ms | 9.5 ms | 10.6 ms | [0] |
| **post-tool-use** | 1075.2 ms | 1115.5 ms | 1078.8 ms | 1135.3 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 1086.2 ms | 1126.1 ms | 1088.3 ms | 1144.9 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `css`
- **Lanes:** `npx stylelint "**/*.{css,scss}"`
