# Benchmark & Gap Audit: [snapframework/snap](https://github.com/snapframework/snap)

- [← Back to Matrix](../matrix.md)
- **Language:** Haskell | **Framework:** Snap | **Tier:** Sehr viel
- **Commit:** `93ac0673b6d20ae4c7778e2323188f1a93cfc52c`
- **Sample file:** `extra/haddock.css`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 10.0 ms | 9.5 ms | 10.7 ms | [0] |
| **post-tool-use** | 1295.8 ms | 1156.3 ms | 1080.8 ms | 1483.4 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 1306.3 ms | 1165.8 ms | 1091.5 ms | 1493.4 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `haddock.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `pull.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `pullLatestMaster.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `runTestsAndCoverage.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `css, shell`
- **Lanes:** `npx stylelint "**/*.{css,scss}", shellcheck **/*.sh`
