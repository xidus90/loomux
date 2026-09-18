# Benchmark & Gap Audit: [GenieFramework/Genie.jl](https://github.com/GenieFramework/Genie.jl)

- [← Back to Matrix](../matrix.md)
- **Language:** Julia | **Framework:** Genie | **Tier:** Sehr viel
- **Commit:** `61ebfc822abd833593fba510a15de236398ec5c5`
- **Sample file:** `CHANGELOG.html`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 18.7 ms | 9.5 ms | 9.5 ms | 9.5 ms | [0] |
| **post-tool-use** | 989.2 ms | 963.6 ms | 960.4 ms | 1008.2 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1007.9 ms | 973.1 ms | 969.9 ms | 1017.7 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `html`
- **Lanes:** `npx htmlhint "**/*.html"`
