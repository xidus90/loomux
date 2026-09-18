# Benchmark & Lücken-Audit: [jasontaylordev/NorthwindTraders](https://github.com/jasontaylordev/NorthwindTraders)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C# | **Framework:** Entity Framework Core | **Tier:** Sehr viel
- **Commit:** `647fafc87c4c34bcb9fc67a08db422a3018a0cab`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.0 ms | 8.5 ms | 8.5 ms | 9.5 ms | [0] |
| **post-tool-use** | 14.0 ms | 12.5 ms | 12.0 ms | 13.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 25.0 ms | 22.0 ms | 20.5 ms | 22.0 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `dotnet-test` | test | `Northwind.sln` | `*keine*` | Ja | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **0.0 %**
- **Identifizierte Lücken:**
  - ⚠️ dotnet-test deklariert (Northwind.sln), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
