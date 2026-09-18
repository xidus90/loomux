# Benchmark & Lücken-Audit: [sinatra/sinatra](https://github.com/sinatra/sinatra)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Ruby | **Framework:** Sinatra | **Tier:** Sehr viel
- **Commit:** `cb22afd7902b566b6eaba6c4ea89739494a65d12`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.2 ms | 9.0 ms | 8.1 ms | 9.6 ms | [0] |
| **post-tool-use** | 18.5 ms | 16.0 ms | 15.4 ms | 16.9 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 30.8 ms | 25.6 ms | 23.5 ms | 25.9 ms | [0] |

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
