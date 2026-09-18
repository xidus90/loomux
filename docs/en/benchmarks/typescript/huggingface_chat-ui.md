# Benchmark & Gap Audit: [huggingface/chat-ui](https://github.com/huggingface/chat-ui)

- [← Back to Matrix](../matrix.md)
- **Language:** TypeScript | **Framework:** SvelteKit | **Tier:** Sehr viel
- **Commit:** `7b2e38bbe747227872496e5ffaa0fc80e8b046c4`
- **Sample file:** `playwright.config.ts`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 13.0 ms | 10.2 ms | 9.5 ms | 10.5 ms | [0] |
| **post-tool-use** | 939.1 ms | 944.5 ms | 937.7 ms | 946.1 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 952.1 ms | 954.0 ms | 947.9 ms | 956.6 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `entrypoint.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `eslint` | lint | `package.json` | `npx eslint --cache .` | No | ❌ Missing from PATH |
| `prettier` | format | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `tsc` | typecheck | `package.json` | `npx tsc --noEmit` | No | ❌ Missing from PATH |
| `vitest` | test | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `tsc` | typecheck | `tsconfig.json` | `npx tsc --noEmit` | No | ❌ Missing from PATH |

- **Coverage Rate:** **66.7 %**
- **Identified Gaps:**
  - ⚠️ Lane für eslint vorhanden (npx eslint --cache .), aber Werkzeug nicht im PATH
  - ⚠️ prettier deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH
  - ⚠️ vitest deklariert (package.json), aber keine Lane in Loomux vorhanden
  - ⚠️ Lane für tsc vorhanden (npx tsc --noEmit), aber Werkzeug nicht im PATH

## 3. Detected Stacks & Lanes

- **Stacks:** `docker, html, shell, svelte, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit, npx svelte-check, npx htmlhint "**/*.html", shellcheck **/*.sh`
