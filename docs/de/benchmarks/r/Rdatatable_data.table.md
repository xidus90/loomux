# Benchmark & Lücken-Audit: [Rdatatable/data.table](https://github.com/Rdatatable/data.table)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** R | **Framework:** Data.table | **Tier:** Sehr viel
- **Commit:** `a3eeb4c0caec51afd66bdf9bbc16817e12a47a40`
- **Beispieldatei:** `site/_navbar.html`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 16.0 ms | 10.0 ms | 9.0 ms | 11.1 ms | [0] |
| **post-tool-use** | 942.0 ms | 937.7 ms | 931.7 ms | 946.3 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 958.0 ms | 948.8 ms | 941.7 ms | 955.3 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `html`
- **Lanes:** `npx htmlhint "**/*.html"`
