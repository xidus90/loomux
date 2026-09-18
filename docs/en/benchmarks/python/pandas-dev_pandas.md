# Benchmark & Gap Audit: [pandas-dev/pandas](https://github.com/pandas-dev/pandas)

- [← Back to Matrix](../matrix.md)
- **Language:** Python | **Framework:** Pandas | **Tier:** Sehr viel
- **Commit:** `cf85d5d203826cab8e164e3ccbe755e49f328d99`
- **Sample file:** `asv_bench/benchmarks/__init__.py`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 9.5 ms | 9.0 ms | 9.5 ms | [0] |
| **post-tool-use** | 100.7 ms | 94.8 ms | 90.6 ms | 99.7 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 112.2 ms | 104.3 ms | 100.1 ms | 108.7 ms | [0] |

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

- **Stacks:** `cpp, meson, pyright, python`
- **Lanes:** `ruff check --output-format=concise ., pyright, clang-format -i, cmake --build build --parallel`
