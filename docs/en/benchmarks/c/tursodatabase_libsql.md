# Benchmark & Gap Audit: [tursodatabase/libsql](https://github.com/tursodatabase/libsql)

- [← Back to Matrix](../matrix.md)
- **Language:** C | **Framework:** SQLite | **Tier:** Sehr viel
- **Commit:** `d6c75af6353bb1c34985399608e37cd272a35aa1`
- **Sample file:** `bindings/c/build.rs`

## 1. Performance & Latencies

| Component | Cold (1st run) | Warm Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 16.5 ms | 10.5 ms | 10.5 ms | 11.2 ms | [0] |
| **post-tool-use** | 49.1 ms | 45.9 ms | 44.0 ms | 47.0 ms | [0] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Total** | 65.6 ms | 56.4 ms | 54.5 ms | 58.2 ms | [0] |

## 2. Check & Gap Audit (Gap Analysis)

| Tool | Category | Native in Project | Loomux Lane | In PATH | Status |
|---|---|---|---|---|---|
| `cargo-clippy` | lint | `Cargo.toml` | `*none*` | No | ⚠️ Missing in Loomux |
| `cargo-test` | test | `Cargo.toml` | `*none*` | No | ⚠️ Missing in Loomux |
| `cargo-fmt` | format | `Cargo.toml` | `*none*` | No | ⚠️ Missing in Loomux |
| `shellcheck` | lint | `docker-entrypoint.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |
| `shellcheck` | lint | `docker-wrapper.sh` | `shellcheck **/*.sh` | Yes | ✅ Active |

- **Coverage Rate:** **40.0 %**
- **Identified Gaps:**
  - ⚠️ cargo-clippy deklariert (Cargo.toml), aber keine Lane in Loomux vorhanden
  - ⚠️ cargo-test deklariert (Cargo.toml), aber keine Lane in Loomux vorhanden
  - ⚠️ cargo-fmt deklariert (Cargo.toml), aber keine Lane in Loomux vorhanden

## 3. Detected Stacks & Lanes

- **Stacks:** `docker, rust, shell`
- **Lanes:** `shellcheck **/*.sh, cargo clippy -- -D warnings, cargo fmt --check`
