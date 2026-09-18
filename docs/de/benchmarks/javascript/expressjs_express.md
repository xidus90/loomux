# Benchmark & Lücken-Audit: [expressjs/express](https://github.com/expressjs/express)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** JavaScript | **Framework:** Express | **Tier:** Sehr viel
- **Commit:** `9a34acf03cb818ff3f8bc40e44176e277a25cbb9`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 10.0 ms | 9.5 ms | 10.5 ms | [0] |
| **post-tool-use** | 23.5 ms | 24.5 ms | 22.5 ms | 27.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 34.0 ms | 35.0 ms | 32.5 ms | 37.0 ms | [0] |

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
