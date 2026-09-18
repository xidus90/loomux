# Benchmark & Lücken-Audit: [gobuffalo/buffalo](https://github.com/gobuffalo/buffalo)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Go | **Framework:** Buffalo | **Tier:** Sehr viel
- **Commit:** `2aa9868365cdcaa28036efd76e7aac4b7df7bbfc`
- **Beispieldatei:** `app.go`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 10.1 ms | 9.5 ms | 10.5 ms | [0] |
| **post-tool-use** | 391.2 ms | 407.9 ms | 394.5 ms | 426.4 ms | [0] |
| **graph build** | 62.1 ms | 63.5 ms | 62.6 ms | 65.8 ms | [0] |
| **Gesamt** | 463.8 ms | 479.9 ms | 470.4 ms | 500.4 ms | [0] |

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
