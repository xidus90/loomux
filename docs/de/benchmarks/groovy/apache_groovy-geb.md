# Benchmark & Lücken-Audit: [apache/groovy-geb](https://github.com/apache/groovy-geb)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Groovy | **Framework:** Geb | **Tier:** Sehr viel
- **Commit:** `9c73494607825cd8900d51681919737e17c6fb12`
- **Beispieldatei:** `build-in-docker.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 9.0 ms | 8.7 ms | 9.5 ms | [0] |
| **post-tool-use** | 73.1 ms | 72.3 ms | 70.0 ms | 76.4 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 84.1 ms | 81.0 ms | 79.5 ms | 85.4 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `build-in-docker.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
