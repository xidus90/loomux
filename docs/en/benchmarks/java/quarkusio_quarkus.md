# Benchmark & Gap Audit: [quarkusio/quarkus](https://github.com/quarkusio/quarkus)

- [← Back to Matrix](../matrix.md)
- **Language:** Java | **Framework:** Quarkus | **Tier:** Sehr viel
- **Commit:** `83809ecb7a3ba8389c207cb1c453aeb09041512c`
- **Sample file:** `coverage-report/prepare.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 33.0 ms | 19.2 ms | 16.0 ms | 24.6 ms | [0] |
| **post-tool-use** | 147.1 ms | 155.8 ms | 122.6 ms | 191.2 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 180.1 ms | 180.4 ms | 138.6 ms | 210.4 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `mvn-test` | test | `pom.xml` | `*none*` | No | ⚠️ Missing in Loomux |
| `shellcheck` | lint | `update-extension-dependencies.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `update-version.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **66.7 %**
- **Identified Gaps:**
  - ⚠️ mvn-test deklariert (pom.xml), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
