# Benchmark & Lücken-Audit: [curl/curl](https://github.com/curl/curl)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C | **Framework:** curl | **Tier:** Sehr viel
- **Commit:** `7f364029b861d064caa128f4a8cb7b34696f7db3`
- **Beispieldatei:** `CMake/CurlTests.c`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.5 ms | 10.0 ms | 10.5 ms | [0] |
| **post-tool-use** | 52.0 ms | 47.0 ms | 46.2 ms | 50.2 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 64.0 ms | 57.0 ms | 56.7 ms | 60.7 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `appveyor.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `cmake, cpp, shell`
- **Lanes:** `clang-format -i, cmake --build build --parallel, shellcheck **/*.sh`
