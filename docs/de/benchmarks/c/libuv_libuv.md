# Benchmark & Lücken-Audit: [libuv/libuv](https://github.com/libuv/libuv)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C | **Framework:** libuv | **Tier:** Sehr viel
- **Commit:** `84af0b18c5aee743a9d4182fa9c18a18fd9825e3`
- **Beispieldatei:** `docs/code/cgi/main.c`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 17.0 ms | 12.0 ms | 10.6 ms | 12.5 ms | [0] |
| **post-tool-use** | 44.0 ms | 36.4 ms | 35.8 ms | 38.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 61.0 ms | 48.4 ms | 46.4 ms | 51.0 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `clang-tidy` | lint | `.clang-tidy` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `autogen.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **66.7 %**
- **Identifizierte Lücken:**
  - ⚠️ clang-tidy deklariert (.clang-tidy), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `clang-tidy, cmake, cpp, python, shell`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, clang-format -i, cmake --build build --parallel, shellcheck **/*.sh`
