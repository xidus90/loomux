# Benchmark & Lücken-Audit: [pandas-dev/pandas](https://github.com/pandas-dev/pandas)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Python | **Framework:** Pandas | **Tier:** Sehr viel
- **Commit:** `cf85d5d203826cab8e164e3ccbe755e49f328d99`
- **Beispieldatei:** `asv_bench/benchmarks/__init__.py`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 9.5 ms | 9.0 ms | 9.5 ms | [0] |
| **post-tool-use** | 100.7 ms | 94.8 ms | 90.6 ms | 99.7 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 112.2 ms | 104.3 ms | 100.1 ms | 108.7 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `ruff` | lint | `pyproject.toml` | `ruff check --output-format=concise .` | Ja | ✅ Aktiv |
| `mypy` | typecheck | `pyproject.toml` | `*keine*` | Ja | ⚠️ Fehlt in Loomux |
| `pytest` | test | `pyproject.toml` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **33.3 %**
- **Identifizierte Lücken:**
  - ⚠️ mypy deklariert (pyproject.toml), aber keine Lane in Loomux vorhanden
  - ⚠️ pytest deklariert (pyproject.toml), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `cpp, meson, pyright, python`
- **Lanes:** `ruff check --output-format=concise ., pyright, clang-format -i, cmake --build build --parallel`
