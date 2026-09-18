# Benchmark & Gap Audit: [pytorch/pytorch](https://github.com/pytorch/pytorch)

- [← Back to Matrix](../matrix.md)
- **Language:** Python | **Framework:** PyTorch | **Tier:** Sehr viel
- **Commit:** `5c6918c45adefb8eaf40e7f665ccd00f9879fbef`
- **Sample file:** `android/pytorch_android/generate_test_asset.cpp`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.4 ms | 8.5 ms | 8.2 ms | 9.0 ms | [0] |
| **post-tool-use** | 53.7 ms | 46.6 ms | 45.1 ms | 47.1 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 65.1 ms | 54.7 ms | 53.6 ms | 56.0 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `clang-format` | format | `.clang-format` | `clang-format -i` | No | ❌ Missing from PATH |
| `clang-tidy` | lint | `.clang-tidy` | `*none*` | No | ⚠️ Missing in Loomux |
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Yes | ✅ Active |
| `shellcheck` | lint | `codex_setup.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `ruff` | lint | `pyproject.toml` | `ruff check --output-format=concise .` | Yes | ✅ Active |

- **Coverage Rate:** **80.0 %**
- **Identified Gaps:**
  - ⚠️ Lane für clang-format vorhanden (clang-format -i), aber Werkzeug nicht im PATH
  - ⚠️ clang-tidy deklariert (.clang-tidy), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `clang-format, clang-tidy, cmake, cpp, python, shell`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, clang-format -i, cmake --build build --parallel, shellcheck **/*.sh`
