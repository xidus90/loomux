# Benchmark & Lücken-Audit: [tornadoweb/tornado](https://github.com/tornadoweb/tornado)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Python | **Framework:** Tornado | **Tier:** Sehr viel
- **Commit:** `85b6917d05a84a6d26b7488b2b56d0b52c107a0f`
- **Beispieldatei:** `demos/blog/blog.py`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.0 ms | 10.8 ms | 10.1 ms | 16.0 ms | [0] |
| **post-tool-use** | 265.3 ms | 238.4 ms | 224.2 ms | 256.7 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 278.3 ms | 248.5 ms | 235.0 ms | 272.8 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `runtests.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `python, shell`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, shellcheck **/*.sh`
