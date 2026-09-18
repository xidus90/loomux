# Benchmark & Gap Audit: [Alamofire/Alamofire](https://github.com/Alamofire/Alamofire)

- [← Back to Matrix](../matrix.md)
- **Language:** Swift | **Framework:** Alamofire | **Tier:** Sehr viel
- **Commit:** `bda9ed57d72988a3a2ada33d824583541f86eac6`
- **Sample file:** `docs/Classes/Adapter.html`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.1 ms | 9.6 ms | 9.1 ms | 10.0 ms | [0] |
| **post-tool-use** | 999.3 ms | 956.5 ms | 932.5 ms | 978.1 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1012.4 ms | 966.5 ms | 941.7 ms | 987.7 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `html`
- **Lanes:** `npx htmlhint "**/*.html"`
