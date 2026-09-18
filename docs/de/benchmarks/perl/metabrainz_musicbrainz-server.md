# Benchmark & Lücken-Audit: [metabrainz/musicbrainz-server](https://github.com/metabrainz/musicbrainz-server)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Perl | **Framework:** Catalyst | **Tier:** Sehr viel
- **Commit:** `15b555fbd838b3cc69e6847fdb21f109b5f14816`
- **Beispieldatei:** `admin/CalculateRelatedTags.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 9.0 ms | 8.5 ms | 10.0 ms | [0] |
| **post-tool-use** | 79.7 ms | 86.1 ms | 80.6 ms | 94.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 90.2 ms | 95.1 ms | 89.1 ms | 104.0 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `shellcheck` | lint | `upgrade.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **50.0 %**
- **Identifizierte Lücken:**
  - ⚠️ eslint deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
