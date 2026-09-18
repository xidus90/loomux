# Benchmark & Gap Audit: [mitchellh/libxev](https://github.com/mitchellh/libxev)

- [← Back to Matrix](../matrix.md)
- **Language:** Zig | **Framework:** Libxev | **Tier:** Sehr viel
- **Commit:** `9ce8e8e6ff89e583258a7f8e7adeeeaeae8611bf`
- **Sample file:** `website/next.config.js`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 9.0 ms | 10.0 ms | 9.5 ms | 10.1 ms | [0] |
| **post-tool-use** | 952.4 ms | 925.5 ms | 922.6 ms | 940.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 961.5 ms | 935.5 ms | 932.1 ms | 950.1 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit`
