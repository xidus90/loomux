# Benchmark & Gap Audit: [rrousselGit/riverpod](https://github.com/rrousselGit/riverpod)

- [← Back to Matrix](../matrix.md)
- **Language:** Dart | **Framework:** Provider | **Tier:** Sehr viel
- **Commit:** `0313d051713158b11aeb6b698c57981563021dae`
- **Sample file:** `website/docs/concepts2/auto_dispose/cache_for_usage/index.ts`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 9.5 ms | 9.5 ms | 10.5 ms | [0] |
| **post-tool-use** | 992.7 ms | 913.0 ms | 912.5 ms | 927.9 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1003.2 ms | 923.0 ms | 922.6 ms | 937.4 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit, shellcheck **/*.sh`
