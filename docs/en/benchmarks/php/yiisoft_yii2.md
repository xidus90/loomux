# Benchmark & Gap Audit: [yiisoft/yii2](https://github.com/yiisoft/yii2)

- [← Back to Matrix](../matrix.md)
- **Language:** PHP | **Framework:** Yii2 | **Tier:** Sehr viel
- **Commit:** `8ee8986b32e574a2d91a1f64f89185b4339d7593`
- **Sample file:** `tests/test-local.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.6 ms | 9.0 ms | 8.5 ms | 9.5 ms | [0] |
| **post-tool-use** | 78.4 ms | 73.1 ms | 70.0 ms | 75.1 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 91.0 ms | 82.1 ms | 78.5 ms | 84.6 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `phpunit` | test | `composer.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `phpstan` | typecheck | `composer.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `eslint` | lint | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ phpunit deklariert (composer.json), aber keine Lane in Loomux vorhanden
  - ⚠️ phpstan deklariert (composer.json), aber keine Lane in Loomux vorhanden
  - ⚠️ eslint deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `docker, shell`
- **Lanes:** `shellcheck **/*.sh`
