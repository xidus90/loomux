# Benchmark & Gap Audit: [go-chi/chi](https://github.com/go-chi/chi)

- [← Back to Matrix](../matrix.md)
- **Language:** Go | **Framework:** Chi | **Tier:** Sehr viel
- **Commit:** `3d1777a1ef8881f7d1da0b02c76ca8f0a29cd2bc`
- **Sample file:** `_examples/custom-handler/main.go`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 10.5 ms | 10.0 ms | 11.0 ms | [0] |
| **post-tool-use** | 259.6 ms | 263.8 ms | 257.6 ms | 268.4 ms | [0] |
| **graph build** | 58.1 ms | 57.0 ms | 56.9 ms | 57.2 ms | [0] |
| **Total** | 328.7 ms | 330.6 ms | 325.2 ms | 336.5 ms | [0] |

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
