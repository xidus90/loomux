# Benchmark & Gap Audit: [microsoft/AirSim](https://github.com/microsoft/AirSim)

- [← Back to Matrix](../matrix.md)
- **Language:** C++ | **Framework:** Unreal Engine | **Tier:** Sehr viel
- **Commit:** `1ca93f6f77e4e8a39b2b241c1fe2764da4d7dd41`
- **Sample file:** `AirLib/include/api/ApiProvider.hpp`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.4 ms | 9.0 ms | 8.1 ms | 9.1 ms | [0] |
| **post-tool-use** | 53.4 ms | 48.3 ms | 47.5 ms | 49.7 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 65.8 ms | 56.5 ms | 56.4 ms | 58.8 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `clang-format` | format | `.clang-format` | `clang-format -i` | No | ❌ Missing from PATH |
| `dotnet-test` | test | `AirSim.sln` | `*none*` | Yes | ⚠️ Missing in Loomux |
| `shellcheck` | lint | `build.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `build_docs.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `clean.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `clean_rebuild.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `install_run_all.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `install_unreal.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `setup.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **88.9 %**
- **Identified Gaps:**
  - ⚠️ Lane für clang-format vorhanden (clang-format -i), aber Werkzeug nicht im PATH
  - ⚠️ dotnet-test deklariert (AirSim.sln), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `clang-format, cmake, cpp, python, shell`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, clang-format -i, cmake --build build --parallel, shellcheck **/*.sh`
