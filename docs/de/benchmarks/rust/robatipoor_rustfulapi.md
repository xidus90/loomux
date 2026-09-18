# Benchmark & Lücken-Audit: [robatipoor/rustfulapi](https://github.com/robatipoor/rustfulapi)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Rust | **Framework:** SeaORM | **Tier:** Sehr viel
- **Commit:** `dd9a9cdcab3826c65367e9a496fe4e6f583e2435`
- **Beispieldatei:** `src/bin/app.rs`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.7 ms | 9.0 ms | 8.5 ms | 9.5 ms | [0] |
| **post-tool-use** | 29.5 ms | 28.5 ms | 27.0 ms | 32.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 40.2 ms | 38.0 ms | 35.5 ms | 41.5 ms | [0] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `cargo-clippy` | lint | `Cargo.toml` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `cargo-test` | test | `Cargo.toml` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `cargo-fmt` | format | `Cargo.toml` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **0.0 %**
- **Identifizierte Lücken:**
  - ⚠️ cargo-clippy deklariert (Cargo.toml), aber keine Lane in Loomux vorhanden
  - ⚠️ cargo-test deklariert (Cargo.toml), aber keine Lane in Loomux vorhanden
  - ⚠️ cargo-fmt deklariert (Cargo.toml), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `rust, shell`
- **Lanes:** `shellcheck **/*.sh, cargo clippy -- -D warnings, cargo fmt --check`
