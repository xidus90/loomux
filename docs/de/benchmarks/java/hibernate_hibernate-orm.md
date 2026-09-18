# Benchmark & Lücken-Audit: [hibernate/hibernate-orm](https://github.com/hibernate/hibernate-orm)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Java | **Framework:** Hibernate | **Tier:** Sehr viel
- **Commit:** `c3ef32170fa99d3efc92262a69acb1084c1a92c0`
- **Beispieldatei:** `ci/before-cache.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.5 ms | 9.6 ms | 9.5 ms | 10.1 ms | [0] |
| **post-tool-use** | 145.3 ms | 118.7 ms | 117.2 ms | 137.9 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 157.8 ms | 128.9 ms | 126.7 ms | 147.5 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `db.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
