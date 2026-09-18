# Benchmark & Lücken-Audit: [mitchellh/libxev](https://github.com/mitchellh/libxev)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Zig | **Framework:** Libxev | **Tier:** Sehr viel
- **Commit:** `9ce8e8e6ff89e583258a7f8e7adeeeaeae8611bf`
- **Beispieldatei:** `website/next.config.js`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 9.0 ms | 10.0 ms | 9.5 ms | 10.1 ms | [0] |
| **post-tool-use** | 952.4 ms | 925.5 ms | 922.6 ms | 940.0 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 961.5 ms | 935.5 ms | 932.1 ms | 950.1 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit`
