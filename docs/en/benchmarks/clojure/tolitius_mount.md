# Benchmark & Gap Audit: [tolitius/mount](https://github.com/tolitius/mount)

- [← Back to Matrix](../matrix.md)
- **Language:** Clojure | **Framework:** Mount | **Tier:** Sehr viel
- **Commit:** `ae94e7d85cbf17b17789d2e42df2b39eb2bfc302`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 9.0 ms | 8.6 ms | 10.1 ms | [0] |
| **post-tool-use** | 18.4 ms | 16.0 ms | 15.9 ms | 16.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 29.9 ms | 24.9 ms | 24.6 ms | 26.1 ms | [0] |

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
