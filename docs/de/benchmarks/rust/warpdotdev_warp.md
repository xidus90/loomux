# Benchmark & Lücken-Audit: [warpdotdev/warp](https://github.com/warpdotdev/warp)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Rust | **Framework:** Warp | **Tier:** Sehr viel
- **Commit:** `a0f5eb31a2ba46e46898f41d8c0e256ae7d20dda`
- **Beispieldatei:** `app/build.rs`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.0 ms | 8.6 ms | 8.0 ms | 9.0 ms | [0] |
| **post-tool-use** | 36.0 ms | 32.6 ms | 32.0 ms | 35.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 46.0 ms | 41.0 ms | 40.6 ms | 44.1 ms | [0] |

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
