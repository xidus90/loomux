# Benchmark & Gap Audit: [metabrainz/musicbrainz-server](https://github.com/metabrainz/musicbrainz-server)

- [← Back to Matrix](../matrix.md)
- **Language:** Perl | **Framework:** Catalyst | **Tier:** Sehr viel
- **Commit:** `15b555fbd838b3cc69e6847fdb21f109b5f14816`
- **Sample file:** `admin/CalculateRelatedTags.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.5 ms | 9.0 ms | 8.5 ms | 10.0 ms | [0] |
| **post-tool-use** | 79.7 ms | 86.1 ms | 80.6 ms | 94.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 90.2 ms | 95.1 ms | 89.1 ms | 104.0 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `eslint` | lint | `package.json` | `*none*` | No | ⚠️ Missing in Loomux |
| `shellcheck` | lint | `upgrade.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **50.0 %**
- **Identified Gaps:**
  - ⚠️ eslint deklariert (package.json), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
