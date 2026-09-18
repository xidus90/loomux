# Benchmark & Lücken-Audit: [arrow-kt/arrow](https://github.com/arrow-kt/arrow)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Kotlin | **Framework:** Arrow | **Tier:** Sehr viel
- **Commit:** `f1ce3a45cacf8058c22bb39aa14f60921a5707a8`
- **Beispieldatei:** `test-optics-gradle-plugin.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.7 ms | 9.0 ms | 8.5 ms | 9.0 ms | [0] |
| **post-tool-use** | 68.0 ms | 63.1 ms | 62.6 ms | 64.6 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 79.7 ms | 72.1 ms | 71.1 ms | 73.6 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `test-optics-gradle-plugin.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
