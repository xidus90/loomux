# Benchmark & Gap Audit: [playframework/play1](https://github.com/playframework/play1)

- [← Back to Matrix](../matrix.md)
- **Language:** Java | **Framework:** Play Framework | **Tier:** Sehr viel
- **Commit:** `2dbefe699d493c5c08830b5db4259763f3faee61`
- **Sample file:** `framework/pym/play/__init__.py`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 10.5 ms | 9.5 ms | 11.0 ms | [0] |
| **post-tool-use** | 257.9 ms | 225.3 ms | 219.2 ms | 247.7 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 268.9 ms | 234.8 ms | 229.7 ms | 258.7 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `python`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty`
