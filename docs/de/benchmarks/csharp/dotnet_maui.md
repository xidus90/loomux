# Benchmark & Lücken-Audit: [dotnet/maui](https://github.com/dotnet/maui)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C# | **Framework:** .NET MAUI | **Tier:** Sehr viel
- **Commit:** `49b62062536b4a2b957d965f76035eb56806615c`
- **Beispieldatei:** `build.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.0 ms | 9.5 ms | 9.0 ms | 10.1 ms | [0] |
| **post-tool-use** | 73.3 ms | 71.4 ms | 68.9 ms | 73.6 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 86.3 ms | 80.9 ms | 79.0 ms | 82.6 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `dotnet-test` | test | `Microsoft.Maui-dev.sln` | `*keine*` | Ja | ⚠️ Fehlt in Loomux |
| `dotnet-test` | test | `Microsoft.Maui-vscode.sln` | `*keine*` | Ja | ⚠️ Fehlt in Loomux |
| `dotnet-test` | test | `Microsoft.Maui.LegacyControlGallery.sln` | `*keine*` | Ja | ⚠️ Fehlt in Loomux |
| `dotnet-test` | test | `Microsoft.Maui.sln` | `*keine*` | Ja | ⚠️ Fehlt in Loomux |
| `shellcheck` | lint | `build.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `helix.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **33.3 %**
- **Identifizierte Lücken:**
  - ⚠️ dotnet-test deklariert (Microsoft.Maui-dev.sln), aber keine Lane in Loomux vorhanden
  - ⚠️ dotnet-test deklariert (Microsoft.Maui-vscode.sln), aber keine Lane in Loomux vorhanden
  - ⚠️ dotnet-test deklariert (Microsoft.Maui.LegacyControlGallery.sln), aber keine Lane in Loomux vorhanden
  - ⚠️ dotnet-test deklariert (Microsoft.Maui.sln), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
