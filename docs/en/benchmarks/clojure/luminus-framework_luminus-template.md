# Benchmark & Gap Audit: [luminus-framework/luminus-template](https://github.com/luminus-framework/luminus-template)

- [← Back to Matrix](../matrix.md)
- **Language:** Clojure | **Framework:** Luminus | **Tier:** Sehr viel
- **Commit:** `eb40d439a3189082a8610347dfe2ded1b21b5062`
- **Sample file:** `publish.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.2 ms | 9.2 ms | 10.5 ms | [0] |
| **post-tool-use** | 74.9 ms | 68.2 ms | 67.6 ms | 76.3 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 86.9 ms | 77.7 ms | 77.3 ms | 86.8 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `publish.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
