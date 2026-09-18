# Benchmark & Lücken-Audit: [rstudio/rmarkdown](https://github.com/rstudio/rmarkdown)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** R | **Framework:** Rmarkdown | **Tier:** Sehr viel
- **Commit:** `ea64bc44b86da766448aa69ff4e28a3a00faac48`
- **Beispieldatei:** `tools/install-pandoc.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 8.5 ms | 8.5 ms | 9.9 ms | [0] |
| **post-tool-use** | 73.1 ms | 71.1 ms | 67.0 ms | 80.8 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 84.1 ms | 81.0 ms | 75.5 ms | 89.3 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
