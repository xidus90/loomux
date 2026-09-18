# Benchmark & Lücken-Audit: [flutter/flutter](https://github.com/flutter/flutter)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Dart | **Framework:** Flutter | **Tier:** Sehr viel
- **Commit:** `2cb5ae40e66710e75a7fe0285dd874db820db2c0`
- **Beispieldatei:** `dev/a11y_assessments/ios/Runner/Runner-Bridging-Header.h`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 8.0 ms | 8.0 ms | 8.5 ms | [0] |
| **post-tool-use** | 31.0 ms | 31.0 ms | 30.5 ms | 34.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 42.0 ms | 39.0 ms | 39.0 ms | 42.5 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `.autoroller-preupload.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `clang-format, cpp, shell`
- **Lanes:** `clang-format -i, cmake --build build --parallel, shellcheck **/*.sh`
