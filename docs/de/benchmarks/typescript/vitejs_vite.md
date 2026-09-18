# Benchmark & Lücken-Audit: [vitejs/vite](https://github.com/vitejs/vite)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** TypeScript | **Framework:** Vite | **Tier:** Sehr viel
- **Commit:** `e9078f865cdff6bed77cd729214a7e2868f126b5`
- **Beispieldatei:** `docs/_data/acknowledgements.data.ts`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.5 ms | 10.5 ms | 13.2 ms | [0] |
| **post-tool-use** | 1039.2 ms | 962.9 ms | 953.0 ms | 974.4 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 1051.2 ms | 973.4 ms | 966.2 ms | 984.9 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `npx eslint --cache .` | Nein | ❌ Nicht im PATH |
| `tsc` | typecheck | `package.json` | `npx tsc --noEmit` | Nein | ❌ Nicht im PATH |
| `vitest` | test | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **66.7 %**
- **Identifizierte Lücken:**
  - ⚠️ Lane für eslint vorhanden (npx eslint --cache .), aber Werkzeug nicht im PATH
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH
  - ⚠️ vitest deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `pnpm, shell, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit, shellcheck **/*.sh`
