# Benchmark & Lücken-Audit: [nagadomi/waifu2x](https://github.com/nagadomi/waifu2x)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Lua | **Framework:** Torch | **Tier:** Sehr viel
- **Commit:** `cc385f97a9debfe611316aabfd5d8bb30ba2dbeb`
- **Beispieldatei:** `appendix/benchmark.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 9.1 ms | 9.1 ms | 9.5 ms | [0] |
| **post-tool-use** | 82.4 ms | 82.5 ms | 81.2 ms | 86.2 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 93.5 ms | 92.0 ms | 90.3 ms | 95.3 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `install_lua_modules.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `css, html, shell`
- **Lanes:** `npx stylelint "**/*.{css,scss}", npx htmlhint "**/*.html", shellcheck **/*.sh`
