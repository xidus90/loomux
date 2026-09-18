# Benchmark & Gap Audit: [digitallyinduced/ihp](https://github.com/digitallyinduced/ihp)

- [← Back to Matrix](../matrix.md)
- **Language:** Haskell | **Framework:** IHP | **Tier:** Sehr viel
- **Commit:** `2bea296d3657693a595e3d5298c51882dfb7f40d`
- **Sample file:** `Guide/layout.html`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 44.2 ms | 24.0 ms | 12.5 ms | 30.5 ms | [0] |
| **post-tool-use** | 2033.0 ms | 1705.5 ms | 1297.2 ms | 1904.4 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 2077.2 ms | 1736.0 ms | 1309.7 ms | 1928.4 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `update-nix-from-cabal.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `css, html, shell`
- **Lanes:** `npx stylelint "**/*.{css,scss}", npx htmlhint "**/*.html", shellcheck **/*.sh`
