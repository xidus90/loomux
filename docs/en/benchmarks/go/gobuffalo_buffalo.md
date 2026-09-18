# Benchmark & Gap Audit: [gobuffalo/buffalo](https://github.com/gobuffalo/buffalo)

- [← Back to Matrix](../matrix.md)
- **Language:** Go | **Framework:** Buffalo | **Tier:** Sehr viel
- **Commit:** `2aa9868365cdcaa28036efd76e7aac4b7df7bbfc`
- **Sample file:** `app.go`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 10.1 ms | 9.5 ms | 10.5 ms | [0] |
| **post-tool-use** | 391.2 ms | 407.9 ms | 394.5 ms | 426.4 ms | [0] |
| **graph build** | 62.1 ms | 63.5 ms | 62.6 ms | 65.8 ms | [0] |
| **Total** | 463.8 ms | 479.9 ms | 470.4 ms | 500.4 ms | [0] |

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
