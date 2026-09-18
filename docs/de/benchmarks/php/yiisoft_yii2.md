# Benchmark & Lücken-Audit: [yiisoft/yii2](https://github.com/yiisoft/yii2)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** PHP | **Framework:** Yii2 | **Tier:** Sehr viel
- **Commit:** `8ee8986b32e574a2d91a1f64f89185b4339d7593`
- **Beispieldatei:** `tests/test-local.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.6 ms | 9.0 ms | 8.5 ms | 9.5 ms | [0] |
| **post-tool-use** | 78.4 ms | 73.1 ms | 70.0 ms | 75.1 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 91.0 ms | 82.1 ms | 78.5 ms | 84.6 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `phpunit` | test | `composer.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `phpstan` | typecheck | `composer.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `eslint` | lint | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **0.0 %**
- **Identifizierte Lücken:**
  - ⚠️ phpunit deklariert (composer.json), aber keine Lane in Loomux vorhanden
  - ⚠️ phpstan deklariert (composer.json), aber keine Lane in Loomux vorhanden
  - ⚠️ eslint deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `docker, shell`
- **Lanes:** `shellcheck **/*.sh`
