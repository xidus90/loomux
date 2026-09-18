# Benchmark & Lücken-Audit: [Perfare/AssetStudio](https://github.com/Perfare/AssetStudio)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C# | **Framework:** Unity | **Tier:** Sehr viel
- **Commit:** `d158e864b556b5970709c2a52e47944d53aa98a2`
- **Beispieldatei:** `AssetStudioFBXNative/api.cpp`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.0 ms | 10.0 ms | 9.0 ms | 10.5 ms | [0] |
| **post-tool-use** | 36.5 ms | 33.6 ms | 31.5 ms | 34.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 49.5 ms | 43.0 ms | 41.5 ms | 44.1 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `dotnet-test` | test | `AssetStudio.sln` | `*keine*` | Ja | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **0.0 %**
- **Identifizierte Lücken:**
  - ⚠️ dotnet-test deklariert (AssetStudio.sln), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `cpp`
- **Lanes:** `clang-format -i, cmake --build build --parallel`
