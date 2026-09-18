# Benchmark & Lücken-Audit: [FluxML/Flux.jl](https://github.com/FluxML/Flux.jl)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Julia | **Framework:** Flux | **Tier:** Sehr viel
- **Commit:** `b617b334a07a241027b660b1714dd96ceb331f24`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.9 ms | 9.0 ms | 8.5 ms | 9.5 ms | [0] |
| **post-tool-use** | 24.6 ms | 18.5 ms | 18.5 ms | 18.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 37.5 ms | 27.5 ms | 27.0 ms | 28.0 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
