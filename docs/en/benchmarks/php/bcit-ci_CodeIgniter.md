# Benchmark & Gap Audit: [bcit-ci/CodeIgniter](https://github.com/bcit-ci/CodeIgniter)

- [← Back to Matrix](../matrix.md)
- **Language:** PHP | **Framework:** CodeIgniter | **Tier:** Sehr viel
- **Commit:** `3658d731eaabe6117298a105ffb5b9dd59e190ce`
- **Sample file:** `application/cache/index.html`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.0 ms | 9.7 ms | 9.5 ms | 10.0 ms | [0] |
| **post-tool-use** | 1103.4 ms | 896.0 ms | 887.5 ms | 897.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1116.4 ms | 905.5 ms | 897.2 ms | 907.5 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `build-release.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `phpunit` | test | `composer.json` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **50.0 %**
- **Identified Gaps:**
  - ⚠️ phpunit deklariert (composer.json), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `html, shell`
- **Lanes:** `npx htmlhint "**/*.html", shellcheck **/*.sh`
