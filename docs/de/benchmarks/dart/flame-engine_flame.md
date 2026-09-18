# Benchmark & Lücken-Audit: [flame-engine/flame](https://github.com/flame-engine/flame)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Dart | **Framework:** Flame | **Tier:** Sehr viel
- **Commit:** `04358b21d2233a2f84f6b03ebe2d91fd528adb77`
- **Beispieldatei:** `examples/games/padracing/scripts/merge_files.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.5 ms | 9.5 ms | 9.2 ms | 9.5 ms | [0] |
| **post-tool-use** | 88.9 ms | 73.1 ms | 69.5 ms | 77.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 102.5 ms | 82.6 ms | 78.6 ms | 86.5 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
