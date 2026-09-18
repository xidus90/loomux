# Benchmark & Gap Audit: [openssl/openssl](https://github.com/openssl/openssl)

- [← Back to Matrix](../matrix.md)
- **Language:** C | **Framework:** OpenSSL | **Tier:** Sehr viel
- **Commit:** `859aea422b5be17ee9fc0f7678e9de302eb67b72`
- **Sample file:** `apps/asn1parse.c`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 14.4 ms | 12.2 ms | 10.0 ms | 12.7 ms | [0] |
| **post-tool-use** | 71.2 ms | 69.9 ms | 63.5 ms | 72.8 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 85.6 ms | 82.1 ms | 76.2 ms | 82.8 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `clang-format` | format | `.clang-format` | `clang-format -i` | No | ❌ Missing from PATH |

- **Coverage Rate:** **100.0 %**
- **Identified Gaps:**
  - ⚠️ Lane für clang-format vorhanden (clang-format -i), aber Werkzeug nicht im PATH

## 3. Detected Stacks & Lanes

- **Stacks:** `clang-format, cpp, shell`
- **Lanes:** `clang-format -i, cmake --build build --parallel, shellcheck **/*.sh`
