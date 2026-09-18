# Benchmark & Lücken-Audit: [pallets/flask](https://github.com/pallets/flask)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Python | **Framework:** Flask | **Tier:** Sehr viel
- **Commit:** `d73fa1cdcbd8b1465c151db8924ba58b1dd14e35`
- **Beispieldatei:** `docs/conf.py`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.3 ms | 9.7 ms | 9.6 ms | 10.1 ms | [0] |
| **post-tool-use** | 1982.6 ms | 1929.9 ms | 1884.1 ms | 1980.2 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 1992.9 ms | 1939.5 ms | 1894.2 ms | 1989.8 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `ruff` | lint | `pyproject.toml` | `ruff check --output-format=concise .` | Ja | ✅ Aktiv |
| `mypy` | typecheck | `pyproject.toml` | `*keine*` | Ja | ⚠️ Fehlt in Loomux |
| `pytest` | test | `pyproject.toml` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **33.3 %**
- **Identifizierte Lücken:**
  - ⚠️ mypy deklariert (pyproject.toml), aber keine Lane in Loomux vorhanden
  - ⚠️ pytest deklariert (pyproject.toml), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `pyright, python, uv`
- **Lanes:** `ruff check --output-format=concise ., uv run pyright`
