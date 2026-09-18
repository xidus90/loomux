# Benchmark & Lücken-Audit: [typelevel/cats-effect](https://github.com/typelevel/cats-effect)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Scala | **Framework:** Cats Effect | **Tier:** Sehr viel
- **Commit:** `e07609556fbeeb7f21e0e5e51acfc851707acca1`
- **Beispieldatei:** `example/test-js.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 9.0 ms | 8.5 ms | 9.0 ms | [0] |
| **post-tool-use** | 83.8 ms | 89.0 ms | 84.5 ms | 171.5 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 95.3 ms | 98.1 ms | 93.0 ms | 180.5 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
