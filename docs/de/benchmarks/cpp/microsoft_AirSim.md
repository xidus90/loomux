# Benchmark & Lücken-Audit: [microsoft/AirSim](https://github.com/microsoft/AirSim)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C++ | **Framework:** Unreal Engine | **Tier:** Sehr viel
- **Commit:** `1ca93f6f77e4e8a39b2b241c1fe2764da4d7dd41`
- **Beispieldatei:** `AirLib/include/api/ApiProvider.hpp`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.4 ms | 9.0 ms | 8.1 ms | 9.1 ms | [0] |
| **post-tool-use** | 53.4 ms | 48.3 ms | 47.5 ms | 49.7 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 65.8 ms | 56.5 ms | 56.4 ms | 58.8 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `clang-format` | format | `.clang-format` | `clang-format -i` | Nein | ❌ Nicht im PATH |
| `dotnet-test` | test | `AirSim.sln` | `*keine*` | Ja | ⚠️ Fehlt in Loomux |
| `shellcheck` | lint | `build.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `build_docs.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `clean.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `clean_rebuild.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `install_run_all.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `install_unreal.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `setup.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **88.9 %**
- **Identifizierte Lücken:**
  - ⚠️ Lane für clang-format vorhanden (clang-format -i), aber Werkzeug nicht im PATH
  - ⚠️ dotnet-test deklariert (AirSim.sln), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `clang-format, cmake, cpp, python, shell`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, clang-format -i, cmake --build build --parallel, shellcheck **/*.sh`
