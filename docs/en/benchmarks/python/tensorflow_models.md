# Benchmark & Gap Audit: [tensorflow/models](https://github.com/tensorflow/models)

- [← Back to Matrix](../matrix.md)
- **Language:** Python | **Framework:** TensorFlow | **Tier:** Sehr viel
- **Commit:** `8b12ae202a3ccf8f965c730a4e7617204e32000b`
- **Sample file:** `official/__init__.py`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 10.5 ms | 10.0 ms | 15.6 ms | [0] |
| **post-tool-use** | 794.5 ms | 765.7 ms | 751.5 ms | 850.5 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 805.0 ms | 781.3 ms | 761.5 ms | 861.0 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `python`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty`
