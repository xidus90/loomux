# Benchmark & Lücken-Audit: [nuxt/create-nuxt-app](https://github.com/nuxt/create-nuxt-app)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** JavaScript | **Framework:** Nuxt | **Tier:** Sehr viel
- **Commit:** `2f04969c2f30b16b85a93bea52e489109c0a7210`
- **Beispieldatei:** `packages/cna-template/template/frameworks/vuetify/assets/variables.scss`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.0 ms | 10.0 ms | 10.5 ms | [0] |
| **post-tool-use** | 1199.4 ms | 1093.7 ms | 1058.5 ms | 1229.6 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 1211.4 ms | 1104.2 ms | 1068.5 ms | 1239.6 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `jest` | test | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **0.0 %**
- **Identifizierte Lücken:**
  - ⚠️ eslint deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ jest deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `css, stylelint`
- **Lanes:** `npx stylelint "**/*.{css,scss}"`
