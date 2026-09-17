# Schema dieses Bundles

Vor jeder Wiki-Arbeit gelesen. Sechs Regeln (Architektur §9.1):

1. Nur die KI schreibt hier.
2. Rohquellen sind unantastbar.
3. Jede Seite ist vernetzt.
4. Widersprüche werden markiert, nie aufgelöst.
5. Jede Änderung endet mit einem Log-Eintrag.
6. Verdichtet wird immer gegen die Originalquelle, nie gegen eine ältere
   Zusammenfassung.

## Die vier Seitentypen

- `Source` — eine Quelle, verdichtet; trägt `sources[]` mit `doc_id`,
  `content_hash` und `revision`.
- `Topic` — ein Thema über mehrere Quellen hinweg.
- `Entity` — eine Person, ein Werkzeug, ein Ort, ein Begriff.
- `Synthesis` — eine eigene Ableitung aus mehreren Seiten.

Diese vier bleiben stehen, auch unter loomux. Sie sind die **zweite Achse**
(Typkatalog §3.5): sie sagen, *wie* eine Seite entstanden ist, nicht worüber
sie handelt. Der Wiki-Lint von loomux fragt dafür nicht das Manifest, sondern
eine eigene Liste von zwölf Typen; `Topic` steht darin als `topic` und geht
durch, `Source`, `Entity` und `Synthesis` melden je eine Warnung
`missing-type: unknown document type`. Das hält nichts auf: `wiki-gate` bricht
nur bei `Error` ab. Die Warnungen verschwinden, sobald der Wiki-Lint das
Manifest fragt: `config.Manifest.KnowsType` kennt die vier Herkunftstypen
bereits, eine Zeile `[wiki] types` braucht es für sie nicht. Umgeschrieben wird
kein Typ; insbesondere ist `Entity` nicht auf `Concept` abzubilden, das der
Typkatalog in §3.4 ausdrücklich ablehnt.

## Die Form eines Konflikts

Fest und maschinell auffindbar (Architektur §9.3); `open_conflicts: <n>` in
der Frontmatter zählt die Kästen, und der Lint hält die Zahl dagegen.

```markdown
> [!conflict] <Kurztitel>
> [Quelle A](/pfad/a.md) sagt X.
> [Quelle B](/pfad/b.md) sagt Y.
> Beide Stände bleiben stehen. Entscheidung offen.
```

Aufgelöst wird die Quelle, nie der Kasten: wer nur die Markierung löscht,
findet denselben Widerspruch im nächsten Durchlauf wieder.

## Wo das Identitätsregister liegt

Nicht hier. `_identities.tsv` neben dieser Datei ist nur der leere Rahmen,
den in ultra-brain `brain wiki init` anlegte; loomux kennt den Befehl nicht,
beim Umzug wurde die Kopfzeile von Hand gesetzt. Die `doc_id` jeder Seite
dieses Bundles gehört in das Register des **Bereichs**, also in
`_identities.tsv` der Repo-Wurzel. Diese Datei schreibt erst `reindex` ab
Stufe 3; bis dahin gibt es sie nicht, und keine Seite ist dort eingetragen. Der
Grund für den Ort: das Bundle ist kein eigener Bereich, sondern ein Teilbaum
von `project/loomux`, und der Indexer führt ein Register je Bereich.
