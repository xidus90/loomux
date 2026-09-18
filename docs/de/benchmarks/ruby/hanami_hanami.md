# Benchmark & Lücken-Audit: [hanami/hanami](https://github.com/hanami/hanami)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Ruby | **Framework:** Hanami | **Tier:** Sehr viel
- **Commit:** `c31f9cf8e51e5c6c5231f45fcaea625202c30ebd`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.4 ms | 8.5 ms | 8.5 ms | 8.6 ms | [0] |
| **post-tool-use** | 13.0 ms | 11.5 ms | 11.0 ms | 16.4 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 24.4 ms | 20.0 ms | 19.6 ms | 24.9 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `rspec` | test | `Gemfile` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **0.0 %**
- **Identifizierte Lücken:**
  - ⚠️ rspec deklariert (Gemfile), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
