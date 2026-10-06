# Protokoll

> Jede Änderung an einer Seite endet hier mit einer Zeile: Datum, Seite,
> was sich geändert hat und aus welcher Quelle. Neueste zuerst, je Tag eine
> Überschrift (OKF §9); `loomux approve` schreibt seine Zeilen genauso.

## 2026-10-06

- 2026-10-06 — `topics/brain-maintenance.md`: Konflikt zu `--reject`
  aufgelöst, die Seite war veraltet. Eine Ablehnung schiebt `revision` und
  `content_hash` in `sources[]` und den Registern vor; die Referenz tat das
  nicht. Kasten entfernt, `open_conflicts` wieder 0. Quelle:
  `docs/de/cli-reference.md` (`loomux approve`).
- 2026-10-06 — `topics/code-graph.md`: Konflikt zu G4c aufgelöst, veraltet war
  die Nutzerdoku. `docs/de/architecture.md` und `docs/en/architecture.md`
  nennen als offen nur noch G5b bis G5d. Kasten entfernt, `open_conflicts`
  wieder 0. Quelle: `docs/de/architecture.md`, `docs/de/cli-reference.md`
  (`check stop`).
- 2026-10-06 — `topics/suche-und-profile.md`: Konflikt zur Tokenmenge in
  `status` aufgelöst; der Absatz nennt sie als Absicht des Vorgängers, gebaut
  ist sie nicht. Kasten entfernt, `open_conflicts` wieder 0. Quelle:
  `docs/de/cli-reference.md` (`loomux brain status`).

## 2026-10-05

- 2026-10-05 — alle Inhaltsseiten mit `sources[]`: die Quellen zeigen auf die
  Nutzerdoku unter `docs/de/` oder, bei den Synthesen, auf Wiki-Seiten, weil
  die Arbeitspapiere ins Archiv-Release gehen; die Vorgänger werden nicht mehr
  genannt. Wo eine Seite einer neuen Quelle widerspricht, steht ein
  Konfliktkasten (`brain-maintenance.md`, `code-graph.md`,
  `suche-und-profile.md`). `_schema.md` nennt die neuen Rohquellen. Quelle:
  Code von loomux (`.loomux/config.toml`, `[index]`).
- 2026-10-05 — `topics/architektur-grundsaetze.md`: sagt jetzt ausdrücklich,
  dass keine Nutzerdoku unter `docs/de/` die Grundsätze, die Vertrauenskette
  und die Fehlerstellen belegt; sie sind aus dem Architektur-Design und der
  Fusions-Spec des Vorgängers verdichtet, die heute in den Arbeitspapieren des
  Archiv-Release `archive/parity-recordings` liegen. `docs/de/architecture.md`
  bleibt die formale Quelle. Quelle: `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md`
  in den Arbeitspapieren des Archiv-Release `archive/parity-recordings`.
- 2026-10-05 — `topics/datenmodell-und-bereiche.md`: der Konfliktkasten zum
  Ort des Wikis entfällt, `open_conflicts` wieder 0. Die Aufteilung
  Wissens-Vault gegen Code-Repo und ihre vier Gründe sind als Stand bis zum
  Umzug am 2026-09-16 datiert; seit dem Umzug liegt das Wiki im Repo. Quelle:
  `docs/de/cli-reference.md` (`loomux area add`), `docs/de/configuration.md`
  (`[layout] wiki`).
- 2026-10-05 — `topics/wiki-schicht.md`: der Konfliktkasten zu den Achsen von
  `loomux brain check` entfällt, `open_conflicts` wieder 0; es sind zwei
  Achsen, `okf` und `house`, und `house` trägt die Föderationsregeln. Quelle:
  `docs/de/cli-reference.md` (`loomux brain check`).
- 2026-10-05 — `topics/code-graph.md`: nennt das Vorbild nur noch über
  `NOTICE.md`. Quelle: `internal/notices/NOTICE.md`.

## 2026-09-24

- 2026-09-24 — `topics/architektur-grundsaetze.md`: Konflikt zu Grundsatz 5
  aufgelöst — beides gilt in verschiedenem Kontext: die Wissensschicht
  degradiert, Wächter und Tore scheitern geschlossen. Kasten entfernt,
  `open_conflicts` auf 0. Die Entscheidung steht zuerst in der Quelle:
  `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md` in den
  Arbeitspapieren des Archiv-Release `archive/parity-recordings`,
  „Fehlerverhalten“. Der Titel „Die Stufen und ihre Abnahme“ ist in allen
  Verweisen nachgezogen.
- 2026-09-24 — `audit.md`: die Fälle `loomux-2026-09-24-5565` und
  `loomux-2026-09-24-947f` zweimal abgelehnt; sie meldeten Änderungen der
  Fusions-Spec, gegen die beide Seiten schon verdichtet waren.

- 2026-09-24 — `topics/datenmodell-und-bereiche.md`,
  `topics/suche-und-profile.md`: verweisen auf die beiden neuen Seiten.
- 2026-09-24 — `topics/schreibschranke.md`: neu. Registry, Manifest und
  Policy, verknüpfte Worktrees und `open.toml`. Quelle:
  `docs/.superpowers/specs/2026-09-15-loomux-schranke-worktrees-design.md`,
  `docs/.superpowers/specs/2026-09-16-loomux-schranke-samerepo-design.md` und
  `docs/.superpowers/specs/2026-09-24-schranke-open-toml-design.md` in den
  Arbeitspapieren des Archiv-Release `archive/parity-recordings`; Code von
  loomux (`internal/brain/guard`, `internal/config`).
- 2026-09-24 — `topics/code-graph.md`: neu. Extraktion, Rang und
  Blast-Radius, `graph`-Befehle und die sieben `graph_*`-Werkzeuge; wo die Spec
  überholt ist, steht der gebaute Stand. Quelle:
  `docs/.superpowers/specs/2026-09-14-loomux-code-graph-design.md` in den
  Arbeitspapieren des Archiv-Release `archive/parity-recordings`, Code von
  loomux (`internal/code`), Migrationsplan.
- 2026-09-24 — `topics/wiki-schicht.md`: gegen die Spec der Stufe 3
  verdichtet — zwei Regelsätze im Lint, die Wiki-Werkzeuge, der Commit einer
  Freigabe, die Skills noch nicht in loomux; `implemented_in` auf `db780a0`.
  Quelle: `docs/.superpowers/specs/2026-09-19-loomux-stufe-3-design.md` und
  `docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md` in den
  Arbeitspapieren des Archiv-Release `archive/parity-recordings`.
- 2026-09-24 — `topics/brain-maintenance.md`: gegen die Specs der Stufen 3
  und 4 verdichtet — Teilstufen 3a bis 3c, Auffangdurchgang vor `reindex`,
  Prüfzentrum, Ablauf einer Freigabe, der geerbte Fehler von `--reject`, die
  Kandidaten des Merge-Auslösers. Quelle: wie oben.
- 2026-09-24 — `topics/architektur-grundsaetze.md`: Grundsätze der Fusion
  ergänzt, das lokale Modell als Absicht datiert, `realization` auf
  `in_progress`; Konfliktkasten zu Grundsatz 5 gegen das Fehlerverhalten der
  Fusion, `open_conflicts` auf 1. Quelle:
  `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md` in den
  Arbeitspapieren des Archiv-Release `archive/parity-recordings`.
- 2026-09-24 — `topics/scheiben-und-abnahme.md`: heißt „Die Stufen und ihre
  Abnahme“; die Scheiben des Altprojekts durch die Stufen von loomux ersetzt,
  mit Fertig-Bedingungen, Stand jeder Stufe, Reihenfolge und Paritätsnachweis.
  Quelle: `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md` in den
  Arbeitspapieren des Archiv-Release `archive/parity-recordings`,
  Migrationsplan.
- 2026-09-24 — `sources/`: alle elf Source-Seiten gelöscht. Die Rohquellen
  liegen im selben Repo und stehen im Register; die Seiten zitieren sie direkt
  (`_schema.md`). Die Links auf die gelöschten Seiten sind Text geworden.
- 2026-09-24 — `topics/wiki-schicht.md`: `loomux lint --scope` und
  `loomux brain check` aus Stufe 3c ergänzt; die Regeln, die es in Go „noch
  nicht“ gab, sind gebaut. Quelle: Code von loomux (`internal/brain/wiki`,
  `internal/brain/check`), Migrationsplan.
- 2026-09-24 — `topics/brain-maintenance.md`: Stufe 3 ist gebaut
  (`reconcile`, `cases`, `case`, `approve`, Nachholen im Dienst),
  `realization` auf `in_progress`; offen bleiben der Prüfvorschlag des lokalen
  Modells und der `post-merge`-Hook (Stufe 4). Quelle: Code von loomux
  (`internal/brain/maintenance`, `internal/brain/apply`), Migrationsplan.
- 2026-09-24 — `log.md`: nach Tagen gruppiert, neueste zuerst, wie OKF §9 es
  verlangt. Quelle: OKF v0.2, §9.
- 2026-09-24 — alle Seiten mit `sources[]`: jede `doc_id` auf die ID gesetzt,
  die das Register von `project/loomux` für ihre `resource` führt. Die alten
  IDs stammten aus dem Register des Vorgängers, und die Quellen unter
  `docs/.superpowers/` (heute in den Arbeitspapieren des Archiv-Release
  `archive/parity-recordings`) standen in keinem Register; `reconcile` konnte eine
  geänderte Quelle so keiner Seite zuordnen. `content_hash` und `revision`
  bleiben der Stand, aus dem die Seiten verdichtet wurden. Quelle:
  `_identities.tsv` nach `loomux reindex`.

## 2026-09-17

- 2026-09-17 — `topics/datenmodell-und-bereiche.md`: die Aussage, ein Wiki im
  Code-Repo sei keine Option, bis zum Umzug datiert.
- 2026-09-17 — `topics/wiki-schicht.md`: `loomux lint` und `loomux wiki-gate`
  mit ihren fünf Regeln beschrieben, der Rest der alten Liste als nicht gebaut
  benannt. Quelle: Code von loomux (`internal/brain/wiki`).
- 2026-09-17 — `topics/suche-und-profile.md`: die drei Ketten am MCP-Daemon und
  die Messung von `brain search --profile fast` vom 2026-09-16 ergänzt. Quelle:
  `docs/de/benchmarks.md`.
- 2026-09-17 — `topics/scheiben-und-abnahme.md`: Scheiben heißen Stufen, die
  Abnahme einer Stufe samt Mutationsrunde ab 1b ergänzt. Quelle: Fusions-Spec,
  Abnahme.
- 2026-09-17 — `topics/datenmodell-und-bereiche.md`: Manifestnamen und Orte von
  loomux ergänzt, die Lint-Regel `wrong-direction` als nur bis zum Umzug
  gültig datiert. Quelle: Code von loomux (`internal/config`, `internal/brain`).
- 2026-09-17 — `topics/brain-maintenance.md`: die Scheibe heißt seit dem Umzug
  Stufe 3, und in Go ist davon nichts gebaut. Quelle: Fusions-Spec, Stufenplan.
- 2026-09-17 — `syntheses/warum-fast-die-vorgabe-bleibt.md`: der Beleg zur
  Entscheidung 46 zeigt auf seinen neuen Ort im Bereich `project/loomux`.
  Quelle: das Messprotokoll zur Entscheidung 46 des Vorgängers
  (`entscheidung-46.md` in den Arbeitspapieren des Archiv-Release
  `archive/parity-recordings`).
- 2026-09-17 — `sources/architektur-spec.md`: Konfliktkasten gesetzt,
  `open_conflicts` auf 1 — der `content_hash` der umgezogenen Spec stimmt nicht
  mehr mit dem verdichteten Stand. Quelle: die Architektur-Spec des Vorgängers
  vom 2026-08-18 (heute in den Arbeitspapieren des Archiv-Release
  `archive/parity-recordings`).
- 2026-09-17 — `entities/qmd.md`: der Aufruf über die Kommandozeile datiert;
  loomux fragt qmds MCP-Daemon, die Reihenfolge trägt der Score nur auf den
  beiden rerankerfreien Wegen. Quelle: Code von loomux (`internal/brain`).
- 2026-09-17 — `entities/brain-daemon.md`: datiert, dass die Seite bis zum
  Umzug gilt; loomux fragt keinen brain-Daemon und spricht nur qmds Daemon über
  HTTP an. Quelle: Code von loomux (`internal/brain`).

## 2026-09-16

- 2026-09-16 — Bündel des Vorgängers übernommen: 24 Inhaltsseiten und das
  Regelwerk; Kataloge, Protokoll, Audit und Register neu angelegt.
