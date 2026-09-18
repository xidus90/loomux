# Benchmark & Lücken-Audit: [dotnet/aspnetcore](https://github.com/dotnet/aspnetcore)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C# | **Framework:** ASP.NET Core | **Tier:** Sehr viel
- **Commit:** `13bb0ebb8a7e1c2cabf08239b4e4c813d49b0de6`
- **Beispieldatei:** `activate.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 10.0 ms | 9.5 ms | 10.2 ms | [0] |
| **post-tool-use** | 82.7 ms | 89.4 ms | 83.9 ms | 91.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 94.7 ms | 98.9 ms | 94.0 ms | 101.0 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `activate.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `clean.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `restore.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `startvscode.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
