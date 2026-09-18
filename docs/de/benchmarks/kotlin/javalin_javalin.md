# Benchmark & Lücken-Audit: [javalin/javalin](https://github.com/javalin/javalin)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Kotlin | **Framework:** Javalin | **Tier:** Sehr viel
- **Commit:** `1ff18e9e03e69cc3759abc57f1f12c32285eded3`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 8.5 ms | 8.0 ms | 11.0 ms | [0] |
| **post-tool-use** | 19.5 ms | 19.0 ms | 18.5 ms | 19.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 30.0 ms | 27.5 ms | 27.5 ms | 29.5 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `mvn-test` | test | `pom.xml` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **0.0 %**
- **Identifizierte Lücken:**
  - ⚠️ mvn-test deklariert (pom.xml), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
