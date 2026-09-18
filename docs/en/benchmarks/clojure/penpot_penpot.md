# Benchmark & Gap Audit: [penpot/penpot](https://github.com/penpot/penpot)

- [← Back to Matrix](../matrix.md)
- **Language:** Clojure | **Framework:** ClojureScript | **Tier:** Sehr viel
- **Commit:** `b402637fe4c35a31eac4007356d3a750ba6187c8`
- **Sample file:** `backend/resources/app/assets/swagger-ui-4.18.3.js`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 9.1 ms | 9.0 ms | 9.5 ms | [0] |
| **post-tool-use** | 966.3 ms | 965.3 ms | 933.4 ms | 972.6 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 976.8 ms | 974.8 ms | 942.4 ms | 981.7 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `manage.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `html, pnpm, rust, shell, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit, npx htmlhint "**/*.html", shellcheck **/*.sh, cargo clippy -- -D warnings, cargo fmt --check`
