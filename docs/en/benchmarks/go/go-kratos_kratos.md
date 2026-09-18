# Benchmark & Gap Audit: [go-kratos/kratos](https://github.com/go-kratos/kratos)

- [← Back to Matrix](../matrix.md)
- **Language:** Go | **Framework:** Kratos | **Tier:** Sehr viel
- **Commit:** `668db92c2c001e9552594ba5a8aede8456af6d7e`
- **Sample file:** `app.go`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.5 ms | 10.0 ms | 11.0 ms | [0] |
| **post-tool-use** | 570.6 ms | 550.7 ms | 550.7 ms | 555.0 ms | [2] |
| **graph build** | 176.2 ms | 181.8 ms | 173.8 ms | 182.7 ms | [0] |
| **Total** | 758.9 ms | 743.0 ms | 734.5 ms | 748.7 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `golangci-lint` | lint | `.golangci.yml` | `*none*` | No | ⚠️ Missing in Loomux |
| `go-vet` | lint | `go.mod` | `go vet ./...` | Yes | ✅ Active |
| `go-test` | test | `go.mod` | `*none*` | Yes | ⚠️ Missing in Loomux |

- **Coverage Rate:** **33.3 %**
- **Identified Gaps:**
  - ⚠️ golangci-lint deklariert (.golangci.yml), aber keine Lane in Loomux vorhanden
  - ⚠️ go-test deklariert (go.mod), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `go, golangci-lint, shell`
- **Lanes:** `shellcheck **/*.sh, go vet ./...`
