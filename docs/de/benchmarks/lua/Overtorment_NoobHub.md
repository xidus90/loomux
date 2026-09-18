# Benchmark & Lücken-Audit: [Overtorment/NoobHub](https://github.com/Overtorment/NoobHub)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Lua | **Framework:** Gideros | **Tier:** Sehr viel
- **Commit:** `bea904c9aecd31b4c7f3b2b09bac2533291d8011`
- **Beispieldatei:** `run-tests.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 8.5 ms | 8.5 ms | 9.0 ms | [0] |
| **post-tool-use** | 63.9 ms | 61.0 ms | 58.5 ms | 67.5 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 74.4 ms | 70.0 ms | 67.0 ms | 76.1 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `run-tests.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
