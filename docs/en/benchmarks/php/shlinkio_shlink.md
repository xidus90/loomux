# Benchmark & Gap Audit: [shlinkio/shlink](https://github.com/shlinkio/shlink)

- [← Back to Matrix](../matrix.md)
- **Language:** PHP | **Framework:** Laminas | **Tier:** Sehr viel
- **Commit:** `c303aff18f8108ca1e17ec5c697a63afd8efd4fe`
- **Sample file:** `bin/test/run-api-tests.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 9.5 ms | 8.6 ms | 10.0 ms | [0] |
| **post-tool-use** | 77.2 ms | 75.4 ms | 72.1 ms | 77.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 88.2 ms | 84.9 ms | 82.1 ms | 85.7 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `build.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `phpunit` | test | `composer.json` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **50.0 %**
- **Identified Gaps:**
  - ⚠️ phpunit deklariert (composer.json), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `docker, shell`
- **Lanes:** `shellcheck **/*.sh`
