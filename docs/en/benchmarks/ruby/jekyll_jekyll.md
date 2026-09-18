# Benchmark & Gap Audit: [jekyll/jekyll](https://github.com/jekyll/jekyll)

- [← Back to Matrix](../matrix.md)
- **Language:** Ruby | **Framework:** Jekyll | **Tier:** Sehr viel
- **Commit:** `541d8b2ee75c8907de4744d4b87a7a6f1f997cae`

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
| `rubocop` | lint | `Gemfile` | `*none*` | No | ⚠️ Missing in Loomux |
| `rspec` | test | `Gemfile` | `*none*` | No | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ rubocop deklariert (Gemfile), aber keine Lane in Loomux vorhanden
  - ⚠️ rspec deklariert (Gemfile), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
