# Benchmark & Lücken-Audit: [remix-run/remix](https://github.com/remix-run/remix)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** TypeScript | **Framework:** Remix | **Tier:** Sehr viel
- **Commit:** `03cd3404b11fc64cc5c172def4542bf92122586e`
- **Beispieldatei:** `demos/assets/app/actions/controller.ts`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 15.5 ms | 9.5 ms | 9.5 ms | 9.5 ms | [0] |
| **post-tool-use** | 964.5 ms | 952.8 ms | 952.7 ms | 968.6 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 980.0 ms | 962.3 ms | 962.2 ms | 978.1 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `pnpm, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit`
