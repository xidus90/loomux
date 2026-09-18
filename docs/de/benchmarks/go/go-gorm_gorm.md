# Benchmark & Lücken-Audit: [go-gorm/gorm](https://github.com/go-gorm/gorm)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Go | **Framework:** GORM | **Tier:** Sehr viel
- **Commit:** `b3d3bf219f0283f8e2e985bac509cb643170f729`
- **Beispieldatei:** `association.go`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 10.0 ms | 10.0 ms | 10.1 ms | [0] |
| **post-tool-use** | 326.0 ms | 303.1 ms | 290.4 ms | 308.5 ms | [2] |
| **graph build** | 149.0 ms | 138.1 ms | 137.3 ms | 140.3 ms | [0] |
| **Gesamt** | 486.6 ms | 450.4 ms | 438.5 ms | 458.7 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `golangci-lint` | lint | `.golangci.yml` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `go-vet` | lint | `go.mod` | `go vet ./...` | Ja | ✅ Aktiv |
| `go-test` | test | `go.mod` | `*keine*` | Ja | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **33.3 %**
- **Identifizierte Lücken:**
  - ⚠️ golangci-lint deklariert (.golangci.yml), aber keine Lane in Loomux vorhanden
  - ⚠️ go-test deklariert (go.mod), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `go, golangci-lint, shell`
- **Lanes:** `shellcheck **/*.sh, go vet ./...`
