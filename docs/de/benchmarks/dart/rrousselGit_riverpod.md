# Benchmark & Lücken-Audit: [rrousselGit/riverpod](https://github.com/rrousselGit/riverpod)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Dart | **Framework:** Provider | **Tier:** Sehr viel
- **Commit:** `0313d051713158b11aeb6b698c57981563021dae`
- **Beispieldatei:** `website/docs/concepts2/auto_dispose/cache_for_usage/index.ts`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 9.5 ms | 9.5 ms | 10.5 ms | [0] |
| **post-tool-use** | 992.7 ms | 913.0 ms | 912.5 ms | 927.9 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 1003.2 ms | 923.0 ms | 922.6 ms | 937.4 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit, shellcheck **/*.sh`
