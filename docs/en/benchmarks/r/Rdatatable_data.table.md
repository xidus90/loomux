# Benchmark & Gap Audit: [Rdatatable/data.table](https://github.com/Rdatatable/data.table)

- [← Back to Matrix](../matrix.md)
- **Language:** R | **Framework:** Data.table | **Tier:** Sehr viel
- **Commit:** `a3eeb4c0caec51afd66bdf9bbc16817e12a47a40`
- **Sample file:** `site/_navbar.html`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 16.0 ms | 10.0 ms | 9.0 ms | 11.1 ms | [0] |
| **post-tool-use** | 942.0 ms | 937.7 ms | 931.7 ms | 946.3 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 958.0 ms | 948.8 ms | 941.7 ms | 955.3 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `html`
- **Lanes:** `npx htmlhint "**/*.html"`
