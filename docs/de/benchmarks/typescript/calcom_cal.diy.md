# Benchmark & Lücken-Audit: [calcom/cal.diy](https://github.com/calcom/cal.diy)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** TypeScript | **Framework:** tRPC | **Tier:** Sehr viel
- **Commit:** `6bc45298226f96ff79e0c070c8b2ce39727e8477`
- **Beispieldatei:** `__checks__/calcom-dashboard.check.js`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 11.0 ms | 10.5 ms | 12.0 ms | [0] |
| **post-tool-use** | 1315.1 ms | 1162.0 ms | 1112.4 ms | 1194.8 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 1326.6 ms | 1173.9 ms | 1123.4 ms | 1205.3 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `tsc` | typecheck | `package.json` | `npx tsc --noEmit` | Nein | ❌ Nicht im PATH |
| `vitest` | test | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `jest` | test | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **33.3 %**
- **Identifizierte Lücken:**
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH
  - ⚠️ vitest deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ jest deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `biome, css, docker, shell, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit, npx stylelint "**/*.{css,scss}", shellcheck **/*.sh`
