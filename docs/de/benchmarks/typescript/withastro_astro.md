# Benchmark & Lücken-Audit: [withastro/astro](https://github.com/withastro/astro)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** TypeScript | **Framework:** Astro | **Tier:** Sehr viel
- **Commit:** `db2eaf17ce84a5f75c5eab30f4ae15af32de1a13`
- **Beispieldatei:** `benchmark/bench/_template.js`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 10.7 ms | 9.5 ms | 15.0 ms | [0] |
| **post-tool-use** | 1036.2 ms | 962.2 ms | 948.1 ms | 982.7 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 1047.7 ms | 971.7 ms | 958.8 ms | 997.7 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `npx eslint --cache .` | Nein | ❌ Nicht im PATH |
| `prettier` | format | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `tsc` | typecheck | `package.json` | `npx tsc --noEmit` | Nein | ❌ Nicht im PATH |
| `vitest` | test | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `tsc` | typecheck | `tsconfig.json` | `npx tsc --noEmit` | Nein | ❌ Nicht im PATH |

- **Abdeckungsquote:** **60.0 %**
- **Identifizierte Lücken:**
  - ⚠️ Lane für eslint vorhanden (npx eslint --cache .), aber Werkzeug nicht im PATH
  - ⚠️ prettier deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH
  - ⚠️ vitest deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH

## 3. Erkannte Stacks & Lanes

- **Stacks:** `pnpm, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit`
