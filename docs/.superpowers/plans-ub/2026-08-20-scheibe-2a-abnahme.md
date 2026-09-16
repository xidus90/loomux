# Abnahme der Teilscheibe 2a — Suchkette gegen die echten Bestände

Datum: 2026-08-20. Zweig `scheibe-2a-suchkette`, Worktree
`C:/Users/micro/Documents/#GIT/ultra-brain/.worktrees/scheibe-2a-suchkette`.
Zustandsverzeichnis: `C:/Users/micro/AppData/Local/brain`. Suchmaschine:
`qmd 2.8.3 (facd35e)`.

Dies ist das Protokoll des ersten Laufs außerhalb des Arbeitsbereichs. Kein
Produktivcode wurde angefasst — auch dort nicht, wo dieser Lauf einen Fehler
gefunden hat.

---

## 1. Was eingerichtet wurde

### Die Bestandsregel

Die drei Bestände sind fremdes Eigentum und wurden ausschließlich gelesen:

- `C:/Users/micro/Documents/#GIT/space`
- `C:/Users/micro/Documents/#GIT/iam_wiki`
- `C:/Users/micro/Documents/#GIT/#Obsidian/AI`

Vor dem ersten Lauf wurde von jedem Baum eine Ausgangsaufnahme genommen — für
die beiden git-Bäume `git status --short`, für `#Obsidian/AI` (kein
git-Repository) eine Liste aller Dateien mit Größe und Änderungszeit. Beleg in
Abschnitt 6.

### Registrierung

`C:/Users/micro/AppData/Local/brain/registry.toml`, ergänzt um drei Einträge.
Der bestehende Eintrag `knowledge` blieb unverändert. Die Scope-Namen sind
genau die Namen der bereits vorhandenen qmd-Sammlungen, damit der Abgleich
keine zweite Sammlung anlegt.

```toml
[[area]]
scope = "knowledge"
path  = "C:/Users/micro/Documents/#GIT/brain-knowledge"

[[area]]
scope    = "space"
path     = "C:/Users/micro/Documents/#GIT/space"
readonly = true

[[area]]
scope    = "iam-wiki"
path     = "C:/Users/micro/Documents/#GIT/iam_wiki"
readonly = true

[[area]]
scope    = "obsidian-ai"
path     = "C:/Users/micro/Documents/#GIT/#Obsidian/AI"
readonly = true
```

Dazu je ein Manifest unter
`C:/Users/micro/AppData/Local/brain/areas/<scope>/.brain.toml` — nicht im
Bestand, sondern im Zustandsverzeichnis, wohin `readonly = true` alles Erzeugte
umlenkt:

```toml
[area]
scope    = "space"
readonly = true

[index]
include = ["**/*.md"]
```

### Sicherung der qmd-Konfiguration

Vor dem ersten Lauf wurde `~/.config/qmd/index.yml` von Hand außerhalb des
Repos kopiert, damit ein Rückweg existiert, der nicht vom eigenen Code abhängt:
`index.yml.manual-safety-copy`, md5 `e16a62b3615e9c7e560571ec87f38b66`,
1018 Bytes, LF-Zeilenenden.

### Der Indexlauf

```
$ uv run brain reindex
updated qmd collections: iam-wiki, knowledge, obsidian-ai, space

real    0m24.231s
```

Zwei weitere Läufe zur Kontrolle: 22,3 s und 21,7 s.

---

## 2. Die sechs Abnahmepunkte

Alle Läufe mit `PYTHONIOENCODING=utf-8` — warum, steht unter Befund B1.

### Punkt 1 — Datenschutz, Kanal: **bestanden**

`obsidian-ai` wurde versuchsweise auf `mode = "local_only"` gesetzt.

```
$ uv run brain search "auto dream image ai manager" --channel cloud -n 5
brain://iam-wiki/projects/backend/apis/ml-worker.md:169  75%  ML-Worker API
brain://iam-wiki/architecture/data-flow.md:71  38%  Datenfluss
brain://iam-wiki/overview.md:59  34%  IAM-Systemüberblick
brain://iam-wiki/llm-wiki.md:21  25%  LLM Wiki
brain://iam-wiki/projects/workers/overview.md:144  15%  ML Worker — Übersicht
exit=0
```

Kein einziger Treffer aus `obsidian-ai`, obwohl die Anfrage wörtlich auf diesen
Bestand zielt. Die Suche hat geantwortet, nur eben ohne den Bereich.

```
$ uv run brain catalog --channel cloud
# brain

* [iam-wiki](brain://iam-wiki/)
* [knowledge](brain://knowledge/)
* [space](brain://space/)
exit=0

$ uv run brain read "Willkommen.md" --scope obsidian-ai --channel cloud
error: unknown scope 'obsidian-ai'; known scopes are: iam-wiki, knowledge, space
exit=1
```

Dieselben drei Aufrufe mit `--channel local`:

```
$ uv run brain search "auto dream image ai manager" --channel local -n 5
brain://obsidian-ai/IAM-Projects/image-ai-manager/feedback_auto_dream.md:2  77%  feedback_auto_dream
    @@ -1,4 @@ (0 before, 8 after)
    ---
    name: Auto-Dream aktiviert
    description: Nutzer möchte, dass Erkenntnisse und wichtige Informationen aus Konversationen automatisch als Memories gespeichert werden
    type: feedback

brain://obsidian-ai/IAM-Projects/image-ai-manager/knowledge_project_deps.md:2  62%  Produziert (andere Projekte konsumieren)
    ...
exit=0

$ uv run brain catalog --channel local
# brain

* [iam-wiki](brain://iam-wiki/)
* [knowledge](brain://knowledge/)
* [obsidian-ai](brain://obsidian-ai/)
* [space](brain://space/)
exit=0

$ uv run brain read "Willkommen.md" --scope obsidian-ai --channel local
  Das ist dein neuer *Vault*.

Notiere dir etwas oder, [[Neuer Link|erstelle einen neuen Link]], oder probiere [die Importer-Erweiterung aus](https://help.obsidian.md/Plugins/Importer)!
...
exit=0
```

Zurückgestellt und belegt:

```
$ cat C:/Users/micro/AppData/Local/brain/areas/obsidian-ai/.brain.toml
[area]
scope    = "obsidian-ai"
readonly = true

[index]
include = ["**/*.md"]

$ uv run brain catalog --channel cloud
# brain

* [iam-wiki](brain://iam-wiki/)
* [knowledge](brain://knowledge/)
* [obsidian-ai](brain://obsidian-ai/)
* [space](brain://space/)
exit=0
```

Anzumerken ist, wie die Verweigerung formuliert ist: `unknown scope`. Der
Bereich existiert für den Wolkenkanal nicht — die Meldung nennt ihn nicht
einmal, um genau das nicht zu verraten, was `local_only` verbirgt. Das ist
absichtlich so und im Kern kommentiert.

### Punkt 2 — Datenschutz, Pfad: **bestanden**

`never = ["overview.md"]` im Manifest von `iam-wiki`, gesetzt auf einen real
vorhandenen Pfad (`iam_wiki/overview.md`). Anfrage, die genau diese Datei zur
Zielquelle hat:

```
$ uv run brain search "IAM-Systemüberblick Ablageort der Repos" --scope iam-wiki --channel local -n 5
brain://iam-wiki/docs/superpowers/specs/2026-07-29-repo-doku-aufraeumen-design.md:66  63%  Design: Doku in den Code-Repos aufräumen
brain://iam-wiki/docs/superpowers/plans/2026-07-28-iam-docs-migration-cutover.md:774  62%  iam_docs → iam_wiki: Migration und Umschaltung — Implementation Plan
brain://iam-wiki/docs/superpowers/specs/2026-07-28-iam-docs-migration-cutover-design.md:33  61%  Design: iam_docs → iam_wiki — Migration und Umschaltung
brain://iam-wiki/docs/superpowers/plans/2026-07-29-repo-doku-aufraeumen.md:17  55%  Doku in den Code-Repos aufräumen — Implementation Plan
brain://iam-wiki/docs/superpowers/plans/2026-07-27-iam-wiki-scaffold.md:55  50%  IAM-Wiki-Gerüst Implementation Plan
```

Dieselbe Anfrage mit `--channel cloud` liefert Zeile für Zeile dasselbe.
`read` auf beiden Kanälen:

```
$ uv run brain read "overview.md" --scope iam-wiki --channel local
error: iam-wiki/overview.md is excluded by [privacy] never
exit=1

$ uv run brain read "overview.md" --scope iam-wiki --channel cloud
error: iam-wiki/overview.md is excluded by [privacy] never
exit=1
```

Nach dem Zurückstellen dieselbe Anfrage:

```
$ uv run brain search "IAM-Systemüberblick Ablageort der Repos" --scope iam-wiki --channel local -n 5
brain://iam-wiki/overview.md:59  100%  IAM-Systemüberblick
brain://iam-wiki/docs/superpowers/specs/2026-07-29-repo-doku-aufraeumen-design.md:66  63%  ...
...

$ uv run brain read "overview.md" --scope iam-wiki --channel cloud
---
type: Overview
title: IAM-Systemüberblick
exit=0
```

Der Beleg ist scharf: die Datei stand ohne `never` auf Platz eins mit 100 %,
mit `never` ist sie auf beiden Kanälen vollständig verschwunden, und die
übrigen fünf Plätze sind identisch besetzt.

### Punkt 3 — Entdopplung: **nicht bestanden**

Die Voraussetzung des Abnahmepunkts trifft nicht zu, und der geprüfte
Mechanismus kann nicht leisten, was der Punkt verlangt. Beides ist belegt.

Erstens: es gibt in keinem der drei Bestände zwei Pfade mit gleichem Inhalt.
Über alle drei Register hinweg kommt kein `content_hash` zweimal vor — die
Suche nach doppelten Hashes in allen drei `_identities.tsv` liefert keine
einzige Zeile.

Die byte-nahen Zwillinge, die es in `space` sehr wohl gibt, liegen unter
`.claude/worktrees/...` und sind von `DEFAULT_EXCLUDES` ausgeschlossen — sie
kommen in keinem der beiden Indizes vor.

Zweitens, und das wiegt schwerer: `_assemble` gruppiert nach
`(collection, doc_id)`, und `doc_id` wird in `_index_area` **pro Pfad** neu
vergeben, ohne den Inhaltshash zu befragen. Zwei Pfade mit gleichem Inhalt
bekommen daher zwei verschiedene `doc_id`. Nachgestellt in einem eigens
angelegten Wegwerf-Bereich außerhalb aller drei Bestände (eigene Registry,
eigenes Zustandsverzeichnis, eigenes `XDG_CONFIG_HOME`, damit die echte
qmd-Konfiguration unberührt bleibt):

```
doc_id                      pfad        content_hash                     revision
01M0FJ9MKPHT7AV0J9GJ4FCN86  a.md        sha256:2517cb5eb7a7df3a94a...1383  1
01M0FJ9MKPWT6ST6K3GHG2F84S  copy/a.md   sha256:2517cb5eb7a7df3a94a...1383  1
```

Gleicher Hash, zwei `doc_id`. Der Treffer erschiene also zweimal, nicht einmal
mit `also at`. Auch die Divergenzmeldung greift nicht: `keys_seen` sammelt je
`doc_id` genau einen `content_key`, und `len(engine_keys) > 1` wird nie wahr.

Der zugehörige Test (`test_same_document_under_two_paths_appears_once`) besteht,
weil die Fixture ein Register von Hand schreibt, in dem `a.md` und `copy/a.md`
denselben `doc_id` `01AAA` tragen. Diesen Zustand kann der Indexer nicht
erzeugen. Der Entdopplungspfad ist damit gegen einen Registerzustand geprüft,
den es im Betrieb nicht gibt.

Was der Mechanismus **tatsächlich** leistet: er fasst mehrere Treffer desselben
Pfades zusammen. Auch das kam in diesem Lauf nicht vor — qmd 2.8.3 liefert bei
`query` je Datei genau eine Zeile (20 Zeilen, 20 verschiedene `docid`, geprüft
am rohen `--json`).

### Punkt 4 — Ausfall der Suchmaschine: **bestanden**

`qmd` wurde nur für die betreffenden Aufrufe aus `PATH` genommen; die
Installation blieb unangetastet.

```
$ PATH="$OHNE_NPM" which qmd
(nicht auf PATH)

$ PATH="$OHNE_NPM" uv run brain search "Ortsmodell" --scope space -n 3
error: cannot find 'qmd' on PATH
exit=1

$ PATH="$OHNE_NPM" uv run brain catalog
# brain

* [iam-wiki](brain://iam-wiki/)
* [knowledge](brain://knowledge/)
* [obsidian-ai](brain://obsidian-ai/)
* [space](brain://space/)
exit=0

$ PATH="$OHNE_NPM" uv run brain read "CLAUDE.md" --scope space
# space

Weltraum-Handels- und Erkundungsspiel in Godot 4 für Android und Windows.
exit=0

$ which qmd
/c/Users/micro/AppData/Roaming/npm/qmd
```

### Punkt 5 — Leere Suche: **bestanden (Profil `keyword`)**, mit Vorbehalt

Beide Fälle nebeneinander, mit getrennt aufgefangenem stdout und stderr:

```
=== A) leeres Ergebnis ===
$ uv run brain search "zzqqxwvk-nichtvorhanden-42" --scope space --profile keyword -n 5
exit=0
stdout: [no matches]
stderr: []

=== B) Suchmaschine weg ===
$ PATH="$OHNE_NPM" uv run brain search "zzqqxwvk-nichtvorhanden-42" --scope space --profile keyword -n 5
exit=1
stdout: []
stderr: [error: cannot find 'qmd' on PATH]
```

Der Unterschied ist an drei Stellen zugleich ablesbar — Rückgabewert 0 gegen 1,
`no matches` gegen leeres stdout, leeres stderr gegen eine benannte Ursache.
Das ist, was §16.14 verlangt.

Der Vorbehalt steht unter Befund B3: mit dem Standardprofil `full` ist
`no matches` praktisch unerreichbar.

### Punkt 6 — Echter Nutzen: **bestanden**

Zwei Fragen mit vorher benannter Zielquelle, Standardprofil, über alle
Bereiche:

```
$ uv run brain search "Warum wurde das Galaxy-Panel abgelöst?" -n 3
brain://space/wiki/architecture/panels-und-zeitsteuerung.md:41  100%  Struktur
brain://space/wiki/architecture/modulschnitt-fundament.md:78  63%  Struktur
brain://space/wiki/architecture/galaxy-panel-abgeloest.md:4  62%  Struktur
                                    ^^^ Zielquelle, Platz 3

$ uv run brain search "Wie funktioniert der Determinismus im Rundlauf?" -n 3
brain://space/wiki/decisions/galaxienreise-sprungantrieb.md:16  75%  Kontext
brain://space/wiki/architecture/spielfluss-und-autosave.md:280  61%  Struktur
brain://space/wiki/architecture/determinismus-rundlauf.md:17  50%  Struktur
                                    ^^^ Zielquelle, Platz 3
```

Beide Male unter den ersten drei — aber beide Male auf Platz drei, hinter zwei
Seiten, die das Thema nur streifen. „Unter den ersten drei" ist erfüllt; ein
Platz eins war es nicht.

---

## 3. Was `status` meldet

```
$ uv run brain status
space: only 7 of 1124 links resolved (anchor=1, external=7, outside_area=2, unknown_target=1107)
exit=0
```

Ein einziger Befund, und er ist gravierend: von 1124 Verweisen in `space`
lösen sich 7 auf. Die Ursache ist gefunden. `space` schreibt seine Verweise
absolut relativ zur Wiki-Wurzel:

```
$ grep -oE '\]\([^)]+\)' wiki/architecture/bahnmechanik.md | head -4
](/balancing/zeitraffer.md)
](/references/masse-leuchtkraft-relation.md)
](/architecture/ortsmodell.md)
](/architecture/modulschnitt-fundament.md)
```

Die Wiki-Wurzel ist `space/wiki`, unsere Bereichswurzel ist `space`. Genau
dafür trägt `Area` das Feld `wiki_path`, und die Registry liest es aus — aber
in 2a benutzt es niemand: `wiki_path` kommt außerhalb von `models.py` und
`registry.py` in keinem Modul vor. Der Knopf existiert und ist nicht
angeschlossen. Der Befund ist damit in 2a nicht konfigurierbar wegzubekommen
und geht unverändert an 2b.

`iam-wiki`, `obsidian-ai` und `knowledge` melden nichts. Auf beiden Kanälen
identische Ausgabe.

---

## 4. Die Zahlen

| Bereich | eigener Index | qmd | Differenz |
|---|---|---|---|
| `space` | 207 | 146 | 61 |
| `iam-wiki` | 68 | 68 | 0 |
| `obsidian-ai` | 23 | 23 | 0 |
| `knowledge` | 0 | 1 | −1 |

qmd insgesamt: 239 Dokumente, 1497 Vektoren, 24 ohne Einbettung, 510
verwaiste Einbettungsbrocken (34 %).

Laufzeiten:

| Vorgang | Zeit |
|---|---|
| `brain reindex`, Lauf 1 | 24,2 s |
| `brain reindex`, Lauf 2 | 22,3 s |
| `brain reindex`, Lauf 3 | 21,7 s |
| `brain search --profile keyword` | 0,40 s |
| `brain search --profile vector` | 2,99 s |
| `brain search --profile fast` | 3,96 s |
| `brain search --profile full`, warm | 3,34 s |
| `brain search --profile full`, kalt | 13–14 s |
| leere Suche, `keyword` (mit Wiederholung) | 0,61 s |
| leere Suche, `full` (mit Wiederholung) | 8,8 s |

Wiederholbarkeit: die Läufe 2 und 3 erzeugten byteweise identische Artefakte
(alle Dateien unter `areas/`, md5 je Datei verglichen — kein Unterschied). Lauf
1 gegen Lauf 2 unterschied sich, weil sich `space` zwischen beiden Läufen
geändert hat: in einer parallelen Sitzung des Nutzers wurde dort während der
Abnahme gearbeitet und um 14:21 committet. Die Nichtwiederholbarkeit liegt am
bewegten Bestand, nicht am Indexer.

---

## 5. Der Eingriff in die qmd-Konfiguration

Der `models:`-Block ist unverändert — der Vergleich des Blocks aus der
Handkopie mit dem Block der jetzigen Datei ergibt keinen Unterschied.

Die Sicherungskopie `index.yml.brain-backup` trägt den ursprünglichen Inhalt —
mit einem Vorbehalt, siehe Befund B2: der Vergleich ist zeichengleich, sobald
Wagenrückläufe entfernt werden, und nur dann.

Vier Sammlungen statt drei: `brain reindex` hat den bereits registrierten
Bereich `knowledge` als vierte qmd-Sammlung angelegt. Das folgt aus Regel und
Freigabe (die Einträge der registrierten Bereiche werden angeglichen), war aber
in der Erwartung des Auftrags nicht genannt.

---

## 6. Die Bestandsregel — Beleg

Kein Schreibzugriff in einen der drei Bäume. Zwei voneinander unabhängige
Nachweise.

**Erstens**, keiner unserer Artefaktnamen wurde in einen der Bäume geschrieben.
`iam_wiki` ist nach dem ganzen Lauf vollständig sauber (`git status --short`
gibt keine Zeile aus), obwohl der Baum 22 git-verfolgte `index.md` enthält —
hätte der Indexer dorthin geschrieben, wären sie als geändert erschienen.
`#Obsidian/AI` ist gegenüber der Ausgangsaufnahme Datei für Datei unverändert,
Größen und Änderungszeiten eingeschlossen.

**Zweitens**, alle erzeugten Dateien liegen im Zustandsverzeichnis:

```
C:/Users/micro/AppData/Local/brain/areas/space/{index.md, graph.json, _identities.tsv, ...}
C:/Users/micro/AppData/Local/brain/areas/iam-wiki/...
C:/Users/micro/AppData/Local/brain/areas/obsidian-ai/...
```

`space` ist am Ende **nicht** sauber, und das ist zu erklären: eine parallele
Arbeitssitzung des Nutzers hat während der Abnahme in diesem Repo gearbeitet.
Ausgangsaufnahme um 14:20 gegen Endstand:

```
Ausgangsaufnahme                       Endstand
 M harness/probe_driver.gd              A  model/ledger.gd
 M wiki/architecture/arbeitsreihen...   A  model/ledger.gd.uid
 ...                                    A  test/model/ledger_test.gd
 ?? core/ledger_rules.gd                A  test/model/ledger_test.gd.uid
 ?? test/core/ledger_rules_test.gd      M  wiki/... (dieselben, plus koerperfarben.md)
```

Die vier vormals unverfolgten Dateien sind nicht verschwunden, sondern durch
einen Commit `290ea9c feat: the four numbers of the cost side` um 14:21:27 in
die Historie gewandert; die Dateien liegen nach wie vor auf der Platte. Neu
hinzugekommen sind `.gd`-Dateien und eine weitere `.md` — alles Arbeit der
parallelen Sitzung, nichts davon aus unserem Lauf. Es wurde vorsichtshalber
gesondert geprüft: unter `space` existiert keine `.brain.toml`, keine
`graph.json` und keine `_identities.tsv` außerhalb von `.claude/worktrees/`
und `.obsidian/`, wo sie schon vorher lagen.

---

## 7. Was überrascht hat

Dies ist der Teil, für den die Abnahme gemacht wurde.

### B1 — `brain search` stürzt auf Windows an einem Gedankenstrich ab

Der allererste echte Suchaufruf endete nicht mit einem Ergebnis, sondern mit
einem Traceback:

```
$ uv run brain search "auto dream image ai manager" --scope obsidian-ai -n 5
Traceback (most recent call last):
  ...
  File ".../src/brain/cli.py", line 215, in _print_search
    print(
        f"brain://{result.scope}/{result.relative}:{result.line}  "
        f"{result.score:.0%}  {result.title}"
    )
  File ".../Lib/encodings/cp1252.py", line 19, in encode
    return codecs.charmap_encode(input,self.errors,encoding_table)[0]
UnicodeEncodeError: 'charmap' codec can't encode character '\u2192' in position 92
exit=1
```

Der Titel einer Seite enthält ein `→`. Pythons stdout ist auf dieser Maschine
cp1252, `→` hat dort keine Entsprechung, und das Programm bricht mitten in der
Trefferliste ab — die ersten drei Treffer waren schon gedruckt, der vierte
riss alles mit. Das ist der schwerste Fund des Laufs, und zwar aus drei
Gründen: er trifft die Standardausgabe des Hauptbefehls, er trat in der ersten
Minute des ersten echten Gebrauchs auf, und er tritt bei jeder Anfrage auf,
die zufällig eine Seite mit einem Nicht-Latin-1-Zeichen im Titel trifft — in
einem deutschsprachigen Bestand mit Gedankenstrichen und Pfeilen ist das kein
Randfall. Die Scheibe hat rund 250 grüne Tests und einen Commit namens „pin
non-ASCII output"; keiner davon hat die Kodierung von `sys.stdout` im echten
Prozess berührt.

Alle weiteren Läufe dieses Protokolls liefen mit `PYTHONIOENCODING=utf-8`.
Der Produktivcode wurde nicht angefasst.

### B2 — Die Sicherungskopie ist nicht byteweise das Original

`sync_collections` schreibt die Sicherungskopie mit
`backup.write_text(raw, encoding="utf-8")` — ohne `newline=""`, das die
eigentliche Konfigurationsdatei zwei Zeilen weiter unten sehr wohl bekommt.
Python übersetzt dabei jedes `\n` in `\r\n`. Das Original hatte LF:

```
Original (Handkopie): 1018 Bytes, ASCII text, md5 e16a62b3615e9c7e560571ec87f38b66
Backup:               ASCII text, with CRLF line terminators, md5 8019e8219d2392947ee52f870c3f2630
```

Inhaltlich identisch, byteweise nicht. Für ein YAML, das ein anderes Werkzeug
liest, ist das folgenlos; für eine Sicherungskopie, deren ganzer Zweck der
unveränderte Ausgangszustand ist, ist es der falsche Anspruch. Der Riegel
hält, aber er hält weniger fest, als er verspricht.

### B3 — Auf dem Standardprofil gibt es kein „nichts gefunden"

Abnahmepunkt 5 hat auf `--profile keyword` sauber bestanden. Auf dem
Standardprofil `full` liefert dieselbe Unsinnsanfrage fünf Treffer:

```
$ uv run brain search "zzqqxwvk-nichtvorhanden-42" --scope space -n 5
brain://space/wiki/architecture/eigene-presets.md:1  75%  Struktur
brain://space/wiki/decisions/zeitmodell-ticks.md:1  38%  Kontext
brain://space/wiki/architecture/eingabeschicht.md:1  25%  Struktur
brain://space/wiki/architecture/hook-system.md:1  15%  Struktur
brain://space/wiki/architecture/gdscript-pruefkette.md:1  12%  Struktur
```

75 % auf eine Zeichenfolge, die im gesamten Bestand nicht vorkommt. qmds
Frageerweiterung dichtet aus dem Unsinn eine plausible Frage, der Reranker
sortiert das Ergebnis und gibt ihm eine Zahl. §16.14 verlangt, dass „nichts
gefunden" von „Fehlschlag" unterscheidbar ist — das ist erfüllt. Aber der
Zustand „nichts gefunden" ist auf dem Vorgabeprofil überhaupt nicht
erreichbar, und was stattdessen kommt, sieht mit 75 % genauso aus wie ein
guter Treffer. Nebenbei erklärt das auch, warum die Wiederholung bei leerem
Ergebnis (bekannte Einschränkung) im Regelbetrieb kaum je greift: leere
Ergebnisse gibt es auf `full` nicht.

### B4 — Der Abgleich der Ausschlussliste hat nichts bewirkt

Der Auftrag erwartete, dass `space` nach dem Abgleich rund 178 Dateien mehr
enthält, weil unsere Liste `**/.superpowers/sdd/**` sagt statt
`**/.superpowers/**`. Eingetreten ist: nichts.

```
$ qmd update
Indexed: 0 new, 2 updated, 144 unchanged, 0 removed
$ qmd ls space | grep -c ".superpowers"
0
```

qmd betritt Punktverzeichnisse gar nicht — die Ausschlussliste ist an dieser
Stelle nicht der Hebel, für den wir sie gehalten haben. Der ganze
Kopplungsaufwand aus Entscheidung 39 (fremde Konfigurationsdatei schreiben,
Sicherungskopie, Vertragstest, Rückweg) ist für die Fälle bezahlt, in denen
er nichts ändert.

Die Folge ist die eigentliche Überraschung: **61 Dateien stehen in unserem
Index und können von der Suche nie gefunden werden.**

```
$ comm -23 unsere-pfade.txt qmd-pfade.txt | sed 's|\(docs/[^/]*/[^/]*\)/.*|\1|' | sort | uniq -c
     28 docs/.superpowers/plans
     33 docs/.superpowers/specs
```

Specs und Pläne — genau die Dokumente, die die Spec unter §5.3 ausdrücklich zu
Quellen erklärt. Sie stehen in `index.md` und in `graph.json`, `brain read`
gibt sie heraus, `brain search` findet sie nie. Und `brain status` sagt dazu
kein Wort: die Divergenz zwischen den beiden Indizes wird nur dann gemeldet,
wenn ein Treffer im Register fehlt — der umgekehrte Fall, Register kennt mehr
als die Suchmaschine, hat keine Meldung. Entscheidung 40 wollte, dass das
Auseinanderlaufen sichtbar wird. In der Richtung, die hier eingetreten ist,
ist es unsichtbar.

### B5 — `**/index.md` schließt fremde Wiki-Seiten aus

`DEFAULT_EXCLUDES` enthält `**/index.md`, weil der Indexer selbst Dateien
dieses Namens erzeugt und ein zweiter Lauf sonst läse, was der erste schrieb.
Für einen fremden Bestand ist das eine Namenskollision mit Folgen:

```
$ git -C iam_wiki ls-files | grep -c "index.md"
22
$ grep -c "index.md" areas/iam-wiki/_identities.tsv
0
$ qmd ls iam-wiki | grep -c "index.md"
0
```

22 von Hand geschriebene, eingecheckte Wiki-Seiten — in einem Wiki genau die
Navigationsebene — sind in beiden Indizes unsichtbar. Auch dazu schweigt
`status`. Bei einem `readonly`-Bereich schreiben wir ohnehin nichts in den
Baum, der Ausschluss ist dort also nicht nur schädlich, sondern
gegenstandslos.

### B6 — Der Entdopplungspfad ist im Betrieb unerreichbar

Siehe Punkt 3. Was daran überrascht, ist nicht die fehlgeschlagene Abnahme,
sondern dass die Lücke unter einem grünen Test lag: die Fixture schreibt ein
Register, in dem zwei Pfade denselben `doc_id` tragen, und der Indexer kann
ein solches Register nicht erzeugen. Der Test prüft eine Welt, die es nicht
gibt. Er ist nicht falsch — er ist unverankert.

### B7 — `qmd update` bettet nicht ein

Nach jedem `brain reindex` meldet qmd:

```
Pending:  24 need embedding (run 'qmd embed')
Orphaned: 510 embedding chunks (34%) — run 'qmd cleanup'
```

`refresh()` ruft `qmd update`, und das schreibt nur den Volltextindex fort.
Die Vektoren entstehen in einem zweiten Befehl, den `brain` nicht kennt. Nach
einem `reindex` sind neue Dokumente also über `search` erreichbar, über
`vsearch` aber nicht — die Suchkette ist nach ihrem eigenen
Aktualisierungslauf halb aktuell, und nichts in unserer Ausgabe sagt das. Das
gehört zu den Kosten, die 2b messen soll; hier ist es zuerst ein Befund, den
`status` nicht meldet.

### B8 — Der Bestand steht nicht still

Während der Abnahme hat eine parallele Sitzung in `space` gearbeitet und
committet. Dass die byteweise Gleichheit zweier Läufe darüber stolpert, ist
kein Fehler des Indexers — aber es ist eine Eigenschaft der echten Umgebung,
die im Arbeitsbereich nicht vorkommt und die jede Aussage der Form „zwei Läufe
sind identisch" ab jetzt an einen Zeitpunkt bindet. Die Läufe 2 und 3, dicht
hintereinander, waren identisch; Lauf 1 und 2 nicht.

---

## 8. Bilanz

| Punkt | Ergebnis |
|---|---|
| 1 Datenschutz, Kanal | bestanden |
| 2 Datenschutz, Pfad | bestanden |
| 3 Entdopplung | **nicht bestanden** |
| 4 Ausfall der Suchmaschine | bestanden |
| 5 Leere Suche | bestanden auf `keyword`, unerreichbar auf `full` |
| 6 Echter Nutzen | bestanden (Zielquelle je Platz 3) |

Zusätzlich blockiert Befund B1 den normalen Gebrauch auf dieser Maschine, bis
die Ausgabekodierung festgelegt ist.

Die Bestandsregel wurde eingehalten. Es wurde kein Produktivcode geändert.

---

# Nachtrag nach der Nacharbeit

Datum: 2026-08-20, später Abend. Stand `9619c5f`. Dieselbe Maschine, dasselbe
Zustandsverzeichnis `C:/Users/micro/AppData/Local/brain`, dieselben drei
Bestände. Alle Läufe mit `PYTHONIOENCODING=utf-8` — Befund B1 ist unverändert
offen, und der Produktivcode wurde auch diesmal nicht angefaßt.

Nachgefahren wurden die drei Punkte, deren Bedeutung sich durch die Nacharbeit
geändert hat, dazu alles, was an `status`, `embed` und der Profilvorgabe neu
ist.

## N.1 Was eingerichtet wurde

Geändert wurde genau eine Datei außerhalb des Repos:
`C:/Users/micro/AppData/Local/brain/areas/space/.brain.toml`.

```toml
[area]
scope    = "space"
readonly = true

[index]
include = ["**/*.md"]
# Specs und Pläne liegen in einem Punktverzeichnis, das die Suchmaschine
# grundsätzlich nicht betritt. Sie bleiben in unserem Index (`read` gibt sie
# heraus), sind aber nicht durchsuchbar — das ist gewollt und wird hier erklärt,
# damit `status` es nicht als Divergenz meldet.
unsearched = ["docs/.superpowers/**"]
```

Für `iam-wiki` und `obsidian-ai` wurde **nichts** gesetzt, und das ist
nachgesehen statt angenommen:

```
$ find "#Obsidian/AI" -path '*/.*/*' -name '*.md'
(keine Zeile)

$ find iam_wiki -path '*/.*/*' -name '*.md' | sed -E 's|^(\.superpowers/[^/]+)/.*|\1|' | sort | uniq -c
     26 .superpowers/sdd
```

`obsidian-ai` trägt überhaupt kein Punktverzeichnis mit Quellen. `iam-wiki`
trägt eines, aber alle 26 Dateien liegen unter `.superpowers/sdd/`, und das
steht bereits in `ALWAYS_EXCLUDES` — sie sind in unserem Register gar nicht
erst enthalten, es gibt also nichts abzuziehen. Beleg: die Suche nach
Punktverzeichnis-Pfaden in den drei Registern liefert **nur** für `space`
Treffer (61 unter `docs/.superpowers`), für `iam-wiki` und `obsidian-ai` keine
einzige Zeile.

```
$ uv run brain reindex
updated qmd collections: iam-wiki, knowledge, obsidian-ai, space

real    0m23.491s
```

## N.2 Die drei nachgefahrenen Punkte

### Punkt 1 — `status` schweigt über die 61 Dokumente: **bestanden**

Vorher, unmittelbar vor der Manifeständerung:

```
$ uv run brain status
space: only 7 of 1124 links resolved (anchor=1, external=7, outside_area=2, unknown_target=1107)
space: 61 of 207 indexed documents are unknown to the search engine, so `brain search` can never return them; e.g. docs/.superpowers/plans/2026-07-29-wiki-und-hooks.md, docs/.superpowers/plans/2026-07-30-fundament.md, docs/.superpowers/plans/2026-07-31-handover-manuelle-schritte.md, …
24 documents are indexed but not yet searchable; run `brain embed`
exit=0

real    0m1.315s
```

Nachher:

```
$ uv run brain status
space: only 157 of 1320 links resolved (anchor=1, external=7, outside_area=2, unknown_target=1153)
36 documents are indexed but not yet searchable; run `brain embed`
exit=0

real    0m1.269s
```

Die Divergenzzeile ist verschwunden. `unsearched` greift. Die Zahlen gehen
sauber auf: unser Register führt für `space` 222 Dokumente, davon 63 unter
`docs/.superpowers`; qmd führt 159. 222 − 63 = 159.

Und die Erklärung erklärt, ohne zu verstecken — das Dokument bleibt lesbar und
bleibt unauffindbar, beides belegt:

```
$ uv run brain read "docs/.superpowers/plans/2026-08-20-kostenseite.md" --scope space
# Die Kostenseite (B1b) — Implementation Plan
...
exit=0

$ uv run brain search "Kostenseite Unterhalt Implementation Plan" --scope space --profile keyword -n 3
brain://space/wiki/balancing/unterhalt.md:11  92%  Formel
brain://space/wiki/design/ownership/kostenseite.md:10  92%  Zweck
brain://space/wiki/balancing/credit-kurve.md:11  90%  Formel
```

### Punkt 2 — `iam-wiki` hat 22 Dokumente mehr: **bestanden**

Register vorher und nachher (`wc -l`; die erste Zeile ist die Kopfzeile):

| Bereich | vorher | nachher | Differenz |
|---|---|---|---|
| `space` | 208 (207 Dok.) | 223 (222 Dok.) | +15 |
| `iam-wiki` | 69 (68 Dok.) | 91 (90 Dok.) | **+22** |
| `obsidian-ai` | 24 (23 Dok.) | 24 (23 Dok.) | 0 |

```
$ grep -c "index.md" areas/iam-wiki/_identities.tsv
22
```

Exakt die 22, die Befund B5 benannt hat. Und sie sind nicht nur im Register,
sondern auch in der Suchmaschine (`qmd ls iam-wiki | wc -l` → 90, vorher 68)
und über `brain search` erreichbar:

```
$ uv run brain search "Projekte Backend Frontend Workers Wissen zum Repo" --scope iam-wiki --profile keyword -n 5
brain://iam-wiki/index.md:20  93%  IAM-Wiki
brain://iam-wiki/docs/superpowers/plans/2026-07-27-iam-wiki-scaffold.md:7  93%  IAM-Wiki-Gerüst Implementation Plan
brain://iam-wiki/projects/index.md:3  93%  Projekte
brain://iam-wiki/docs/superpowers/specs/2026-07-29-repo-doku-aufraeumen-design.md:10  92%  Design: Doku in den Code-Repos aufräumen
brain://iam-wiki/docs/superpowers/plans/2026-07-28-iam-docs-migration-cutover.md:5  92%  iam_docs → iam_wiki: Migration und Umschaltung — Implementation Plan
```

Zwei der drei ersten Plätze sind Navigationsseiten, die vorher in beiden
Indizes nicht existierten. Der Punkt ist damit nicht nur gezählt, sondern
benutzt.

Die +15 bei `space` sind vollständig aufgeschlüsselt: **+11** sind
`wiki/**/index.md` und gehen auf dieselbe Änderung zurück; **+4** stammen aus
dem bewegten Bestand (Befund B8) und stehen in der git-Historie von `space` —
`wiki/balancing/unterhalt.md` und
`wiki/decisions/keine-internen-einheiten-vor-dem-spieler.md` (Commit `399044b`,
19:59) sowie zwei Dateien unter `docs/.superpowers/` (Commits `ad5ad3f`, 21:56
und `d739b09`, 22:04), die auch die 61 auf 63 hochgezählt haben.

### Punkt 3 — Trefferliste nicht mehr entdoppelt, `status` meldet Hashes: **bestanden, mit einem Fund**

Innerhalb jedes Bereichs gibt es weiterhin keine zwei Pfade mit gleichem
Inhalt, und `status` schweigt folgerichtig:

```
$ for a in space iam-wiki obsidian-ai; do awk -F'\t' 'NR>1{print $3}' $a/_identities.tsv | sort | uniq -d | wc -l; done
0
0
0
```

Die Aussage des ersten Laufs gilt also unverändert. Der Fund steht in N.4 unter
B10: **über Bereichsgrenzen hinweg gibt es den Fall sehr wohl**, und dort ist er
unsichtbar.

## N.3 Was neu ist

### `status` insgesamt

Die Meldung, die Befund B4 trug, ist weg; die Meldung, die Befund B7 trug, ist
da. Jede Zeile, die dieser Lauf gesehen hat, steht oben unter Punkt 1 im
Wortlaut. Zusammengefaßt:

| Zeile | vorher | nachher |
|---|---|---|
| Verweisauflösung `space` | `only 7 of 1124` | `only 157 of 1320` |
| Divergenz `space` (61 Dok.) | gemeldet | schweigt (`unsearched`) |
| Einbettungsrückstand | `24 documents` | `36 documents`, nach `embed` weg |
| geteilte Inhaltshashes | — | keine (innerhalb der Bereiche gibt es keine) |

`status` läuft in 1,3 s — die bekannte Einschränkung „ein Prozeß je Bereich" ist
auf dieser Maschine nicht spürbar.

### `brain embed`

```
$ uv run brain status
...
36 documents are indexed but not yet searchable; run `brain embed`

$ uv run brain embed
embedded 4 area(s)

real    0m18.679s
exit=0

$ uv run brain status
space: only 157 of 1320 links resolved (anchor=1, external=7, outside_area=2, unknown_target=1153)
exit=0
```

`status` nennt den Rückstand vorher und schweigt nachher. Der Befehl braucht
18,7 s, nicht Minuten — die Modelle lagen warm.

### Die Profilvorgabe

`brain search` ohne `--profile` benutzt `fast`. Zeile für Zeile derselbe
Wortlaut wie `--profile fast`, und deutlich anders als `--profile full`:

```
$ uv run brain search "Warum wurde das Galaxy-Panel abgelöst?" -n 3
brain://space/wiki/architecture/galaxy-panel-abgeloest.md:4  51%  Struktur
brain://space/wiki/architecture/panels-und-zeitsteuerung.md:532  51%  Struktur
brain://space/wiki/architecture/ebenenmodell.md:219  43%  Struktur

$ ... --profile fast -n 3      → identisch
$ ... --profile vector -n 3    → identisch

$ ... --profile full -n 3
brain://space/wiki/architecture/galaxy-panel-abgeloest.md:4  100%  Struktur
brain://space/wiki/architecture/modulschnitt-fundament.md:78  63%  Struktur
brain://space/wiki/architecture/panels-und-zeitsteuerung.md:41  62%  Struktur
```

Laufzeiten, warm:

| Profil | Zeit |
|---|---|
| ohne `--profile` | 3,20 s |
| `fast` | 3,08 s |
| `full` | 3,20 s |
| `keyword` | 0,41 s |

## N.4 Was diesmal überrascht hat

### B9 — Das Standardprofil kostet den Nutzenpunkt die Hälfte

Die Umstellung der Vorgabe auf `fast` hat Abnahmepunkt 6 verschlechtert, und
zwar in beide Richtungen zugleich:

```
Frage 1, Standardprofil (jetzt fast)   Zielquelle auf Platz 1        (erster Lauf: Platz 3)
Frage 2, Standardprofil (jetzt fast)   Zielquelle NICHT unter den ersten drei
Frage 2, --profile full                Zielquelle auf Platz 3        (wie im ersten Lauf)
```

im Wortlaut:

```
$ uv run brain search "Wie funktioniert der Determinismus im Rundlauf?" -n 3
brain://space/wiki/decisions/galaxienreise-sprungantrieb.md:16  46%  Kontext
brain://space/wiki/open-questions/ressourcen-aus-der-physik.md:3  35%  Frage
brain://space/wiki/architecture/ebenenmodell.md:45  35%  Struktur

$ uv run brain search "Wie funktioniert der Determinismus im Rundlauf?" --profile full -n 3
brain://space/wiki/decisions/galaxienreise-sprungantrieb.md:16  75%  Kontext
brain://space/wiki/architecture/spielfluss-und-autosave.md:280  62%  Struktur
brain://space/wiki/architecture/determinismus-rundlauf.md:17  55%  Struktur
```

`wiki/architecture/determinismus-rundlauf.md` — die Seite, die die Frage im
Titel trägt — ist auf dem neuen Vorgabeprofil verschwunden. Punkt 6 wäre in
dieser Form heute **nicht bestanden**, und zwar nicht wegen des Bestands,
sondern wegen der Vorgabe. Der Reranker war das, was diese Seite nach oben
gezogen hat.

Das wiegt schwerer, weil die Begründung im Code („die 100–500 ms des Rerankers
sind es nicht wert", `cli.py`) sich warm nicht messen läßt: `full` kostete
3,20 s, `fast` 3,08 s. Die 120 ms Unterschied sind Rauschen zwischen zwei
Läufen. Der Preis, den die Umstellung tatsächlich spart, ist das kalte Laden
des Reranker-Modells (im ersten Lauf 13–14 s gegen 3,3 s warm) — eine Aussage
über den ersten Aufruf nach einem Kaltstart, nicht über den Regelbetrieb.
Bezahlt wird sie mit jeder Anfrage.

### B10 — Geteilte Inhaltshashes gibt es, nur nicht dort, wo hingesehen wird

Der erste Lauf stellte fest, es gebe in keinem der Bestände zwei Pfade mit
gleichem Inhalt. Das stimmt — solange man je Bereich zählt. Über die drei
Register hinweg gezählt:

```
== sha256:5a3311d270bebb16d558010e75064f5b75323f284992641732b1c8097511f948
   space     SPEC.md
   iam-wiki  SPEC.md
== sha256:dc3efe98ae62f23dd08acad13aba2e95287beb20b6bec2f4af0423557fe37401
   space     llm-wiki.md
   iam-wiki  llm-wiki.md
```

Zwei Dateipaare, byteweise identisch, in zwei verschiedenen Bereichen. Eine
Anfrage über alle Bereiche liefert sie zweimal, und `status` sagt nichts, weil
`_shared_hashes(area, state_dir)` je Bereich aufgerufen wird und nur dessen
Register liest. Der Fall, für den die zurückgenommene Entdopplung gedacht war,
existiert also im echten Bestand — nur eine Ebene höher als die Stelle, an der
die Ersatzmeldung nachsieht. Die bekannte Einschränkung sagt, das Problem käme
zurück, „registriert jemand eine Spiegelung — dann aber mit einem Bestand als
Beleg". Der Beleg lag schon vor, ungesehen.

Das ist ausdrücklich **kein** Ruf nach der Entdopplung zurück. Es ist die
Feststellung, daß die Aussage „es gibt den Fall nicht" enger galt, als sie
formuliert war.

### B11 — Die Navigationsseiten haben nebenbei die Verweisauflösung vervielfacht

Erwartet war eine Zahl: 22 Dokumente mehr in `iam-wiki`. Nicht erwartet war,
was dieselbe Änderung in `space` anrichtet:

```
vorher:   space: only   7 of 1124 links resolved
nachher:  space: only 157 of 1320 links resolved
```

Von 7 auf 157 aufgelöste Verweise — Faktor 22. Der Grund: die 11
`wiki/**/index.md` sind nicht nur Quellen, sondern **Ziele**. Ein Wiki verlinkt
seine Ordner (`[Backend](backend/)`), und ein Ordnerverweis löst sich auf das
`index.md` darin auf. Solange die `index.md` nicht im Register standen, war
jeder solche Verweis ein `unknown_target`. Befund B5 war als Sichtbarkeitsfrage
protokolliert („22 Seiten sind unsichtbar"); er war zugleich, unbemerkt, die
zweitgrößte Ursache für den schwersten offenen Befund des Protokolls. Die
verbleibenden 1153 unaufgelösten Verweise sind unverändert das
`wiki_path`-Problem aus Abschnitt 3 — die absolute Schreibweise ist die größere
Ursache, aber eben nicht die einzige.

### B12 — `fast` und `vector` sind derselbe Aufruf

`_SUBCOMMAND` bildet `Profile.FAST` und `Profile.VECTOR` beide auf `vsearch`
ab. Die Ausgaben sind oben Zeile für Zeile identisch. Vier Profile an der
Oberfläche, drei Verhaltensweisen darunter. Das ist kein Fehler und vermutlich
Absicht (`fast` benennt eine Zusage, `vector` ein Verfahren), aber im ersten
Protokoll stehen beide als getrennte Messungen — 2,99 s gegen 3,96 s — und
dieser Unterschied kann nichts anderes gewesen sein als Rauschen. Wer die
Tabelle in Abschnitt 4 liest, liest dort einen Unterschied, den es nicht gibt.

### B13 — `status` ist schnell, und niemand hatte das gemessen

Die bekannte Einschränkung sagt, `status` starte einen Prozeß je Bereich, das
sei „bei vier Bereichen spürbar, ungemessen". Gemessen: 1,3 s, beide Male, vor
und nach der Nacharbeit. Die Einschränkung beschreibt eine Sorge, die auf
dieser Maschine nicht eintritt. Sie bleibt für 2b stehen — aber als Zahl, nicht
als Vermutung.

## N.5 Die Bestandsregel — Beleg

Kein Schreibzugriff in einen der drei Bäume.

```
$ git -C "#GIT/space" status --short          (vorher wie nachher, Zeile für Zeile gleich)
 M core/colony_rules.gd
 M data/resource_catalog.gd
 M data/resource_entry.gd
 M test/core/colony_rules_test.gd
 M test/data/resource_catalog_test.gd

$ git -C "#GIT/iam_wiki" status --short
(keine Zeile)

$ find "#Obsidian/AI" -type f | wc -l
51        (vorher wie nachher)
```

`space` zeigt fünf geänderte `.gd`-Dateien — die parallele Arbeitssitzung des
Nutzers, unverändert gegenüber der Ausgangsaufnahme dieses Laufs, keine `.md`
darunter. Die Suche nach unseren Artefaktnamen (`graph.json`,
`_identities.tsv`, `index.intro.md`, `.brain.toml`) außerhalb von `.claude/`
und `.obsidian/` liefert in allen drei Bäumen keine einzige Datei.

Der schärfste Beleg ist diesmal ein anderer: seit dieser Nacharbeit liest der
Indexer die 22 git-verfolgten `index.md` in `iam_wiki` und die 11 in `space` —
und `git status` bleibt in beiden leer beziehungsweise ohne `.md`-Zeile. Er
liest sie und schreibt sie nicht.

Der `models:`-Block in `~/.config/qmd/index.yml` ist unverändert: die md5-Summe
des Blocks vor `brain embed` und danach ist beide Male
`ad62017f08c503ad70dc11288a4e4a94`.

## N.6 Bilanz des Nachtrags

| Punkt | erster Lauf | dieser Lauf |
|---|---|---|
| 1 `status` schweigt über die 61 | (Befund B4) | **bestanden** |
| 2 `iam-wiki` +22 Dokumente | (Befund B5) | **bestanden**, auch über `search` |
| 3 Entdopplung zurückgenommen | nicht bestanden | **bestanden**, mit Fund B10 |
| 6 Echter Nutzen (nachgeprüft) | bestanden, Platz 3 / Platz 3 | Frage 1 Platz 1, **Frage 2 nicht unter den ersten drei** |

Neu offen: B9 (Standardprofil kostet Treffer), B10 (geteilte Hashes über
Bereichsgrenzen hinweg), B12 (`fast` und `vector` identisch). Unverändert
offen: B1 (Ausgabekodierung), B2 (Sicherungskopie mit CRLF), B3 (kein „nichts
gefunden" auf `full`), das `wiki_path`-Problem aus Abschnitt 3.

Es wurde kein Produktivcode geändert.
