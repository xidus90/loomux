# Benchmark & Lücken-Audit: [snapframework/snap](https://github.com/snapframework/snap)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Haskell | **Framework:** Snap | **Tier:** Sehr viel
- **Commit:** `93ac0673b6d20ae4c7778e2323188f1a93cfc52c`
- **Beispieldatei:** `extra/haddock.css`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 10.0 ms | 9.5 ms | 10.7 ms | [0] |
| **post-tool-use** | 1295.8 ms | 1156.3 ms | 1080.8 ms | 1483.4 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 1306.3 ms | 1165.8 ms | 1091.5 ms | 1493.4 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `haddock.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `pull.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `pullLatestMaster.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `runTestsAndCoverage.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `css, shell`
- **Lanes:** `npx stylelint "**/*.{css,scss}", shellcheck **/*.sh`
