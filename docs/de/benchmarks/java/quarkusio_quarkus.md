# Benchmark & Lücken-Audit: [quarkusio/quarkus](https://github.com/quarkusio/quarkus)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Java | **Framework:** Quarkus | **Tier:** Sehr viel
- **Commit:** `83809ecb7a3ba8389c207cb1c453aeb09041512c`
- **Beispieldatei:** `coverage-report/prepare.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 33.0 ms | 19.2 ms | 16.0 ms | 24.6 ms | [0] |
| **post-tool-use** | 147.1 ms | 155.8 ms | 122.6 ms | 191.2 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 180.1 ms | 180.4 ms | 138.6 ms | 210.4 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `mvn-test` | test | `pom.xml` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `shellcheck` | lint | `update-extension-dependencies.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `update-version.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **66.7 %**
- **Identifizierte Lücken:**
  - ⚠️ mvn-test deklariert (pom.xml), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
