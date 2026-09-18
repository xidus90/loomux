# Benchmark & Lücken-Audit: [Kitura/Kitura](https://github.com/Kitura/Kitura)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Swift | **Framework:** Kitura | **Tier:** Sehr viel
- **Commit:** `60334372274bd78522af7d488d6eb0920a591a8d`
- **Beispieldatei:** `Scripts/generate_router_verb_tests.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.0 ms | 10.1 ms | 9.5 ms | 12.0 ms | [0] |
| **post-tool-use** | 80.5 ms | 76.0 ms | 75.9 ms | 109.1 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 90.5 ms | 88.0 ms | 85.4 ms | 119.2 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `docker, html, shell`
- **Lanes:** `npx htmlhint "**/*.html", shellcheck **/*.sh`
