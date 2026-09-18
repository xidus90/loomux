# Benchmark & Gap Audit: [apache/spark](https://github.com/apache/spark)

- [← Back to Matrix](../matrix.md)
- **Language:** Scala | **Framework:** Spark | **Tier:** Sehr viel
- **Commit:** `6aa2688a21e3fbc9db9abc2a66e47a951fa94702`
- **Sample file:** `connector/kinesis-asl/src/main/python/examples/streaming/kinesis_wordcount_asl.py`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 11.1 ms | 10.5 ms | 11.4 ms | [0] |
| **post-tool-use** | 345.2 ms | 354.5 ms | 352.4 ms | 354.6 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 357.2 ms | 365.1 ms | 363.8 ms | 365.6 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `mvn-test` | test | `pom.xml` | `*none*` | No | ⚠️ Missing in Loomux |
| `ruff` | lint | `pyproject.toml` | `ruff check --output-format=concise .` | Yes | ✅ Active |
| `mypy` | typecheck | `pyproject.toml` | `mypy --no-error-summary --no-pretty` | Yes | ✅ Active |
| `pytest` | test | `pyproject.toml` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **50.0 %**
- **Identified Gaps:**
  - ⚠️ mvn-test deklariert (pom.xml), aber keine Lane in Loomux vorhanden
  - ⚠️ pytest deklariert (pyproject.toml), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `html, python, shell`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, npx htmlhint "**/*.html", shellcheck **/*.sh`
