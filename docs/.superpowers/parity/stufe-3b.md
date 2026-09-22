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
