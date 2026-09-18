# Benchmark & Lücken-Audit: [simolus3/drift](https://github.com/simolus3/drift)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Dart | **Framework:** Drift | **Tier:** Sehr viel
- **Commit:** `9676564c67ebd0b8a462f38fee94d3b2757957f3`
- **Beispieldatei:** `docs/tool/build.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 9.0 ms | 8.5 ms | 10.0 ms | [0] |
| **post-tool-use** | 79.1 ms | 81.2 ms | 80.1 ms | 85.2 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 90.1 ms | 90.1 ms | 89.8 ms | 94.2 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
