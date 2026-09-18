# Benchmark & Gap Audit: [ChartsOrg/Charts](https://github.com/ChartsOrg/Charts)

- [← Back to Matrix](../matrix.md)
- **Language:** Swift | **Framework:** Charts | **Tier:** Sehr viel
- **Commit:** `1bad5469f57628782110b05996d4fa00473abf06`
- **Sample file:** `carthage.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 9.0 ms | 8.3 ms | 8.0 ms | 9.0 ms | [0] |
| **post-tool-use** | 65.0 ms | 64.5 ms | 63.0 ms | 66.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 74.0 ms | 72.8 ms | 72.0 ms | 74.0 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `carthage.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
