# Benchmark & Lücken-Audit: [cakephp/cakephp](https://github.com/cakephp/cakephp)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** PHP | **Framework:** CakePHP | **Tier:** Sehr viel
- **Commit:** `356619db99a13278e32e7065d43cce950d6ed3c1`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.0 ms | 8.7 ms | 8.7 ms | 9.0 ms | [0] |
| **post-tool-use** | 18.3 ms | 14.8 ms | 14.0 ms | 15.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 31.3 ms | 23.8 ms | 22.7 ms | 24.2 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `phpunit` | test | `composer.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `phpstan` | typecheck | `composer.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **0.0 %**
- **Identifizierte Lücken:**
  - ⚠️ phpunit deklariert (composer.json), aber keine Lane in Loomux vorhanden
  - ⚠️ phpstan deklariert (composer.json), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
