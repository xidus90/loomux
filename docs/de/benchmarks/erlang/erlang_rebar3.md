# Benchmark & Lücken-Audit: [erlang/rebar3](https://github.com/erlang/rebar3)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Erlang | **Framework:** Rebar3 | **Tier:** Sehr viel
- **Commit:** `e277c503562e313ef2829a6bcf0aa8e7dd6cc09c`
- **Beispieldatei:** `pr2relnotes.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 8.1 ms | 8.0 ms | 8.5 ms | [0] |
| **post-tool-use** | 62.4 ms | 60.5 ms | 57.9 ms | 60.5 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 74.4 ms | 68.5 ms | 66.4 ms | 68.6 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `pr2relnotes.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `vendor_hex_core.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
