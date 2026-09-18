# Benchmark & Gap Audit: [fastapi/fastapi](https://github.com/fastapi/fastapi)

- [← Back to Matrix](../matrix.md)
- **Language:** Python | **Framework:** FastAPI | **Tier:** Sehr viel
- **Commit:** `50113da16fec53b66b80d75e80a89296de4fa5a5`
- **Sample file:** `docs_src/additional_responses/__init__.py`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.5 ms | 10.5 ms | 9.5 ms | 14.0 ms | [0] |
| **post-tool-use** | 215.0 ms | 228.6 ms | 207.6 ms | 238.7 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 227.5 ms | 242.6 ms | 217.1 ms | 249.2 ms | [0, 2] |

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

- **Stacks:** `python, shell, uv`
- **Lanes:** `ruff check --output-format=concise ., dmypy run -- --no-error-summary --no-pretty, shellcheck **/*.sh`
