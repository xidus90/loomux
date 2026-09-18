# Benchmark & Gap Audit: [spockframework/spock-example](https://github.com/spockframework/spock-example)

- [← Back to Matrix](../matrix.md)
- **Language:** Groovy | **Framework:** Spock | **Tier:** Sehr viel
- **Commit:** `8155d1e2a8f1bdfb1aeacca2e9fe5ded7c992a5e`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | n/a | n/a | n/a | n/a | n/a |
| **post-tool-use** | n/a | n/a | n/a | n/a | n/a |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 0.0 ms | 0.0 ms | 0.0 ms | 0.0 ms | n/a |

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
