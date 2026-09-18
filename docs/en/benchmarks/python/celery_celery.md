# Benchmark & Gap Audit: [celery/celery](https://github.com/celery/celery)

- [← Back to Matrix](../matrix.md)
- **Language:** Python | **Framework:** Celery | **Tier:** Sehr viel
- **Commit:** `208a803655672d63cf2a257d30f39c672a5b4bcf`
- **Sample file:** `celery/__init__.py`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 10.1 ms | 9.5 ms | 10.5 ms | [0] |
| **post-tool-use** | 289.1 ms | 285.2 ms | 276.7 ms | 290.5 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 300.1 ms | 295.3 ms | 287.2 ms | 300.0 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `mypy` | typecheck | `pyproject.toml` | `mypy --no-error-summary --no-pretty` | Yes | ✅ Active |
| `pytest` | test | `pyproject.toml` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **50.0 %**
- **Identified Gaps:**
  - ⚠️ pytest deklariert (pyproject.toml), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `docker, html, python`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, npx htmlhint "**/*.html"`
