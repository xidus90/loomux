# Benchmark & Gap Audit: [jasontaylordev/NorthwindTraders](https://github.com/jasontaylordev/NorthwindTraders)

- [← Back to Matrix](../matrix.md)
- **Language:** C# | **Framework:** Entity Framework Core | **Tier:** Sehr viel
- **Commit:** `647fafc87c4c34bcb9fc67a08db422a3018a0cab`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 8.5 ms | 8.5 ms | 9.5 ms | [0] |
| **post-tool-use** | 14.0 ms | 12.5 ms | 12.0 ms | 13.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 25.0 ms | 22.0 ms | 20.5 ms | 22.0 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `dotnet-test` | test | `Northwind.sln` | `*none*` | Yes | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ dotnet-test deklariert (Northwind.sln), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
