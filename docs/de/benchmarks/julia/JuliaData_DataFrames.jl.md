# Benchmark & Lücken-Audit: [JuliaData/DataFrames.jl](https://github.com/JuliaData/DataFrames.jl)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Julia | **Framework:** DataFrames.jl | **Tier:** Sehr viel
- **Commit:** `0f793c798a6629e70bede0c61b0564d7bc8cbd8b`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 8.8 ms | 8.6 ms | 9.6 ms | [0] |
| **post-tool-use** | 15.8 ms | 13.9 ms | 13.4 ms | 13.9 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 26.8 ms | 22.7 ms | 22.5 ms | 23.0 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
