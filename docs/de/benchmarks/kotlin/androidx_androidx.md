# Benchmark & Lücken-Audit: [androidx/androidx](https://github.com/androidx/androidx)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Kotlin | **Framework:** Android Jetpack | **Tier:** Sehr viel
- **Commit:** `994f2ad5cf175f7cebd39303bbee355d19eeb929`
- **Beispieldatei:** `appfunctions/local_tests.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 9.5 ms | 9.2 ms | 10.0 ms | [0] |
| **post-tool-use** | 231.3 ms | 249.2 ms | 225.7 ms | 256.8 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 243.3 ms | 259.2 ms | 235.2 ms | 266.0 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `cleanBuild.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
