# Benchmark & Lücken-Audit: [pedestal/pedestal](https://github.com/pedestal/pedestal)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Clojure | **Framework:** Pedestal | **Tier:** Sehr viel
- **Commit:** `6ec245e6a194cc80252a026ecbec837ccc9ebda9`
- **Beispieldatei:** `docs/modules/guides/examples/war/run.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.3 ms | 8.1 ms | 10.7 ms | [0] |
| **post-tool-use** | 85.2 ms | 82.6 ms | 81.1 ms | 85.4 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 97.2 ms | 92.9 ms | 89.2 ms | 96.1 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `html, shell`
- **Lanes:** `npx htmlhint "**/*.html", shellcheck **/*.sh`
