# Benchmark & Gap Audit: [pallets/flask](https://github.com/pallets/flask)

- [← Back to Matrix](../matrix.md)
- **Language:** Python | **Framework:** Flask | **Tier:** Sehr viel
- **Commit:** `d73fa1cdcbd8b1465c151db8924ba58b1dd14e35`
- **Sample file:** `docs/conf.py`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.3 ms | 9.7 ms | 9.6 ms | 10.1 ms | [0] |
| **post-tool-use** | 1982.6 ms | 1929.9 ms | 1884.1 ms | 1980.2 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1992.9 ms | 1939.5 ms | 1894.2 ms | 1989.8 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `ruff` | lint | `pyproject.toml` | `ruff check --output-format=concise .` | Yes | ✅ Active |
| `mypy` | typecheck | `pyproject.toml` | `*none*` | Yes | ⚠️ Missing in Loomux |
| `pytest` | test | `pyproject.toml` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **33.3 %**
- **Identified Gaps:**
  - ⚠️ mypy deklariert (pyproject.toml), aber keine Lane in Loomux vorhanden
  - ⚠️ pytest deklariert (pyproject.toml), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `pyright, python, uv`
- **Lanes:** `ruff check --output-format=concise ., uv run pyright`
