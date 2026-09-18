# Benchmark & Lücken-Audit: [Netflix/dgs-framework](https://github.com/Netflix/dgs-framework)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Kotlin | **Framework:** Spring Boot | **Tier:** Sehr viel
- **Commit:** `0f3f258e8d0227ab0de9736f6d8d07f418ce0a0f`
- **Beispieldatei:** `scripts/common.py`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 10.0 ms | 9.5 ms | 11.0 ms | [0] |
| **post-tool-use** | 291.9 ms | 249.5 ms | 245.5 ms | 288.8 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 302.9 ms | 259.5 ms | 255.0 ms | 299.8 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `python`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty`
