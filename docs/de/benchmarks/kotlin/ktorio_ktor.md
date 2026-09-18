# Benchmark & Lücken-Audit: [ktorio/ktor](https://github.com/ktorio/ktor)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Kotlin | **Framework:** Ktor | **Tier:** Sehr viel
- **Commit:** `9ff002937e2f992ce7429f486c5459b7c8710a32`
- **Beispieldatei:** `switch-base-branch.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 10.0 ms | 9.1 ms | 12.0 ms | [0] |
| **post-tool-use** | 246.2 ms | 236.5 ms | 227.8 ms | 266.1 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 257.7 ms | 245.5 ms | 237.8 ms | 278.1 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `switch-base-branch.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `update-artifact-dumps.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
