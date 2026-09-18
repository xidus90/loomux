# Benchmark & Gap Audit: [dotnet/aspnetcore](https://github.com/dotnet/aspnetcore)

- [← Back to Matrix](../matrix.md)
- **Language:** C# | **Framework:** ASP.NET Core | **Tier:** Sehr viel
- **Commit:** `13bb0ebb8a7e1c2cabf08239b4e4c813d49b0de6`
- **Sample file:** `activate.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.0 ms | 9.5 ms | 10.2 ms | [0] |
| **post-tool-use** | 82.7 ms | 89.4 ms | 83.9 ms | 91.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 94.7 ms | 98.9 ms | 94.0 ms | 101.0 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `activate.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `clean.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `restore.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `startvscode.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
