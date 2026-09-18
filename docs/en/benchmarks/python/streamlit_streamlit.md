# Benchmark & Gap Audit: [streamlit/streamlit](https://github.com/streamlit/streamlit)

- [← Back to Matrix](../matrix.md)
- **Language:** Python | **Framework:** Streamlit | **Tier:** Sehr viel
- **Commit:** `5c6cf98c90edd3b91cf86f1b5ca22f7a13646dfa`
- **Sample file:** `e2e_playwright/__init__.py`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 14.5 ms | 10.5 ms | 10.0 ms | 11.0 ms | [0] |
| **post-tool-use** | 838.5 ms | 616.0 ms | 597.2 ms | 620.9 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 853.0 ms | 626.0 ms | 607.7 ms | 631.9 ms | [0, 2] |

- **Baseline Claude Hook:** 344.7 ms vs. loomux hooks 626.0 ms (Speedup: 0.6x, Status: [0])

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

- **Stacks:** `python, shell, typescript, uv`
- **Lanes:** `ruff check --output-format=concise ., dmypy run -- --no-error-summary --no-pretty, npx eslint --cache ., npx tsc --noEmit, shellcheck **/*.sh`
