# Benchmark & Gap Audit: [MudBlazor/MudBlazor](https://github.com/MudBlazor/MudBlazor)

- [← Back to Matrix](../matrix.md)
- **Language:** C# | **Framework:** Blazor | **Tier:** Sehr viel
- **Commit:** `92709d06e510173e52aaaf2c35d24d869c0f7848`
- **Sample file:** `src/MudBlazor/TScripts/MudWindow.ts`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.5 ms | 10.6 ms | 10.2 ms | 17.5 ms | [0] |
| **post-tool-use** | 1037.2 ms | 970.5 ms | 918.7 ms | 1003.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1049.7 ms | 981.1 ms | 928.9 ms | 1020.5 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit`
