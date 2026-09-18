# Benchmark & Lücken-Audit: [dotnet/orleans](https://github.com/dotnet/orleans)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C# | **Framework:** Orleans | **Tier:** Sehr viel
- **Commit:** `dcf2925076e9893a6af6dd13cd489a465bb51472`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 9.0 ms | 9.0 ms | 9.5 ms | [0] |
| **post-tool-use** | 30.5 ms | 28.0 ms | 26.5 ms | 28.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 42.5 ms | 37.0 ms | 35.5 ms | 37.5 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
