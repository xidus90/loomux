# Benchmark & Lücken-Audit: [ChartsOrg/Charts](https://github.com/ChartsOrg/Charts)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Swift | **Framework:** Charts | **Tier:** Sehr viel
- **Commit:** `1bad5469f57628782110b05996d4fa00473abf06`
- **Beispieldatei:** `carthage.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 9.0 ms | 8.3 ms | 8.0 ms | 9.0 ms | [0] |
| **post-tool-use** | 65.0 ms | 64.5 ms | 63.0 ms | 66.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 74.0 ms | 72.8 ms | 72.0 ms | 74.0 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `carthage.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
