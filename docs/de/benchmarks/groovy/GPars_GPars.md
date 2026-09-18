# Benchmark & Lücken-Audit: [GPars/GPars](https://github.com/GPars/GPars)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Groovy | **Framework:** GPars | **Tier:** Sehr viel
- **Commit:** `7cddf7cf2fec1fd66ef800edccfc03315d078a2b`
- **Beispieldatei:** `overview.html`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.0 ms | 9.5 ms | 9.5 ms | 10.5 ms | [0] |
| **post-tool-use** | 938.4 ms | 924.8 ms | 896.8 ms | 935.2 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 948.4 ms | 934.3 ms | 906.3 ms | 945.7 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `html`
- **Lanes:** `npx htmlhint "**/*.html"`
