# Benchmark & Lücken-Audit: [pocoproject/poco](https://github.com/pocoproject/poco)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C++ | **Framework:** Poco | **Tier:** Sehr viel
- **Commit:** `3f0c291d56751b12ba95ca545706061bd4d35aeb`
- **Beispieldatei:** `ActiveRecord/Compiler/src/CodeGenerator.cpp`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 14.5 ms | 12.5 ms | 11.5 ms | 14.0 ms | [0] |
| **post-tool-use** | 93.5 ms | 90.6 ms | 90.0 ms | 92.2 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 108.0 ms | 103.1 ms | 101.5 ms | 106.2 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `build_cmake.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `build_make.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `env.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `gh-cli-for-release-notes.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `runLibTests.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `runVSCode.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `cmake, cpp, html, shell`
- **Lanes:** `clang-format -i, cmake --build build --parallel, npx htmlhint "**/*.html", shellcheck **/*.sh`
