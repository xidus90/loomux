# Benchmark & Gap Audit: [phoenixframework/phoenix](https://github.com/phoenixframework/phoenix)

- [← Back to Matrix](../matrix.md)
- **Language:** Elixir | **Framework:** Phoenix | **Tier:** Sehr viel
- **Commit:** `da518c2f871c3b9049929160c91de87a323b61be`
- **Sample file:** `integration_test/docker.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.4 ms | 9.1 ms | 9.0 ms | 9.5 ms | [0] |
| **post-tool-use** | 80.3 ms | 76.1 ms | 75.0 ms | 77.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 90.7 ms | 85.1 ms | 84.5 ms | 86.1 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `jest` | test | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ eslint deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ jest deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `docker, shell`
- **Lanes:** `shellcheck **/*.sh`
