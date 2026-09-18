# Benchmark & Gap Audit: [GPars/GPars](https://github.com/GPars/GPars)

- [← Back to Matrix](../matrix.md)
- **Language:** Groovy | **Framework:** GPars | **Tier:** Sehr viel
- **Commit:** `7cddf7cf2fec1fd66ef800edccfc03315d078a2b`
- **Sample file:** `overview.html`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.0 ms | 9.5 ms | 9.5 ms | 10.5 ms | [0] |
| **post-tool-use** | 938.4 ms | 924.8 ms | 896.8 ms | 935.2 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 948.4 ms | 934.3 ms | 906.3 ms | 945.7 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `html`
- **Lanes:** `npx htmlhint "**/*.html"`
