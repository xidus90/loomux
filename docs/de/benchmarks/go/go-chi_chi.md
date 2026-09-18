# Benchmark & Lücken-Audit: [go-chi/chi](https://github.com/go-chi/chi)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Go | **Framework:** Chi | **Tier:** Sehr viel
- **Commit:** `3d1777a1ef8881f7d1da0b02c76ca8f0a29cd2bc`
- **Beispieldatei:** `_examples/custom-handler/main.go`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 10.5 ms | 10.0 ms | 11.0 ms | [0] |
| **post-tool-use** | 259.6 ms | 263.8 ms | 257.6 ms | 268.4 ms | [0] |
| **graph build** | 58.1 ms | 57.0 ms | 56.9 ms | 57.2 ms | [0] |
| **Gesamt** | 328.7 ms | 330.6 ms | 325.2 ms | 336.5 ms | [0] |

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
