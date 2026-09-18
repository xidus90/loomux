# Benchmark & Lücken-Audit: [django/django](https://github.com/django/django)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Python | **Framework:** Django | **Tier:** Sehr viel
- **Commit:** `a3f0642f69277f01611d0d3e829fc3b85aded2ec`
- **Beispieldatei:** `Gruntfile.js`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 9.5 ms | 9.5 ms | 10.0 ms | [0] |
| **post-tool-use** | 1003.8 ms | 999.7 ms | 978.1 ms | 1121.5 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 1014.8 ms | 1009.2 ms | 988.1 ms | 1131.0 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `biome, html, python, shell, typescript`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, npx eslint --cache ., npx tsc --noEmit, npx htmlhint "**/*.html", shellcheck **/*.sh`
