# Benchmark & Gap Audit: [k1tbyte/Wand-Enhancer](https://github.com/k1tbyte/Wand-Enhancer)

- [← Back to Matrix](../matrix.md)
- **Language:** C# | **Framework:** WPF | **Tier:** Sehr viel
- **Commit:** `c6ae7a3ad49388ab9ad0ef6e06d235b59cc573cb`
- **Sample file:** `WandEnhancer/Patches/devtools-f12.js`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 10.0 ms | 10.0 ms | 10.5 ms | [0] |
| **post-tool-use** | 935.1 ms | 970.2 ms | 923.7 ms | 1056.4 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 945.6 ms | 980.2 ms | 934.2 ms | 1066.4 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `dotnet-test` | test | `Wand-Enhancer.sln` | `*none*` | Yes | ⚠️ Missing in Loomux |
| `shellcheck` | lint | `build.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **50.0 %**
- **Identified Gaps:**
  - ⚠️ dotnet-test deklariert (Wand-Enhancer.sln), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `biome, html, shell, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit, npx htmlhint "**/*.html", shellcheck **/*.sh, set -eu, script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd), repo_root=$(dirname "$script_dir"), script_path="$repo_root/scripts/validate-release-metadata.ps1", if command -v cygpath >/dev/null 2>&1; then, script_path=$(cygpath -w "$script_path"), fi, powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$script_path"`
