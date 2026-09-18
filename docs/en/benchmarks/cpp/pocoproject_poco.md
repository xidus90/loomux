# Benchmark & Gap Audit: [pocoproject/poco](https://github.com/pocoproject/poco)

- [← Back to Matrix](../matrix.md)
- **Language:** C++ | **Framework:** Poco | **Tier:** Sehr viel
- **Commit:** `3f0c291d56751b12ba95ca545706061bd4d35aeb`
- **Sample file:** `ActiveRecord/Compiler/src/CodeGenerator.cpp`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 14.5 ms | 12.5 ms | 11.5 ms | 14.0 ms | [0] |
| **post-tool-use** | 93.5 ms | 90.6 ms | 90.0 ms | 92.2 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 108.0 ms | 103.1 ms | 101.5 ms | 106.2 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Yes | ✅ Active |
| `shellcheck` | lint | `build_cmake.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `build_make.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `env.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `gh-cli-for-release-notes.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `runLibTests.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `runVSCode.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `cmake, cpp, html, shell`
- **Lanes:** `clang-format -i, cmake --build build --parallel, npx htmlhint "**/*.html", shellcheck **/*.sh`
