# Benchmark & Lücken-Audit: [dropwizard/dropwizard](https://github.com/dropwizard/dropwizard)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Java | **Framework:** Dropwizard | **Tier:** Sehr viel
- **Commit:** `c9aa3f0ad7ccfb7ed83ce568f582a272e6e60427`
- **Beispieldatei:** `docs/source/_ext/dropwizard_literalinclude.py`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.5 ms | 10.0 ms | 10.0 ms | 10.1 ms | [0] |
| **post-tool-use** | 281.2 ms | 266.2 ms | 263.1 ms | 277.6 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 294.7 ms | 276.2 ms | 273.2 ms | 287.6 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `mvn-test` | test | `pom.xml` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `shellcheck` | lint | `prepare_docs.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **50.0 %**
- **Identifizierte Lücken:**
  - ⚠️ mvn-test deklariert (pom.xml), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `python, shell`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, shellcheck **/*.sh`
