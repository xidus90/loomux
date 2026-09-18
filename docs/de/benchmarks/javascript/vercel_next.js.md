# Benchmark & Lücken-Audit: [vercel/next.js](https://github.com/vercel/next.js)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** JavaScript | **Framework:** Next.js | **Tier:** Sehr viel
- **Commit:** `9a20fce8f885992fcc44a107adc41390084c2696`
- **Beispieldatei:** `apps/bundle-analyzer/app/layout.tsx`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.8 ms | 11.0 ms | 10.0 ms | 11.0 ms | [0] |
| **post-tool-use** | 1043.8 ms | 1015.3 ms | 993.7 ms | 1058.7 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 1056.6 ms | 1025.4 ms | 1004.7 ms | 1069.7 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `cargo-clippy` | lint | `Cargo.toml` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `cargo-test` | test | `Cargo.toml` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `cargo-fmt` | format | `Cargo.toml` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `eslint` | lint | `package.json` | `npx eslint --cache .` | Nein | ❌ Nicht im PATH |
| `prettier` | format | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `tsc` | typecheck | `package.json` | `npx tsc --noEmit` | Nein | ❌ Nicht im PATH |
| `jest` | test | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `tsc` | typecheck | `tsconfig.json` | `npx tsc --noEmit` | Nein | ❌ Nicht im PATH |

- **Abdeckungsquote:** **37.5 %**
- **Identifizierte Lücken:**
  - ⚠️ cargo-clippy deklariert (Cargo.toml), aber keine Lane in Loomux vorhanden
  - ⚠️ cargo-test deklariert (Cargo.toml), aber keine Lane in Loomux vorhanden
  - ⚠️ cargo-fmt deklariert (Cargo.toml), aber keine Lane in Loomux vorhanden
  - ⚠️ Lane für eslint vorhanden (npx eslint --cache .), aber Werkzeug nicht im PATH
  - ⚠️ prettier deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH
  - ⚠️ jest deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH

## 3. Erkannte Stacks & Lanes

- **Stacks:** `pnpm, rust, shell, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit, shellcheck **/*.sh, cargo clippy -- -D warnings, cargo fmt --check`
