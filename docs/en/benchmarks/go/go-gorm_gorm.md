# Benchmark & Gap Audit: [go-gorm/gorm](https://github.com/go-gorm/gorm)

- [← Back to Matrix](../matrix.md)
- **Language:** Go | **Framework:** GORM | **Tier:** Sehr viel
- **Commit:** `b3d3bf219f0283f8e2e985bac509cb643170f729`
- **Sample file:** `association.go`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 10.0 ms | 10.0 ms | 10.1 ms | [0] |
| **post-tool-use** | 326.0 ms | 303.1 ms | 290.4 ms | 308.5 ms | [2] |
| **graph build** | 149.0 ms | 138.1 ms | 137.3 ms | 140.3 ms | [0] |
| **Total** | 486.6 ms | 450.4 ms | 438.5 ms | 458.7 ms | [0, 2] |

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
