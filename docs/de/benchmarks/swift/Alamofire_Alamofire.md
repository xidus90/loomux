# Benchmark & Lücken-Audit: [Alamofire/Alamofire](https://github.com/Alamofire/Alamofire)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Swift | **Framework:** Alamofire | **Tier:** Sehr viel
- **Commit:** `bda9ed57d72988a3a2ada33d824583541f86eac6`
- **Beispieldatei:** `docs/Classes/Adapter.html`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.1 ms | 9.6 ms | 9.1 ms | 10.0 ms | [0] |
| **post-tool-use** | 999.3 ms | 956.5 ms | 932.5 ms | 978.1 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 1012.4 ms | 966.5 ms | 941.7 ms | 987.7 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `html`
- **Lanes:** `npx htmlhint "**/*.html"`
