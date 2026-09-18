# Benchmark & Lücken-Audit: [go-kratos/kratos](https://github.com/go-kratos/kratos)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Go | **Framework:** Kratos | **Tier:** Sehr viel
- **Commit:** `668db92c2c001e9552594ba5a8aede8456af6d7e`
- **Beispieldatei:** `app.go`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.5 ms | 10.0 ms | 11.0 ms | [0] |
| **post-tool-use** | 570.6 ms | 550.7 ms | 550.7 ms | 555.0 ms | [2] |
| **graph build** | 176.2 ms | 181.8 ms | 173.8 ms | 182.7 ms | [0] |
| **Gesamt** | 758.9 ms | 743.0 ms | 734.5 ms | 748.7 ms | [0, 2] |

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
