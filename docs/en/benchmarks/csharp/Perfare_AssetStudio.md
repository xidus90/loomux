# Benchmark & Gap Audit: [Perfare/AssetStudio](https://github.com/Perfare/AssetStudio)

- [← Back to Matrix](../matrix.md)
- **Language:** C# | **Framework:** Unity | **Tier:** Sehr viel
- **Commit:** `d158e864b556b5970709c2a52e47944d53aa98a2`
- **Sample file:** `AssetStudioFBXNative/api.cpp`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.0 ms | 10.0 ms | 9.0 ms | 10.5 ms | [0] |
| **post-tool-use** | 36.5 ms | 33.6 ms | 31.5 ms | 34.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 49.5 ms | 43.0 ms | 41.5 ms | 44.1 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `dotnet-test` | test | `AssetStudio.sln` | `*none*` | Yes | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ dotnet-test deklariert (AssetStudio.sln), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `cpp`
- **Lanes:** `clang-format -i, cmake --build build --parallel`
