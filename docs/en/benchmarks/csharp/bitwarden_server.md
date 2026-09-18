# Benchmark & Gap Audit: [bitwarden/server](https://github.com/bitwarden/server)

- [← Back to Matrix](../matrix.md)
- **Language:** C# | **Framework:** SignalR | **Tier:** Sehr viel
- **Commit:** `8aa4813cb93125fc77f82ebae935b3a724c7951a`
- **Sample file:** `bitwarden_license/src/Scim/build.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 9.5 ms | 9.0 ms | 9.5 ms | [0] |
| **post-tool-use** | 73.6 ms | 69.5 ms | 69.1 ms | 71.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 85.6 ms | 79.0 ms | 78.1 ms | 80.5 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `docker, shell`
- **Lanes:** `shellcheck **/*.sh`
