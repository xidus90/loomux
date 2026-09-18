# Benchmark & Lücken-Audit: [bcit-ci/CodeIgniter](https://github.com/bcit-ci/CodeIgniter)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** PHP | **Framework:** CodeIgniter | **Tier:** Sehr viel
- **Commit:** `3658d731eaabe6117298a105ffb5b9dd59e190ce`
- **Beispieldatei:** `application/cache/index.html`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.0 ms | 9.7 ms | 9.5 ms | 10.0 ms | [0] |
| **post-tool-use** | 1103.4 ms | 896.0 ms | 887.5 ms | 897.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 1116.4 ms | 905.5 ms | 897.2 ms | 907.5 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `build-release.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `phpunit` | test | `composer.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **50.0 %**
- **Identifizierte Lücken:**
  - ⚠️ phpunit deklariert (composer.json), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `html, shell`
- **Lanes:** `npx htmlhint "**/*.html", shellcheck **/*.sh`
