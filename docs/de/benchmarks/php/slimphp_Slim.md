# Benchmark & Lücken-Audit: [slimphp/Slim](https://github.com/slimphp/Slim)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** PHP | **Framework:** Slim | **Tier:** Sehr viel
- **Commit:** `3675bf6baac66b07032575b7bef4200b60b7974b`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 11.0 ms | 8.5 ms | 11.5 ms | [0] |
| **post-tool-use** | 12.5 ms | 13.5 ms | 12.0 ms | 14.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 23.0 ms | 23.5 ms | 22.5 ms | 24.5 ms | [0] |

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
