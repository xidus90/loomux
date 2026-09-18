# Benchmark & Lücken-Audit: [EhPanda-Team/EhPanda](https://github.com/EhPanda-Team/EhPanda)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Swift | **Framework:** TCA | **Tier:** Sehr viel
- **Commit:** `37b979965cf7c4778877c795e8df60c2c0ff6eb3`
- **Beispieldatei:** `actions-tool/thin-payload.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.5 ms | 8.5 ms | 8.0 ms | 8.6 ms | [0] |
| **post-tool-use** | 69.1 ms | 65.0 ms | 62.4 ms | 68.4 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 81.6 ms | 73.7 ms | 70.4 ms | 76.9 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
