# Benchmark & Gap Audit: [Difegue/LANraragi](https://github.com/Difegue/LANraragi)

- [← Back to Matrix](../matrix.md)
- **Language:** Perl | **Framework:** Mojolicious | **Tier:** Sehr viel
- **Commit:** `db3106900d90e07f5723da9ed933cc0399a5670d`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 9.5 ms | 8.5 ms | 8.0 ms | 8.7 ms | [0] |
| **post-tool-use** | 20.0 ms | 18.8 ms | 18.0 ms | 20.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 29.5 ms | 26.8 ms | 26.7 ms | 29.0 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ eslint deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
