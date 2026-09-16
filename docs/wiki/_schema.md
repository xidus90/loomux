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

Nicht hier. `_identities.tsv` neben dieser Datei ist der leere Rahmen, den
`brain wiki init` anlegt; die `doc_id` jeder Seite dieses Bundles steht im
Register des **Bereichs**, also in `_identities.tsv` der Repo-Wurzel. Der
Grund: das Bundle ist kein eigener Bereich, sondern ein Teilbaum von
`project/ultra-brain`, und der Indexer führt ein Register je Bereich.
