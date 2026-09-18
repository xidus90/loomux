# Benchmark & Lücken-Audit: [digitallyinduced/ihp](https://github.com/digitallyinduced/ihp)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Haskell | **Framework:** IHP | **Tier:** Sehr viel
- **Commit:** `2bea296d3657693a595e3d5298c51882dfb7f40d`
- **Beispieldatei:** `Guide/layout.html`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 44.2 ms | 24.0 ms | 12.5 ms | 30.5 ms | [0] |
| **post-tool-use** | 2033.0 ms | 1705.5 ms | 1297.2 ms | 1904.4 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 2077.2 ms | 1736.0 ms | 1309.7 ms | 1928.4 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `update-nix-from-cabal.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `css, html, shell`
- **Lanes:** `npx stylelint "**/*.{css,scss}", npx htmlhint "**/*.html", shellcheck **/*.sh`
