# Benchmark & Gap Audit: [slimphp/Slim](https://github.com/slimphp/Slim)

- [← Back to Matrix](../matrix.md)
- **Language:** PHP | **Framework:** Slim | **Tier:** Sehr viel
- **Commit:** `3675bf6baac66b07032575b7bef4200b60b7974b`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 11.0 ms | 8.5 ms | 11.5 ms | [0] |
| **post-tool-use** | 12.5 ms | 13.5 ms | 12.0 ms | 14.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 23.0 ms | 23.5 ms | 22.5 ms | 24.5 ms | [0] |

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
