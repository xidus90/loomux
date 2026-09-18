# Benchmark & Gap Audit: [ruby-grape/grape](https://github.com/ruby-grape/grape)

- [← Back to Matrix](../matrix.md)
- **Language:** Ruby | **Framework:** Grape | **Tier:** Sehr viel
- **Commit:** `f6fff5a0a1dd125caa055ab85bed72e5ab6b794b`
- **Sample file:** `docker/entrypoint.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.0 ms | 9.5 ms | 9.0 ms | 10.5 ms | [0] |
| **post-tool-use** | 74.2 ms | 64.6 ms | 64.5 ms | 68.8 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 84.2 ms | 74.0 ms | 73.6 ms | 79.3 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `rubocop` | lint | `Gemfile` | `*none*` | No | ⚠️ Missing in Loomux |
| `rspec` | test | `Gemfile` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ rubocop deklariert (Gemfile), aber keine Lane in Loomux vorhanden
  - ⚠️ rspec deklariert (Gemfile), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `docker, shell`
- **Lanes:** `shellcheck **/*.sh`
