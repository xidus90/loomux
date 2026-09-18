# Benchmark & Lücken-Audit: [beego/beego](https://github.com/beego/beego)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Go | **Framework:** Beego | **Tier:** Sehr viel
- **Commit:** `939cfde380bb9f15844ad633b84f037f7da21584`
- **Beispieldatei:** `build_info.go`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 10.5 ms | 10.0 ms | 13.1 ms | [0] |
| **post-tool-use** | 812.8 ms | 822.9 ms | 779.9 ms | 841.1 ms | [2] |
| **graph build** | 253.0 ms | 268.9 ms | 257.5 ms | 275.3 ms | [0] |
| **Gesamt** | 1076.8 ms | 1102.4 ms | 1065.3 ms | 1111.7 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `go-vet` | lint | `go.mod` | `go vet ./...` | Ja | ✅ Aktiv |
| `go-test` | test | `go.mod` | `*keine*` | Ja | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **50.0 %**
- **Identifizierte Lücken:**
  - ⚠️ go-test deklariert (go.mod), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `go, shell`
- **Lanes:** `shellcheck **/*.sh, go vet ./...`
