# Benchmark & Gap Audit: [haskell-servant/servant](https://github.com/haskell-servant/servant)

- [← Back to Matrix](../matrix.md)
- **Language:** Haskell | **Framework:** Servant | **Tier:** Sehr viel
- **Commit:** `5f059a40ab0062dd8d179d7de91d63e7218b0e80`
- **Sample file:** `doc/conf.py`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.6 ms | 10.0 ms | 9.6 ms | 10.5 ms | [0] |
| **post-tool-use** | 242.7 ms | 236.1 ms | 229.4 ms | 287.6 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 253.4 ms | 246.6 ms | 239.4 ms | 297.2 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `streaming-benchmark.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `python, shell`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, shellcheck **/*.sh`
