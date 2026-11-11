---
type: Entity
title: zvec-grep
description: Fremde Suche aus ripgrep, BM25 und Vektoren — gegen loomux graph grep gemessen und als Ersatz für exakte Suche verworfen.
open_conflicts: 0
sources:
  - id: benchmarks
    resource: brain://project/loomux/docs/de/benchmarks.md
    doc_id: 01M47W91R5MPENKN7N34H6QTNT
    content_hash: "sha256:5c86c86e56a3171d4c836e06fbe77ce39154833c26cb2846a3e054774e2ef78c"
    revision: 16
---

Fremdes Werkzeug (github.com/zvec-ai/zvec-grep, Befehl `zg`), Rust,
Apache-2.0, aktiv gepflegt (v0.3.0 vom 2026-11-03). Es vereint drei Wege
hinter einer Oberfläche: ein mitgeliefertes ripgrep ohne Index (`--rg`),
BM25 (`--fts`) und Vektorsuche mit einem lokalen Modell (`--vector`, Vorgabe
`local/potion-code-16m-v2`), dazu einen MCP-Server und Installer für Agenten.
Der Index liegt unter `.zvec-grep/` im Projekt.

## Gemessen gegen loomux graph grep

Am 2026-11-11 auf dem loomux-Repo
([Benchmarks](../../de/benchmarks.md), Abschnitt „Vergleich mit
zvec-grep“), ein Lauf:

- Exakt ist loomux schneller: Mit Literal braucht `zg --rg` 95–118 ms,
  loomux im Prüfmodus 49–71 ms und mit `--trust-index` 20–45 ms. Nur beim
  Muster ohne Literal liegt `zg --rg` mit 124 ms knapp vor dem Prüfmodus
  (137 ms) und hinter `--trust-index` (103 ms).
- `zg --fts` ist nach Rang sortiert und nicht vollständig; ein Aufruf kostet
  gegen den laufenden Server rund 200 ms, ohne Server 2,3 s.
- Der Index braucht 24 s statt 2,6 s (loomux) und 0,5 s (tgrep).

Als Ersatz für `graph grep` oder tgrep taugt es daher nicht. Was es darüber
hinaus kann, ist die Suche nach Bedeutung; das gehört neben `brain_search`,
nicht neben grep.
