# Benchmark & Lücken-Audit: [apache/struts](https://github.com/apache/struts)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Java | **Framework:** Struts | **Tier:** Sehr viel
- **Commit:** `85f30e35c5c5fc7c46db04af961e07a5826da32e`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 9.0 ms | 8.5 ms | 9.0 ms | [0] |
| **post-tool-use** | 27.0 ms | 22.5 ms | 22.0 ms | 22.9 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 39.0 ms | 31.4 ms | 31.0 ms | 31.5 ms | [0] |

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
