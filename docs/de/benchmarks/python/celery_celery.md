# Benchmark & Lücken-Audit: [celery/celery](https://github.com/celery/celery)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Python | **Framework:** Celery | **Tier:** Sehr viel
- **Commit:** `208a803655672d63cf2a257d30f39c672a5b4bcf`
- **Beispieldatei:** `celery/__init__.py`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 10.1 ms | 9.5 ms | 10.5 ms | [0] |
| **post-tool-use** | 289.1 ms | 285.2 ms | 276.7 ms | 290.5 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 300.1 ms | 295.3 ms | 287.2 ms | 300.0 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `mypy` | typecheck | `pyproject.toml` | `mypy --no-error-summary --no-pretty` | Ja | ✅ Aktiv |
| `pytest` | test | `pyproject.toml` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **50.0 %**
- **Identifizierte Lücken:**
  - ⚠️ pytest deklariert (pyproject.toml), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `docker, html, python`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, npx htmlhint "**/*.html"`
