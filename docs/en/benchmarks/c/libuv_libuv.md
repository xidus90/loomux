# Benchmark & Gap Audit: [libuv/libuv](https://github.com/libuv/libuv)

- [← Back to Matrix](../matrix.md)
- **Language:** C | **Framework:** libuv | **Tier:** Sehr viel
- **Commit:** `84af0b18c5aee743a9d4182fa9c18a18fd9825e3`
- **Sample file:** `docs/code/cgi/main.c`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 17.0 ms | 12.0 ms | 10.6 ms | 12.5 ms | [0] |
| **post-tool-use** | 44.0 ms | 36.4 ms | 35.8 ms | 38.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 61.0 ms | 48.4 ms | 46.4 ms | 51.0 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `clang-tidy` | lint | `.clang-tidy` | `*none*` | No | ⚠️ Missing in Loomux |
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Yes | ✅ Active |
| `shellcheck` | lint | `autogen.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **66.7 %**
- **Identified Gaps:**
  - ⚠️ clang-tidy deklariert (.clang-tidy), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `clang-tidy, cmake, cpp, python, shell`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, clang-format -i, cmake --build build --parallel, shellcheck **/*.sh`
