# Benchmark & Gap Audit: [remix-run/remix](https://github.com/remix-run/remix)

- [← Back to Matrix](../matrix.md)
- **Language:** TypeScript | **Framework:** Remix | **Tier:** Sehr viel
- **Commit:** `03cd3404b11fc64cc5c172def4542bf92122586e`
- **Sample file:** `demos/assets/app/actions/controller.ts`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 15.5 ms | 9.5 ms | 9.5 ms | 9.5 ms | [0] |
| **post-tool-use** | 964.5 ms | 952.8 ms | 952.7 ms | 968.6 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 980.0 ms | 962.3 ms | 962.2 ms | 978.1 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `pnpm, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit`
