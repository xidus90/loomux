# Benchmark & Lücken-Audit: [hexchat/hexchat](https://github.com/hexchat/hexchat)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C | **Framework:** GTK | **Tier:** Sehr viel
- **Commit:** `b544ac3350e85d4cc41fe3414cbdb82d75ce5d7a`
- **Beispieldatei:** `plugins/checksum/checksum.c`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.4 ms | 9.0 ms | 8.5 ms | 10.1 ms | [0] |
| **post-tool-use** | 33.4 ms | 28.5 ms | 28.5 ms | 30.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 44.8 ms | 37.5 ms | 37.0 ms | 40.6 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `cpp, meson, shell`
- **Lanes:** `clang-format -i, cmake --build build --parallel, shellcheck **/*.sh`
