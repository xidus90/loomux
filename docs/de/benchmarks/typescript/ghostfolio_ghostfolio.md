# Benchmark & Lücken-Audit: [ghostfolio/ghostfolio](https://github.com/ghostfolio/ghostfolio)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** TypeScript | **Framework:** NestJS | **Tier:** Sehr viel
- **Commit:** `f0e2d6c33644cf928bd1317652bb52265770362f`
- **Beispieldatei:** `docker/entrypoint.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 9.0 ms | 8.5 ms | 10.0 ms | [0] |
| **post-tool-use** | 73.6 ms | 67.5 ms | 65.7 ms | 67.9 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 85.1 ms | 76.9 ms | 74.2 ms | 77.5 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `prettier` | format | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `jest` | test | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **0.0 %**
- **Identifizierte Lücken:**
  - ⚠️ eslint deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ prettier deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ jest deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `docker, shell`
- **Lanes:** `shellcheck **/*.sh`
