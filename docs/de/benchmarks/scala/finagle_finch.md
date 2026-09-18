# Benchmark & Lücken-Audit: [finagle/finch](https://github.com/finagle/finch)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Scala | **Framework:** Finch | **Tier:** Sehr viel
- **Commit:** `5834b3c0d42f6ce14dc0ef20d550202468820bc1`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 10.0 ms | 9.5 ms | 10.2 ms | [0] |
| **post-tool-use** | 32.5 ms | 31.5 ms | 28.5 ms | 32.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 44.0 ms | 41.7 ms | 38.5 ms | 42.0 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
