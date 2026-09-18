# Benchmark & Gap Audit: [symfony/symfony](https://github.com/symfony/symfony)

- [← Back to Matrix](../matrix.md)
- **Language:** PHP | **Framework:** Symfony | **Tier:** Sehr viel
- **Commit:** `6df7f701eb37d5257124060536c3169ffd56630e`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 11.5 ms | 10.1 ms | 11.5 ms | [0] |
| **post-tool-use** | 16.0 ms | 12.4 ms | 12.0 ms | 15.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 27.5 ms | 23.9 ms | 22.1 ms | 26.5 ms | [0] |

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
