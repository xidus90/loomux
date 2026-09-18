# Benchmark & Gap Audit: [dotnet/maui](https://github.com/dotnet/maui)

- [← Back to Matrix](../matrix.md)
- **Language:** C# | **Framework:** .NET MAUI | **Tier:** Sehr viel
- **Commit:** `49b62062536b4a2b957d965f76035eb56806615c`
- **Sample file:** `build.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.0 ms | 9.5 ms | 9.0 ms | 10.1 ms | [0] |
| **post-tool-use** | 73.3 ms | 71.4 ms | 68.9 ms | 73.6 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 86.3 ms | 80.9 ms | 79.0 ms | 82.6 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `dotnet-test` | test | `Microsoft.Maui-dev.sln` | `*none*` | Yes | ⚠️ Missing in Loomux |
| `dotnet-test` | test | `Microsoft.Maui-vscode.sln` | `*none*` | Yes | ⚠️ Missing in Loomux |
| `dotnet-test` | test | `Microsoft.Maui.LegacyControlGallery.sln` | `*none*` | Yes | ⚠️ Missing in Loomux |
| `dotnet-test` | test | `Microsoft.Maui.sln` | `*none*` | Yes | ⚠️ Missing in Loomux |
| `shellcheck` | lint | `build.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `helix.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **33.3 %**
- **Identified Gaps:**
  - ⚠️ dotnet-test deklariert (Microsoft.Maui-dev.sln), aber keine Lane in Loomux vorhanden
  - ⚠️ dotnet-test deklariert (Microsoft.Maui-vscode.sln), aber keine Lane in Loomux vorhanden
  - ⚠️ dotnet-test deklariert (Microsoft.Maui.LegacyControlGallery.sln), aber keine Lane in Loomux vorhanden
  - ⚠️ dotnet-test deklariert (Microsoft.Maui.sln), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
