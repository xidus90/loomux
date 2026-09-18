# Benchmark & Lücken-Audit: [softwaremill/tapir](https://github.com/softwaremill/tapir)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Scala | **Framework:** Tapir | **Tier:** Sehr viel
- **Commit:** `d15eb5900fa2c3ef6bd71f090ead8dc3042257fa`
- **Beispieldatei:** `doc/conf.py`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.5 ms | 9.6 ms | 9.5 ms | 10.0 ms | [0] |
| **post-tool-use** | 247.9 ms | 266.9 ms | 232.1 ms | 280.6 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 260.5 ms | 276.4 ms | 241.7 ms | 290.6 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `python, shell`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, shellcheck **/*.sh`
