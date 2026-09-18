# Benchmark & Lücken-Audit: [ruby-grape/grape](https://github.com/ruby-grape/grape)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Ruby | **Framework:** Grape | **Tier:** Sehr viel
- **Commit:** `f6fff5a0a1dd125caa055ab85bed72e5ab6b794b`
- **Beispieldatei:** `docker/entrypoint.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 10.0 ms | 9.5 ms | 9.0 ms | 10.5 ms | [0] |
| **post-tool-use** | 74.2 ms | 64.6 ms | 64.5 ms | 68.8 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 84.2 ms | 74.0 ms | 73.6 ms | 79.3 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `rubocop` | lint | `Gemfile` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |
| `rspec` | test | `Gemfile` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **0.0 %**
- **Identifizierte Lücken:**
  - ⚠️ rubocop deklariert (Gemfile), aber keine Lane in Loomux vorhanden
  - ⚠️ rspec deklariert (Gemfile), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `docker, shell`
- **Lanes:** `shellcheck **/*.sh`
