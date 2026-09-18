# Benchmark & Lücken-Audit: [google/go-cloud](https://github.com/google/go-cloud)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Go | **Framework:** Go-Kit | **Tier:** Sehr viel
- **Commit:** `25fdfb8360def3fa7946a89fb41490a146c4e733`
- **Beispieldatei:** `aws/aws.go`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 12.0 ms | 11.0 ms | 13.5 ms | [0] |
| **post-tool-use** | 1087.9 ms | 1156.2 ms | 1096.2 ms | 1225.7 ms | [2] |
| **graph build** | 352.0 ms | 304.7 ms | 303.8 ms | 307.6 ms | [0] |
| **Gesamt** | 1450.4 ms | 1477.3 ms | 1411.0 ms | 1542.4 ms | [0, 2] |

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
