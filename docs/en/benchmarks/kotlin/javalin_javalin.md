# Benchmark & Gap Audit: [javalin/javalin](https://github.com/javalin/javalin)

- [← Back to Matrix](../matrix.md)
- **Language:** Kotlin | **Framework:** Javalin | **Tier:** Sehr viel
- **Commit:** `1ff18e9e03e69cc3759abc57f1f12c32285eded3`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 8.5 ms | 8.0 ms | 11.0 ms | [0] |
| **post-tool-use** | 19.5 ms | 19.0 ms | 18.5 ms | 19.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 30.0 ms | 27.5 ms | 27.5 ms | 29.5 ms | [0] |

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
