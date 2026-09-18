# Benchmark & Gap Audit: [nitrogen/nitrogen](https://github.com/nitrogen/nitrogen)

- [← Back to Matrix](../matrix.md)
- **Language:** Erlang | **Framework:** Nitrogen | **Tier:** Sehr viel
- **Commit:** `dd9aa84beaf3451dd739de269e26a930aabf4d27`
- **Sample file:** `scripts/convert_plugin_to_rebar3.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 9.5 ms | 8.5 ms | 8.2 ms | 11.2 ms | [0] |
| **post-tool-use** | 65.3 ms | 60.3 ms | 60.2 ms | 63.7 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 74.8 ms | 68.8 ms | 68.4 ms | 74.9 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
