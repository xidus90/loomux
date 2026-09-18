# Benchmark & Lücken-Audit: [JuliaWeb/HTTP.jl](https://github.com/JuliaWeb/HTTP.jl)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Julia | **Framework:** HTTP.jl | **Tier:** Sehr viel
- **Commit:** `6883e336f9d34b47a1378deaca1210b2a431cee5`
- **Beispieldatei:** `bench/all.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 14.0 ms | 11.0 ms | 9.5 ms | 11.5 ms | [0] |
| **post-tool-use** | 90.1 ms | 98.2 ms | 89.0 ms | 102.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 104.1 ms | 109.2 ms | 100.5 ms | 111.5 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
