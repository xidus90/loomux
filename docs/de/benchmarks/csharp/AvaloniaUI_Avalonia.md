# Benchmark & Lücken-Audit: [AvaloniaUI/Avalonia](https://github.com/AvaloniaUI/Avalonia)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C# | **Framework:** Avalonia | **Tier:** Sehr viel
- **Commit:** `334536066a6cf4df5dcfb1da22cb9ed80dd66e8e`
- **Beispieldatei:** `build-native.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 9.5 ms | 9.5 ms | 10.0 ms | [0] |
| **post-tool-use** | 83.5 ms | 78.0 ms | 77.5 ms | 81.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 95.5 ms | 88.0 ms | 87.0 ms | 90.5 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `build-native.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `build.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
