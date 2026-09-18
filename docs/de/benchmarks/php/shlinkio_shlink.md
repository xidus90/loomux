# Benchmark & Lücken-Audit: [shlinkio/shlink](https://github.com/shlinkio/shlink)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** PHP | **Framework:** Laminas | **Tier:** Sehr viel
- **Commit:** `c303aff18f8108ca1e17ec5c697a63afd8efd4fe`
- **Beispieldatei:** `bin/test/run-api-tests.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 9.5 ms | 8.6 ms | 10.0 ms | [0] |
| **post-tool-use** | 77.2 ms | 75.4 ms | 72.1 ms | 77.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 88.2 ms | 84.9 ms | 82.1 ms | 85.7 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `build.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `phpunit` | test | `composer.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **50.0 %**
- **Identifizierte Lücken:**
  - ⚠️ phpunit deklariert (composer.json), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `docker, shell`
- **Lanes:** `shellcheck **/*.sh`
