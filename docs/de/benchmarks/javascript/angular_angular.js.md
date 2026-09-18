# Benchmark & Lücken-Audit: [angular/angular.js](https://github.com/angular/angular.js)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** JavaScript | **Framework:** Angular | **Tier:** Sehr viel
- **Commit:** `d8f77817eb5c98dec5317bc3756d1ea1812bcfbe`
- **Beispieldatei:** `css/angular-scenario.css`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 15.0 ms | 10.0 ms | 10.0 ms | 10.0 ms | [0] |
| **post-tool-use** | 1255.8 ms | 1174.5 ms | 1172.7 ms | 1175.5 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 1270.8 ms | 1184.5 ms | 1182.7 ms | 1185.5 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **0.0 %**
- **Identifizierte Lücken:**
  - ⚠️ eslint deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `css, shell`
- **Lanes:** `npx stylelint "**/*.{css,scss}", shellcheck **/*.sh`
