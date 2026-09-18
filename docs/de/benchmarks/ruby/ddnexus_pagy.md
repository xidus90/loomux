# Benchmark & Lücken-Audit: [ddnexus/pagy](https://github.com/ddnexus/pagy)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Ruby | **Framework:** Padrino | **Tier:** Sehr viel
- **Commit:** `e85430530be38b76d4329c2fea7d870dee60e94f`
- **Beispieldatei:** `assets/nav.html`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 10.1 ms | 9.0 ms | 10.5 ms | [0] |
| **post-tool-use** | 1110.9 ms | 940.5 ms | 919.8 ms | 942.9 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 1121.9 ms | 951.0 ms | 929.9 ms | 951.9 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `rubocop` | lint | `Gemfile` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `eslint` | lint | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **0.0 %**
- **Identifizierte Lücken:**
  - ⚠️ rubocop deklariert (Gemfile), aber keine Lane in Loomux vorhanden
  - ⚠️ eslint deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `html`
- **Lanes:** `npx htmlhint "**/*.html"`
