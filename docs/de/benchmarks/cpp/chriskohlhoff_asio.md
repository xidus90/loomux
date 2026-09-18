# Benchmark & Lücken-Audit: [chriskohlhoff/asio](https://github.com/chriskohlhoff/asio)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C++ | **Framework:** Asio | **Tier:** Sehr viel
- **Commit:** `8806a6803cde7054c3049d3666d3ec36786568c5`
- **Beispieldatei:** `include/asio/any_completion_executor.hpp`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 9.5 ms | 9.0 ms | 10.5 ms | [0] |
| **post-tool-use** | 27.5 ms | 23.5 ms | 22.5 ms | 23.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 39.5 ms | 32.5 ms | 32.0 ms | 34.0 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `autogen.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `cpp, shell`
- **Lanes:** `clang-format -i, cmake --build build --parallel, shellcheck **/*.sh`
