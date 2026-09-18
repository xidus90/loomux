# Benchmark & Lücken-Audit: [t3-oss/create-t3-app](https://github.com/t3-oss/create-t3-app)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** TypeScript | **Framework:** Next.js | **Tier:** Sehr viel
- **Commit:** `4709861f7e67a15564c0460c13e7b4b6cfcae40d`
- **Beispieldatei:** `cli/eslint.config.js`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.5 ms | 10.5 ms | 10.0 ms | 10.5 ms | [0] |
| **post-tool-use** | 975.0 ms | 953.0 ms | 925.0 ms | 962.1 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 987.5 ms | 963.5 ms | 935.5 ms | 972.1 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `npx eslint --cache .` | Nein | ❌ Nicht im PATH |
| `prettier` | format | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `tsc` | typecheck | `tsconfig.json` | `npx tsc --noEmit` | Nein | ❌ Nicht im PATH |

- **Abdeckungsquote:** **66.7 %**
- **Identifizierte Lücken:**
  - ⚠️ Lane für eslint vorhanden (npx eslint --cache .), aber Werkzeug nicht im PATH
  - ⚠️ prettier deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH

## 3. Erkannte Stacks & Lanes

- **Stacks:** `astro, pnpm, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit`
