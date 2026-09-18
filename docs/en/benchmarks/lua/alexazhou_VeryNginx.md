# Benchmark & Gap Audit: [alexazhou/VeryNginx](https://github.com/alexazhou/VeryNginx)

- [← Back to Matrix](../matrix.md)
- **Language:** Lua | **Framework:** OpenResty | **Tier:** Sehr viel
- **Commit:** `b66ef2ee6a108279653316323bd8ac026dd7ccd8`
- **Sample file:** `install.py`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 11.0 ms | 10.3 ms | 11.1 ms | [0] |
| **post-tool-use** | 216.0 ms | 210.7 ms | 209.7 ms | 225.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 227.5 ms | 221.7 ms | 220.7 ms | 235.3 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `python`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty`
