# Benchmark & Gap Audit: [revel/revel](https://github.com/revel/revel)

- [← Back to Matrix](../matrix.md)
- **Language:** Go | **Framework:** Revel | **Tier:** Sehr viel
- **Commit:** `b053175279547526fe914932716bc313558df4bd`
- **Sample file:** `before_after_filter.go`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.0 ms | 9.7 ms | 9.2 ms | 13.0 ms | [0] |
| **post-tool-use** | 354.0 ms | 353.2 ms | 341.4 ms | 373.0 ms | [2] |
| **graph build** | 84.9 ms | 77.0 ms | 74.9 ms | 78.5 ms | [0] |
| **Total** | 448.9 ms | 437.8 ms | 431.4 ms | 460.7 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `go-vet` | lint | `go.mod` | `go vet ./...` | Yes | ✅ Active |
| `go-test` | test | `go.mod` | `*none*` | Yes | ⚠️ Missing in Loomux |

- **Coverage Rate:** **50.0 %**
- **Identified Gaps:**
  - ⚠️ go-test deklariert (go.mod), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `go`
- **Lanes:** `go vet ./...`
