# Benchmark & Lücken-Audit: [DaveGamble/cJSON](https://github.com/DaveGamble/cJSON)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C | **Framework:** cJSON | **Tier:** Sehr viel
- **Commit:** `6d9f2443ab071f86e5d9b43025a40929ec41c46c`
- **Beispieldatei:** `cJSON.c`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 8.5 ms | 8.5 ms | 8.6 ms | [0] |
| **post-tool-use** | 25.5 ms | 22.0 ms | 21.4 ms | 23.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 36.0 ms | 30.6 ms | 29.9 ms | 31.5 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `cmake, cpp, shell`
- **Lanes:** `clang-format -i, cmake --build build --parallel, shellcheck **/*.sh`
