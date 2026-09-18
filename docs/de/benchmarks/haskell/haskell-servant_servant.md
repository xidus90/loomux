# Benchmark & Lücken-Audit: [haskell-servant/servant](https://github.com/haskell-servant/servant)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Haskell | **Framework:** Servant | **Tier:** Sehr viel
- **Commit:** `5f059a40ab0062dd8d179d7de91d63e7218b0e80`
- **Beispieldatei:** `doc/conf.py`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.6 ms | 10.0 ms | 9.6 ms | 10.5 ms | [0] |
| **post-tool-use** | 242.7 ms | 236.1 ms | 229.4 ms | 287.6 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 253.4 ms | 246.6 ms | 239.4 ms | 297.2 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `streaming-benchmark.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `python, shell`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, shellcheck **/*.sh`
