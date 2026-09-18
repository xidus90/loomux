# Benchmark & Gap Audit: [scikit-learn/scikit-learn](https://github.com/scikit-learn/scikit-learn)

- [← Back to Matrix](../matrix.md)
- **Language:** Python | **Framework:** Scikit-learn | **Tier:** Sehr viel
- **Commit:** `7a6ed871e03e95de276b0bc354ffb21e8399de59`
- **Sample file:** `asv_benchmarks/benchmarks/__init__.py`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 9.6 ms | 9.0 ms | 11.0 ms | [0] |
| **post-tool-use** | 275.8 ms | 306.5 ms | 277.5 ms | 397.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 286.8 ms | 315.5 ms | 287.1 ms | 408.0 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `ruff` | lint | `pyproject.toml` | `ruff check --output-format=concise .` | Yes | ✅ Active |
| `mypy` | typecheck | `pyproject.toml` | `mypy --no-error-summary --no-pretty` | Yes | ✅ Active |
| `pytest` | test | `pyproject.toml` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **66.7 %**
- **Identified Gaps:**
  - ⚠️ pytest deklariert (pyproject.toml), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `cpp, meson, python, shell`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, clang-format -i, cmake --build build --parallel, shellcheck **/*.sh`
