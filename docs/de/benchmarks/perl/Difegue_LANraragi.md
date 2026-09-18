# Benchmark & Lücken-Audit: [Difegue/LANraragi](https://github.com/Difegue/LANraragi)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Perl | **Framework:** Mojolicious | **Tier:** Sehr viel
- **Commit:** `db3106900d90e07f5723da9ed933cc0399a5670d`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 9.5 ms | 8.5 ms | 8.0 ms | 8.7 ms | [0] |
| **post-tool-use** | 20.0 ms | 18.8 ms | 18.0 ms | 20.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 29.5 ms | 26.8 ms | 26.7 ms | 29.0 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **0.0 %**
- **Identifizierte Lücken:**
  - ⚠️ eslint deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
