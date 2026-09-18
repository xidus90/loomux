# Benchmark & Gap Audit: [AvaloniaUI/Avalonia](https://github.com/AvaloniaUI/Avalonia)

- [← Back to Matrix](../matrix.md)
- **Language:** C# | **Framework:** Avalonia | **Tier:** Sehr viel
- **Commit:** `334536066a6cf4df5dcfb1da22cb9ed80dd66e8e`
- **Sample file:** `build-native.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 9.5 ms | 9.5 ms | 10.0 ms | [0] |
| **post-tool-use** | 83.5 ms | 78.0 ms | 77.5 ms | 81.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 95.5 ms | 88.0 ms | 87.0 ms | 90.5 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `build-native.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `build.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
