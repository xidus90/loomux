# Lanes konfigurieren statt treiben, und jede gepflegte Version tragen

Godot ist der erste Fall; der Zuschnitt gilt für jede Sprache.

Stand 2026-10-16, Basis `master` 18566e83 (nach v1.7.0). Noch kein eigener
Zweig; der Checkout stand beim Schreiben auf `feat/llm-config-switch`.

## Anlass

loomux erkennt ein Godot-Projekt an `project.godot` und gibt ihm die Lanes des
Stacks `gdscript`. Erkannt wird dabei jedes Godot-Projekt, getragen aber nur
die 4er-Linie — und auch die nur so weit, wie der mitgelieferte Parser reicht.
Drei Befunde aus einer Sitzung an `space` (2026-10-16):

1. **Ein Godot-3-Projekt kommt nicht einmal bis zur Lane.** `projectGodot`
   liest die gewünschte Version ausschließlich aus `config/features`
   (`internal/verify/godot.go:229`). Diese Einstellung gibt es in Godot 3
   nicht: in der ProjectSettings-Dokumentation von 3.6 fehlt
   `application/config/features` an seiner alphabetischen Stelle zwischen
   `description` und `icon`. Ein 3.6-Projekt bekommt daher
   `… names no version in config/features`, `{godot}` löst nicht auf, und
   `test` wie `coverage` fallen aus — nicht als „nicht zuständig", sondern als
   Fehler.
2. **Die Werkzeuge der Presets sind auf Godot 4 genagelt.** Das Lint-Preset
   ruft `uvx --from gdtoolkit` ohne Versionsbindung und holt damit immer die
   4er-Linie, obwohl gdtoolkit getrennte Linien veröffentlicht
   (`gdtoolkit==4.*` für Godot 4, `gdtoolkit==3.*` für Godot 3). Das
   Test-Preset erkennt die Suite über `plugin.cfg:gdUnit4`, und gdUnit4 ist
   ausschließlich Godot 4 (offizielle Tabelle beginnt bei 4.0; `master`
   v7.0.0 trägt 4.7, 4.7.1, 4.8.dev7). Das Coverage-Preset verlangt
   zusätzlich `plugin.cfg:NanoCoverage`, und Nano Coverage ist ebenfalls
   Godot-4-only.
3. **Die Grammatik des Graphen ist ein Blob und hängt in der Vergangenheit.**
   Der Extractor parst mit `gotreesitter v0.55.1`, dessen gdscript-Grammatik
   eine vorkompilierte Datei ohne Quelltext ist (`BlobSHA256`
   `18ac7fa3…`). Laut `THIRD_PARTY_NOTICES` ist das
   `PrestonKnopp/tree-sitter-gdscript @ 89e66b6b`. Dieser Stand kennt
   `remote` und `master` noch als Godot-3-Anweisungsschlüsselwörter: eine
   Anweisung, die mit einer so benannten Variablen beginnt, erzeugt einen
   Fehlerknoten (gemessen: `remote.x(1)` → 1, `q.extractors.append(1)` → 0,
   `var remote := 1` → 0). Im heutigen `master` derselben Grammatik sind diese
   Schlüsselwörter verschwunden; Annotationen laufen generisch über
   `@identifier`. Eine neuere Grammatik heilt also Godot 4 und verliert
   Godot 3.

Gemessen an `space` (706 Dateien, 4.7) sind das 104 Fehlerknoten in drei
Dateien: eine eigene (`godot/test/ui/game_root_colony_test.gd:497`, lokale
Variable `remote`) und zwei aus gdUnit4 (`GdDiffTool.gd` 5, `GdScriptParser.gd`
98). Der Parser ist tolerant — die Dateien stehen mit ihren Symbolen im
Graphen, nur die Region um die Fehlerstelle fällt aus. Der Schaden ist heute
also klein und still; das ist das Problem.

## Ziel und Erfolg

Ein fremdes Godot-Projekt, von dem niemand etwas konfiguriert hat, bekommt auf
jeder offiziell gepflegten Linie die Lanes, die es auf dieser Linie überhaupt
geben kann — und wo es eine nicht geben kann, sagt loomux das als Zustand,
nicht als Fehler.

Erfolg heißt:

- Ein Godot-3.6-Projekt mit GUT bekommt `lint`, `test` und `graph`. Die
  `coverage`-Lane erscheint nur, wenn ein Coverage-Addon gefunden wird, das
  auf dieser Linie läuft; sonst ist sie `unavailable` mit einer Begründung,
  die die Linie nennt.
- Ein Godot-4.8-Beta-Projekt läuft wie ein 4.7-Projekt, solange die
  Frameworks es tragen. Trägt eine Framework-Version die Linie nicht, ist die
  betroffene Lane `unready` und nennt gefundene und verlangte Version.
- `loomux doctor` zeigt pro Lane, gegen welche Godot-Linie sie läuft und
  welches Binary sie dafür nimmt.
- Wer mehrere Godot-Installationen auf der Maschine hat, muss keinen Eintrag
  umbiegen, wenn er zwischen Projekten wechselt.
- Die 104 Fehlerknoten an `space` sind entweder weg oder benannt: `graph
  build` sagt, welche Dateien betroffen sind, statt nur eine Summe zu nennen.
- Die Doku nennt je Stack die gängigen Frameworks und ob und in welchen
  Versionen loomux sie trägt, und die Tests prüfen genau diese Tabelle.

## Die Verträglichkeitsmatrix je Stack

Eine Forderung des Auftraggebers, die über Godot hinausreicht: Für jede
Sprache und jedes Framework gehört in die Doku eine Seite, die die gängigen
Test- und Werkzeug-Frameworks auflistet und zu jedem sagt, ob und in welchen
Versionen loomux es trägt. Was dort steht, muss sich in den Tests
wiederfinden.

Damit bekommt auch die Frage eine Antwort, die ohne sie offen bliebe: woher
loomux weiß, ob eine Framework-Version eine Godot-Linie trägt. Keine Datei
eines Frameworks sagt das von sich aus — gdUnit4 schreibt in seine
`plugin.cfg` die eigene Version, nicht die Spanne der Godot-Linien, die es
verträgt; diese Spanne steht nur in seiner README-Tabelle. Die Matrix ist also
Wissen, das loomux mitbringen muss.

Meine Festlegung zur Form: die Matrix liegt **als Daten** im Repo, eine Zeile
je Framework und Versionsspanne, und beides hängt daran — die Doku-Seiten (en
und de) und die Tests. Eine Tabelle, die nur in Prosa steht, läuft von den
Tests weg; eine, die nur im Go-Quelltext steht, erreicht den Leser nicht.
Welche Form die Daten haben und ob die Doku daraus erzeugt oder nur gegen sie
geprüft wird, bleibt dem Bauenden.

Zwei Dinge, die die Matrix nicht ist: keine Liste unterstützter
Godot-Versionen (die Linie liest loomux aus dem Projekt, siehe Umfang), und
kein Ersatz für `when_found` — die Matrix sagt, was eine gefundene
Framework-Version trägt, nicht ob ein Framework da ist.

**Das Muster steht schon im Repo.** `internal/verify/linecov/tools.toml`
erklärt Coverage-Werkzeuge als Daten: je Eintrag die Endungen, die
Berichtsdateien, die Einstellungsdateien in ihrer Suchreihenfolge und die
Schlüssel darin. Im Kopf der Datei steht sogar die Regel, die der Auftraggeber
verlangt — „der Lint-Test in `fixtures_test.go` verlangt ein Fixture-Projekt
für jeden Eintrag". Die Matrix je Stack ist dasselbe Muster, eine Ebene höher.

## Universeller Zuschnitt: Lanes wie tox

Der Auftraggeber verlangt, dass Lint, Typen, Unittests, Coverage und was sonst
kommt **über loomux konfiguriert** werden, ähnlich wie tox es für Python tut —
universell genug, dass im Grunde alles Vorhandene getragen wird. Daraus folgt
die Antwort auf die Frage, die das Vorhaben am stärksten im Umfang bewegt:
**ein neues Framework bekommt keinen Treiber in Go, sondern eine Lane in der
Konfiguration.** GUT wird also konfiguriert, nicht gebaut.

Der Bestand ist dafür besser aufgestellt als erwartet, aber nicht fertig:

- **Coverage ist schon formatgetrieben.** `check linecov` liest Cobertura,
  JaCoCo, LCOV, Go-Profile und strukturierte Formate und kennt die Werkzeuge
  als Daten. Ein fremdes Coverage-Addon braucht kein Go, sofern es ein
  bekanntes Format schreibt.
- **Die Lane-Schlüssel tragen viel:** `commands`, `on_file`, `measuring`,
  `measure`, `after`, `needs`, `when_found`, `skip_when_only`, `lock`,
  `threaded`, dazu die Platzhalter `{file}`, `{area}`, `{coverprofile}`,
  `{coverdata}`, `{loomux}`, `{godot}`.
- **Die Lücke ist der Testbericht.** Ein Urteil aus einem Bericht statt aus
  einem Exitcode gibt es nur innerhalb von `godotsuite`
  (`internal/verify/godotsuite/junit.go`); außerhalb existiert kein JUnit-Leser.
  Genau das braucht aber jedes Testframework, dessen Exitcode lügt — und das
  Preset hält als Regel fest, dass gdUnit4 eines davon ist.

Damit ist die zentrale Bauaufgabe benannt: ein formatgetriebener
**Testbericht-Leser** nach dem Vorbild von `linecov`, den eine Lane über eine
Konfigurationsangabe benutzt, statt dass loomux pro Framework Go-Code bekommt.
`check godot-suite` bleibt als vorhandener Treiber bestehen, ist aber ab jetzt
die Ausnahme und nicht das Vorbild; was es über JUnit weiß, gehört in den
gemeinsamen Leser.

## Versionspolitik: dem Hersteller folgen

Welche Versionen loomux trägt, entscheidet nicht loomux, sondern wer die
Sprache pflegt: getragen wird, was der Hersteller noch versorgt — bei Godot
also 3.6 LTS, 4.5 bis 4.7 und das laufende Beta, bei Go und Python
entsprechend deren eigene Politik. Das ist der Grund, warum keine Liste
einprogrammiert wird: sie wäre nach jedem Beta falsch. Die Matrix hält den
jeweiligen Stand als Daten, die Doku zeigt ihn, die Tests prüfen ihn.

## Leitlinien

Wie in den vorangehenden Godot-Specs: erkennen statt konfigurieren, die
Dateien der Werkzeuge sind die Quelle, erweiterbar über Daten, laut statt
still. Dazu zwei neue, aus dieser Sitzung:

- **Keine zweite Wahrheit über die Version.** Welche Version ein Projekt will,
  sagt das Projekt. Eine Versionsangabe in `.loomux/config.toml` oder
  `registry.toml` wäre ein zweiter Ort, der auseinanderläuft, sobald jemand
  das Projekt im Editor hochzieht.
- **Nur vorhandene Frameworks.** loomux fährt, was es gibt, und baut nichts
  nach. Gibt es für eine Linie keinen Testläufer oder kein Coverage-Addon,
  dann gibt es dort diese Lane nicht.

## Das Vorgehen gilt allgemein

Vom Auftraggeber freigegeben: was hier für Godot entschieden wird, ist die
Regel für jede Sprache und jedes Framework, das danach kommt. Godot ist der
erste Fall, nicht der Sonderfall. Wer künftig eine Sprache oder ein Framework
aufnimmt, geht denselben Weg:

1. **Version aus dem Projekt lesen**, nie aus der Konfiguration, und keine
   Liste unterstützter Versionen einprogrammieren.
2. **Getragen wird, was der Hersteller noch versorgt** — dessen Politik
   entscheidet, nicht loomux.
3. **Eine Zeile in der Matrix** als Daten: Framework, Versionsspanne, welche
   Linien sie trägt.
4. **Eine Doku-Seite je Stack** (en und de), die aus diesen Daten lebt.
5. **Ein Fixture je Matrixzeile** im Tor, nach dem Vorbild von
   `linecov/fixtures_test.go`.
6. **Die Lane wird konfiguriert, nicht getrieben.** Go-Code bekommt nur, was
   formatgetrieben allen dient — ein Berichtsleser, kein Framework-Treiber.
7. **Echte Fremdprojekte über den Korpus**, außerhalb des Tors.
8. **Über Installationen führt loomux keinen Zustand.** Wo ein
   Versionsverwalter die Fassung auf den `PATH` legt, ist das die Antwort; wo
   es keinen gibt, nennt eine Umgebungsvariable den Ordner und loomux fragt
   die Kandidaten. Keine Datei je Framework — bei zwanzig Frameworks wären das
   zwanzig Dateien mit zwanzig Lesern, zwanzig Schreibpfaden und zwanzig
   Gelegenheiten zu veralten.
8. **Was nicht laufen kann, sagt das:** `unavailable` mit Begründung, wenn es
   das Framework auf dieser Linie nicht gibt, `unready` samt beiden Versionen,
   wenn es da ist und nicht passt. Nie still grün.

## Drei Dateien, drei Eigentümer

Damit die Frage „wohin gehört das?" nicht bei jedem neuen Werkzeug neu
verhandelt wird: es gibt drei Orte und keinen vierten. Eine eigene
Projektdatei neben `.loomux/config.toml` kommt nicht dazu — sie wäre der Ort,
an dem die Version ein zweites Mal stünde.

| Datei | Eigentümer | Inhalt | Haltbarkeit |
|---|---|---|---|
| `project.godot` (bzw. die Datei des jeweiligen Werkzeugs) | das Framework | welche Version das Projekt will | mit dem Projekt |
| `.loomux/config.toml` | das Projekt, eingecheckt | was das Projekt von loomux will: Lanes, Abweichungen vom Preset, notfalls ein fester Binärpfad | mit dem Repo |
| Umgebung (`GODOT_BIN`, `GODOT_BIN_DIR`) | die Maschine und die Sitzung | wo die Binaries hier liegen | nichts wird aufgezeichnet |

`registry.toml` bleibt daneben, was es ist: die Liste der Bereiche. Es trägt
keine Werkzeuge. Eine vierte Datei über Installationen gibt es nicht — siehe
die Entscheidung weiter unten.

## Umfang: welche Linien

Verlangt sind alle offiziell gepflegten Linien plus Beta. Am 2026-10-16 sind
das:

| Linie | Stand | Pflege |
|---|---|---|
| 4.8 | beta1 (15.10.2026) | Beta |
| 4.7 | 4.7.2 stabil (18.08.2026) | aktiv |
| 4.6 | 4.6.3 | aktiv |
| 4.5 | — | nur Sicherheit/Plattform |
| 3.6 | 3.6.3 LTS (21.08.2026) | Best-Effort, solange Beitragende |

Die Tabelle ist ein Stand, keine Liste zum Einbauen: Godot bringt laut
eigener Politik alle drei bis sechs Monate eine Nebenversion. Der Entwurf
verlangt deshalb, dass loomux die Linie **aus dem Projekt** liest und
unbekannte Linien nach vorn offen behandelt — eine fest eingebaute Liste
unterstützter Versionen wäre nach dem nächsten Beta falsch.

Ausdrücklich außerhalb: Godot 3.5 und älter, 4.4 und älter (beide am Ende
ihrer Pflege), Godot 2.

## Nicht-Ziele und Anti-Ziele

- **Kein eigener Testläufer, keine eigene Instrumentierung.** Auch nicht für
  Godot 3, wo Coverage dünn ist. (Entscheidung des Auftraggebers.)
- **Keine Gabelung der Grammatik**, solange eine vorhandene reicht. loomux
  pflegt keine GDScript-Grammatik.
- **Keine Versionsangabe in der Konfiguration.** Siehe Leitlinien. Was in
  `registry.toml` darf, ist die Maschinenseite: wo welche Binaries liegen.
- **Anti-Ziel: eine stille Teilunterstützung.** Eine Lane, die auf einer Linie
  nicht laufen kann, darf nicht als grün durchgehen und nicht als Fehler
  erscheinen. Beides wäre schlimmer als der heutige Zustand, weil es den
  Nutzer in die falsche Richtung schickt.
- **Anti-Ziel: ein Godot-4-Projekt verliert, was es heute hat.** Keine
  Regression an `space` (4.7): Lanes, Graphzahlen und Laufzeit bleiben
  mindestens gleich.

## Randbedingungen aus dem Bestand

- `{godot}` sucht heute in der Reihenfolge `GODOT_BIN`,
  `[verify.gdscript] godot`, dann `godot`/`godot4` auf dem `PATH`, fragt jedes
  Binary einmal `--version` (höchstens 30 s, gegen das Budget) und verlangt
  Übereinstimmung in Haupt- und Nebenversion sowie einen Mono-Build bei
  `[dotnet]`. Diese Mechanik ist tragfähig und soll bleiben; sie bekommt nur
  eine Versionsquelle, die auch für Godot 3 antwortet, und eine Auswahl, die
  mehr als ein Binary kennt.
- `[verify.gdscript] godot` nennt genau einen Pfad. Für mehrere
  Installationen fehlt eine Liste; sie gehört in `registry.toml`, weil
  Installationspfade maschinenabhängig sind und `config.toml` eingecheckt ist.
- `when_found` (`plugin.cfg:gdUnit4`, `plugin.cfg:NanoCoverage`) ist der
  vorhandene Weg, eine Lane an ein gefundenes Framework zu binden. Für GUT und
  ein Godot-3-Coverage-Addon braucht es entsprechende Muster, nicht einen
  neuen Mechanismus.
- Der Extractor zählt Fehlerknoten pro Datei (`ParseErrors`) und sammelt die
  betroffenen Dateien (`ErrorFiles`), die CLI zeigt davon fünf Namen
  (`internal/cli/graph.go:238`). Die Datenwege stehen also; es fehlt die
  Sichtbarkeit und eine Entscheidung über die Grammatik.

## Die Werkzeuge pro Linie, belegt

| | Godot 3.6 | Godot 4.5–4.8 |
|---|---|---|
| Lint/Format | `gdtoolkit==3.*` | `gdtoolkit==4.*` |
| Test | GUT 7.x | gdUnit4 (Tabelle ab 4.0; `master` bis 4.8.dev7) |
| Coverage | `jamie-pate/godot-code-coverage` über `gut_cmdln.gd`, dokumentiert für **3.5**, 3.6 unbestätigt | Nano Coverage → `check linecov` |

Die Coverage-Zelle links ist die einzige Unsicherheit des Vorhabens. Sie ist
zu prüfen, bevor eine Lane dafür gebaut wird; scheitert die Prüfung, bleibt
Godot 3 nach der Leitlinie „nur vorhandene Frameworks" ohne Coverage-Lane, und
das ist dann das richtige Ergebnis, nicht ein Mangel.

## Was entschieden ist

Vom Auftraggeber:

- Alle offiziell gepflegten Linien plus Beta, für fremde Projekte wie für
  `space`; getragen wird, was der Hersteller der Sprache noch versorgt.
- Voller Satz, wo es Frameworks gibt; keine Eigenbauten.
- Die Version kommt nicht in die Konfiguration.
- Lint, Typen, Tests, Coverage und was folgt werden über loomux
  **konfiguriert**, wie tox es für Python tut. Kein Treiber je Framework: GUT
  wird konfiguriert, nicht gebaut.
- Je Sprache und Framework eine Doku-Seite mit der Verträglichkeitsmatrix, in
  den Tests gespiegelt.

Meine Festlegungen (als solche zu lesen, nicht als Zusage des Auftraggebers):

- **Versionsquelle:** `config/features` bleibt der erste Griff. Fehlt es,
  entscheidet `config_version` über die Hauptlinie (`5` → 4.x, `4` → 3.x).
  `config_version=4` nennt keine Nebenversion; ein 3er-Projekt bekommt
  deshalb eine Linienangabe ohne Nebenversion, und die Prüfung des Binaries
  vergleicht dort nur die Hauptversion.
- **Mehrere Installationen:** eine Liste in `registry.toml`, pro Eintrag Pfad
  und die Version, die das Binary gemeldet hat; `{godot}` wählt daraus das
  passende, bevor es auf `PATH` zurückfällt. `GODOT_BIN` und
  `[verify.gdscript] godot` behalten ihren Vorrang.
- **Grammatik:** der Graph bleibt versionsblind und tolerant. Statt einer
  zweiten Grammatik wird der heutige Blob beibehalten und `graph build` macht
  die betroffenen Dateien sichtbar. Ein Wechsel auf eine neuere Grammatik ist
  erst dann richtig, wenn gotreesitter sie liefert und gemessen ist, was sie
  auf einem 3.6-Projekt verliert.
- **Nicht laufbare Lanes:** `unavailable` mit Begründung, nicht `unready` und
  nicht grün. `unready` bleibt dem Fall vorbehalten, dass das Framework da
  ist, aber Version oder Zustand nicht passen.
- **Linien außerhalb des Umfangs** (3.5 und älter, 4.4 und älter) werden nicht
  abgewiesen. Es gibt keine Mindestversionsprüfung und keine eingebaute
  Liste; „außerhalb" heißt, dass loomux diese Linien nicht prüft und nichts
  über sie zusagt, nicht dass es sie blockiert. Wessen Framework dort läuft,
  bekommt seine Lane.
- **Die Matrix entscheidet über Framework-Versionen, nicht über Godot-Linien.**
  Trägt die gefundene Framework-Version die gelesene Linie laut Matrix nicht,
  ist die Lane `unready` und nennt beide Versionen. Fehlt die Framework-Version
  in der Matrix (eine neuere, als loomux kennt), läuft die Lane und sagt, dass
  sie ungeprüftes Gebiet betritt — nach vorn offen, wie der Umfang es
  verlangt.
- **Keine Datei über Installationen.** Ein Ordner genügt: eine
  Umgebungsvariable (`GODOT_BIN_DIR`) nennt das Verzeichnis, in dem die
  Engines liegen, loomux nimmt daraus die Kandidaten und wählt den ersten,
  dessen Fassung zur Linie passt. Eine Aufzeichnung hätte nur die
  `--version`-Abfrage gespart — Millisekunden — und dafür Schreibpfad, Sperre,
  atomaren Tausch, einen Pflegebefehl und Veraltung nach sich gezogen. Für
  Sprachen mit Versionsverwalter (mise, asdf, nvm, pyenv, SDKMAN) ist ohnehin
  der `PATH` die Antwort, und loomux wird kein zweiter Versionsverwalter
  neben dem, den der Nutzer schon hat; Godot ist der Sonderfall einer
  heruntergeladenen Datei ohne Verwalter.
  Entschieden wird über die gelebte
  `--version`-Abfrage, damit keine zweite Wahrheit über die Maschine entsteht,
  wenn jemand Godot an derselben Stelle hochzieht. Gefragt wird das zuerst
  passende Binary, und jedes verschiedene Binary nur einmal je Lauf: mehrere
  Projekte verschiedener Linien in einem Lauf sind der Normalfall, also kostet
  ein gemischter Lauf so viele Abfragen, wie er Linien berührt.
- **Ein genanntes Binary wird an die Linie gehalten, nicht übergangen.**
  Hier stand zuerst, `GODOT_BIN` werde übergangen, wenn es die Linie nicht
  trägt, „und die Meldung sagt, warum". Bei der Umsetzung (2026-10-17) zeigte
  sich, dass es diese Meldung nicht geben kann: eine Notiz erreicht den Bericht
  allein über `job.Note`, und `run.go` zeigt die ausschließlich für eine Lane,
  die **nicht** läuft. Ein Übergehen wäre also stillschweigend gewesen, gegen
  die Leitlinie „laut statt still". Deshalb gilt: `GODOT_BIN` und
  `[verify.gdscript] godot` behalten ihre Autorität — ein Binary der falschen
  Linie macht die Lane `unready`, und die Meldung nennt `GODOT_BIN_DIR` als
  Ausweg. Die Reihum-Suche gilt für die Quellen, die niemand für dieses Projekt
  genannt hat: den Ordner und den `PATH`.
  Eine Notiz für eine laufende Lane bräuchte ein Feld in `Job`/`Outcome` und am
  Berichtsformat. Das ist eine eigene Änderung, und erst mit ihr wäre ein
  Übergehen vertretbar.
- **`doctor` zeigt die Linie nur, wo eine Lane sie ohnehin auflöst** (`test`,
  `coverage`). Kein zusätzlicher `--version`-Aufruf in jedem Lauf, erst recht
  nicht in Projekten ohne Godot.
- **Die Linie kommt nicht in die `detect`-Signale.** Kein Stackname `godot3`:
  die Version ist erst zur Planzeit bekannt, also gehört sie in die
  Platzhalter-Auflösung, nicht in die statische Erkennung.
- **Hauptversionsvergleich nur im 3er-Zweig.** Ein 4er-Projekt ohne
  `config/features` bleibt der Fehler, der es heute ist; `config_version=5`
  spannt 4.0 bis 4.8 und taugt nicht als Vergleichsmaß.
- **Die Sichtbarkeit im Graphen ist mehr als heute.** `graph build` nennt
  bereits bis zu fünf betroffene Dateien; verlangt sind Zahl je Datei und die
  vollständige Liste auf Abruf. Die eine Fundstelle in `space`
  (`game_root_colony_test.gd:497`) ist Sache von `space`, nicht dieses
  Vorhabens.

## Prüfbarkeit: was bewiesen werden kann und was nicht

Zur Verfügung steht genau ein echtes Projekt: `space`, Godot 4.7, gdUnit4,
Nano Coverage. Keine 3.6-Installation, kein GUT-Projekt, kein 4.8-beta1.
Daraus folgt eine harte Trennung, die im Plan sichtbar bleiben muss:

- **Ohne Engine prüfbar und darum im Tor:** Versionserkennung aus
  `project.godot` (beide Formen), die Matrix samt Doku-Spiegelung, die Auswahl
  des Binaries aus mehreren Kandidaten, der Testbericht-Leser gegen
  Berichtsdateien, `unavailable`/`unready` samt Begründungen, der Graph auf
  Dateien jeder Linie. Dafür Fixtures je Matrixzeile, nach dem Vorbild von
  `linecov/fixtures_test.go`.
- **Nur gegen `space` prüfbar:** dass die 4.7-Kette von Ende zu Ende grün
  bleibt — keine Regression an Lanes, Graphzahlen und Laufzeit.
- **Nicht prüfbar, also offenes Tor:** ob `jamie-pate/godot-code-coverage` auf
  3.6 läuft, und ob GUT und gdUnit4 sich an die Matrix halten, die loomux über
  sie behauptet. Der Plan baut diese Zeilen als **Daten und Doku**, nicht als
  Go-Code, und markiert sie als unerprobt. Das ist die Deckung, die der
  tox-Zuschnitt gibt: eine falsche Datenzeile ist eine Zeile, kein Fehler im
  Programm.
- **Echte Fremdprojekte** kommen über die bestehende Korpus-Mechanik
  (`internal/dev/benchcorpus`, `Cloner`, Matrix in
  `docs/*/open-source-matrix.md`), um die es eine GDScript-Scheibe getrennt
  nach Linie erweitert — rund zehn Projekte je Linie, Auswahl nach Sternen wie
  bei den Benchmarks, mit gdUnit4- und GUT-Projekten darunter. Sie beantworten
  Erkennung, Graph und Lint an echtem Code, **nie** eine Lane, die eine Engine
  braucht, und laufen über `loomux dev`, nicht im Tor.

Wer das offene Tor schließen will, braucht eine 3.6- und eine
4.8-beta-Installation auf der Maschine oder in CI. Solange die fehlen, gilt
für diese Zeilen die Leitlinie „laut statt still": loomux behauptet dort
nichts, was es nicht geprüft hat.

## Was dem Bauenden bleibt

- Die Form des Testbericht-Lesers: welche Formate zuerst (JUnit ist gesetzt,
  da gdUnit4 und GUT es schreiben), über welchen Lane-Schlüssel eine Lane ihn
  benennt, und wie viel von `godotsuite/junit.go` dorthin wandert, ohne die
  bestehende gdUnit4-Lane zu verändern.
- Wie die Linie intern dargestellt wird (Haupt-/Nebenversion, Beta-Zusatz) und
  wo diese Darstellung sitzt.
- Welche `when_found`-Muster GUT und ein Godot-3-Coverage-Addon erkennen.
- Ob `gdtoolkit` je Linie über einen Pin im Preset oder über zwei
  Preset-Varianten gebunden wird.
- Die Auswahl der Korpus-Projekte je Linie und die Form der GDScript-Scheibe
  in der Matrix.
- Die Form, in der `graph build` und `doctor` Linie, Binary und betroffene
  Dateien zeigen.
- Prüfprojekte: je Linie ein Fixture mit `project.godot`, einem Testframework
  und einer Datei, die die Syntax dieser Linie ausreizt (Godot 3:
  `remote var`, `setget`, `onready`, `export(Type)`; Godot 4: Annotationen,
  Lambdas, typisierte Arrays, `await`).
