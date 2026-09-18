# Benchmark & Lücken-Audit: [JetBrains/Exposed](https://github.com/JetBrains/Exposed)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Kotlin | **Framework:** Exposed | **Tier:** Sehr viel
- **Commit:** `0e4d81a5896540acd2c8da5ca851ba34aed2b79d`
- **Beispieldatei:** `docs/about.html`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 9.5 ms | 9.0 ms | 9.5 ms | [0] |
| **post-tool-use** | 983.3 ms | 970.8 ms | 941.7 ms | 1065.4 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 995.3 ms | 980.3 ms | 951.2 ms | 1074.4 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `html`
- **Lanes:** `npx htmlhint "**/*.html"`
