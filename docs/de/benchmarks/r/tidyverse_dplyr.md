# Benchmark & Lücken-Audit: [tidyverse/dplyr](https://github.com/tidyverse/dplyr)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** R | **Framework:** Dplyr | **Tier:** Sehr viel
- **Commit:** `d5e94e7fa8fd4a5f79c1a707d1842216bb4c691f`
- **Beispieldatei:** `src/chop.cpp`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 9.5 ms | 9.0 ms | 9.5 ms | [0] |
| **post-tool-use** | 41.1 ms | 36.5 ms | 32.3 ms | 37.2 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 52.6 ms | 46.0 ms | 41.3 ms | 46.7 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `cpp`
- **Lanes:** `clang-format -i, cmake --build build --parallel`
