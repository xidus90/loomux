# Benchmark & Lücken-Audit: [JetBrains/compose-multiplatform](https://github.com/JetBrains/compose-multiplatform)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Kotlin | **Framework:** Compose Multiplatform | **Tier:** Sehr viel
- **Commit:** `200695dde45e1109d0437f9bac777d0bfc32c40e`
- **Beispieldatei:** `benchmarks/multiplatform/iosApp/run_ios_benchmarks.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 21.0 ms | 9.5 ms | 9.0 ms | 13.0 ms | [0] |
| **post-tool-use** | 84.2 ms | 81.1 ms | 79.1 ms | 84.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 105.2 ms | 90.6 ms | 88.1 ms | 97.0 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
