# Benchmark & Lücken-Audit: [sveltejs/svelte](https://github.com/sveltejs/svelte)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** JavaScript | **Framework:** Svelte | **Tier:** Sehr viel
- **Commit:** `6eb720a1b7cafca3ebe0ab5c76674272cd3044f9`
- **Beispieldatei:** `benchmarking/analyze-compiler-profile.js`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 9.5 ms | 9.5 ms | 10.5 ms | [0] |
| **post-tool-use** | 944.9 ms | 992.9 ms | 991.3 ms | 1004.5 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 955.9 ms | 1002.4 ms | 1000.8 ms | 1015.0 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `npx eslint --cache .` | Nein | ❌ Nicht im PATH |
| `prettier` | format | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `vitest` | test | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **33.3 %**
- **Identifizierte Lücken:**
  - ⚠️ Lane für eslint vorhanden (npx eslint --cache .), aber Werkzeug nicht im PATH
  - ⚠️ prettier deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ vitest deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `pnpm, svelte, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit, npx svelte-check`
