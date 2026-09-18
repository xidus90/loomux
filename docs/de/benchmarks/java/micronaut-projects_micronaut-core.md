# Benchmark & Lücken-Audit: [micronaut-projects/micronaut-core](https://github.com/micronaut-projects/micronaut-core)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Java | **Framework:** Micronaut | **Tier:** Sehr viel
- **Commit:** `79c063aca0df388b0138015da814b6ca54a000e0`
- **Beispieldatei:** `setup.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 9.5 ms | 9.2 ms | 13.1 ms | [0] |
| **post-tool-use** | 230.9 ms | 204.2 ms | 193.8 ms | 232.9 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 242.9 ms | 213.4 ms | 203.3 ms | 246.1 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `setup.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
