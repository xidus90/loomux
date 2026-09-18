# Benchmark & Gap Audit: [ghostfolio/ghostfolio](https://github.com/ghostfolio/ghostfolio)

- [← Back to Matrix](../matrix.md)
- **Language:** TypeScript | **Framework:** NestJS | **Tier:** Sehr viel
- **Commit:** `f0e2d6c33644cf928bd1317652bb52265770362f`
- **Sample file:** `docker/entrypoint.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 9.0 ms | 8.5 ms | 10.0 ms | [0] |
| **post-tool-use** | 73.6 ms | 67.5 ms | 65.7 ms | 67.9 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 85.1 ms | 76.9 ms | 74.2 ms | 77.5 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `prettier` | format | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `jest` | test | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ eslint deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ prettier deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ jest deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `docker, shell`
- **Lanes:** `shellcheck **/*.sh`
