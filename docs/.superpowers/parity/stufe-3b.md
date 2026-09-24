# Paritätsakte Stufe 3b

**Quelle:** ultra-brain `loomux-3-source` (`3cc72d2`), derselbe Tag wie in
Stufe 3a.
**Spec:** `docs/.superpowers/specs/2026-09-19-loomux-stufe-3-design.md`,
Abschnitt „3b im Einzelnen“
**Plan:** `docs/.superpowers/plans/2026-09-22-loomux-stufe-3b.md`

`loomux cases`, `loomux case <id>` und `loomux approve` entscheiden die Fälle,
die `reconcile` seit 3a ins Prüfzentrum legt. Referenz ist die Python-Form
(`src/brain/maintenance/apply.py`, `vcs.py`, `cli.py`); die alte Go-Form
(`pkg/maintenance/approve.go`, `cmd/brain/approve.go`, `cmd/brain/main.go`)
ist gelesen, aber nicht Maßstab. Alle Zeilenangaben unten beziehen sich auf
die Dateien im Stand des Tags.

## Vorbedingungen, festgestellt am 2026-09-22

| Frage | Antwort | Belegstelle |
|---|---|---|
| Zeigt der Tag auf den erwarteten Commit? | **Ja**, `3cc72d2` | `git -C ../ultra-brain rev-parse --short loomux-3-source` |
| Hat sich `src/brain/maintenance` oder `src/brain/cli.py` seit dem Tag bewegt? | **Nein**, die Ausgabe ist leer | `git -C ../ultra-brain log --oneline loomux-3-source..master -- src/brain/maintenance src/brain/cli.py` |
| Stimmen die gelesenen Dateien mit dem Tag überein? | **Ja.** `HEAD` von ultra-brain ist `3cc72d2`, und `pkg/maintenance`, `cmd/brain`, `src/brain` sind im Arbeitsbaum unverändert | `git -C ../ultra-brain status --short -- pkg/maintenance cmd/brain src/brain` (leer) |

**Folgerung:** der Tag bleibt, wo er ist. Die Zeilenangaben dieser Akte gelten
für `3cc72d2`.

## Abweichungen der Go-Form, gelesen

Die Spec hat den Gleichheitsnachweis am 2026-09-22 für `approve.go` als
negativ erklärt: die Go-Form läuft nicht durch den Fallsatz. Ihre
Abweichungen stehen hier einmal, gegen den Code gelesen. **Für jeden Punkt
gilt die Python-Form.** Wo die Spec etwas anderes sagt als der Code, steht
das beim Punkt als „Spec: … / Code: …“.

### 1. `approve.go` ist keine Portierung von `apply.py`

`approve.go` hat 640 Zeilen, `apply.py` 1.378. Es fehlt:

- **Die Schreibschranke.** `_preflight` (`apply.py:467-484`) prüft `log.md`,
  `audit.md`, das Register und das Fallverzeichnis, bevor ein Byte
  geschrieben wird. `_gate` (`:556-633`) hält jeden Schreib- und Löschaufruf
  auf: kein Glied des Pfades ein Link, jedes Glied im Tresor aufgelöst, kein
  Gerüstname, auch nicht als 8.3-Kurzname (`:629`). Dazu gehören `_anchor`
  (`:449-464`) und `_normalised` (`:635-653`). Go schreibt ungeprüft mit
  `os.WriteFile` und `os.RemoveAll` (`approve.go:174-175`, `:456`, `:606`,
  `:621`).
  Spec: `apply.py:556-635` für `_gate`/`_preflight` / Code: `_preflight`
  steht bei `:467-484`, `_gate` bei `:556-633`.
- **Die Prüfung des Ziels.** `_check_target` (`apply.py:841-847`) lehnt ein
  leeres Ziel ab und prüft die Unicode-Kategorien Cc, Cf, Zl und Zp
  (`:222`). `checkTarget` (`approve.go:106-128`) lässt ein leeres Ziel durch
  und prüft nur `unicode.IsControl`, also Cc. `..` erkennt es nur als
  Teilstring `../` oder als ganzes Ziel (`:117`), `a/..` geht durch. Die
  Prüfung nach dem Auflösen und die Link-Suche aus `_target`
  (`apply.py:887-899`) fehlen ganz.
- **`[layout].review`, zur Hälfte.** Go prüft, dass der Schlüssel gesetzt ist
  (`approve.go:432-434`), aber nicht, ob der Wert absolut ist oder `..`
  enthält (`apply.py:671-675`, `_layout` bei `:656-676`).
  Spec: „fehlt“ / Code: fehlt zur Hälfte.
- **`_is_bundle`** (`apply.py:437-446`, gerufen bei `:388`). Python lehnt ein
  Wiki außerhalb des Tresors ab, wenn es kein `_schema.md` trägt. Go prüft
  das nicht und fügt einen relativen `wiki_path` an die Tresorwurzel an
  (`approve.go:155-158`). Python nimmt den Pfad aus der Registry, wie er
  steht (`apply.py:434`).
- **Die Quellsuche über alle Bereiche.** `_resolve_sources`
  (`apply.py:155-186`) sucht jede `doc_id` im Register jedes registrierten
  Bereichs. Der Quellwächter (`:961-966`) und der Registervorschub
  (`_advance_register`, `:1181-1214`) arbeiten mit dem Pfad dieses Bereichs.
  Go liest nur `_identities.tsv` des Tresors (`approve.go:495-496`), setzt den
  Quellpfad an die Tresorwurzel (`:500`) und schiebt nur dieses Register vor
  (`:609-617`).
- **Die Liste der berührten Dateien.** Python führt `place.touched`
  (`apply.py:295`, `_touch` bei `:511-529`) und hängt sie an jeden Abbruch
  (`:357`), tresorrelativ (`_relative`, `:1311-1315`). Go füllt `Dirty` nur
  auf den Pfaden „Zielseite bewegt“, „Quelle bewegt“ und „verworfen“
  (`approve.go:482-491`, `:507-516`, `:558-561`), mit absoluten Pfaden. Alle
  Schreibfehler verwirft es mit `_ =` (`:174-175`, `:456`, `:481`, `:506`,
  `:606`, `:614`, `:621`), bei `:556` ohne Zuweisung.
- **`_unrecorded`, zur Hälfte.** Python schreibt einen Auditblock nur, wenn
  sich der Vermerk geändert hat (`apply.py:984-1010`), und das auf allen drei
  Abbruchpfaden (`:930`, `:970`, `:1051`). Go tut es auf den beiden
  Bewegt-Pfaden (`approve.go:484`, `:509`), bei der Verwerfung aber nicht
  (`:556-557`): ein wiederholtes `approve` hängt jedes Mal einen weiteren
  Block an.
  Spec: „fehlt“ / Code: fehlt nur bei der Verwerfung.
- **Die Entscheidung `defer` im Kern.** `Approve` nimmt `defer` an
  (`approve.go:398`, `:410-416`). `DECISIONS` kennt nur `approve` und
  `reject` (`apply.py:189`, `:329-330`); zurückgestellt wird in der CLI
  (`cli.py:1337-1348`). Über die CLI ist der Go-Pfad nicht erreichbar
  (`cmd/brain/approve.go:110-122`).

Texte, die abweichen:

- **`_safe`.** Python lässt Buchstaben, Ziffern und ` .,:;/-_()…·` durch,
  ersetzt alles andere durch `·` und kürzt auf 200 Zeichen
  (`apply.py:216-218`, `:1224-1264`). Go lässt Buchstaben, Ziffern und
  ` -_./:` durch, ersetzt durch `_` und kürzt auf 120 (`approve.go:70-89`).
- **Die drei Vermerke** in `case.toml`:
  - Zielseite bewegt: `_MOVED_NOTE` (`apply.py:201-204`) gegen „Zielseite hat
    sich seit der Fallbildung geändert“ (`approve.go:478`).
  - Verwerfung: `_REFUSED_NOTE` (`apply.py:205`) gegen „keine Behauptung hielt
    der Evidenzprüfung stand“ (`approve.go:550`).
  - Verworfene Nachbesserung: `_AMEND_NOTE` (`apply.py:206`) gegen „Prüfung der
    geänderten Fassung scheiterte an der Evidenz“ (`approve.go:552`).

  Der Vermerk „Quelle bewegt“ ist auf beiden Seiten gleich (`apply.py:968`,
  `approve.go:503`).
- **Die Zeile `- entschieden:` einer Verwerfung.** Python schreibt „verworfen
  (Evidenzbindung)“ (`apply.py:1057`), Go „nicht geschrieben
  (Evidenzprüfung gescheitert)“ (`approve.go:557`). Die Meldung von
  `ProposalRefused` nennt in Python die Fall-ID (`apply.py:1062`), in Go den
  Pfad des Vorschlags (`approve.go:559`).
  Spec: „die Zeile `- entschieden:` einer Ablehnung“ / Code: die Ablehnung
  (`--reject`) schreibt auf beiden Seiten „abgelehnt durch …“
  (`apply.py:710`, `approve.go:455`). Es weicht die **Verwerfung** ab.
- **Die Zahl der Behauptungen auf den Bewegt-Pfaden.** Python zählt die
  Behauptungen des Vorschlags in den Auditblock (`_claims`, `apply.py:938`,
  `:978`). Go übergibt `nil` (`approve.go:485`, `:510`) und schreibt darum
  „0 Behauptung(en)“.
- **Zeitstempel.** Python schreibt `now.isoformat()` in den Auditblock und
  in die Frontmatter (`apply.py:1290`, `:1160`), also `+00:00` und
  Mikrosekunden. Go schreibt `time.RFC3339` (`approve.go:181`,
  `frontmatter.go:70`), also `Z` ohne Bruchteil.

### 2. Die Commit-Seite ist in Go unsicher

`commitPaths` (`approve.go:294-394`) ruft dieselbe Plumbing wie
`vcs.commit_paths` (`vcs.py:108-228`). Es fehlt:

- **Die Weigerung bei laufendem Rebase oder Merge.**
  `_reject_operation_in_progress` (`vcs.py:265-275`, gerufen bei `:185`,
  Namen bei `:69`).
- **Die Auswertung des Vergleichs-und-Tauschs.** Go übergibt `update-ref`
  den alten Wert (`approve.go:385-388`), ignoriert aber den Exitcode (`:391`).
  Ein verlorener Tausch meldet darum Erfolg, mit einem Commit, auf den kein
  Ref zeigt. Python liest den Ref neu und hebt `RefMoved` (`vcs.py:223-227`).
  Vor dem Tausch prüft es außerdem, ob HEAD noch auf demselben Ref steht
  (`:220-221`), und `_commit` versucht es genau einmal neu
  (`apply.py:1351-1359`).
  Spec: „ohne Vergleich-und-Tausch beim `update-ref`“ / Code: der Tausch wird
  gestellt, sein Ergebnis wird verworfen.
- **`created=False`.** Python vergleicht den neuen Baum mit dem des alten
  Commits und legt keinen leeren Commit an (`vcs.py:209-211`). `_report`
  meldet dann „nothing to commit … the case belongs back in the queue“
  (`apply.py:1373-1377`). Go ruft `commit-tree` in jedem Fall
  (`approve.go:378-383`) und legt einen leeren Commit an.
- **`:(literal)` und der Fehler bei unverfolgtem Pfad.** `_expand` fragt
  `ls-files` mit `:(literal)` (`vcs.py:247`, `_literal` bei `:254-262`) und
  hebt „path to remove is not tracked“, wenn nichts verfolgt ist (`:249-250`).
  Der Commit scheitert dann mit einer Warnung (`apply.py:1360-1364`). Go fragt
  ohne Präfix (`approve.go:350`) und überspringt ein unverfolgtes
  Fallverzeichnis still (`:354`). Der Commit landet dann ohne die Löschung.
- **Die Prüfung der Pfade.** `_reject_path_outside` (`vcs.py:278-282`) und
  die Weigerung bei einem Pfad, der zugleich hinzugefügt und entfernt wird
  (`:172-179`). Python lässt außerdem jeden geschriebenen Pfad außerhalb des
  Tresors aus dem Commit (`_staged`, `apply.py:683-695`). Go reicht ihn als
  `../…` an `update-index` weiter (`approve.go:341-342`), und der Aufruf
  scheitert still.
- **Die Exitcodes.** `read-tree`, `update-index`, `write-tree`, `commit-tree`
  und `update-ref` laufen mit verworfenem Fehler (`approve.go:337`, `:345`,
  `:361`, `:370`, `:382`, `:391`). Python läuft jeden Aufruf durch `_must`
  (`vcs.py:297`).
- **Die Warnung ohne Repository.** Go meldet „no git repository in the vault;
  <Ausgabe von git>“ (`approve.go:307`). Python meldet „<Fall>: no git
  repository in the vault; written but not committed“ bzw. „decision recorded
  but not committed“ (`apply.py:1346`, `:1371-1372`).

### 3. Zwei Wege zu `ProposalRefused`

- **Die gescheiterte Belegprüfung** (`_refuse`, `apply.py:1032-1062`)
  schreibt den Vermerk nach `case.toml`, bei einem eigenen Vorschlag
  `manual = true` (`:1049`), und einen Auditblock, wenn der Vermerk neu ist
  (`:1051-1061`).
- **Ein Diff ohne Zaun, ein Hunk ohne Kopf, überlappende oder unpassende
  Hunks** (`_collect`, `_hunks`, `_patch`, `apply.py:1065-1140`) heben
  ebenfalls `ProposalRefused`, schreiben aber **nichts**.
- Go wirft auf dem ersten Weg `ProposalRefused` (`approve.go:549-561`), auf
  dem zweiten nur einen einfachen Fehler (`:574`, `:579`, `:588`). Der Exit ist
  in beiden Fällen 1. Die Meldungen von `ParseUnifiedDiff` und `ApplyHunks`
  sind nicht gegen `_hunks` und `_patch` gelesen.

### 4. Nach einem geschriebenen `approve`

- Python fährt `_technical_update` (`cli.py:1432-1516`, gerufen bei `:1428`).
  Zuerst läuft `reconcile` (`:1484`). Scheitert es, warnt es und hält an
  (`:1485-1501`). Neue Fälle meldet es auf stderr (`:1502-1507`). Dann läuft
  `reindex` (`:1509`), und nach Erfolg wird der Daemon zum Neuladen gebeten
  (`:1510`).
- Go fährt nur `reindex` (`cmd/brain/approve.go:47-56`). Fehlt die Registry,
  kehrt es still zurück (`:49-51`). Es bittet keinen Daemon um ein Neuladen.
- Der Exit ist in beiden Formen 0 (`cli.py:1429`,
  `cmd/brain/approve.go:161`).

### 5. `brain case` und `--package`

- Python liest `case` mit argparse (`cli.py:601-611`). `--package` und
  `--state-dir` stehen vor oder nach der ID. Ein fehlendes Argument und ein
  unbekanntes Argument enden mit Exit 2.
- Go liest mit `flag.Parse` (`cmd/brain/main.go:288`). Das Parsen hört bei der
  ersten Position auf. Ein `--package` nach der ID wird mit „--package must be
  typed before the case identifier“ und Exit 1 abgelehnt (`:298-302`). Jedes
  andere Argument nach der ID, auch `--state-dir`, wird **still ignoriert**.
  Eine fehlende ID endet mit Exit 1 (`:292-295`). `approve` hat dasselbe
  Problem nicht, dort liest `parseInterspersed` die Flaggen auf beiden Seiten
  (`cmd/brain/approve.go:68`).

## Geerbt

**`--reject` schiebt Revision und Hash nicht vor.** Eingetragen in
`OFFENE_AUFGABEN.md:213` („Entwurfslücke 2“): die Ablehnung schreibt
`audit.md`, löscht das Fallverzeichnis und committet, schiebt aber weder die
`sources[]` der Seite noch das Register vor. Der nächste Abgleich eröffnet
darum denselben Fall wieder.

- Der Fehler ist nicht nur einer der Go-Form. Pythons `_reject`
  (`apply.py:698-727`) ruft weder `_advance` noch `_advance_register`, genau
  wie Gos Ablehnungspfad (`approve.go:448-468`).
- Der Eintrag nennt `pkg/maintenance/approve.go:375`. Am Tag liegt diese Zeile
  in `commitPaths`; der Ablehnungspfad steht bei `:448-468`.
- `OFFENE_AUFGABEN.md` ist in ultra-brain **nicht versioniert**
  (`git status`: `??`). Der Eintrag gehört also nicht zum Stand des Tags,
  sondern zum Arbeitsbaum vom 2026-09-22.

3b übernimmt das Verhalten unverändert (Spec, Befunde 3b). Eine Heilung ist
ein Nachtrag der Fusions-Spec, nicht Teil dieser Stufe.

**Eine schließende Frontmatter-Zeile mit Leerraum verliert die alte
Frontmatter.** `_advance` sucht den Block mit `apply._FRONTMATTER`
(`\A---\n(.*?)\n---[ \t]*\n`, `apply.py:224`) und liest das YAML über
`parse_frontmatter`, dessen Muster (`document.py:10`) nach `---` keinen
Leerraum zulässt. Steht die schließende Zeile als `--- `, passt das erste
Muster und das zweite nicht: `parse_frontmatter` liefert `{}`, und `_advance`
schreibt eine Frontmatter nur aus `generated` und `verified`. Jeder andere
Schlüssel der Seite ist danach weg (Golden `s13-closing-blanks`).

3b bildet das absichtlich nach (`frontmatter.go`, `advanceBlock` und
`documentBlock`), weil die Python-Form gilt. Die Heilung ist ein Nachtrag der
Fusions-Spec und eine Entscheidung des Nutzers.

## Abweichungsliste

| Fall | Alt | Neu | Begründung |
|---|---|---|---|
| Commit bei laufendem Rebase oder Merge | Go-Form: committet (`approve.go:294-394` prüft nichts). Python: Weigerung (`vcs.py:265-275`) | `vcs.CommitPaths` weigert sich mit „<pfad> exists: finish the rebase or merge first“, wörtlich wie Python | Python gilt. Ein Cherry-Pick (`CHERRY_PICK_HEAD`) wird **nicht** geprüft, auf beiden Seiten nicht (`vcs.py:69`); übernommen, nicht geheilt |
| Verlorener Vergleich-und-Tausch | Go-Form: Exitcode von `update-ref` verworfen, Erfolg gemeldet (`approve.go:391`) | Ref wird neu gelesen; weicht er von `old` ab, `vcs.ErrRefMoved`, sonst ein Fehler mit git's stderr. HEAD wird vor dem Tausch neu gelesen, ein Zweigwechsel ist ebenfalls `ErrRefMoved` | Python gilt (`vcs.py:220-227`) |
| Unveränderter Baum | Go-Form: `commit-tree` in jedem Fall, leerer Commit (`approve.go:378-383`) | `Commit{Head: old, Created: false}`, kein Commit | Python gilt (`vcs.py:209-211`) |
| Löschpfade | Go-Form: `ls-files` ohne `:(literal)`, unverfolgter Pfad still übersprungen (`approve.go:350-354`) | `:(literal)`, ungekürzte `-z`-Liste, unverfolgter Pfad ⇒ „path to remove is not tracked: …“ | Python gilt (`vcs.py:231-262`) |
| Pfadprüfung | Go-Form: `../…` geht an `update-index` (`approve.go:341-342`) | absolut oder `..` ⇒ `vcs.ErrOutsideRepository`; Pfad in `add` und `remove` ⇒ „path is both added and removed: …“ | Python gilt (`vcs.py:172-179`, `:278-282`). **Neu gegen Python:** der Doppelt-Vergleich läuft über die bereinigte Schreibweise (`a\b` = `a/b`), und an git geht die bereinigte Form; Python vergleicht und übergibt die Rohzeichenketten. Strenger, nie lockerer |
| Exitcodes | Go-Form: `read-tree`, `update-index`, `write-tree`, `commit-tree`, `update-ref` mit verworfenem Fehler | jeder Aufruf geprüft; der erste Fehler bricht ab und wird mit git's stderr gemeldet | Python gilt (`_must`, `vcs.py:297-308`) |
| Wortlaut der Commit-Fehler | Python: `update-ref <ref> failed: <stderr>` (`vcs.py:227`), `HEAD left <ref> while committing` (`:221`), `<ref> moved away from <alt> while committing` (`:226`) | `git update-ref <ref> <neu> <alt> failed: <stderr>` (der gemeinsame Weg von `must`); `the branch moved while committing: HEAD left <ref>` und `the branch moved while committing: <ref> moved away from <alt>` (`ErrRefMoved` vorangestellt). Die Warnung nach einem zweiten verlorenen Tausch trägt diesen Text in Klammern: `<fall>: written, but not committed (the branch moved while committing: …)`, Python `(… while committing)` (`apply.py:1364`) | **Abweichung im Wortlaut, stehengelassen.** Befund, Exit und Rahmen der Warnung sind gleich; stderr vergleicht der Fallsatz nicht, und kein Fall verliert den Tausch |
| Ort des Scratch-Index | Python: `scratch` **ist** die Indexdatei. Go-Form: `<scratch>/scratch_index` | `<scratch>/index`, vor Gebrauch gelöscht, absolut gemacht | Schnittstelle der Stufe (Plan, Task 3). Eine unbrauchbare Datei (etwa ein nicht leeres Verzeichnis) ist ein Fehler wie in Python (`vcs.py:193-197`); ein **leeres** Verzeichnis räumt `os.Remove` still weg |
| Zeilenenden der Commit-Nachricht | Python auf Windows: `subprocess.run(text=True, input=…)` schreibt `\n` als `\r\n` nach `commit-tree`; gemessen 2026-09-22, `cat-file commit HEAD` endete auf `Line one\r\n\r\nBody.` | Die Nachricht geht Byte für Byte an `commit-tree`, `\n` bleibt `\n` | Plattformfehler der Referenz; auf POSIX schreibt Python LF. `test_the_commit_message_is_the_one_given` liest mit `text=True` zurück und kann den Unterschied nicht sehen |
| Leerraum in `evidence:`, Behauptungs- und Segmentkopf | Go-Form: `\s`/`\S` (`evidence.go:38`, `:37`, `:45`), in Go nur `[\t\n\f\r ]`. `evidence: D1\v` zitiert Segment `D1\v`, `evidence:<NBSP>D1` Segment `<NBSP>D1`, `## B1\x1c — x` ist eine Behauptung `B1\x1c` | eine Klasse aus den 29 Zeichen von `pytext.IsSpace` und ihre Verneinung; `D1` wird gelesen, `## B1\x1c — x` ist ein Kopf außerhalb des Musters | Python gilt (`evidence.py:107`, `:108`, `:558`; `\s` eines str-Musters ist Unicode-weit). Gemessen 2026-09-22 mit dem Python-Lauf, festgehalten in `unicode_test.go` |
| `strip()` | Go-Form: `strings.TrimSpace` (Info-Zeichenkette des Zauns, Abschnittsrumpf, Kopfzeile, Zitat in `CheckEvidence`) lässt `\x1c`–`\x1f` stehen. Ein Schließer ```` ```\x1c ```` ist darum keiner, und der Zaun läuft weiter | `pytext.Strip` an allen zwölf Stellen | Python gilt (`str.strip()`, `evidence.py:227`, `:400`, `:418`, `:430`, `:433`, `:435`, `:482`–`:491`, `:518`) |
| Ziffern | Go-Form: `\d` in der Listenmarke und in `segments:` ist ASCII (`evidence.go:42`, `:44`). `١. item` gilt als Prosa, `segments: ١` als fehlende Angabe | `\p{Nd}`; der Wert wie `int()`, eine Zahl jenseits von `int` ist ungleich jeder Segmentzahl | Python gilt (`\d` ist Unicode-Nd, `evidence.py:140`, `:550`, `:635`). Go-Tabellen können Python 3.14 um eine Unicode-Version nachhängen; keine Eingabe, die dieses System schreibt, erreicht den Unterschied |
| Zahl in „package declares N segments …“ | Go-Form: `%d` des mit `strconv.Atoi` gelesenen Werts (`evidence.go:400-403`): `02` wird `2`, ein Überlauf die größte `int` | die Gruppe, wie sie steht | Python gilt (`evidence.py:637`, formatiert `declared.group(1)`) |
| `FencedBlocks` auf CRLF | Python: `fenced_blocks` paart auf dem gefalteten Text, schneidet aber aus dem ungefalteten (`evidence.py:673-677`); gemessen 2026-09-22: ```` x\r\n```a\r\nbody\r\n```\r\n ```` ergibt `('a', 'a\r\nbod')` | **Neu gegen Python:** Go schneidet aus dem gefalteten Text, wie die Go-Form | Fehler der Referenz, nicht geheilt, sondern nicht übernommen. Der einzige Aufrufer (`apply.py:1069`) übergibt `claim.body`, und der ist schon gefaltet: auf jedem erreichbaren Weg sind beide gleich |
| Fehlerart beim Anwenden | Go-Form: nackte Fehler aus `ParseUnifiedDiff` und `ApplyHunks` (`patch.go:48`, `:67`, `:87`, `:90`, `:94`, `:98`) und aus der Zaunsuche (`approve.go:574`) | jede Weigerung ist `*apply.RefusedError`, der Text wörtlich der Python-Text hinter `{heading}: ` | Python gilt (`ProposalRefused`, `apply.py:1071`, `:1095`, `:1111`, `:1128`, `:1133`, `:1135`). Das Voranstellen der Überschrift ist Sache von `Approve`; die Meldungen aus `_patch` tragen in Python keine |
| Mehrere `diff`-Zäune in einer Behauptung | Plan (Task 5): nur der erste Zaun, `CollectDiff` gibt eine Zeichenkette | `CollectDiff` gibt den Rumpf **jedes** Zauns, dessen Info-Zeichenkette mit `diff` beginnt, in Reihenfolge; jeder wird für sich gelesen und muss mit einem eigenen Kopf beginnen | Python gilt (`_collect`, `apply.py:1069-1073`), die Go-Form tat dasselbe (`approve.go:566-582`). Ruling 2026-09-22: Planfehler, Schnittstelle `([]string, error)` |
| `ParseUnifiedDiff`: Schnitt und Zeilenenden | Go-Form: `strings.Trim(Normalised(diff), "\n")` (`patch.go:26`) | geteilt an `\n`, sonst nichts: ein abschließendes `\n` ist eine weitere leere Kontextzeile, ein `\r` bleibt in der Zeile | Python gilt (`_hunks`, `apply.py:1086`), gemessen 2026-09-22: `"@@ -10,2 +10,2 @@\n-a\n+b\n"` ergibt `old ['a', '']`. Auf dem echten Weg nicht erreichbar, `FencedBlocks` streift und faltet den Rumpf schon |
| Leerzeilen vor dem ersten Kopf | Go-Form: `strings.TrimSpace` (`patch.go:47`); eine Zeile aus `\x1c` wird verweigert | `pytext.Strip`; `\x1c` gilt als leer | Python gilt (`line.strip()`, `apply.py:1094`), gemessen 2026-09-22 |
| Zahlen im Hunk-Kopf | Go-Form: ASCII-`\d` und `strconv.Atoi` mit verworfenem Fehler (`patch.go:19`, `:31`, `:34`). `@@ -١٢,٠ +1 @@` ist kein Kopf; ein Überlauf wird zur größten `int`, gemessen: `@@ -99999999999999999999999 +1 @@` meldet Zeile `9223372036854775807`, mit `,0` Zeile `-9223372036854775808` | `\p{Nd}`, der Wert wie `int()` über `big.Int`; eine Zahl jenseits von `int` liegt hinter jedem Seitenende und wird in voller Länge gemeldet, mehrere davon nach ihrem vollen Wert geordnet | Python gilt (`\d` ist Unicode-Nd, `int()` ohne Obergrenze; `apply.py:225`, `:1089-1090`), gemessen 2026-09-22: Zeile `99999999999999999999999` bzw. `100000000000000000000000` |
| Reihenfolge der Hunks | Go-Form: `sort.Slice` (`patch.go:78`), nicht stabil | `sort.SliceStable` | Python gilt (`sorted` ist stabil, `apply.py:1126`). Nicht beobachtet: 14 reine Einfügungen an einem Anker blieben auch mit `sort.Slice` in Ordnung; der Test hält die Eigenschaft fest, nicht einen gemessenen Fehler |
| Ausgabe der Frontmatter | Go-Form: yaml.v3-Encoder mit `SetIndent(2)` (`frontmatter.go:144-147`): Folgen unter einem Schlüssel eingerückt, kein Umbruch bei 80 Spalten, Anführung nach YAML 1.2 | `apply/pyyaml_emit.go`, PyYAMLs Emitter (6.0.3) für das, was `safe_dump(sort_keys=False, allow_unicode=True, default_flow_style=False)` aus einem `safe_load`-Ergebnis macht: Folgen ohne Einrückung, Umbruch bei 80 Codepunkten, Anführung nach YAML 1.1 (`'2026-09-22T08:16:27.936837+00:00'`, `'yes'`), Kurzform `!!str` im 128er-Limit einfacher Schlüssel, leere Sammlungen als `[]`/`{}` | Python gilt (`apply.py:1177`). Gemessen 2026-09-22: 110 Eingaben durch `_advance` (46 Ausgaben, 56 Weigerungen, 8 Ausgaben, die der Port bewusst verweigert) (`internal/brain/apply/testdata/frontmatter`), dazu ein Differenzlauf über alle 10.580 Markdown-Seiten mit Frontmatter unter `#GIT` (Doppelte aus Worktrees und die eigenen Goldens eingeschlossen): 10.477 byte-gleich, 97 auf beiden Seiten verweigert, keine Abweichung außer den Weigerungen unten. Der Parser ist yaml.v3; seine Unterschiede zu PyYAMLs Scanner sind nur so weit abgedeckt, wie Goldens und Differenzlauf reichen |
| Lesen der Frontmatter | Go-Form: yaml.v3 löst nach YAML 1.2 auf: `yes`, `010`, `1:30` und ein ungequoteter Zeitstempel bleiben, wie sie stehen | Auflöser und Konstruktoren von PyYAMLs `SafeLoader` (YAML 1.1): `yes`/`on` sind Wahrheitswerte, `010` ist 8, `1:30` ist 90, `2026-01-01T00:00:00Z` wird `2026-01-01 00:00:00+00:00`; Schlüssel der obersten Ebene werden `str()` (`'1'`, `None`), gleiche Schlüssel nach Pythons `==` fallen zusammen (`1`, `true`, `1.0`), jedes NaN ist ein Schlüssel; roh nicht druckbare Zeichen verweigert wie `Reader.check_printable` | Python gilt (`document.py:31`, `:44`) |
| Tabulatoren in der Frontmatter | Go-Form: yaml.v3 liest einen Tabulator als Trenner (`a:\tb`, `c: "x"\t`, `a: [x,\ty]`) und schreibt die Seite | Geweigert wie PyYAML („found character '\t' that cannot start any token“): ein Tabulator ist nur Text in einem gequoteten Skalar, in einem Kommentar und im Rumpf eines Blockskalars ab dessen Einrückung (`pyyaml_tabs.go`, `scan_block_scalar` nachgemessen). Sonst ist er eine Weigerung, auch in einem ungequoteten Skalar (`a: x\ty`) und in einer Zeile nur aus Leerraum | Python gilt (`scanner.py`, `scan_to_next_token` überspringt nur Leerzeichen). Gemessen 2026-09-22: 25 Formen als `e23`–`e47`, die erlaubten in `s17`, `s18`, `s21`. Vor der Heilung schrieb der Port 15 dieser 25 Seiten. Zeilen zählt die Prüfung wie yaml.v3 und PyYAML an CRLF, CR, LF, NEL, LS und PS; mit nur LF lag eine Spanne hinter einem NEL, LS, PS oder einzelnen CR auf der falschen Zeile oder außerhalb des Textes (Absturz). Goldens `r-*`: 30, davon 9 Weigerungen (`.err`). CR, LS, NEL und PS je sieben (gequotet, Kommentar, Blockskalar, Tab im Blockrumpf, nackt, und gequotet wie nackt hinter dem Umbruch als Trenner); CRLF nur zwei (gequotet, nackt) |
| Zwei Frontmatter-Muster | Go-Form: ein Muster `\A---\r?\n(.*?)\r?\n---(?:\r?\n(.*))?\z` (`frontmatter.go:18`) für Block und Rumpf | wie Python zwei: `apply._FRONTMATTER` (`\A---\n(.*?)\n---[ \t]*\n`) bestimmt den Rumpf, `document._FRONTMATTER` (`\A---\r?\n(.*?)\r?\n---\r?\n`) das gelesene YAML | Python gilt, **Fehler der Referenz übernommen**, siehe „Geerbt“: eine schließende Zeile mit Leerraum (`--- `) verliert die alte Frontmatter (Golden `s13-closing-blanks`) |
| Revision in `sources[]` | Plan (Task 6): `SourceUpdate` ohne Revision, „`revision` + 1“ | `SourceUpdate.Revision` ist die Revision des Falls, der Eintrag bekommt `Revision+1`, wie in der Go-Form | Python gilt (`state.revision + 1`, `apply.py:1171`); die Revision der Seite zählt nicht. Die Goldens geben jedem Update Revision 7, die keine Seite trägt |
| `doc_id`, die keine Zeichenkette ist | Go-Form: `docIDNode.Value`, die rohe Schreibweise | `str()` des geladenen Skalars: `doc_id: true` trifft `True`, ein fehlendes `doc_id` trifft `None`; eine Liste oder Abbildung trifft nichts | Python gilt für jeden Skalar (`str(entry.get("doc_id"))`, `apply.py:1168`). **Neu gegen Python** bei einer Sammlung: Python vergleicht deren Repr (`str(['x'])` ist `"['x']"`), der Port lässt sie nie treffen. Unerreichbar, solange die `doc_id`s eines Falls ULIDs sind (`pyyaml.go`, `pythonStr`) |
| Weigerungen, **neu gegen Python** | Python schreibt eine Seite mit: einem Alias auf eine Sammlung oder einen Zeitstempel (mit Anker `&id001`, wo beide Vorkommen den Dump erreichen; ohne Anker, wo eines davon ersetzt wird, etwa `generated: *g`), einem expliziten Tag (`!!int "12"`), einem Merge-Schlüssel `<<`, dem Escape `\/` in doppelten Anführungszeichen, und einem Tabulator als erstem Zeichen der ersten Zeile eines Blockskalars (`a: |` gefolgt von `  <TAB>x`) | `AdvanceFrontmatter` verweigert alle fünf. Die ersten drei weist der Port selbst ab, die letzten beiden schon yaml.v3 („found unknown escape character“, „found a tab character where an indentation space is expected“). Ein Alias auf `None`, `str`, `bool`, `int` oder `float` wird wie in Python ausgeschrieben | In Wiki-Frontmatter nicht vorhanden (Differenzlauf: nur die eigenen Goldens `x01`–`x04`). Goldens `x01`–`x08`. Strenger: der Port schreibt in diesen Fällen nichts statt etwas anderem |
| Meldungen der Weigerung | Python: Text von PyYAML bzw. `ValueError` von `int()`/`datetime` (letzterer ist kein `ApplyError` und bricht `approve` mit Traceback ab) | Fehlertext von yaml.v3 bzw. ein eigener („invalid date …“); `the target page has no frontmatter to advance` und `frontmatter is not a mapping` wörtlich | Die Weigerung selbst ist gleich (alle 56 Weigerungs-Goldens, `e*` und `r-*`), nur ihr Text nicht. Das Voranstellen des Ziels ist Sache von `Approve` |
| `Safe`, Zeichenklassen | Go-Form: `unicode.IsLetter`/`IsDigit` und ` -_./:`, Ersatz `_`, Schnitt bei 120 (`approve.go:70-89`); `²`, `½`, `Ⅻ` (Kategorie `No`/`Nl`) werden ersetzt | `unicode.IsLetter`/`IsNumber` (ganz `N`) und ` .,:;/-_()…·`, Ersatz `·`, über 200 Zeichen die ersten 199 und `…` | Python gilt (`_safe`, `apply.py:216-218`, `:1256-1264`). Gemessen 2026-09-22: 21 Eingaben durch `_safe` (`internal/brain/apply/testdata/protocol.json`), alle gleich. Gos Tabellen sind Unicode 17.0, Pythons 3.14.7 `unicodedata` 16.0: ein Buchstabe, der erst in 17.0 kam, bleibt in Go und wird in Python ersetzt. Ungültiges UTF-8 wird je Byte ein `·`; Python kennt es im `str` nicht |
| Datum der `log.md`-Zeile | Go-Form: `now.Format("2006-01-02")` in der Zone des Aufrufers (`approve.go:194`) | `LogLine` datiert in UTC | Python gilt (`now.date()`, `apply.py:1270`, bei `now = datetime.now(UTC)`, `cli.py:1355`) |
| Platz der `log.md`-Zeile, **neu gegen Python** (2026-09-24) | Python: `_append` hängt die Zeile mit einer Leerzeile ans Ende (`apply.py:1301-1307`), wie den Auditblock | `LogInsert`: die Zeile kommt vor den ersten Eintrag, unter die Überschrift `## YYYY-MM-DD` des Tages (UTC); fehlt sie, entsteht davor eine neue Tagesgruppe. Die Zeile selbst ist unverändert `LogLine` | OKF §9 verlangt „a flat list of date-grouped entries, newest first“; die Referenz verletzt das. `audit.md` bleibt bei `Append`, OKF regelt die Datei nicht. `approve/success`, `amend`, `rebase` und `no-repo` weichen darum in `repo-a/wiki/log.md` ab; `TestCases3b` nimmt die Tagesüberschrift heraus und verlangt dann die Aufzeichnung |
| Kopf des Auditblocks | Go-Form: `time.RFC3339`, also `Z` ohne Bruchteil (`approve.go:181`) | `IsoFormat`: `+00:00` und Mikrosekunden, wenn nicht null | Python gilt (`now.isoformat()`, `apply.py:1290`); gemessen mit drei Zeiten in `protocol.json` |
| `Unrecorded` | Plan (Task 7): `Unrecorded(audit, block string)`, „verglichen wird der Block ohne Kopfzeile“ | `Unrecorded(last, note string) bool` ist `last != note`: der letzte Vermerk in `case.toml` gegen den neuen | Python gilt (`_unrecorded`, `apply.py:984-1010`, `return case.note != note`); `audit.md` wird dafür nicht gelesen. Planfehler, Schnittstelle bleibt zwei Zeichenketten |
| Schreibschranke | Go-Form: keine. Nur das Ziel wird in `checkTarget` auf Gerüstnamen geprüft (`approve.go:106-127`, `ToLower`, kein 8.3), jeder andere Schreibweg geht ungeprüft auf die Platte; kein Link-, kein Einschlusstest | `place.gate` vor jedem Schreiben und Löschen: jede Komponente ab dem Anker kein Link, jede löst im Anker auf, keine trägt einen Gerüstnamen, weder geschrieben noch aufgelöst (8.3), `:`-Strom und Punkt/Leerzeichen am Ende abgeschnitten, Faltung über `EqualFold` (`ſ` = `s` wie `casefold`) | Python gilt (`_gate`, `_normalised`, `apply.py:556-653`). Die Auflösung ist `guard.ResolvePath` (Junctions, 8.3), der Einschluss `guard.IsRelativeTo`; darum braucht `place.go` keine eigene Windows-Datei. Eine Junction **in** den Tresor bleibt erlaubt, wie in Python (`apply.py:585-586`) |
| Anker und fremde Register | Brief (Task 8): `place{anchor, wiki, touched}`, ein Anker | `anchorFor` misst einen Schreibzugriff unter einem Wiki außerhalb des Tresors gegen dieses Wiki; `place.registers` nimmt die `_identities.tsv` aller Bereiche auf, ein Gerüstschreiben dorthin prüft nur, dass der Pfad selbst kein Link ist | Python gilt (`_anchor`, `_is_external_register`, `_gate_external_register`, `apply.py:449-553`). Python liest die Registry bei jedem Aufruf; hier füllt der Aufrufer `registers` einmal |
| Gerüstschreiben und Vorabprüfung | Brief (Task 8): `write(path, text)`, `preflight(paths ...string)` | `write` (Seite) und `writeScaffold` (`log.md`, `audit.md`, `_identities.tsv`; die letzte Komponente muss ein Gerüstname sein); `preflight(caseDir)` leitet die vier Pfade selbst ab, das Register aus dem Tresor, nicht aus dem Wiki | Python gilt (`_write(..., scaffold=True)`, `_preflight`, `apply.py:467-490`) |
| `touched` | Go-Form: kein Protokoll, ein Abbruch nennt nichts | vor dem Schreiben ergänzt, bei unveränderten Bytes zurückgenommen; ein Fehler beim Lesen, Anlegen oder Tauschen lässt den Eintrag stehen; `remove` trägt immer ein | Python gilt (`_touch`, `_remove`, `apply.py:499-529`) |
| Löschen des Fallverzeichnisses | Go-Form: `_ = os.RemoveAll(caseDir)` (`approve.go:456`, `:621`) | Fehler geprüft; ein fehlendes Verzeichnis ist ein Fehler (`os.Lstat` davor), weil `os.RemoveAll` dort `nil` gibt | Python gilt (`rmtree` wirft `FileNotFoundError`) |
| `AdvanceRegister` schreibt selbst | Python: `_advance_register` schreibt über `_write(..., scaffold=True)` (`apply.py:1211`) | `AdvanceRegister(tsvPath, caseStates)` liest das Register nur und gibt den gerenderten Text zurück; geschrieben wird er von `Approve` über `place.writeScaffold` | **Geschlossen.** Der AST-Test `TestNoWritePrimitiveIsCalledOutsideTheBarrier` führt keine Ausnahme mehr |
| Fallverzeichnis, das keins ist | Python: `rmtree` wirft bei einer Junction (`_rmtree_islink` prüft unter Windows auch `isjunction`: „Cannot call rmtree on a symbolic link“) und bei einer Datei (`NotADirectoryError`) | `os.RemoveAll` löscht den Junction-Eintrag bzw. die Datei und meldet Erfolg | **Neu gegen Python, nicht geheilt.** Die Schranke lässt eine Junction **in** den Tresor durch, wie Python; erst `rmtree` weigert sich dort. Kein Schreibweg legt ein Fallverzeichnis als Junction oder Datei an; eingetragen ist der Pfad in `touched` in beiden Fällen |
| Register, das sich nicht auflösen lässt | Python: `_resolved(valid)` in `_is_external_register` wirft (`apply.py:543-547`) und bricht `_gate` mit `OSError` ab | Ein Fehler von `resolvePath` für den Pfad oder ein eingetragenes Register gilt als kein Treffer; der Pfad nimmt den Gang durch den Anker und wird außerhalb des Tresors verweigert | **Neu gegen Python.** Schließt statt zu öffnen, Meldung aber eine andere (`not inside the vault` statt des Auflösungsfehlers). `TestGateReadsAnUnresolvableRegisterAsNoMatch` hält es fest |
| Tresormarke | Python: der nächste Vorfahr mit `.brain.toml` (`_MANIFEST`, `apply.py:193`, `:373-376`), gelesen mit `read_manifest` | der nächste Vorfahr, in dem `config.ReadAreaManifestUntilStage4` eine Erklärung findet: `.loomux/config.toml`, sonst `.ultra-brain/config.toml`, sonst `.brain.toml`. Eine `.loomux/config.toml` ohne `[area]` ist reine Policy und keine Marke, der Weg geht an ihr vorbei nach oben (bzw. zum nächsten Namen im selben Verzeichnis); eine Erklärung, die da ist und nicht liest (auch `[area]` ohne `scope`), wird verweigert. Ein Fallverzeichnis `.` fragt keinen Vorfahr, wie `Path(".").parents` | Python gilt, **mit Verhaltensänderung**: Python kannte nur `.brain.toml`; loomux erkennt bis Stufe 4 alle drei Namen, ein Tresor mit `.loomux/config.toml` ist neu. Die Meldung ohne Tresor nennt keinen Dateinamen mehr („no area declaration above this case“) |
| `[layout] review` | Go-Form: nicht geprüft | `layoutEntry`: leer, gewurzelt (`/`, `\`), absolut, jede `..`-Komponente mit beiden Trennern verweigert; Meldung mit `repr` wie Python | Python gilt (`_layout`, `apply.py:656-676`). **Strenger:** ein Wert mit Laufwerk ohne Wurzel (`C:x`) wird auch verweigert; pathlib setzte ihn auf das aktuelle Verzeichnis des Laufwerks, `filepath.Join` klebte ihn an den Tresor |
| Wiki eines Falls | Go-Form: das Wiki des Tresormanifests | das Wiki des Registry-Eintrags mit `scope == case.area`; unbekannter Bereich und Bereich ohne Wiki mit Namen verweigert; ein Wiki außerhalb des Tresors nur, wenn es `_schema.md` als Datei trägt | Python gilt (`_wiki`, `_is_bundle`, `apply.py:383-446`). Die Meldung „… which the registry does not register“ nennt den Pfad der Registry nicht mehr, `resolve` bekommt die Bereiche statt des Zustandsverzeichnisses |
| Zeichen im Ziel | Go-Form: nur Kategorie `Cc` | `Cc`, `Cf`, `Zl`, `Zp` (`unicode.In`); Leerraum am Rand und leeres Ziel nach `pytext.Strip` (`str.strip()`) | Python gilt (`_check_target`, `_FORBIDDEN_CATEGORIES`, `apply.py:222`, `:814-847`). `U+200E` (Cf) fiel in der Go-Form durch. Gos Unicode-Tabellen (17.0) gegen Pythons (16.0): ein erst in 17.0 vergebenes Format-Zeichen wird nur in Go verweigert |
| `..` im Ziel | Go-Form: Suche nach `"../"`, `a/..` und `a\..` gingen durch | jede `..`-Komponente nach `\` → `/` verweigert, dazu gewurzelt, absolut und (strenger, wie oben) Laufwerk ohne Wurzel | Python gilt (`_target`, `apply.py:879-881`) |
| Fehlende Zielseite | Python: geprüft in `_apply` direkt nach `_target` (`apply.py:739-741`) | geprüft am Ende von `targetPath`, Meldung wörtlich | Gleiches Verhalten, weil `_target` genau einen Aufrufer hat |
| Quellen eines Falls | Go-Form: nur das Register des Tresors | `resolveSources` sucht jede `doc_id` in den `_identities.tsv` aller Bereiche in Registry-Reihenfolge (lesend über `config.ResolvedAreaDir`, wie `area_artifact_dir`), überspringt ein Register, das keine Datei ist, und unbekannte `doc_id`s; hört auf, sobald alles gefunden ist | Python gilt (`_resolve_sources`, `apply.py:155-186`). Zusätzlicher Parameter `config.ArtifactLookup`, weil ein schreibgeschützter Bereich sein Register im Zustandsverzeichnis hält. Jede Quelle trägt zwei Registerpfade: `readFrom` (gelesen, über `ResolvedAreaDir`, notfalls der alte Zustandsordner) und `register` (geschrieben, immer `ManifestDir(area, Primary)` wie `index`); `place.registers` hält die Schreibpfade |
| Doppelte `doc_id` in einem Register | Python: die erste Zeile in Dateireihenfolge | die Zeile mit dem kleinsten `relative` | **Neu gegen Python.** `identity.ReadIdentities` liefert eine Map ohne Reihenfolge; sortiert statt zufällig. Ein Register mit doppelter `doc_id` schreibt `reindex` nicht |
| Wiederholung und Warnungen des Commits | Go-Form: kein zweiter Versuch; ohne Repository „no git repository in the vault; <Ausgabe von git>“ ohne Fall-ID (`approve.go:307`); jeder andere Fehler verworfen | `commit` versucht nach `vcs.ErrRefMoved` genau einmal neu (`errors.Is`, der Fehler kommt umhüllt). Jeder andere Fehler, auch ein zweites `ErrRefMoved`, wird zur Warnung „<id>: <said>, but not committed (<err>)“; ohne Repository „<id>: no git repository in the vault; <said> but not committed“; unveränderter Baum „<id>: nothing to commit on the first attempt\|after the retry; the case belongs back in the queue“. `said` ist `written` bzw. `decision recorded`, die Fall-ID steht roh wie in Python | Python gilt (`_commit`, `_report`, `apply.py:1318-1378`) |
| Ablehnung: Pfade an git | Go-Form: `audit.md` und das Fallverzeichnis gehen absolut an `commitPaths` (`approve.go:459`); das `audit.md` eines Wikis außerhalb des Tresors wird mitgegeben | `add` ist `audit.md` relativ zum Tresor, `remove` das Fallverzeichnis relativ zum Tresor; ein `audit.md` außerhalb des Tresors wird geschrieben, aber nicht gestagt (`place.staged`) | Python gilt (`_staged`, `apply.py:683-695`; `_reject`, `:715-721`) |
| Ablehnung: Schreib- und Löschfehler | Go-Form: `appendProtocol` und `os.RemoveAll` mit verworfenem Fehler (`approve.go:174-175`, `:456`); ein unlesbares `audit.md` gilt als leer und wird überschrieben | jeder Schreibvorgang durch die Schranke (`writeScaffold`, `remove`); ein Fehler bricht ab und steht in `place.touched`. `audit.md` wird roh und streng als UTF-8 gelesen, ein anderes Byte ist ein Fehler, bevor etwas geschrieben ist | Python gilt (`_append`, `apply.py:1301-1307`, `read_text(encoding="utf-8", newline="")`; `_remove`, `:499-508`) |
| Ablehnung: Vorschlag, der kein UTF-8 ist | Go-Form: die rohen Bytes gehen an `ReadProposal` (`approve.go:451-452`) | ungültige Bytes werden zu U+FFFD (`strings.ToValidUTF8`), dann gezählt | Python gilt dem Sinn nach (`_claims`, `read_text(errors="replace")`, `apply.py:1217-1221`). **Neu gegen Python:** `ToValidUTF8` fasst eine Folge ungültiger Bytes zu einem U+FFFD zusammen, Python ersetzt jede ungültige Teilfolge einzeln. In den Auditblock geht nur die Zahl der Behauptungen, und die ändert sich dadurch nicht |
| Erstes Schreiben in einen schreibgeschützten Bereich | Python: `area_artifact_dir` kennt nur einen Zustandsordner | `moveStock` kopiert, bevor ein Register unter `Primary/areas/<scope>` geschrieben wird, den ganzen Bereich aus dem alten Zustandsordner in einen Staging-Ordner daneben und benennt ihn an seinen Platz um; ein halber Tausch von `index` wird vorher mit `lock.Recover` zu Ende geführt. Ein Bereich, den der neue Ort zum Zeitpunkt des Tauschs schon hält, wird **nie ersetzt**: veröffentlicht ein gleichzeitiger `reindex` ihn zwischen Entscheidung und Tausch (es gibt keine gemeinsame Sperre), bleibt dessen neuerer Bestand stehen und die eigene Kopie wird verworfen. Darum ein schlichtes `os.Rename`, das ein vorhandenes (Windows) bzw. nicht leeres (POSIX) Ziel verweigert, und nicht `lock.ReplaceDir`, das den frischen Bestand beiseitelegen und löschen würde (`TestMoveStockKeepsAnAreaPublishedBeforeTheSwap`, gemessen 2026-09-22: mit `ReplaceDir` verschwindet der veröffentlichte Bestand). Idempotent: hält `Primary` den Bereich schon (oder keiner der beiden Orte), geschieht nichts | **Neu gegen Python**, eine Folge der zwei Zustandsordner bis Stufe 4: `ResolvedAreaDir` entscheidet am Verzeichnis, ein einzelnes Register unter `Primary` ließe den ganzen Bereich ohne Erklärung erscheinen, und `privacy.VisibleAreas` verweigerte dann jeden Bereich. Außerhalb der Schreibschranke, weil nur unter dem Zustandsordner geschrieben wird; der AST-Test führt dafür eine eigene Stufe: `copyStock` ist selbst ein Primitiv, das nur `moveStock` rufen darf, `moveStock` darf nur `os.RemoveAll`, `copyStock` nur `os.MkdirAll` und `os.WriteFile`. Dass `copyStock` mit `os.WriteFile` und nicht über `lock.ReplaceText` schreibt, bricht die Regel des Plans „jede Datei über `lock.ReplaceText`“ (`stock.go:110`); das Ziel ist ein frischer Staging-Ordner, den ein einziges Rename als Ganzes einsetzt, ein halb geschriebener Ordner wird also nie gelesen. Anders als für `WriteGitAfter` deckt kein Entscheid des Plans diese Ausnahme; sie steht hier, damit der Nutzer sie einmal sieht |
| Zustandsverzeichnis an `Approve` | Python: `approve(..., state_dir=...)`, Pflicht; Brief (Task 11): `Options` ohne Zustandsort | `Options.Lookup config.ArtifactLookup`; die Registry kommt als `areas`, der Zustandsort für `resolveSources` und `registersOf` über `Lookup`. `Approve` ruft `config.NewArtifactLookup()` nicht selbst, damit ein Test seinen Zustandsort isoliert | Python gilt dem Sinn nach (`apply.py:307-316`); Schnittstelle um ein Feld erweitert |
| Scratch-Index | Python: `scratch=None` fällt auf eine temporäre Datei zurück | `Options.Scratch` ist ein Verzeichnis (vcs legt `<scratch>/index` an); leer wird es als `*ApplyError` verweigert, bevor der Fall gelesen ist | **Neu gegen Python** (Controller-Entscheid): ohne Rückfall käme ein leerer Wert erst als Commit-Warnung zurück. Die CLI übergibt `<state>/maintenance` |
| `defer` | Go-Form: `defer` war ein Entscheid von `Approve` | `Decisions = ["approve", "reject"]`; `defer` wird mit „decision must be one of approve, reject, found 'defer'“ verweigert | Python gilt (`DECISIONS`, `apply.py:189`); Zurückstellen behandelt die CLI, ohne `Approve` zu rufen |
| Fehlerarten und `Dirty` | Python: nur `ApplyError` und `OSError` bekommen `dirty` (`apply.py:343-358`); ein `UnicodeDecodeError` (Paket oder Seite kein UTF-8) oder ein Fehler von `read_identities` verlässt `approve` ohne `dirty` | jeder Fehler verlässt `Approve` als `*ApplyError` oder eine seiner drei Arten, mit `Dirty = place.touched` (nil, wenn nichts berührt ist); eine Weigerung der Schranke wird ein schlichter `*ApplyError`, jeder andere Fehler behält seine Ursache hinter `Unwrap` (`errors.Is(err, fs.ErrNotExist)` hält) | **Neu gegen Python**, enger: auch die Fehler, die Python ohne Hinweis durchreicht, nennen die berührten Dateien. Für die CLI: `errors.As` auf `*ApplyError` trifft die drei Arten nicht, sie tragen `Dirty` im eingebetteten Feld. Darum die Methode `DirtyFiles()` an `*ApplyError`, die auf alle drei Arten befördert wird: die CLI prüft eine Schnittstelle (`interface{ DirtyFiles() []string }` über `errors.As`), keine vier Typen |
| Behauptungszahl in den Wächterblöcken | Go-Form: `nil` als Behauptungen (`approve.go`), also immer „0 Behauptung(en)“ | Ziel- und Quellwache zählen die Behauptungen von `<fall>/proposal.md` über `claimHeadings`, auch wenn eine Nachbesserung übergeben ist | Python gilt (`_guard`, `_guard_sources`, `apply.py:931-939`, `:971-979`) |
| Quellen zweimal auflösen | Python: `_resolve_sources` in `_guard_sources` und noch einmal in `_advance_register` | einmal in der Quellwache, das Ergebnis geht an den Registervorschub | Gleiches Verhalten, solange sich zwischen Wache und Vorschub kein Register ändert; die Lücke dazwischen ist dieselbe wie unten |
| Register vorschieben ohne Sperre | Python: `_advance_register` nimmt keine Sperre | `advanceRegisters` nimmt keine: `moveStock`, dann `AdvanceRegister` (lesen) und `place.writeScaffold` (atomar tauschen) | Python gilt, **Lücke festgehalten:** `reindex` nimmt in loomux ebenfalls keine Sperre. Läuft ein `reindex` zugleich, kann er das Register zwischen Lesen und Tausch neu schreiben (dann gewinnt der spätere Tausch, eine Zeile geht verloren), oder `moveStock` und der Staging-Tausch von `index` treffen sich auf `Primary/areas/<scope>`. Der nächste `reconcile` sieht eine verlorene Zeile als geänderte Quelle und eröffnet den Fall neu; eine Sperre braucht beide Seiten |
| Zwei `approve` zugleich | Python: beide nutzen `<zustand>/maintenance/index` (`cli.py:1364`), und `commit_paths` löscht die Datei vor Gebrauch (`vcs.py:193-197`) | ebenso: `vcs.scratchIndex` löscht `<scratch>/index`, bevor es ihn füllt (`commit.go:223-237`); zwei gleichzeitige Freigaben teilen die eine Datei, und jede kann den halb gefüllten Index der anderen löschen | Python gilt, **Lücke festgehalten, nicht geheilt:** nicht gemessen; denkbar ist ein Lauf, der an einem `git`-Aufruf scheitert, oder ein Commit, der Pfade trägt, die der andere Lauf in die Datei geschrieben hat. Der Vergleich-und-Tausch lässt nur einen der beiden auf den Ref, der zweite geht in die eine Wiederholung. Wie die fehlende Sperre zwischen `reindex` und `approve` eine Entscheidung des Nutzers |
| Seite lesen vor dem Patch | Python: `read_text(encoding="utf-8", newline="")`, dann `\r\n` → `\n` | `readPage`: Bytes, streng UTF-8 („<seite>: not valid UTF-8“ als `*ApplyError`), nur `\r\n` gefaltet, ein einzelnes `\r` bleibt | Python gilt (`apply.py:769-772`); Python beendet sich bei ungültigem UTF-8 mit einem `UnicodeDecodeError` ohne `dirty` (siehe „Fehlerarten“) |
| `Bewusst ausgeben:` im zurückgehaltenen Block von `case` | Python: `brain case --package <id>` (`cli.py:1305`) | `loomux case --package <id>` | **Übersetzung**, nicht Abweichung: der Befehl heißt in loomux so. Die `[[stdout]]`-Regel des Importers (`brain case --package ` → `loomux case --package `) übersetzt die aufgezeichnete Erwartung; alle vier `case`-Einträge und alle acht `cases`-Einträge des Fallsatzes von ultra-brain (`bench/cases/case`, `bench/cases/cases`) sind am 2026-09-22 damit bytegleich gelaufen |
| Reihenfolge der Fälle | Python: `sorted(rglob)`, unter Windows Teil für Teil in Kleinschreibung, unter POSIX nach Bytes. Go-Form: `lessCasePath` in `ListCases`, aber `sort.Strings` über die ganzen Pfade in `FindCase` (`lookup.go:66`), also `alpha-later` vor `alpha/…` | `maintenance.CaseFiles`: Teil für Teil in Kleinschreibung (`strings.ToLower`), stabil, auf **jedem** System; `cases` und `FindCase` nutzen dieselbe Liste, die Mehrdeutigkeitsmeldung nennt die beiden Pfade darum in der Reihenfolge der Liste | **Bewusste Abweichung** gegen Python unter POSIX: die Referenz lief unter Windows, und nach ihrem Wegfall definiert dieser Vergleich allein die Reihenfolge |
| Schreibweise von `case.toml` | Python: `root.rglob("case.toml")` (`cli.py:1386`) vergleicht unter Windows ohne Rücksicht auf Groß- und Kleinschreibung; eine `Case.toml` steht in der Liste (gemessen 2026-09-23 mit Python 3.14.7; der Pfad trägt dann die Schreibweise des Musters, `case.toml`) | `CaseFiles` nimmt nur den Namen `case.toml` genau so (`lookup.go:29`), auf jedem System | **Abweichung, stehengelassen:** `reconcile` legt jede Akte als `case.toml` an; nur eine von Hand umbenannte Datei fällt unter Windows aus der Liste, unter POSIX auf beiden Seiten |
| Leerer `note` oder `superseded_proposal` in `case` | Python: `is not None` (`cli.py:1272`, `:1279`); ein von Hand geschriebenes `note = ""` druckt `Vermerk: `, ein leeres `superseded_proposal` zeigt `superseded-proposal.md` | `Case.Note` und `Case.SupersededProposal` sind Zeichenketten, `""` ist der fehlende Schlüssel (`case.go:36-39`); `case` druckt dann keinen Vermerk und zeigt die Datei nicht (`cases.go:157`, `:165`) | **Abweichung, stehengelassen:** `reconcile` und `approve` schreiben nie einen leeren Wert; nur eine von Hand bearbeitete Akte zeigt den Unterschied |
| Anführung der ID in Meldungen | Go-Form: `'%s'` (`lookup.go:74`, `:90`, `:93`, `cases.go:153`) | `pytext.Repr`: `it's` wird `"it's"` | Python gilt (`{identifier!r}`, `{case.id!r}`, `cli.py:1176`, `:1210`) |
| Unlesbare `case.toml` in `cases` | Python: nur `CaseError` gilt als unlesbar; ein `OSError` (etwa ein Verzeichnis namens `case.toml`, das `rglob` mitliefert) bricht mit `error:` und Exit 1 ab, bevor eine Zeile gedruckt ist | jeder Fehler von `ReadCase` gilt als unlesbar (`unreadable case: …`, Exit 1), die lesbaren Fälle werden trotzdem gelistet; ein Verzeichnis namens `case.toml` wird von `CaseFiles` gar nicht erst aufgenommen | **Neu gegen Python**, wie die Go-Form (`cases.go` `ListCases`): der Pfad kam Augenblicke vorher aus dem Durchgang, ein Lesefehler ist dort derselbe Befund wie ein kaputter Inhalt |
| Unlesbare Erklärung beim Datenschutzblick von `case` | Python: `area_manifest` wirft (`reconcile.py:242-250`) | `maintenance.AreaManifest` antwortet `nil`; der Fall gilt dann als zurückgehalten, mit der Zeile „Datenschutzmodus unbekannt“ | **Neu gegen Python, nicht beobachtbar:** `case` fragt vorher `ReviewRoot`, das dieselben Erklärungen liest und eine kaputte verweigert. Nur eine Datei, die zwischen beiden Lesevorgängen bricht, erreicht die Stelle, und dann schließt sie den Fall, statt ihn zu öffnen |
| Aufruf von `case` | Python: argparse, `--package` und `--state-dir` vor oder nach der ID, Präfixe wie `--pack` werden angenommen, `-h` endet mit 0. Go-Form: siehe Abschnitt 5 | `parseInterspersed` (geteilt mit `approve`): `--package` vor oder nach der ID, auch `-package`; nach `--` ist alles ID. Keine ID oder mehr als eine ⇒ Exit 2 mit der argparse-Meldung. Kein `--state-dir` (der Zustandsort kommt aus der Umgebung, wie bei `reconcile`), keine Präfixe, `-h` endet mit 2 wie bei jedem loomux-Befehl | Python gilt für die Stellung der Flagge; die übrigen Punkte folgen den loomux-Regeln aller Befehle |
| Datei eines Falls, die kein UTF-8 ist | Python: `UnicodeDecodeError`, Traceback, Exit 1, nach den bereits gedruckten Kopfzeilen | `pytext.ReadText` verweigert, `error: <pfad>: not valid UTF-8`, Exit 1, ebenfalls nach den Kopfzeilen | Python gilt dem Sinn nach; der Traceback wird zur einen `error:`-Zeile |
| Aufruf von `approve` | Python: argparse, `--amend`, `--reject`, `--defer` und `--state-dir` vor oder nach der ID, die drei Entscheide in einer sich ausschließenden Gruppe (`cli.py:613-623`) | `parseInterspersed` wie bei `case`; zwei gesetzte Entscheide ⇒ Exit 2 mit „loomux approve: argument --<spätere>: not allowed with argument --<frühere>“ in der Reihenfolge der Befehlszeile. `--reject=false` entscheidet nichts und kollidiert mit nichts. `--amend --` ⇒ Exit 2 mit „argument --amend: expected one argument“, wie argparse (Controller-Entscheid; die Prüfung sitzt im geteilten Helfer, gilt also für jede Flagge mit Wert); `--amend=--` nimmt `--` als Wert, wie argparse. Ebenso jeder Wert, den argparse für eine Option hält (Controller-Entscheid R2): `--amend --reject`, `--amend -x.md`, `--amend -inf` ⇒ Exit 2 mit derselben Meldung. Als Wert gelten wie bei argparse (`_parse_optional`) ein einzelnes `-`, ein Wort mit Leerzeichen und jedes Wort, auf das Pythons 3.14 `_negative_number_matcher` `^-\.?\d` als Präfix passt, mit Unicode-Ziffern (`unicode.IsDigit`): also auch `-1e5`, `-1.`, `-5x.md`, `-2026-notes.md`, `-.5x`, `-١x`. Gemessen am 2026-09-23 mit argparse aus Python 3.14.7 der Referenz (`uv run --project ultra-brain python`): diese sechs als Pfad genommen, `-inf`, `-.x`, `-x.md`, `--reject` mit „expected one argument“ abgewiesen. `--amend=` wird `.`, wie Python `Path("")` liest. Kein `--state-dir` | Python gilt; `--state-dir` fehlt wie bei `case` |
| Prüfer | Python: `getpass.getuser()` (`cli.py:1323-1331`), das zuerst `LOGNAME`, `USER`, `LNAME`, `USERNAME` liest und erst dann das Konto | `os/user.Current().Username`, der Domänenteil vor dem letzten `\` abgeschnitten. Kann das Konto nicht gelesen werden: `error: <meldung>`, Exit 1, `apply` wird nicht gerufen | Unter Windows fallen beide für ein angemeldetes Konto zusammen (`USERNAME` ist der Kontoname ohne Domäne). **Neu gegen Python:** eine Umgebungsvariable kann den Prüfer nicht mehr umbenennen |
| Nachlauf nach einem geschriebenen `approve` | Python: `_technical_update` (`cli.py:1432-1516`): eigene Aufholung; scheitert sie mit irgendeinem `ReconcileError` oder `OSError`, auch mit `NoReviewCentreError` (`reconcile.py:155`), warnt sie „warning: die Aufholung vor der technischen Aktualisierung ist fehlgeschlagen (…). Die Freigabe steht, …“ und hält an, ohne zu indizieren; sonst listet sie neue Fälle und unlesbare Falldateien auf stderr und ruft die Funktion `reindex` ohne zweite Aufholung; scheitert die, folgt „warning: die technische Aktualisierung ist fehlgeschlagen; …“. Go-Form: siehe Abschnitt 4 | `technicalUpdate` in `internal/cli/approve.go`, genau diese Form (Controller-Entscheid R1, Fixrunde 1; er ersetzt den früheren Entscheid, `reindexCommand` einmal zu rufen): `catchUp` über die gelesene Registry, bei jedem Fehler Pythons Aufholungswarnung wörtlich (`brain` als `loomux`) und Halt, sonst `reportCatchUp` (mit `reindex` geteilt) und `indexAreas`, der Indexlauf ohne eigene Aufholung, den `reindexCommand` nach seiner Aufholung ebenso ruft. Scheitert er, Pythons Zeile der technischen Aktualisierung. Exit von `approve` bleibt 0 | Python gilt. Die Aufholung läuft genau einmal; „kein Prüfzentrum“ ist hier ein Fehlschlag, anders als beim Befehl `reindex` (`cli.py:1004-1027`), der warnt und weiterindiziert. `reindexCommand` verhält sich unverändert. Das „updated qmd collections: …“ auf stderr druckt auch Pythons `reindex` (`cli.py:320`) |
| Daemon neu laden nach dem Nachlauf | Python: `_ask_daemon_to_reload(state_dir)` nach einem grünen `reindex` (`cli.py:1510`) | entfällt | Am Code von `internal/serve` geprüft, 2026-09-22: `handlers` reicht nur `RegistryDir` und `LegacyDir` an die Werkzeuge weiter (`serve.go:208-217`), `answer.RunWith` liest Registry, Katalog und Index bei jedem Aufruf, und die Ports werden je Aufruf gebaut (`answer.go:97-109`). Pythons Daemon hielt Graph und Registry ab seinem Start (`cli.py:1802-1810`). Der loomux-Dienst hält weder Index noch Graph noch Registry über einen Aufruf hinaus; die einzigen `sync.Once` in `internal/serve` und `internal/brain/search` sind die Stoppanfrage und ein Warmlauf-Hinweis. Es gibt nichts neu zu laden. loomux' eigenes `reindex` ruft ihn aus demselben Grund schon seit Stufe 3a nicht |
| Scratch-Index im Fallsatz | Python: `scratch` ist `<zustand>/maintenance/index` und bleibt nach dem Commit liegen (`vcs.py:193-197`, gelöscht wird nur **vor** Gebrauch) | `<zustand>/maintenance/index`, ebenfalls liegen gelassen | **Keine Abweichung im Verhalten**, nur im Vergleich: beide Seiten hinterlassen die Datei am selben Ort, ihre Bytes tragen aber die Stat-Daten des Laufs (mtime, ctime, Inode der gestagten Dateien) und können nie gleich sein. `expected3b` führt `content mismatch: maintenance/index` bei `approve/success`, `approve/amend` und `approve/reject`; was der Index gestagt hat, hält `git.after` (Baum von HEAD, Status). Bei `approve/rebase` und `approve/no-repo` legt keine Seite ihn an, und der Fallsatz hält auch das fest |
| Register über gestempelte Seiten | Python: `_technical_update` ruft `reindex`, das `wiki/page.md`, `wiki/audit.md` und `wiki/log.md` mit ihrem `content_hash` ins Register schreibt | dasselbe, über `indexAreas` | **Keine Abweichung im Verhalten**, nur im Vergleich: Seite und `audit.md` tragen den Zeitstempel des Laufs mit Mikrosekunden, ihr Hash ist darum auf jeder Seite ein anderer. Der Fallsatz führt `content mismatch: repo-a/_identities.tsv` bei den vier geschriebenen Freigaben (`success`, `amend`, `rebase`, `no-repo`) und prüft stattdessen (`rehashed` in `cases_3b_test.go`): jeder `content_hash` beider Seiten ist der Hash der Datei, die dieselbe Seite danebenlegt, und die Zeilen stimmen ohne ihn überein (Pfad, Revision, `doc_id` bzw. „neu vergeben“) |

## Fallsatz 3b, aufgezeichnet am 2026-09-22, approve neu am 2026-09-23

24 Fälle, aufgezeichnet mit `stufe-3b-orakel/record_all.sh` gegen
`brain-mcp.exe` am Tag `loomux-3-source`, übersetzt mit
`testdata/cases/3b-map.toml`, abgespielt von `TestCases3b`.

- **19 Fälle ohne Unterschied nach der Normalisierung**: alle sechs `cases`,
  alle sechs `case`, und `approve` mit `--defer`, bewegtem Ziel, bewegter
  Quelle, gescheiterter Belegprüfung, unpassendem Hunk, leeren Argumenten und
  unbekanntem Fall. „Ohne Unterschied“ heißt: null Abweichungen, nachdem
  `NormalizeState` Zeitstempel, Prüfer, Commit-SHA und neu vergebene
  `doc_id`s gefaltet hat; stderr
  vergleicht der Fallsatz in **keinem** der 24 Fälle (die Spec erlaubt das).
  Rohe Byte-Gleichheit ist es nicht. Die drei Weigerungen, die schreiben
  (`case.toml` mit Vermerk und `manual`, ein Auditblock), sind gleich nach
  der Normalisierung; die Faltung des Prüfers schreibt jedes `human:<x>` in
  `audit.md` um, auch in älteren Blöcken, und verdeckte darum einen falschen
  Prüfer dort. `git.after` belegt, dass HEAD stehen bleibt (Betreff `base`)
  und im Arbeitsbaum genau diese Dateien von HEAD abweichen. Bei
  `approve/unknown`, `approve/empty-args`, `case/unknown`, `case/ambiguous`
  und der Warnung von `approve/rebase` hält der Fallsatz darum Exitcode,
  Stdout und Welt fest, nicht den Wortlaut der Meldung.
- **5 Fälle mit freigegebenen Unterschieden**, alle aus den beiden Zeilen
  oben und den Indexformaten aus Stufe 3a (`graph.json`, `index.yml`,
  `qmd-collections.json`, dekodiert gleich): `approve/success`,
  `approve/amend`, `approve/reject`, `approve/rebase`, `approve/no-repo`.
  Seite, `audit.md`, `log.md`, Fallverzeichnis und `git.after` sind in allen
  fünf gleich. `git.after` trägt seit Fixrunde 1 von Task 14 die Zeilen aus
  `git diff --name-status HEAD` und Autor/Committer: nach `approve/success`
  weichen nur `notes/source.md` (nie Teil des Commits) und `_identities.tsv`
  (vom Nachlauf nach dem Commit neu geschrieben, wie in Python) von HEAD ab,
  also trägt HEAD Seite, `audit.md` und `log.md` genau so, wie `world_after`
  sie vergleicht. Ein Mutant, der die Seite nur für den Commit um eine Zeile
  verlängert, lässt `approve/success` und `approve/amend` allein an
  `git.after` scheitern; die alten Statuszeilen (`MM`) hätten ihn nicht
  gesehen.
- **Kein Code geheilt**: der Fallsatz fand keinen Unterschied in einer
  geschriebenen Datei oder in `git.after`.
- Die Übersetzung `Bewusst ausgeben:` (Zeile oben) ist über die
  `[[stdout]]`-Regel angewandt, bei `case/withheld`.
- Die Commit-Nachricht (Zeile „Zeilenenden der Commit-Nachricht“) erreicht
  den Vergleich nicht: `git.after` hält nur die Betreffzeile, und die ist auf
  beiden Seiten gleich.
- Die Welten der Freigaben tragen das Paket, das `reconcile` in
  `3a-source/reconcile/changed-source-baseline` geschrieben hat, einen Tag
  zurückdatiert. Diesen Fall hält Stufe 3a ohne Abweichung gegen
  `maintenance.RenderPackage`; die Segmentgrenzen sind also die aus 3a.

## Selbstnutzung (2026-09-23)

Bedingung 5 der Stufe, gegen die echte Registry dieses Rechners und das
Prüfzentrum des Tresors (`brain-knowledge/95 Prüfzentrum`, unversioniert,
ohne Remote). Verglichen wurde `bin/loomux.exe` mit der Python-Referenz
`ultra-brain/.venv/Scripts/brain-mcp.exe`.

| Aufruf | Ergebnis |
|---|---|
| `cases` | Beide Exit 0, 14 wartende Fälle (1 in `hub`, 13 in `project/ultra-brain`, alle `source_changed`). stdout gleich, nachdem die Zeilenenden gefaltet sind: Python schreibt unter Windows CRLF |
| `case hub-2026-09-04-a673` (702 Zeilen) | Beide Exit 0, gleich nach Faltung der CRLF |
| `case ultra-brain-2026-09-12-4401` (2243 Zeilen) | Beide Exit 0, gleich nach Faltung der CRLF |
| `approve --defer ultra-brain-2026-09-12-4401` | Beide geben `Fall ultra-brain-2026-09-12-4401 zurückgestellt; er bleibt unverändert in der Warteschlange.` aus, Exit 0. Das Fallverzeichnis ist danach Byte für Byte dasselbe (sha256 vorher und nachher) |

**Warum nur `--defer`.** Keiner der 14 Fälle trägt ein `proposal.md`; ein
schlichtes `approve` hielte mit „no proposal to approve“ an. Der Nutzer hat
entschieden, nur `approve --defer` zu fahren und keine schreibende Freigabe
(`approve`, `--reject`, `--amend`) gegen die unversionierten echten Daten.
Die schreibenden Freigaben belegt allein der aufgezeichnete Fallsatz (oben).

## Überlebende Mutanten

**Die Runde mit `loomux dev mutants` (2026-09-23).** Gefahren mit
`bin/loomux.exe dev mutants <paket>` gegen `a882946`, acht Arbeiter (bei
`evidence` vier, weil `apply` daneben lief), `LOOMUX_STATE_DIR` und
`LOOMUX_LEGACY_BRAIN_DIR` im Scratchpad der Sitzung. Die erste Runde über
`vcs` ist verunreinigt und wird nur als Suche gelesen: ab etwa Mutant 110 von
220 standen die ersten neuen Tests schon im Baum. Die zweite Runde lief
sauber über den Stand nach den drei Test-Commits (`07d0881`), mit einer Kopie
des Binärs, damit das Tor der Commits daneben bauen konnte. Ein Zeitüberlauf
zählt in `dev mutants` als getötet; `go test ./internal/brain/apply` braucht
allein 13 s und unter der vollen Last der Runde 14 s, weit unter den 60 s von
`goTimeout`.

| Paket | erzeugt | nicht übersetzbar | erste Runde: getötet / überlebt | zweite Runde: getötet / stehen |
|---|---:|---:|---:|---:|
| `internal/brain/vcs` | 220 | 24 | 183 / 13 | 194 / 2 |
| `internal/brain/evidence` | 170 | 28 | 127 / 15 | 136 / 6 |
| `internal/brain/apply` | 1284 | 221 | 963 / 100 | 1013 / 50 |

Jeder Überlebende der ersten Runde ist einzeln per `go test -overlay` gegen
den neuen Stand gespielt worden (ein Skript im Scratchpad ersetzt die Zeile
und fährt die Suite des Pakets); was dort stirbt, stirbt an dem Test, der
unten steht. Die neuen Goldens unter
`internal/brain/apply/testdata/frontmatter/m*` hat der Generator der Stufe
(`frontmatter_goldens.py`) aus `_advance` der Referenz geschrieben; beim
Neuschreiben blieb jedes vorhandene Golden bytegleich.

**Getötet in `vcs` (11 von 13).**
`TestCommitPathsStopsAtAFailedCommitTreeBeforeTheWindow` (`commit.go:134`),
`TestCommitPathsMakesAMissingScratchDirectory` und
`TestCommitPathsRefusesAScratchPathThatIsAFile` (`:225`, drei Formen, und
`:228`), die geschärfte Meldung in `TestCommitPathsRefusesAMovedRef`
(`:241`), die geschärfte Meldung in
`TestCommitPathsReportsAGitLostInTheWindow` (`:282`: der erste Fehler einer
Sitzung ist der gemeldete),
`TestCommitPathsReportsTheFailedAddAndNotTheRemovalAfterIt` (`:333`),
`TestObjectNameTakesFourToSixtyFourDigits` (`vcs.go:205`, beide Grenzen) und
`TestSaidAddsNothingForASilentFailure` (`vcs.go:251`).

**Getötet in `evidence` (9 von 15).** Neue Datei `boundaries_test.go`, jede
Erwartung am 2026-09-23 mit `read_proposal` und `read_package` der Referenz
nachgemessen: Infostring des Öffners im Zaun (`:193`, `>=`), Zeile nur aus
Leerraum (`:204`), erste Zeile nach der Frontmatter (`:227`), Zeile nur dem
eigenen Abschnitt angelastet (`:333`, `< end` weggelassen), gewöhnliche
Überschrift als Name (`:341`, `true`: ein Index außerhalb), versteckte
Auszeichnung auf einer Überschriftenzeile (`:307`, `:333` mit `<=`),
Nicht-Segment-Überschrift im Paket (`:408`) und Segment ohne eigenen Zaun
(`:413`).

**Getötet in `apply` (50 von 100).** 36 an neuen Goldens (`m01` bis `m18`):
Datums- und Zeitgrenzen, `+0x`, `+0` und `+0b`, ein gequoteter
`<<`-Schlüssel, ein Mapping als Schlüssel, die Ränder der druck- und
schlichtbaren Bereiche im Emitter, Falten an der Breite in allen drei Stilen,
verschachtelte und nicht einfache Schlüssel, Tabs um Blockskalare, Kommentare
und gequotete Skalare, ein datumsförmiger `str`. 14 an Tests: drei Nähte,
die jeden Lese- oder Auflösungsaufruf scheitern ließen und damit den
Mutanten verdeckten, schneiden jetzt nur den gemeinten Pfad
(`approve.go:320`, `:449`, `commit.go:64`);
`TestTwoSourcesInOneRegisterBothAdvance` (`approve.go:422`),
`TestSameFileIsFalseWhenEitherPathIsNoFile` (`:501`),
`TestReplaceIfChangedCreatesAnEmptyFile` (`place.go:150`),
`TestNormalisedNameCutsAStreamAtTheFirstCharacter` (`:322`),
`TestRelativeDropsADotComponent` (`:292`),
`TestAnOutsideWikiPassesOnAResolverFailure` (`:251`),
`TestAWritableAreasNeighbourIsLeftAlone` (`stock.go:56`: ohne die Abfrage
räumt `lock.Recover` ein `<tresor>.loomux-aside` neben einem beschreibbaren
Tresor ab) und `TestAdvanceFrontmatterNamesWhyItRefuses` (`pyyaml.go:130`
zweimal, `:187` zweimal: die Weigerung nennt ihren Grund).

**Nachgetragen in der Fixrunde (2026-09-23).** Die Zeile zu `targetParts`
hieß zuerst für alle drei Formen „äquivalent“; das hat die Durchsicht
widerlegt. Mit einem Ziel `./…` fragt der Gang über die Teile
(`resolve.go:210-216`) unter den Mutanten `true` und `part != ""` auch
`isLink(<wiki>)`, das Original nicht, und Python ebenso wenig
(`apply.py:890-899` geht `PurePosixPath(…).parts`, das `.` fallen lässt).
`TestTargetPathWalksNoDotComponent` tötet beide, einzeln per `-overlay`
nachgeprüft (`. is a link, not a page`). Die dritte Form, `part != "."`,
hieß danach noch „äquivalent“, weil ein leerer Teil nur hinter einem
verbotenen führenden `/` stünde; auch das hat die zweite Durchsicht
widerlegt: das leere Ziel `""` ist `plainRelative` und zerfällt in genau
einen leeren Teil. Das Original fragt dann keinen Teil und meldet `<wiki>: the
case's target page is gone`, wie Python (`PurePosixPath("").parts` ist leer);
der Mutant prüft das Wiki selbst auf einen Link. Derselbe Test ruft darum
auch `targetPath(r, "")` und tötet sie. In `apply` stehen damit 47 statt
50; die Tabelle oben gibt die zweite Runde wieder, wie sie lief.

**Geerbt, festgehalten und nicht geheilt.** Versteckte Auszeichnung (`%%`,
`<!--`, Bidi-Steuerzeichen) auf einer **Überschriftenzeile** eines Vorschlags
wird keinem Abschnitt angelastet: ein Abschnitt beginnt hinter seiner
Überschrift, der vorige endet vor ihr. Die Referenz tut dasselbe
(`read_proposal`, gemessen am 2026-09-23), obwohl ihr Kommentar in
`_is_allowed_line` sagt, eine Überschrift könne einen Kommentar ebenso
verbergen wie ein Absatz. Python gilt;
`TestHiddenMarkupOnAHeadingLineIsChargedToNoSection` hält das Verhalten fest,
damit eine Heilung bewusst geschieht.

| Paket | Mutant | Entscheidung |
|---|---|---|
| `internal/brain/vcs` | `scratchIndex`, `commit.go:225`: `if err == nil` → `true` | **Nicht erreichbar, stehengelassen.** `filepath.Abs` scheitert nur, wenn `os.Getwd` für einen relativen Pfad scheitert; der einzige Aufrufer reicht `<zustand>/maintenance`, und unter Windows lässt sich das Arbeitsverzeichnis eines laufenden Prozesses nicht löschen |
| `internal/brain/vcs` | `session.call`, `commit.go:286`: `if g.index != ""` → `true` | **Äquivalent, stehengelassen.** Vor `scratchIndex` laufen nur `rev-parse --absolute-git-dir`, `symbolic-ref --quiet HEAD` und `rev-parse --verify`; keiner liest den Index, ein leeres `GIT_INDEX_FILE` ändert an ihnen nichts. Die ganze Suite bleibt unter dem Mutanten grün |
| `internal/brain/evidence` | `inside`, `evidence.go:193`: `pos < s[1]` → `<=` | **Äquivalent, stehengelassen.** Das Ende einer Spanne ist das Ende der Schließerzeile vor ihrem Umbruch (`(?m)$`), eine Zeile beginnt also nie dort; bei einem offenen Zaun ist es `len(text)`, und dort beginnt nur die leere letzte Zeile, die ohnehin erlaubt ist |
| `internal/brain/evidence` | `isAllowedLine`, `:207`: Überschrift und `evidence:`-Zeile aus der Ausnahme genommen | **Äquivalent, stehengelassen.** Eine `evidence:`-Zeile beginnt mit `e` und trifft `blockStart` nie. Eine Überschriftenzeile würde anstößig, aber `ReadProposal` lastet sie keinem Abschnitt an (siehe „Geerbt, festgehalten und nicht geheilt“ oben): ihr Offset ist der Anfang ihrer Überschrift, also weder vor der ersten noch innerhalb eines Abschnitts. Dieselbe Redundanz trägt `_is_allowed_line` |
| `internal/brain/evidence` | `parseClaim`, `:279`: `opener.start >= ev[1]` → `>` | **Äquivalent, stehengelassen.** `evidenceLine` endet mit `\s*$` im Mehrzeilenmodus, also vor einem Umbruch; ein Öffner beginnt am Zeilenanfang, mindestens ein Byte dahinter |
| `internal/brain/evidence` | `ReadProposal`, `:333`: `offset >= start` → `>` | **Äquivalent, stehengelassen.** `start` ist das Ende der Überschriftenzeile vor ihrem Umbruch; eine Zeile beginnt dort nie |
| `internal/brain/evidence` | `ReadProposal`, `:341`: `len(claimSub) > 1` → `>= 1` | **Äquivalent, stehengelassen.** `FindStringSubmatch` antwortet `nil` oder zwei Elemente |
| `internal/brain/evidence` | `decimal`, `:493`: `>` → `>=` | **Nicht beobachtbar, stehengelassen.** Der Mutant nennt die Zahlen von 9223372036854775790 bis 9223372036854775799 „passt nicht“, obwohl sie passen; eine so große Segmentzahl ist nie die Zahl gelesener Segmente, und die Meldung zitiert `m[1]` wie geschrieben, in beiden Fällen dieselbe |
| `internal/brain/apply` | `AdvanceFrontmatter`, `frontmatter.go:73` (`&& entries.kind == pyList` weggelassen) und `:75` (`entry.kind != pyDict` → `false`) | **Äquivalent, stehengelassen.** Nur eine Liste trägt `items`; ein Mapping oder Skalar unter `sources` durchläuft die Schleife nicht. Ein Eintrag, der kein Mapping ist, hat keine `keys`, `get` antwortet `nil`, `pythonStr(nil)` ist `None`, und passt ein Update auf `None`, setzt `set` Schlüssel an einem Wert, den der Emitter nach seiner Art als Skalar oder Liste schreibt, ohne `keys` je zu lesen |
| `internal/brain/apply` | `anchored`, `patch.go:117`: `< MaxInt` → `<=` | **Äquivalent, stehengelassen.** Bei genau `MaxInt` bleibt `line` leer statt `"9223372036854775808"`; `lineOf` rechnet dieselbe Zahl über `big.Int`, und in `before` ordnet ein leeres `line` vor jedem 19-stelligen wie der Text selbst, weil jeder größere Anker einen größeren Text trägt |
| `internal/brain/apply` | `before`, `patch.go:153` und `:156`: `<` → `<=` | **Äquivalent, stehengelassen.** Beide Vergleiche stehen hinter `!=` derselben Werte |
| `internal/brain/apply` | `isExternalRegister`, `place.go:234`: `samePath(path, valid)` → `false` | **Äquivalent, stehengelassen.** Ist die Schreibweise gleich, lösen beide Pfade gleich auf, und der zweite Vergleich derselben Schleife antwortet `true` |
| `internal/brain/apply` | `components`, `place.go:305` (sechs Formen), `:309` (vier Formen), `:313` (`<= 0x80`) | **Äquivalent, stehengelassen.** `lexicalParts` liest nur den Schwanz hinter `len(components(base))`, und `IsRelativeTo` hat vorher festgestellt, dass Basis und Pfad dasselbe Laufwerk und dieselbe Wurzel tragen; jeder Mutant ändert den Kopf beider Zerlegungen gleich. Die Formen, die `rest[0]` auf leerem `rest` lesen, bräuchten einen Pfad, der nur ein Laufwerk ist, und kein Tresor ist das. `0x80` ist kein Trenner |
| `internal/brain/apply` | `pythonStr`, `pyyaml.go:108` (drei Formen) | **Nicht beobachtbar über einen Aufrufer, stehengelassen.** Eine Liste oder ein Mapping als `doc_id` gibt dann `""` statt „kein str“. Ein Update mit leerer `doc_id` kommt aus keinem Fall: `ReadCase` weist eine leere `doc_id` ab (`required`, `maintenance/case.go:309`). Das exportierte `AdvanceFrontmatter` nimmt ein `SourceUpdate{DocID: ""}` allerdings an; nur dort, an der API selbst, wäre der Mutant sichtbar |
| `internal/brain/apply` | `constructInt`, `pyyaml.go:320`: `sign < 0` → `<= 0` | **Äquivalent, stehengelassen.** `sign` ist 1 oder −1 |
| `internal/brain/apply` | `newFloat`, `pyyaml.go:378`: `&& strings.Contains(repr, "e")` weggelassen | **Äquivalent, stehengelassen.** Eine endliche `floatRepr` ohne `.` ist immer die Exponentenform; die Festkommaform bekommt sonst `.0` angehängt |
| `internal/brain/apply` | `constructTimestamp`, `pyyaml.go:442`: `m[9] != ""` → `true` | **Äquivalent, stehengelassen.** Leer ist `m[9]` nur bei `Z`; dann sind `m[10]` und `m[11]` leer, `atoi` gibt 0, der Versatz bleibt 0 |
| `internal/brain/apply` | `analyzeScalar`, `pyyaml_emit.go:192` und `:193`: `index == …` → `!=` | **Äquivalent, stehengelassen.** `leadingBreak` und `trailingBreak` verbieten nur den schlichten Stil, und jeder Umbruch setzt schon `lineBreaks`, das dasselbe verbietet |
| `internal/brain/apply` | `writeIndent`, `pyyaml_emit.go:230`: `column < indent` → `true` und `<=` | **Äquivalent, stehengelassen.** Nach der Zeile davor gilt `column <= indent`, und bei Gleichheit ist `whitespace` schon wahr, sonst hätte sie umgebrochen |
| `internal/brain/apply` | `writeSingleQuoted`, `pyyaml_emit.go:319` (`ch == -1 \|\|` weggelassen), `:324` (vier Formen), `:333` (`ch != -1` → `true`) | **Äquivalent, stehengelassen.** `isBreak(-1)` ist falsch; `:324` schreibt dieselben Zeichen in kleineren Stücken oder ein leeres Stück, `writeText` hängt sie gleich an und zählt dieselben Spalten; nach `ch == -1` endet die Schleife |
| `internal/brain/apply` | `writeDoubleQuoted`, `pyyaml_emit.go:359` (`true`, `<=`) und `:372` (`<=`) | **Äquivalent, stehengelassen.** Bei `start == end` wird ein leeres Stück geschrieben oder vor den Rückstrich gehängt |
| `internal/brain/apply` | `escape`, `pyyaml_emit.go:394`: `<= 0xff` → `<` | **Nicht erreichbar, stehengelassen.** U+00FF liegt im schlichten Bereich ab U+00A0 und wird nie maskiert |
| `internal/brain/apply` | `checkTabs`, `pyyaml_tabs.go:33` (Abkürzung ohne Tab weg), `:57` (`<=` im Sortieren), `:63` (`start < i`) | **Äquivalent, stehengelassen.** Ohne Tab findet die Schleife keinen; die Spannen beginnen an verschiedenen Stellen; an `start` selbst steht ein Anführungszeichen oder der Beginn eines Blockkörpers, nie `#` oder ein Tab, und einen leeren Körper überspringt die Schleife davor |
| `internal/brain/apply` | `scalarSpan`, `pyyaml_tabs.go:158` und `:172`: `< len` → `<=` | **Nicht erreichbar, stehengelassen.** yaml.v3 hat den gequoteten Skalar gelesen, das schließende Zeichen steht also im Text, und die Schleife hält davor |
| `internal/brain/apply` | `blockBody`, `pyyaml_tabs.go:193`: `width == 0` → `false` | **Äquivalent, stehengelassen.** Ohne Umbruch nach dem Kopf ist `bodyStart == len(text)`; beide Schleifen laufen nicht, die Spanne ist dieselbe leere |
| `internal/brain/apply` | `blockBody`, `pyyaml_tabs.go:212` (drei Formen): die Einrückungserkennung hält am ersten Zeichen | **Nicht beobachtbar, stehengelassen.** Unterscheiden würde eine Zeile, deren Leerzeichen zwischen der Mindest- und der erkannten Einrückung liegen und die einen Tab trägt. Gemessen am 2026-09-23 an fünf Formen: yaml.v3 weist jede solche Zeile selbst ab („found a tab character where an indentation space is expected“), bevor `checkTabs` läuft; eine Kommentarzeile dort ist in beiden Formen Kommentar |
| `internal/brain/apply` | `blockBody`, `pyyaml_tabs.go:224`: `spaces < indent` → `<=` | **Äquivalent, stehengelassen.** Ein Leerzeichen mehr gezählt ändert nur `at`, und die Abbruchbedingung fragt `spaces < indent`, das bei beiden falsch ist |
