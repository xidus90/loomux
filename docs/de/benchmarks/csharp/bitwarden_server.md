# Benchmark & Lücken-Audit: [bitwarden/server](https://github.com/bitwarden/server)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** C# | **Framework:** SignalR | **Tier:** Sehr viel
- **Commit:** `8aa4813cb93125fc77f82ebae935b3a724c7951a`
- **Beispieldatei:** `bitwarden_license/src/Scim/build.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 9.5 ms | 9.0 ms | 9.5 ms | [0] |
| **post-tool-use** | 73.6 ms | 69.5 ms | 69.1 ms | 71.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 85.6 ms | 79.0 ms | 78.1 ms | 80.5 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `docker, shell`
- **Lanes:** `shellcheck **/*.sh`
