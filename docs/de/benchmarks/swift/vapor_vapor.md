# Benchmark & Lücken-Audit: [vapor/vapor](https://github.com/vapor/vapor)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Swift | **Framework:** Vapor | **Tier:** Sehr viel
- **Commit:** `e142b03ad0b779db718e4517343b3dd89f4fdd58`
- **Beispieldatei:** `Performance/run-wrk.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 9.0 ms | 8.2 ms | 9.0 ms | [0] |
| **post-tool-use** | 58.8 ms | 59.1 ms | 58.5 ms | 64.5 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 69.3 ms | 68.1 ms | 66.7 ms | 73.5 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
