# Benchmark & Lücken-Audit: [apache/spark](https://github.com/apache/spark)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Scala | **Framework:** Spark | **Tier:** Sehr viel
- **Commit:** `6aa2688a21e3fbc9db9abc2a66e47a951fa94702`
- **Beispieldatei:** `connector/kinesis-asl/src/main/python/examples/streaming/kinesis_wordcount_asl.py`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 11.1 ms | 10.5 ms | 11.4 ms | [0] |
| **post-tool-use** | 345.2 ms | 354.5 ms | 352.4 ms | 354.6 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 357.2 ms | 365.1 ms | 363.8 ms | 365.6 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `mvn-test` | test | `pom.xml` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `ruff` | lint | `pyproject.toml` | `ruff check --output-format=concise .` | Ja | ✅ Aktiv |
| `mypy` | typecheck | `pyproject.toml` | `mypy --no-error-summary --no-pretty` | Ja | ✅ Aktiv |
| `pytest` | test | `pyproject.toml` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **50.0 %**
- **Identifizierte Lücken:**
  - ⚠️ mvn-test deklariert (pom.xml), aber keine Lane in Loomux vorhanden
  - ⚠️ pytest deklariert (pyproject.toml), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `html, python, shell`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, npx htmlhint "**/*.html", shellcheck **/*.sh`
