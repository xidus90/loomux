# Benchmark & Lücken-Audit: [spockframework/spock-example](https://github.com/spockframework/spock-example)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Groovy | **Framework:** Spock | **Tier:** Sehr viel
- **Commit:** `8155d1e2a8f1bdfb1aeacca2e9fe5ded7c992a5e`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | n/a | n/a | n/a | n/a | n/a |
| **post-tool-use** | n/a | n/a | n/a | n/a | n/a |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 0.0 ms | 0.0 ms | 0.0 ms | 0.0 ms | n/a |

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
