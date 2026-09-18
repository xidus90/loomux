# Benchmark & Gap Audit: [micronaut-projects/micronaut-profiles](https://github.com/micronaut-projects/micronaut-profiles)

- [← Back to Matrix](../matrix.md)
- **Language:** Groovy | **Framework:** Micronaut | **Tier:** Sehr viel
- **Commit:** `891799331c9be570f2231bea25fd9c6d9a462bbd`
- **Sample file:** `base/features/aws-api-gateway-graal/skeleton/deploy.sh`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 9.7 ms | 9.5 ms | 10.0 ms | [0] |
| **post-tool-use** | 87.2 ms | 86.5 ms | 82.6 ms | 91.1 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 98.7 ms | 96.5 ms | 92.2 ms | 100.6 ms | [0, 2] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `shellcheck` | lint | `test-profile-tests.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `travis-build.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **100.0 %**

## 3. Detected Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
