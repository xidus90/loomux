# Benchmark & Gap Audit: [apache/struts](https://github.com/apache/struts)

- [← Back to Matrix](../matrix.md)
- **Language:** Java | **Framework:** Struts | **Tier:** Sehr viel
- **Commit:** `85f30e35c5c5fc7c46db04af961e07a5826da32e`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 9.0 ms | 8.5 ms | 9.0 ms | [0] |
| **post-tool-use** | 27.0 ms | 22.5 ms | 22.0 ms | 22.9 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 39.0 ms | 31.4 ms | 31.0 ms | 31.5 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `mvn-test` | test | `pom.xml` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ mvn-test deklariert (pom.xml), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
