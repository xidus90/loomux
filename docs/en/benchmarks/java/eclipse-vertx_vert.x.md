# Benchmark & Gap Audit: [eclipse-vertx/vert.x](https://github.com/eclipse-vertx/vert.x)

- [← Back to Matrix](../matrix.md)
- **Language:** Java | **Framework:** Vert.x | **Tier:** Sehr viel
- **Commit:** `132c5bc0c3f77f99bcec88ecae0d512d0b0206bd`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.5 ms | 9.0 ms | 8.5 ms | 9.4 ms | [0] |
| **post-tool-use** | 15.5 ms | 14.0 ms | 13.0 ms | 14.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 28.0 ms | 22.5 ms | 22.0 ms | 23.4 ms | [0] |

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
