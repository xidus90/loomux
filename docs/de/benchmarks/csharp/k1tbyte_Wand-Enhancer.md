# Benchmark & Lücken-Audit: [k1tbyte/Wand-Enhancer](https://github.com/k1tbyte/Wand-Enhancer)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C# | **Framework:** WPF | **Tier:** Sehr viel
- **Commit:** `c6ae7a3ad49388ab9ad0ef6e06d235b59cc573cb`
- **Beispieldatei:** `WandEnhancer/Patches/devtools-f12.js`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 10.0 ms | 10.0 ms | 10.5 ms | [0] |
| **post-tool-use** | 935.1 ms | 970.2 ms | 923.7 ms | 1056.4 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 945.6 ms | 980.2 ms | 934.2 ms | 1066.4 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `dotnet-test` | test | `Wand-Enhancer.sln` | `*keine*` | Ja | ⚠️ Fehlt in Loomux |
| `shellcheck` | lint | `build.sh` | `shellcheck **/*.sh` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **50.0 %**
- **Identifizierte Lücken:**
  - ⚠️ dotnet-test deklariert (Wand-Enhancer.sln), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `biome, html, shell, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit, npx htmlhint "**/*.html", shellcheck **/*.sh, set -eu, script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd), repo_root=$(dirname "$script_dir"), script_path="$repo_root/scripts/validate-release-metadata.ps1", if command -v cygpath >/dev/null 2>&1; then, script_path=$(cygpath -w "$script_path"), fi, powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$script_path"`
