# Benchmark & Lücken-Audit: [honojs/hono](https://github.com/honojs/hono)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** TypeScript | **Framework:** Hono | **Tier:** Sehr viel
- **Commit:** `098e11912ab244c5c33931de007f04dc8e3c2929`
- **Beispieldatei:** `benchmarks/http-server/benchmark.ts`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.0 ms | 9.5 ms | 10.0 ms | [0] |
| **post-tool-use** | 960.1 ms | 965.1 ms | 942.1 ms | 967.1 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 972.1 ms | 975.1 ms | 951.6 ms | 977.1 ms | [0, 2] |

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

- **Stacks:** `typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit`
