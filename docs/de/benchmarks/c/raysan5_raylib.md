# Benchmark & Lücken-Audit: [raysan5/raylib](https://github.com/raysan5/raylib)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C | **Framework:** Raylib | **Tier:** Sehr viel
- **Commit:** `2525c16f4bbb5f1753939fc0eebe0883a4f52be8`
- **Beispieldatei:** `examples/audio/audio_amp_envelope.c`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 15.0 ms | 9.5 ms | 8.5 ms | 9.5 ms | [0] |
| **post-tool-use** | 33.5 ms | 26.5 ms | 26.5 ms | 27.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 48.5 ms | 36.0 ms | 35.0 ms | 37.0 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `cmake, cpp, html`
- **Lanes:** `clang-format -i, cmake --build build --parallel, npx htmlhint "**/*.html"`
