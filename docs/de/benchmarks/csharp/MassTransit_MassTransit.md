# Benchmark & Lücken-Audit: [MassTransit/MassTransit](https://github.com/MassTransit/MassTransit)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C# | **Framework:** MassTransit | **Tier:** Sehr viel
- **Commit:** `62ab339afa3bac2e9b3fe1769d0d35d7e44778e9`
- **Beispieldatei:** `README.md`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.0 ms | 9.0 ms | 8.5 ms | 9.0 ms | [0] |
| **post-tool-use** | 15.0 ms | 12.5 ms | 12.0 ms | 13.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 28.0 ms | 21.5 ms | 20.5 ms | 22.5 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `dotnet-test` | test | `MassTransit.sln` | `*keine*` | Ja | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **0.0 %**
- **Identifizierte Lücken:**
  - ⚠️ dotnet-test deklariert (MassTransit.sln), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** ``
- **Lanes:** ``
