# Benchmark & Lücken-Audit: [rails/rails](https://github.com/rails/rails)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Ruby | **Framework:** Ruby on Rails | **Tier:** Sehr viel
- **Commit:** `74d13d9845857db065051a42461e9d54f6ab4e39`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.0 ms | 9.5 ms | 9.0 ms | 9.5 ms | [0] |
| **post-tool-use** | 38.1 ms | 35.6 ms | 32.8 ms | 36.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 48.1 ms | 44.6 ms | 42.3 ms | 45.5 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `rubocop` | lint | `Gemfile` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **0.0 %**
- **Identifizierte Lücken:**
  - ⚠️ rubocop deklariert (Gemfile), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
