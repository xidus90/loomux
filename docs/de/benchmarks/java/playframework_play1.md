# Benchmark & Lücken-Audit: [playframework/play1](https://github.com/playframework/play1)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Java | **Framework:** Play Framework | **Tier:** Sehr viel
- **Commit:** `2dbefe699d493c5c08830b5db4259763f3faee61`
- **Beispieldatei:** `framework/pym/play/__init__.py`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 10.5 ms | 9.5 ms | 11.0 ms | [0] |
| **post-tool-use** | 257.9 ms | 225.3 ms | 219.2 ms | 247.7 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 268.9 ms | 234.8 ms | 229.7 ms | 258.7 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `python`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty`
