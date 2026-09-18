# Benchmark & Lücken-Audit: [akka/akka-core](https://github.com/akka/akka-core)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Scala | **Framework:** Akka | **Tier:** Sehr viel
- **Commit:** `e2441c7ae1b0e500ac30a121863cc7e54d919a79`
- **Beispieldatei:** `akka-remote/src/test/resources/ssl/gen-artery-nodes.example.com.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 9.6 ms | 9.5 ms | 12.0 ms | [0] |
| **post-tool-use** | 127.5 ms | 132.5 ms | 126.3 ms | 133.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 139.5 ms | 142.5 ms | 135.9 ms | 144.5 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
