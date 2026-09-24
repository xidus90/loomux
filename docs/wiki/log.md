# Protokoll

> Jede Änderung an einer Seite endet hier mit einer Zeile: Datum, Seite,
> was sich geändert hat und aus welcher Quelle.

- 2026-09-16 — Bündel aus `ultra-brain` übernommen: 24 Inhaltsseiten und das
  Regelwerk; Kataloge, Protokoll, Audit und Register neu angelegt.
- 2026-09-17 — `entities/brain-daemon.md`: datiert, dass die Seite bis zum
  Umzug gilt; loomux fragt keinen brain-Daemon und spricht nur qmds Daemon über
  HTTP an. Quelle: Code von loomux (`internal/brain`).
- 2026-09-17 — `entities/qmd.md`: der Aufruf über die Kommandozeile datiert;
  loomux fragt qmds MCP-Daemon, die Reihenfolge trägt der Score nur auf den
  beiden rerankerfreien Wegen. Quelle: Code von loomux (`internal/brain`).
- 2026-09-17 — `sources/architektur-spec.md`: Konfliktkasten gesetzt,
  `open_conflicts` auf 1 — der `content_hash` der umgezogenen Spec stimmt nicht
  mehr mit dem verdichteten Stand. Quelle:
  `docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md`.
- 2026-09-17 — `syntheses/warum-fast-die-vorgabe-bleibt.md`: der Beleg zur
  Entscheidung 46 zeigt auf seinen neuen Ort im Bereich `project/loomux`.
  Quelle: `docs/.superpowers/bench-ub/entscheidung-46.md`.
- 2026-09-17 — `topics/brain-maintenance.md`: die Scheibe heißt seit dem Umzug
  Stufe 3, und in Go ist davon nichts gebaut. Quelle: Fusions-Spec, Stufenplan.
- 2026-09-17 — `topics/datenmodell-und-bereiche.md`: Manifestnamen und Orte von
  loomux ergänzt, die Lint-Regel `wrong-direction` als nur bis zum Umzug
  gültig datiert. Quelle: Code von loomux (`internal/config`, `internal/brain`).
- 2026-09-17 — `topics/scheiben-und-abnahme.md`: Scheiben heißen Stufen, die
  Abnahme einer Stufe samt Mutationsrunde ab 1b ergänzt. Quelle: Fusions-Spec,
  Abnahme.
- 2026-09-17 — `topics/suche-und-profile.md`: die drei Ketten am MCP-Daemon und
  die Messung von `brain search --profile fast` vom 2026-09-16 ergänzt. Quelle:
  `docs/de/benchmarks.md`.
- 2026-09-17 — `topics/wiki-schicht.md`: `loomux lint` und `loomux wiki-gate`
  mit ihren fünf Regeln beschrieben, der Rest der alten Liste als nicht gebaut
  benannt. Quelle: Code von loomux (`internal/brain/wiki`).
- 2026-09-17 — `topics/datenmodell-und-bereiche.md`: die Aussage, ein Wiki im
  Code-Repo sei keine Option, bis zum Umzug datiert.
- 2026-09-24 — alle Seiten mit `sources[]`: jede `doc_id` auf die ID gesetzt,
  die das Register von `project/loomux` für ihre `resource` führt. Die alten
  IDs stammten aus dem Register von `ultra-brain`, und die Quellen unter
  `docs/.superpowers/` standen in keinem Register; `reconcile` konnte eine
  geänderte Quelle so keiner Seite zuordnen. `content_hash` und `revision`
  bleiben der Stand, aus dem die Seiten verdichtet wurden. Quelle:
  `_identities.tsv` nach `loomux reindex`.
