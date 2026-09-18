# Benchmark & Lücken-Audit: [vaadin/framework](https://github.com/vaadin/framework)

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Sprache:** Java | **Framework:** Vaadin | **Tier:** Sehr viel
- **Commit:** `f7d61fc6d0de0f8e37c97eab85b7e554b24a7114`
- **Beispieldatei:** `scripts/cleanWhitespace.sh`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 12.0 ms | 9.0 ms | 9.0 ms | 9.5 ms | [0] |
| **post-tool-use** | 98.4 ms | 104.4 ms | 98.1 ms | 108.1 ms | [2] |
| **graph build** | n/a | n/a | n/a | n/a | n/a (non-Go) |
| **Gesamt** | 110.4 ms | 113.9 ms | 107.1 ms | 117.1 ms | [0, 2] |

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `mvn-test` | test | `pom.xml` | `*keine*` | Nein | ⚠️ Fehlt in Loomux |

- **Abdeckungsquote:** **0.0 %**
- **Identifizierte Lücken:**
  - ⚠️ mvn-test deklariert (pom.xml), aber keine Lane in Loomux vorhanden

## 3. Erkannte Stacks & Lanes

- **Stacks:** `shell`
- **Lanes:** `shellcheck **/*.sh`
