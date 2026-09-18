# Benchmark & Lücken-Audit: [filamentphp/filament](https://github.com/filamentphp/filament)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** PHP | **Framework:** Livewire | **Tier:** Sehr viel
- **Commit:** `2f160150a0f1d4007e4e1d1c4518646526cf53ca`
- **Beispieldatei:** `bin/issue-reproduction-template.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 9.1 ms | 8.5 ms | 9.5 ms | [0] |
| **post-tool-use** | 85.5 ms | 82.0 ms | 80.5 ms | 82.6 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 97.5 ms | 91.1 ms | 89.0 ms | 92.1 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `phpunit` | test | `composer.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `phpstan` | typecheck | `composer.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `prettier` | format | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **0.0 %**
- **Identifizierte Lücken:**
  - ⚠️ phpunit deklariert (composer.json), aber keine Lane in Loomux vorhanden
  - ⚠️ phpstan deklariert (composer.json), aber keine Lane in Loomux vorhanden
  - ⚠️ prettier deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
