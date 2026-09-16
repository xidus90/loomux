# Prüfkette: Reihenfolge, Prozessgruppe, mehrere Kommandos

**Stand:** 2026-08-22 · **Status:** zur Freigabe · **Zielgruppe:** intern (Entscheidungsgrundlage)

Dieses Dokument ist der Vertrag für den Umfang, der zwischen Teilprojekt 2 und
Teilprojekt 3 steht. Taucht beim Bauen eine Streitfrage auf, gilt, was hier
steht — nicht die Erinnerung und nicht die Auslegung durch ein Modell.

> **Nachtrag (Task 7).** Die Abschnitte 4, 8 und 9 sind in der Ausführung von
> diesem Vertrag abgewichen: Abschnitt 4 verspricht einen Rückfall, den es in
> einem Fall nicht gibt, Godot hat kein `coverage`-Preset bekommen, und die
> Tabelle der knappen Modi nennt eine Flagge nicht, die im Code steht. Alle drei
> Stellen sind unten markiert.

---

## 1. Warum dieser Umfang, und warum jetzt

Teilprojekt 3 ist laut §16 des Kern-Designs der Spec→Code→Wiki-Kreis für space.
Er setzt auf der Prüfkette auf. Die Prüfkette hat aus Teilprojekt 2 drei Befunde
mitgebracht, die zusammen der Grund sind, warum in space **kein grüner
`precommit`-Lauf steht**:

1. **Zwischen Prüfungen gibt es keine Reihenfolge.** Spec 9.4 nimmt an,
   Prüfungen seien unabhängig, und lässt sie nebenläufig laufen. In space ist
   der LCOV-Bericht ein Nebenprodukt des Suitenlaufs: das Coverage-Tor liest
   den Bericht, den die Suite erst acht Minuten später schreibt, und meldet
   „no coverage report".
2. **Die Zeitgrenze tötet nur das Kind, nicht die Enkel.** `subprocess.run`
   beendet bei Fristablauf den direkten Kindprozess und ruft danach
   `communicate()`. Hält ein überlebender Enkel dieselben Pipe-Enden offen,
   kommt dieser Aufruf nicht zurück — der Lauf hängt genau dort unbegrenzt, wo
   die Grenze ihn davor bewahren sollte.
3. **Eine Prüfart, ein Kommando.** space lintet mit `gdlint` *und*
   `gdformat --check`; `config._KINDS` erlaubt je Art genau eine Zeichenkette.
   Die Konfiguration von space fährt heute nur `gdlint`.

Alle drei sitzen in `checks.py` und `config.py`. Sie werden zusammen gebaut,
weil sie sich gegenseitig bedingen: die Reihenfolge ändert, was `run_all` ist;
mehrere Kommandos ändern, worauf die Zeitgrenze sich bezieht.

**Nicht in diesem Umfang** und ausdrücklich weiterhin im Backlog: der
Agentenpfad (SDK-Pin, `setting_sources`, geerbte MCP-Server), die Grundlinie
der Testsperre, der Journal-Cache, `_marker` bei mehreren Markern.

---

## 2. Ein gemeinsamer Scheduler

Die Nebenläufigkeit steht heute zweimal da: `checks.run_all` hat einen
Thread-Pool, und `flows/verify_until_green.make_check` hat einen zweiten, mit
einem Kommentar, der erklärt, warum er nicht `run_all` ruft — `run_all` läuft
über alle Prüfarten, der Knoten über die angeforderten.

Würden Stufen nur in `run_all` gebaut, liefe `ultraloom check all` gestuft und
der Ablauf, für den die Reparatur gedacht ist, weiterhin ungestuft. Genau der
Lauf, der in space rot war, bliebe rot.

```python
def run_kinds(
    kinds: Sequence[str],
    config: Config,
    runner: CheckRunner = run_check,
) -> tuple[CheckResult, ...]
```

- `run_all(config)` wird zu `run_kinds(KINDS, config)`.
- `make_check` ruft `run_kinds(state.kinds, config, runner)` und verliert seinen
  eigenen Pool.
- Die Übersetzung von `CheckUnavailableError` in ein rotes Ergebnis, die heute
  an zwei Stellen steht (`checks._run_or_report` und
  `flows/verify_until_green._result_for`), zieht nach `run_kinds`. Die
  Asymmetrie war nur nötig, weil es zwei Pools gab.
- Der `runner`-Parameter bleibt, damit die Tests des Ablaufs kein echtes
  Werkzeug starten: ein Test, der ruff aufruft, misst ruff.
- Die Ergebnisreihenfolge ist die Reihenfolge von `kinds`, nie die
  Abschlussreihenfolge. Ein Bericht, dessen Zeilen zwischen Läufen wandern,
  ist nicht vergleichbar.

`run_kinds` mit leerer `kinds`-Folge ist ein Programmierfehler und wirft; der
Ablaufknoten fängt den Fall weiterhin vorher mit seinem eigenen `FlowExit` ab,
weil er dort eine Bedeutung hat („der Zustand nennt keine Prüfung").

---

## 3. Stufen

`run_kinds` führt nicht mehr eine nebenläufige Menge aus, sondern eine **Folge
von Stufen**: innerhalb einer Stufe nebenläufig, zwischen den Stufen
sequentiell.

Die Stufen entstehen aus Abhängigkeitskanten „Prüfart *nach* Prüfart". Die
Vorgabe kommt aus dem Preset, also aus der Sprache; ein Projekt überschreibt
sie mit `[verify.after]`.

| Sprache | Stufe 0 | Stufe 1 |
|---|---|---|
| Python | lint, types, test | coverage |
| Node | lint, types, test, coverage | — |
| GDScript | lint, types, test | coverage |

Node bleibt einstufig: `vitest run --coverage` misst und berichtet in einem
Lauf.

**Prüfarten, die nicht angefordert wurden, fallen aus den Stufen heraus, ohne
die übrigen zu verschieben.** `run_kinds(["coverage"])` läuft sofort, nicht
nach einer leeren Stufe 0.

**Ein Zyklus in den Kanten ist ein `ConfigError` beim Laden**, kein Hänger zur
Laufzeit. Ebenso ein `after`-Eintrag, der eine unbekannte Prüfart nennt.

---

## 4. Wer misst — die `alongside`-Regel

*Abgewichen: der `measure`-Rückfall am Ende dieses Abschnitts greift nicht
immer — siehe die Markierung dort.*

Stufen allein bringen Python nichts: `coverage report` nach `test` zu stellen
hilft nur, wenn `test` gemessen hat, und `uv run pytest` misst nichts. Damit die
Abhängigkeit etwas wert ist, muss `test` **in diesem Lauf** unter Coverage
laufen.

Das Preset bekommt dafür drei Felder mit getrennten Rollen:

```python
"pyproject.toml": {
    "test": Preset(
        argv=("uv", "run", "pytest", "-q", "--tb=short", "--no-header"),
        measuring=("uv", "run", "coverage", "run", "-m", "pytest",
                   "-q", "--tb=short", "--no-header"),
    ),
    "coverage": Preset(
        argv=("uv", "run", "coverage", "report", "--skip-covered", "--skip-empty"),
        after="test",
        measure=("uv", "run", "coverage", "run", "-m", "pytest",
                 "-q", "--tb=short", "--no-header"),
    ),
}
```

- **`measuring`** — „diese Prüfung kann als Nebenprodukt messen, wenn jemand
  die Messung braucht"
- **`after`** — „ich lese, was jene Prüfung hinterlässt"
- **`measure`** — „misst niemand für mich, messe ich selbst" (das heutige Feld,
  unverändert)

**Die Regel, in einem Satz:** Läuft die Prüfung, auf die ich warte, in diesem
Lauf mit und kann sie messen, dann misst sie — sonst messe ich selbst.

| Angefordert | `test` läuft als | `coverage` läuft als | Suitenläufe |
|---|---|---|---|
| test + coverage | `coverage run -m pytest` | `coverage report`, Stufe danach | 1 |
| nur test | `uv run pytest` | — | 1, ohne Messaufschlag |
| nur coverage | — | `measure` + `report` | 1 |
| `check all` | `coverage run -m pytest` | `coverage report`, Stufe danach | **1 statt heute 2** |

Umgesetzt über eine erweiterte Signatur:

```python
def resolve_check(
    kind: str,
    config: Config,
    *,
    alongside: frozenset[str] = frozenset(),
) -> Command
```

Die Vorgabe ist leer, also verhält sich jeder bestehende Aufruf wie heute.
`run_kinds` füllt `alongside` mit der Menge, die es tatsächlich fährt.

**Die Auflösungskette bleibt unangetastet:** Konfiguration schlägt Skript
schlägt Preset. `measuring` und `after` sind Preset-Eigenschaften. Ein Projekt,
das `test` selbst konfiguriert, hat kein `measuring` — dann greift für
`coverage` der `measure`-Rückfall, weil ultraloom nicht wissen kann, ob das
fremde Testkommando misst. Kein Raten.

*Abgewichen: der Rückfall greift nicht, wenn das Projekt auch `coverage` selbst
konfiguriert. Ein konfiguriertes Kommando hat kein `measure`, und weil `test` im
Lauf angefordert ist, bleibt auch die Warnung aus — `coverage` berichtet dann
still über die Daten des vorigen Laufs. Nachgetragen im Backlog unter „Der eine
verbliebene Weg zu Grün über alte Daten" und im Docstring von `_measures_for`.*

**Stille Präzedenz, benannt statt versteckt:** Wer `resolve_check` einzeln ruft,
bekommt eine korrekte, aber möglicherweise langsamere Auflösung als der
Scheduler. Gehört in den Docstring und auf die Ablaufseite.

---

## 5. Konfiguration

```toml
[verify]
lint = "gdlint ."                      # Zeichenkette: wie bisher
types = ["mypy", "pyright"]            # Liste: mehrere, sequentiell
max_parallel = 4                       # Vorgabe: os.process_cpu_count()

[verify.lint]                          # Tabelle: mehrere, mit Schaltern
commands = ["gdlint .", "gdformat --check ."]
threaded = true

[verify.after]                         # Stufen überschreiben
coverage = "test"
```

**Drei Gestalten für `lint`, `types`, `test`**, nach Typ unterschieden:
Zeichenkette und Liste sind Kurzformen, die Tabelle ist die volle Form mit
`commands` (Pflicht, nicht leer) und `threaded` (Vorgabe `false`). Zeichenkette
und Tabelle unter demselben Namen kann TOML nicht ausdrücken; der Parser lehnt
es ab, bevor `load_config` es sieht.

`commands = []` und jede leere Zeichenkette darin sind ein `ConfigError`, aus
demselben Grund wie heute: ein `[exec].prefix` allein, der mit 0 endet, wäre
eine grüne Prüfung, die nichts geprüft hat.

**`coverage` bekommt die Tabellenform nicht.** Es hat mit `[verify.coverage]`
schon eine eigene Tabelle mit `report` und `threshold`; `commands` und
`threaded` dort zusätzlich würden die stille Präzedenz zwischen `report` und
einem Kommando verdoppeln, die bereits als offener Punkt im Backlog steht.

**`[verify.after]`** bildet Prüfart auf **einen** Vorgänger ab. Kein
Vorgänger-Tupel, keine Ketten von Hand: die einzige gemessene Abhängigkeit ist
`coverage → test`, und für mehr gibt es keinen Bedarfsnachweis.

**`max_parallel`** ist eine positive Ganzzahl, Vorgabe `os.process_cpu_count()`.
Sie deckelt die gleichzeitig laufenden **Prüfprozesse** über alle drei Ebenen —
Stufe, Prüfart, Kommando —, nicht die Threads je Ebene. Umgesetzt als ein
`Semaphore`, den sich alle teilen und der genau um den `Popen`-Aufruf herum
gehalten wird. Ohne diesen Deckel ist `threaded = true` ein Fuß, auf den man
sich selbst schießt: bei space wären es sonst vier Godot-Prozesse gleichzeitig.

**Presets dürfen `after` setzen, `threaded` nicht.** Die Kommandos in einem
Preset sind `measure`-Ketten und müssen sequentiell bleiben; `threaded` ist
ausschließlich eine Projektentscheidung.

`_KINDS` in `config.py` bleibt eine bewusste Kopie von `checks.KINDS` und kein
Import — `config` sitzt unter `checks`, und `test_module_boundary` hält es dort.

---

## 6. Mehrere Kommandos je Prüfart

**Alle laufen, auch nach dem ersten roten.** Der Zweck der Prüfkette ist, dem
Reparateur eine vollständige Mängelliste zu geben; eine halbe Liste erzeugt eine
zusätzliche Runde durch das Modell, und die kostet mehr als das zweite
Lint-Kommando.

Das ist bewusst **nicht** die Semantik von `measure`. `measure` bricht ab, weil
ein Bericht über nicht gemessene Daten bedeutungslos wäre; zwei gleichrangige
Linter haben diese Beziehung nicht.

`threaded = true` ist deshalb ein reiner Geschwindigkeitsschalter: das Ergebnis
ist mit und ohne Nebenläufigkeit identisch. (Mit „Abbruch beim ersten roten"
wäre `threaded` widersprüchlich gewesen — das ist der Grund, warum die beiden
Entscheidungen zusammengehören.)

**Die Zeitgrenze gilt je Kommando**, nicht je Prüfart und nicht je Stufe. Sonst
hinge die Frist eines Linters daran, wie viele Geschwister er hat.

---

## 7. Die Zeitgrenze und was sie tötet

Ein neues Modul, `src/ultraloom/process.py`, mit genau einer öffentlichen
Funktion:

```python
def run(argv, *, cwd, timeout) -> Completed   # returncode, stdout, stderr, timed_out
```

`checks._run` ruft nur noch diese. Der Rest von `checks.py` bleibt frei von
Prozessführung — der Grund für ein eigenes Modul statt 70 weiterer Zeilen in
`checks.py`.

**Die Form, die den Hänger auflöst.** Kein `communicate()`, nirgends:

1. `Popen` mit Pipes, plattformabhängig in eine eigene Gruppe gesetzt.
2. **Zwei Lesefäden je Prozess**, die stdout und stderr fortlaufend leerlaufen
   lassen. Die Ausgabe ist damit eingesammelt, bevor irgendetwas stirbt — und
   die Pipe-Puffer laufen bei einem geschwätzigen Werkzeug nicht voll.
3. `wait(timeout)`. Bei Fristablauf: **den ganzen Baum töten**, dann die
   Lesefäden mit einer zweiten, kurzen Frist einholen (5 s).
4. Kommen die Fäden nicht zurück — ein Enkel hält die Pipe wirklich fest —,
   werden sie als Daemon-Fäden aufgegeben und berichtet wird, was im Puffer
   steht. **Der Lauf hängt nicht.** Ein aufgegebener Faden wird im Ergebnis
   benannt, damit die Ausgabe nicht stillschweigend abgeschnitten aussieht.

**Die Plattformweiche**, testbar statt `if sys.platform` mitten im Ablauf:

```python
_spawn_kwargs() -> dict          # POSIX:   start_new_session=True
                                 # Windows: CREATE_SUSPENDED + Job-Zuweisung
_terminate_tree(handle) -> None  # POSIX:   os.killpg(os.getpgid(pid), SIGKILL)
                                 # Windows: TerminateJobObject
```

Beide werden über eine gewöhnliche Auswahlfunktion gewählt, die in den Tests auf
beide Plattformen gestellt werden kann.

**Windows: suspendiert starten, dem Job zuweisen, fortsetzen.** Das ist nicht
Umständlichkeit, sondern das Fenster: startet der Prozess zuerst und wird danach
zugewiesen, kann ein sehr schnelles Kind vorher schon ein Enkelkind erzeugt
haben, das dem Job nie angehört. Umgesetzt über `ctypes`, ohne neue
Abhängigkeit — `taskkill /F /T` wurde verworfen, weil es über Eltern-PIDs geht
und verwaiste Enkel überleben lässt; `pywin32` wurde verworfen, weil der Kern
das an einer einzigen Stelle braucht.

**Ein abgelaufenes Kommando bleibt rot**, mit derselben Meldung wie heute plus
der Teilausgabe. Neu ist nur, dass die Teilausgabe zuverlässig eingesammelt
wird, statt beim Einsammeln zu hängen.

Die Lesefäden zählen nicht gegen `max_parallel` — der Deckel begrenzt Prozesse,
nicht Threads.

---

## 8. Ergebnisse, Quellen, Bericht

**`CheckResult` bleibt flach:** `kind`, `ok`, `output`, `source`. Kein neues
Feld, weil Journal, CLI und Ablauf daran hängen.

Mehrere Kommandos werden in `output` zusammengeführt, mit einer Überschrift je
Kommando:

```
$ gdlint .
...

$ gdformat --check .
...
```

Bei genau einem Kommando entfällt die Überschrift — der heutige Bericht bleibt
Zeichen für Zeichen derselbe. `ok` ist die Konjunktion.

**Eine neue Quelle: `blocked`.** Eine Prüfung, deren Vorgänger rot war, läuft
nicht und meldet: läuft nicht, weil `test` rot war. Rot, nicht grün, nicht
übersprungen — Grundsatz 4. Ihr Platz in der Ergebnisreihenfolge bleibt.

Blockiert wird über **jedes** rote Ergebnis des Vorgängers, auch über ein
`unavailable` oder `unready`. Ein Godot-Projekt ohne Import hat ein rotes
`test`, und ein Coverage-Bericht darüber wäre so wertlos wie über eine
gescheiterte Suite. Die Ansteckung ist transitiv: hängt C an B und B an A, und
A ist rot, sind B und C beide `blocked`.

**`blocked` gehört ausdrücklich nicht zu `_out_of_reach`.** `unavailable` und
`unready` heißen „keine Reparaturrunde schließt das". Eine blockierte
Coverage-Prüfung ist das Gegenteil — sie schließt sich, sobald `test` grün ist.
Wäre sie außer Reichweite, gäbe `verify_until_green` bei jedem gewöhnlichen
roten Test sofort auf.

**`_render`** listet die roten Prüfungen wie heute; blockierte kommen ans Ende
unter eine eigene Zeile:

```
Nicht gelaufen, weil ein Vorgänger rot war: coverage
```

Nicht in der Mängelliste, weil es kein Mangel ist, den der Reparateur anfassen
kann — aber sichtbar, damit ein Bericht mit grünem `lint`, grünem `types` und
rotem `test` nicht so aussieht, als wäre Coverage geprüft worden.

**Die Warnung ohne Messer.** Wird eine Prüfung angefordert, hat sie ein `after`,
läuft der Vorgänger in diesem Lauf nicht mit, und gibt es keinen
`measure`-Rückfall (der Godot-Fall), dann läuft sie trotzdem, und ihre Ausgabe
trägt eine vorangestellte Zeile:

```
Achtung: `test` lief in diesem Lauf nicht; dieser Bericht kann von einem
älteren Lauf stammen.
```

Sie bleibt eine Warnung und wird nie zur roten Prüfung. Ob der Bericht alt ist,
weiß ultraloom nicht — space beantwortet das mit einem eigenen Zeitstempel, und
ein Rateversuch hier wäre schlechter als der Satz.

*Abgewichen (Task 7): „der Godot-Fall" entsteht aus keinem Preset mehr.* Der
`coverage`-Eintrag für `project.godot` ist entfallen, weil es kein allgemein
aufrufbares GDScript-Coverage-Kommando gibt: in space schreibt die Nano-
Coverage-Editorerweiterung `lcov.info` als Nebenprodukt des Suite-Laufs, und
die Schwelle erzwingt ein projekteigenes Skript. Ein erfundenes Kommando im
Preset hätte nach einer Prüfung ausgesehen, die es nicht gibt.

Die Konstellation „`after` gesetzt, kein `measure`" ist damit nur noch von Hand
erreichbar — über `[verify.coverage].report` zusammen mit `[verify.after]`. Die
Warnung ist **nicht** tot und das Preset **nicht** versehentlich verloren; wer
Task 8 oder 9 baut, findet den Fall nur nicht mehr in der Tabelle.

Der Preis steht ausdrücklich hier: die Aussage „ein Godot-Coverage-Bericht folgt
immer auf die Suite" ist allgemein wahr und war im Preset einmal darstellbar.
Jedes Godot-Projekt muß sie jetzt einzeln in `[verify.after]` wiederholen. Das
ist vertretbar — eine allgemein wahre Reihenfolge ohne allgemein aufrufbares
Kommando kann ultraloom nicht ausführen —, aber es ist eine Wiederholung, die
ein Preset erspart hätte.

---

## 9. Die Menge der Ausgabe

Zwei deterministische Maßnahmen, keine dritte.

**Knappe Werkzeugmodi in den Presets.** Fast jedes Werkzeug bringt einen
knappen oder maschinenlesbaren Modus mit; den zu benutzen ist robuster als ein
eigener Parser, der beim nächsten Werkzeug-Update bricht.

| Werkzeug | Modus |
|---|---|
| coverage | `report --skip-covered --skip-empty -m` — nur die Dateien unter 100 %, mit den fehlenden Zeilen |
| ruff | `check . --output-format=concise` |
| mypy | `--no-error-summary --no-pretty` |
| pytest | `-q --tb=short --no-header` |
| vitest | unverändert |
| gdlint | ist schon knapp |

*Abgewichen (Task 7): `-m` stand nicht in dieser Tabelle und ist beim Bauen
hinzugekommen.* Es ist die einzige Stelle, an der die knappen Modi verbergen,
was der Reparateur braucht: `--skip-covered` läßt genau die Dateien stehen, die
unter 100 % liegen, und ohne `-m` nennt der Bericht die Datei, aber nicht die
fehlende Zeile. `show_missing` ist per Voreinstellung aus, und daß ultraloom es
in der eigenen `pyproject.toml` setzt, sagt nichts über ein fremdes Projekt.

Ebenfalls gemessen und hier festgehalten: `coverage report` bestimmt seinen
Rückgabewert allein über `fail_under`. Ohne diesen Schlüssel in der
Projektkonfiguration ist der Lauf bei 83 % **grün** — keine Wirkung der knappen
Flaggen, sondern die Eigenschaft, die §7 des Kern-Designs meint, wenn es sagt,
daß ultraloom die Schwelle nicht selbst erzwingt.

**Eine Obergrenze für die Ausgabe Richtung Modell**, mit Kopf und Fuß erhalten
und einer Zeile dazwischen, die sagt, wie viel fehlt. Der Fuß wiegt schwerer als
der Kopf: pytest schreibt die Zusammenfassung ans Ende.

**Gekürzt wird nur, was an den Reparateur geht.** `CheckResult.output` und damit
das Journal behalten die vollständige Ausgabe. Sonst geht verloren, was einen
Lauf im Nachhinein auswertbar macht — eine der vier Fähigkeiten, die dieses
Projekt rechtfertigen.

---

## 10. Verworfen, mit Begründung

Damit diese Fragen nicht in einem halben Jahr von vorn gestellt werden.

**Werkzeugspezifische Ausgabe-Parser.** Sie brauchen die Zuordnung „welches
Werkzeug ist das eigentlich", und die ist mit `[exec].prefix`, Wrapper-Skripten
und `uv run` davor nicht zuverlässig zu beantworten. Die knappen Modi aus §9
liefern denselben Nutzen ohne Regexe. Falls je nötig, wird es billiger, sobald
echte Läufe zeigen, welche Ausgabe wirklich weh tut.

**Ein Modell, das den Prüfbericht verdichtet** — weder lokal noch über den
Adapter, weder in `checks.py` noch als Ablaufknoten.

- In `checks.py` scheidet es aus: das Modul läuft auch unter `ultraloom check`
  ohne Journal und soll ohne Netz vollständig prüfbar bleiben (Grundsatz 6).
- Als Ablaufknoten wäre es mechanisch sauber — Knoten sind journalisiert, und
  der Reparateur ist selbst ein Modellknoten. Ausgeschlossen wird es trotzdem,
  weil die Prüfkette deterministisch bleiben soll: eine Verdichtung, die die
  eine Zeile verliert, auf die es ankam, lässt die Schleife an der falschen
  Stelle weiterdrehen, und der Bericht sieht dabei ordentlich aus. Das ist
  teurer als die Token, die sie spart.

**`taskkill /F /T`** statt Job-Objekt: geht über Eltern-PIDs, verwaiste Enkel
überleben — genau der Fall, für den die Reparatur geschrieben wird.

**`pywin32`** als Abhängigkeit: dieselbe Semantik wie das Job-Objekt, aber ein
plattformbedingtes Extra für etwas, das der Kern an einer Stelle braucht.

**Abbruch beim ersten roten Kommando** einer Prüfart: siehe §6.

**Ein Vorgänger-Tupel in `[verify.after]`**: kein Bedarfsnachweis, siehe §5.

---

## 11. Was den Entwurf beweist

**Unit-Tests gegen Attrappen.** Der Scheduler wird gegen einen eingesetzten
`runner` getestet, ohne ein echtes Werkzeug. Abgedeckt: Stufenbildung aus
Presets und `[verify.after]`, Zyklus beim Laden, `blocked` bei rotem Vorgänger,
`blocked` **nicht** in `_out_of_reach`, die vier Fälle der `alongside`-Regel,
`max_parallel` als geteilter Deckel, Berichtsreihenfolge unabhängig von der
Abschlussreihenfolge, die drei Konfigurationsgestalten samt ihrer Fehler.

**`process.py` gegen echte, aber triviale Prozesse.** Ein Skript, das ein
Enkelkind startet, welches die Pipe offen hält, und selbst sofort endet — genau
die Form, an der `subprocess.run` hängt. Der Test beweist, dass `run` innerhalb
der Frist plus der zweiten kurzen Frist zurückkommt und dass der Enkel tot ist.
Er ist der einzige Test dieses Umfangs, der Wanduhrzeit misst; großzügige
Grenzen, und eine Notiz, dass er der zweite Flake-Kandidat der Suite ist.

**Die Plattformweiche in beide Richtungen.** Auf jeder Maschine wird die
Auswahlfunktion für beide Plattformen geprüft, die Systemaufrufe selbst nur für
die eigene. `pragma: no cover` sitzt auf zwei bis drei Zeilen, jede mit Grund.

**Abnahmekriterium: ein grüner `precommit`-Lauf in space.** Nicht optional. Die
Unit-Tests können grün sein, während die Kette an genau dem Fall scheitert, für
den sie geschrieben wurde — das ist in diesem Projekt schon einmal passiert
(die Gate-Fixture, die den Fehler in `resume` fünf Runden lang verdeckt hat).
Der Lauf misst zugleich, was `threaded` und die eingesparte zweite Python-Suite
tatsächlich bringen.

---

## 12. Was dieser Umfang nicht anfasst

Weiterhin Backlog, unverändert: der Journal-Cache und sein Schlüssel; die
Grundlinie der Testsperre (`guard` misst den Arbeitsbaum) und das Pausenfenster;
der Agentenpfad (SDK-Pin beziehungsweise CLI-Diagnose, `setting_sources`, die
geerbten MCP-Server); `_marker` als First-Hit bei mehreren Markern; die stille
Präzedenz zwischen `[verify.coverage].report` und einem `coverage`-Kommando; die
nicht durchgesetzte `threshold`; die allgemeine Form der Vorbedingungen jenseits
des Godot-Imports.
