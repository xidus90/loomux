# Benchmark & Lücken-Audit: [yewstack/yew](https://github.com/yewstack/yew)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Rust | **Framework:** Yew | **Tier:** Sehr viel
- **Commit:** `bfa6c19af971084f9547495388bde9aceeb81aaa`
- **Beispieldatei:** `examples/actix_ssr_router/src/bin/ssr_router_hydrate.rs`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.0 ms | 8.0 ms | 8.0 ms | 9.0 ms | [0] |
| **post-tool-use** | 27.5 ms | 25.0 ms | 25.0 ms | 25.5 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 37.5 ms | 33.0 ms | 33.0 ms | 34.5 ms | [0] |

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

- **Stacks:** `css, html, rust, shell, typescript`
- **Lanes:** `npx eslint --cache ., npx tsc --noEmit, npx stylelint "**/*.{css,scss}", npx htmlhint "**/*.html", shellcheck **/*.sh, cargo clippy -- -D warnings, cargo fmt --check`
