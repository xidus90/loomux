# Benchmark & Gap Audit: [cakephp/cakephp](https://github.com/cakephp/cakephp)

- [← Back to Matrix](../matrix.md)
- **Language:** PHP | **Framework:** CakePHP | **Tier:** Sehr viel
- **Commit:** `356619db99a13278e32e7065d43cce950d6ed3c1`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.0 ms | 8.7 ms | 8.7 ms | 9.0 ms | [0] |
| **post-tool-use** | 18.3 ms | 14.8 ms | 14.0 ms | 15.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 31.3 ms | 23.8 ms | 22.7 ms | 24.2 ms | [0] |

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

- **Stacks:** ``
- **Lanes:** ``
