# Durchsicht: `feature/artefakte-nach-lebensdauer` gegen loomux

Nachtrag #20 der Fusions-Spec verlangt vor 4e, den nie gemergten Zweig von
ultra-brain Commit für Commit gegen `internal/brain/index` und die
Registerschreibung zu lesen: Was loomux nicht schon anders löst, wird ein
Fix-PR oder eine eigene Zeile, der Rest fällt mit Begründung weg.

- **Quelle:** ultra-brain, Zweig `feature/artefakte-nach-lebensdauer` (44
  Commits vor `master`, Spitze `33d34ed`), Spec
  `2026-09-11-artefakte-nach-lebensdauer-design.md` auf
  `docs/artefakte-nach-lebensdauer`.
- **Gelesen gegen:** loomux `origin/master` `a7805df8` (v4.0.0), am 2026-09-27.
- **Nicht Zeile für Zeile gelesen:** die Diffs der Umzugscommits `de95053`,
  `80e2248`, `e33174c` und `33d34ed`, die Commits zu `pkg/vcs`, zum Katalog des
  eigenen Wikis (`9b1e175`, `079b8c6`) und zu `.gitattributes`; gelesen sind
  ihre Betreffe und die Spec. Sollte eine der Entscheidungen unten für den Bau
  fallen, liest dessen Plan diese Commits vollständig.

## Ergebnis

Zwei Themen löst loomux schon, eines fällt weg, keines braucht einen Fix-PR.
Sechs Themen sind Entscheidungen des Nutzers; drei davon hängen an der ersten
oder an der zweiten.

| # | Thema (Commits) | loomux heute | Verdikt |
|---|---|---|---|
| 1 | `graph.json` und `layout.json` ins Zustandsverzeichnis, nur noch der Wurzelkatalog (`5dc9275`, `47c5d0e`, `5e0e4eb`, `68ce190`, `489645a`, `009f5a1`, `9b1e175`, `079b8c6`) | Wie ultra-brain `master`: Ein schreibbarer Bereich bekommt einen Katalog je Verzeichnis, `graph.json` und das Register im Baum (`internal/brain/index/staging.go`, `ManifestDir` in `internal/config/manifest.go`, `WriteCatalogs` in `catalogwrite.go`); Zustandsverzeichnis und Tausch nur für schreibgeschützte Bereiche. Eine `layout.json` gibt es nicht | **Entscheidung:** ob schreibbare Bereiche nur noch den Wurzelkatalog schreiben und `graph.json` samt Wurzelkatalog ins Zustandsverzeichnis gehen. Dann wären `staging.go`, `internal/brain/graph/read.go` und `internal/brain/status/status.go` auf einen gemeinsamen Artefaktort umzustellen. Der Teil zu `layout.json` fällt weg |
| 2 | Der Indexlauf schreibt nur Geburten und Umbenennungen ins Register, keine Revision je Lauf (`5dd3d59`, `73579ab`) | `carryForward` schreibt bei jeder Hashänderung Hash und Revision + 1 fort (`internal/brain/index/reindex.go`). Das Argument der Spec, der Indexlauf schiebe eine Änderung am Prüftor vorbei, löst loomux anders: vor jedem Indexlauf läuft zwingend `reconcile` (`internal/cli/index.go`, `catchUpBeforeIndexing`) | **Entscheidung:** ob die Revision nur noch zählt, wie oft geprüft wurde. Zwei Rechner mit derselben Geschichte schreiben heute verschiedene Zeilen in ein versioniertes Register; loomux versioniert `_identities.tsv` an der eigenen Wurzel |
| 3 | Register mit Aliasen, Grabsteinen und Union-Merge (`1584af8`, `23d657e`, `e7b0d9f`, `7d4548d`) | `ReadIdentities` überschreibt einen doppelten Pfad still (`internal/brain/identity/identity.go`); verschwundene Dateien verlieren ihre Zeile, weil `carryForward` das Register nur aus den gefundenen Dokumenten baut; kein `merge=union` in `.gitattributes` | **Entscheidung:** ob das Register die Invariante „eine `doc_id` wird nie entfernt, Aliase erlaubt“ bekommt. Fällt die Entscheidung dafür, folgt ein Fix-PR in `internal/brain/identity` und `reindex.go` mit einem Test, in dem ein Register mit zwei Zeilen desselben Pfads nach `reindex` beide `doc_id`s behält |
| 4 | Mehrdeutige Umbenennung, verlorene oder doppelte Zeile (`441f7ce`, `4591b19`, `3bb6bab`, `8db9685`) | `MatchRenames` überträgt eine `doc_id` nur bei genau einem Kandidaten und genau einer Ankunft je Hash (`identity.go`) | **Löst loomux schon** |
| 5 | Bereichsübergreifende Registerauflösung in `approve` (Spec §5.6) | `approve` führt je Zeile ein Lese- und ein Schreibregister und kennt die Register aller registrierten Bereiche (`internal/brain/apply/resolve.go`, `place.go`) | **Löst loomux schon** |
| 6 | Bereichsauflösung aus einem verknüpften Worktree, ein Worktree-Lauf schreibt nur das Register des Zweigs (`afdf198`, `5f6e61a`, `5dffda8`, `4d35035`, `3e33b35`, `9ca743b`, `4044249`, `51d402f`, `88fc4a7`, `f0fe34e`) | `reindex` kennt keinen Worktree und indexiert immer den registrierten Pfad, also den Hauptcheckout; Worktree-Wissen hat nur die Schreibschranke (`internal/brain/guard`) | **Entscheidung**, abhängig von #3: ob `reindex` aus einem verknüpften Worktree dessen Bereich über das gemeinsame Git-Verzeichnis auflöst und nur das Register des Zweigs schreibt. Kein Fehler, sondern eine neue Fähigkeit |
| 7 | Ein Worktree-Lauf schreibt nicht in den Hauptcheckout; ein Bereichspfad mit abschließendem Trenner (`785e3b8`, `f19aaad`, `adb42d1`, `cdb2531`) | Beide Fehler entstanden in der Worktree-Auflösung des Zweigs (`pkg/config/places.go`), die loomux nicht hat; der Indexlauf säubert Bereichs- und Wikipfad schon (`reindex.go`, `walk.go`) | **Fällt weg.** Wird #6 gebaut, gehören die Tests aus `adb42d1` und `785e3b8` in dessen Plan |
| 8 | Frühere Indexausgabe aus dem Baum räumen (`909295a`, `de95053`, `b68813f`, `80e2248`, `720d1f8`, `3a2885d`, `e33174c`, `33d34ed`) | Nichts zu räumen, solange die Artefakte im Baum bleiben | **Entscheidung**, hängt an #1: Wird die Verlegung beschlossen, räumt ein eigener Schritt des Umzugs die alten Kataloge und `graph.json`. Dabei gelten die Lehren aus `80e2248` und `33d34ed`: den Katalog des Bündels stehen lassen, erst nach dem Schreiben des Registers löschen, bei einem Abbruch jede schon entfernte Datei nennen |
| 9 | Die Schreibschranke verweigert `_identities.tsv` im Bündel, der Registercommit im Tresor (Spec §5.5, §6; `c5b978e`, `ad4b992`) | Die Schranke nennt `_identities.tsv` nicht; ein Agent darf das Register eines schreibbaren Bündels per Edit ändern. Einen isolierten Commit gibt es schon (`internal/brain/vcs/commit.go`) | **Entscheidung:** ob die Schranke jedes Schreiben auf eine Datei dieses Namens in einem registrierten Bereich verweigert und ob `reindex` Registeränderungen im Tresor über `internal/brain/vcs` committet |

## Offen

Die sechs Entscheidungen (#1, #2, #3, #6, #8, #9) trifft der Nutzer. Keine
davon hält 4e auf, außer der Nutzer will das Register vor der Umstellung der
Wirte versioniert und mehrzweigig machen (#3).
