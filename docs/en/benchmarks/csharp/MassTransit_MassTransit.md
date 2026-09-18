# Benchmark & Gap Audit: [MassTransit/MassTransit](https://github.com/MassTransit/MassTransit)

- [← Back to Matrix](../matrix.md)
- **Language:** C# | **Framework:** MassTransit | **Tier:** Sehr viel
- **Commit:** `62ab339afa3bac2e9b3fe1769d0d35d7e44778e9`
- **Sample file:** `README.md`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.0 ms | 9.0 ms | 8.5 ms | 9.0 ms | [0] |
| **post-tool-use** | 15.0 ms | 12.5 ms | 12.0 ms | 13.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 28.0 ms | 21.5 ms | 20.5 ms | 22.5 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `dotnet-test` | test | `MassTransit.sln` | `*none*` | Yes | ⚠️ Missing in Loomux |

- **Coverage Rate:** **0.0 %**
- **Identified Gaps:**
  - ⚠️ dotnet-test deklariert (MassTransit.sln), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
