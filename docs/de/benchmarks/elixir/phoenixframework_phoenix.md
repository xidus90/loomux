# Benchmark & Lücken-Audit: [phoenixframework/phoenix](https://github.com/phoenixframework/phoenix)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Elixir | **Framework:** Phoenix | **Tier:** Sehr viel
- **Commit:** `da518c2f871c3b9049929160c91de87a323b61be`
- **Beispieldatei:** `integration_test/docker.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.4 ms | 9.1 ms | 9.0 ms | 9.5 ms | [0] |
| **post-tool-use** | 80.3 ms | 76.1 ms | 75.0 ms | 77.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 90.7 ms | 85.1 ms | 84.5 ms | 86.1 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `jest` | test | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **0.0 %**
- **Identifizierte Lücken:**
  - ⚠️ eslint deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ jest deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `docker, shell`
- **Lanes:** `shellcheck **/*.sh`
