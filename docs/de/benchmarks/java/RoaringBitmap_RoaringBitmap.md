# Benchmark & Lücken-Audit: [RoaringBitmap/RoaringBitmap](https://github.com/RoaringBitmap/RoaringBitmap)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Java | **Framework:** Apache Spark | **Tier:** Sehr viel
- **Commit:** `f2289086f4204df51a1d9c4e494d50ffa3ee8988`
- **Beispieldatei:** `jmh/grabresults.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 9.5 ms | 9.0 ms | 10.5 ms | [0] |
| **post-tool-use** | 71.9 ms | 70.1 ms | 66.0 ms | 77.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 83.4 ms | 79.6 ms | 76.5 ms | 86.0 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
