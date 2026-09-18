# Benchmark & Lücken-Audit: [dani-garcia/vaultwarden](https://github.com/dani-garcia/vaultwarden)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Rust | **Framework:** Rocket | **Tier:** Sehr viel
- **Commit:** `cc67d644f62605cb46f4d16c4a2eed1a861cc8bb`
- **Beispieldatei:** `build.rs`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.5 ms | 8.5 ms | 8.5 ms | 9.0 ms | [0] |
| **post-tool-use** | 29.5 ms | 27.5 ms | 27.0 ms | 28.1 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 41.0 ms | 36.0 ms | 35.5 ms | 37.1 ms | [0] |

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

- **Stacks:** `docker, rust, shell`
- **Lanes:** `shellcheck **/*.sh, cargo clippy -- -D warnings, cargo fmt --check`
