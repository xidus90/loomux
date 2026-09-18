# Benchmark & Lücken-Audit: [dart-lang/shelf](https://github.com/dart-lang/shelf)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Dart | **Framework:** Shelf | **Tier:** Sehr viel
- **Commit:** `e5c8dc663bf1325ad8f997c4a2387923d37a90d9`
- **Beispieldatei:** `pkgs/_shelf_compliance/tool/run_tests_with_dotnet.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 9.5 ms | 9.0 ms | 9.7 ms | [0] |
| **post-tool-use** | 74.1 ms | 64.4 ms | 63.7 ms | 65.4 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 86.1 ms | 73.4 ms | 73.2 ms | 75.1 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
