# Benchmark & Lücken-Audit: [nitrogen/nitrogen](https://github.com/nitrogen/nitrogen)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Erlang | **Framework:** Nitrogen | **Tier:** Sehr viel
- **Commit:** `dd9aa84beaf3451dd739de269e26a930aabf4d27`
- **Beispieldatei:** `scripts/convert_plugin_to_rebar3.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 9.5 ms | 8.5 ms | 8.2 ms | 11.2 ms | [0] |
| **post-tool-use** | 65.3 ms | 60.3 ms | 60.2 ms | 63.7 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 74.8 ms | 68.8 ms | 68.4 ms | 74.9 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
