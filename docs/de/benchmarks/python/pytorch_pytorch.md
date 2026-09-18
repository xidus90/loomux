# Benchmark & Lücken-Audit: [pytorch/pytorch](https://github.com/pytorch/pytorch)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Python | **Framework:** PyTorch | **Tier:** Sehr viel
- **Commit:** `5c6918c45adefb8eaf40e7f665ccd00f9879fbef`
- **Beispieldatei:** `android/pytorch_android/generate_test_asset.cpp`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.4 ms | 8.5 ms | 8.2 ms | 9.0 ms | [0] |
| **post-tool-use** | 53.7 ms | 46.6 ms | 45.1 ms | 47.1 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 65.1 ms | 54.7 ms | 53.6 ms | 56.0 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `clang-format` | format | `.clang-format` | `clang-format -i` | Nein | ❌ Nicht im PATH |
| `clang-tidy` | lint | `.clang-tidy` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `cmake` | build | `CMakeLists.txt` | `cmake --build build --parallel` | Ja | ✅ Aktiv |
| `shellcheck` | lint | `codex_setup.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |
| `ruff` | lint | `pyproject.toml` | `ruff check --output-format=concise .` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **80.0 %**
- **Identifizierte Lücken:**
  - ⚠️ Lane für clang-format vorhanden (clang-format -i), aber Werkzeug nicht im PATH
  - ⚠️ clang-tidy deklariert (.clang-tidy), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `clang-format, clang-tidy, cmake, cpp, python, shell`
- **Lanes:** `ruff check --output-format=concise ., mypy --no-error-summary --no-pretty, clang-format -i, cmake --build build --parallel, shellcheck **/*.sh`
