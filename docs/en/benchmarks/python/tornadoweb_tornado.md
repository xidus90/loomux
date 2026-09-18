# Benchmark & Gap Audit: [tornadoweb/tornado](https://github.com/tornadoweb/tornado)

- [← Back to Matrix](../matrix.md)
- **Language:** Python | **Framework:** Tornado | **Tier:** Sehr viel
- **Commit:** `85b6917d05a84a6d26b7488b2b56d0b52c107a0f`
- **Sample file:** `demos/blog/blog.py`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.0 ms | 10.8 ms | 10.1 ms | 16.0 ms | [0] |
| **post-tool-use** | 265.3 ms | 238.4 ms | 224.2 ms | 256.7 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 278.3 ms | 248.5 ms | 235.0 ms | 272.8 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `runtests.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `python, shell`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, shellcheck **/*.sh`
