# Benchmark & Lücken-Audit: [revel/revel](https://github.com/revel/revel)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Go | **Framework:** Revel | **Tier:** Sehr viel
- **Commit:** `b053175279547526fe914932716bc313558df4bd`
- **Beispieldatei:** `before_after_filter.go`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.0 ms | 9.7 ms | 9.2 ms | 13.0 ms | [0] |
| **post-tool-use** | 354.0 ms | 353.2 ms | 341.4 ms | 373.0 ms | [2] |
| **graph build** | 84.9 ms | 77.0 ms | 74.9 ms | 78.5 ms | [0] |
| **Gesamt** | 448.9 ms | 437.8 ms | 431.4 ms | 460.7 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `go-vet` | lint | `go.mod` | `go vet ./...` | Ja | ✅ Aktiv |
| `go-test` | test | `go.mod` | `*keine*` | Ja | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **50.0 %**
- **Identifizierte Lücken:**
  - ⚠️ go-test deklariert (go.mod), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `go`
- **Lanes:** `go vet ./...`
