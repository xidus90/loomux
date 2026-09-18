# Benchmark & Gap Audit: [vaadin/framework](https://github.com/vaadin/framework)

- [← Back to Matrix](../matrix.md)
- **Language:** Java | **Framework:** Vaadin | **Tier:** Sehr viel
- **Commit:** `f7d61fc6d0de0f8e37c97eab85b7e554b24a7114`
- **Sample file:** `scripts/cleanWhitespace.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 9.0 ms | 9.0 ms | 9.5 ms | [0] |
| **post-tool-use** | 98.4 ms | 104.4 ms | 98.1 ms | 108.1 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 110.4 ms | 113.9 ms | 107.1 ms | 117.1 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `mvn-test` | test | `pom.xml` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ mvn-test deklariert (pom.xml), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
