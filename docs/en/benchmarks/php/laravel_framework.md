# Benchmark & Gap Audit: [laravel/framework](https://github.com/laravel/framework)

- [← Back to Matrix](../matrix.md)
- **Language:** PHP | **Framework:** Laravel | **Tier:** Sehr viel
- **Commit:** `1d9727160ad440d1ccf2fdcbb7e4f4f36efae212`
- **Sample file:** `bin/release.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.0 ms | 9.5 ms | 9.1 ms | 12.5 ms | [0] |
| **post-tool-use** | 83.3 ms | 74.5 ms | 73.5 ms | 80.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 96.3 ms | 84.0 ms | 82.6 ms | 92.5 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `phpunit` | test | `composer.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `phpstan` | typecheck | `composer.json` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ phpunit deklariert (composer.json), aber keine Lane in Loomux vorhanden
  - ⚠️ phpstan deklariert (composer.json), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `docker, shell`
- **Lanes:** `shellcheck **/*.sh`
