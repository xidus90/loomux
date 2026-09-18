# Benchmark & Lücken-Audit: [felangel/bloc](https://github.com/felangel/bloc)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Dart | **Framework:** Bloc | **Tier:** Sehr viel
- **Commit:** `dad671e41c184978cbd70d32212f2bb0867641f0`
- **Beispieldatei:** `docs/src/content.config.ts`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 9.5 ms | 9.1 ms | 10.0 ms | [0] |
| **post-tool-use** | 954.9 ms | 1024.7 ms | 917.3 ms | 1057.2 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 966.4 ms | 1034.7 ms | 926.5 ms | 1066.7 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `astro, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit`
