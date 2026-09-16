# Backlog aus Teilprojekt 1 für Teilprojekt 2

Diese Datei hält fest, was während der Ausführung von Teilprojekt 1 bewusst
liegen gelassen wurde. Sie ist kein Fehlerbericht: jeder Punkt wurde gesehen,
beurteilt und mit Begründung verschoben. Die Arbeitsspuren, aus denen sie
stammt, sind nach dem Abschluss gelöscht worden — diese Datei ist das, was
davon neben dem Code weiterlebt.

## Entwurfsentscheidungen, die offen sind

**Zeitgrenze für Prüfkommandos.** `checks.py` ruft `subprocess.run` ohne
`timeout` auf. Ein hängender Linter blockiert `run_all` unbegrenzt und gibt
keine Ausgabe — die Prüfkette sieht dann aus, als täte sie nichts. Der Plan
hat die Zeitgrenze nicht vorgesehen; sie ist eine Entwurfsentscheidung
(welche Grenze, und wird eine Überschreitung ein rotes `CheckResult` oder ein
Abbruch?), keine Reparatur. **Dies ist der oberste Punkt der Liste.**

**Der Journal-Cache ist unbedingt.** Ein Knoten mit einem `ok`-Eintrag unter
seinem `(Name, input_hash)` liefert dessen Delta zurück, statt zu laufen — auch
außerhalb des Replay-Modus. Folge: ein begrenzter Zyklus ist wirkungslos, wenn
seine Nutzlast sich nicht bei jedem Durchlauf ändert, und ein Bibliotheksnutzer,
der `run()` zweimal auf demselben Journal aufruft, bekommt beim zweiten Mal
nichts ausgeführt. Beides ist jetzt dokumentiert (`Runner`-Klassendocstring,
`CodeNode.max_visits`, README) und die Meldung der Besuchsgrenze sagt, wenn ein
Knoten aus dem Journal bedient wurde. Das Abschlussreview hätte den Cache enger
geschlüsselt — auf „wir betreten einen Lauf erneut" statt „irgendein Eintrag
existiert". Dagegen steht, dass der Schlüssel das ist, was der
Golden-Journal-Test festnagelt; eine Änderung berührt Wiederaufnahme, Wiedergabe
und Besuchsgrenze gleichzeitig. Wenn Teilprojekt 2 Wiederholschleifen braucht,
ist das die erste Frage.

**Schema-Semantik des Modell-Adapters.** `_schema_of` verlangt inzwischen eine
eingefrorene Dataclass aus Skalarfeldern und verweigert alles andere mit einem
`ModelError`, statt es still zu `"string"` zu machen; `required` wird für Felder
ohne Default gesetzt. Was fehlt, ist eine Entscheidung darüber, wie reichhaltig
`AgentNode.schema` sein darf — verschachtelte Dataclasses, Listen, Optionals.

**`mcp__<server>` ohne Werkzeugsegment.** `resolve_tools` erzeugt Bezeichner der
Form `mcp__<server>`, SDK-Bezeichner sind `mcp__<server>__<tool>`. Der Adapter
reicht die serverweite Form unverändert an `allowed_tools` (eine
Berechtigungsregel) und filtert sie aus `tools` (der Obergrenze eingebauter
Werkzeuge, die diesen Namen nicht kennt). Abgeleitet aus dem Parser des SDK,
nicht gegen einen laufenden MCP-Server gemessen.

## Was ein echter Lauf noch beweisen muss

Der Contract-Test (`uv run pytest -m contract`) ist nie gelaufen — er braucht
Credentials und Netz. Introspektion belegt die *Feldnamen und Typen* des SDK;
was sie nicht belegt: dass `usage["output_tokens"]` tatsächlich gefüllt wird und
dass `structured_output` unter `--json-schema` ein `dict` trägt. Solange das
offen ist, ist die Token-Abrechnung unbestätigt. `usage.get("output_tokens", 0)`
liefert bei einer Umbenennung still Kosten von 0 — der Drift-Wächter kann
Dictionary-Schlüssel strukturell nicht abdecken.

## Was das Abschlussreview nachgetragen hat

Vier Punkte, die während der Ausführung von Teilprojekt 2 gesehen und verschoben
wurden und bis zum Abschlussreview nur im Ausführungsledger standen — der ist
git-ignoriert und stirbt mit dem Arbeitsbereich.

**ERLEDIGT** (zum Punkt darunter) — umgesetzt in
[2026-08-22-pruefkette-reihenfolge-design.md](2026-08-22-pruefkette-reihenfolge-design.md)
§7. `checks._run` ruft jetzt `process.run`: `Popen` in einer eigenen Gruppe
(POSIX) beziehungsweise einem Job-Objekt (Windows, über `ctypes`), zwei
Lesefäden je Prozess statt `communicate()`, Baumtötung bei Fristablauf und ein
zweites, kurzes Fenster für das Einsammeln. Kommt ein Lesefaden nicht zurück,
wird er aufgegeben und das Ergebnis sagt, dass die Ausgabe unvollständig ist.
Der Befund von damals steht unverändert darunter, weil er der Grund für den
Umbau ist.

**Die Zeitgrenze tötet nur das Kind, nicht die Enkel.** `checks._run` ruft
`subprocess.run(..., timeout=…)` — die Grenze aus §8 der Spec, Vorgabe 600
Sekunden. `subprocess.run` beendet bei Zeitüberschreitung **den direkten
Kindprozess** und ruft danach `communicate()`, um dessen Ausgabe einzusammeln.
Hält ein überlebender Enkel dieselben Pipe-Enden noch offen, kommt dieser Aufruf
nicht zurück, und der Lauf hängt genau dort unbegrenzt, wo die Grenze ihn
gerade davor bewahren sollte. Das ist kein Randfall: `uv run pytest` ist eine
Kette aus mindestens zwei Prozessen, ein Godot-Starter ebenso, und jede Prüfung,
die über ein Präfix aus `[exec].prefix` läuft, hat dieselbe Form. Verschoben,
weil die Reparatur die Prozessführung austauscht — `Popen` mit einer
Prozessgruppe beziehungsweise einem Job-Objekt unter Windows, `kill` auf die
ganze Gruppe, und ein zweites, kürzeres Zeitfenster für das Einsammeln der
Ausgabe. Kosten des Liegenlassens: die Zeitgrenze ist auf genau den Prüfketten
wirkungslos, für die sie geschrieben wurde. **Dies ist der wichtigste der vier.**

**Committet der Reparateur, sieht die Wache nichts.** `guard` misst über
`git status`, also über den *Arbeitsbaum*. Ein Agent, der seine Änderung
committet, hinterlässt einen sauberen Baum: `changed_files` meldet nichts, kein
Pfad wird geprüft, und eine geänderte Testdatei geht durch. Heute entschärft,
nicht gelöst — das Werkzeugprofil `edit` enthält kein Bash, der Agent hat also
keinen Weg zu `git commit`. Verschoben, weil die härtere Grundlage eine andere
Messung wäre: der Vergleich gegen einen beim Laufstart festgehaltenen
Ausgangs-Commit statt gegen den Arbeitsbaum. Das berührt die Grundlinie, den
Fall „Projekt ohne Repository" und die Frage, was bei einem Lauf auf einem
Detached Head gilt. Kosten: die Sperre hängt an einer Profilentscheidung, die
anderswo aus einem ganz anderen Grund gelockert werden kann.

**Das Pausenfenster gehört niemandem.** Die Grundlinie wird beim `run`
aufgenommen und im `.flow`-Marker mitgeführt. Ändert ein Mensch zwischen `run`
und `resume` eine geschützte Datei — legitim, an seinem eigenen Baum —, steht
sie in keiner Grundlinie: nicht in der aufgezeichneten, denn die ist älter, und
eine neue wird bewusst nicht genommen. Die Wache lastet sie dem Reparateur an,
Exit 4 gegen einen Unschuldigen. Verschoben, weil die naheliegende Abhilfe die
Lücke aufmacht, gegen die die Aufzeichnung überhaupt existiert: nähme `resume`
eine frische Grundlinie, wäre alles, was der Reparateur vor der Pause schon
geändert hat, entschuldigt. Eine ehrliche Lösung müsste die beiden Urheber
auseinanderhalten können, und dazu fehlt jeder Anhaltspunkt. Kosten: gering,
solange `verify_until_green` kein Gate hat — der Ablauf pausiert heute nie. Der
Punkt wird scharf, sobald Teilprojekt 4 die Gate-Variante der Testsperre baut.

**Reihenfolge zwischen Prüfungen** stand schon oben, aus dem space-Lauf, und ist
dort als oberster Punkt der Liste vermerkt.

## Kleinere offene Punkte

- `runner._why_it_looped` erklärt die Besuchsgrenze erst, wenn sie erreicht ist.
  Ein Gate-Zyklus mit gleichbleibender Nutzlast und großzügigem `max_visits`
  dreht viele Durchläufe durch den Cache, bevor irgendetwas gesagt wird.
- Eine Antwort, deren Pause-Hash der Lauf nie erreicht, wird still verworfen.
  Praktisch nur erreichbar durch ein von Hand bearbeitetes Journal oder einen
  zwischen Pause und Wiederaufnahme geänderten Graphen.
- `list_flows` filtert auf Identifier und *verschweigt* damit eine Datei wie
  `my-flow.py`, statt zu erklären, warum sie unbrauchbar ist. Gehört zur
  Diagnose-Arbeit an der CLI.
- `tests/test_config.py` trägt noch einen Grenztest, der Quelltext durchsucht —
  dieselbe Redundanz gegenüber `test_module_boundary.py`, die anderswo bereits
  entfernt wurde.
- Der Nebenläufigkeitstest in `tests/test_checks.py` misst Wanduhrzeit. Er ist
  sorgfältig relativ formuliert, bleibt aber der einzige Flake-Kandidat der
  Suite.
- `src/ultraloom/flows/__init__.py` ist ein leeres Paket, das niemand
  importiert — der Platzhalter für die mitgelieferten Abläufe aus Teilprojekt 2.
- `PendingGate` hat ein drittes Feld bekommen (`input_hash`). Positionale
  Konstruktion durch Fremdcode bricht; vor 0.1.0 unkritisch, aber es ist eine
  Signaturänderung.
- Der Exit-Code für „wartet an einem Freigabepunkt" ist von 2 auf 3 gewandert,
  damit er nicht mit argparses eigenem Nutzungsfehler kollidiert. Dokumentiert in
  der README.

## Eine Lehre, die teurer war als die anderen

Der schwerste Fehler des Teilprojekts — `resume` mit einer Antwort verwarf jedes
Delta vor dem Freigabepunkt und zerbrach die Wiedergabe — hat ein Aufgaben-Review,
eine Fix-Runde und zwei darauf aufbauende Aufgaben überlebt. Der Grund war nicht
Nachlässigkeit: **jede Gate-Fixture im Branch setzte das Gate auf `graph.start`**,
und genau dort hebt sich der Fehler auf. Die Fixture stammte aus dem Plan selbst.

Wenn ein Plan eine Test-Fixture vorgibt, erben alle darauf aufbauenden Aufgaben
deren blinde Flecken. Für Teilprojekt 2 heißt das: die Fixtures des Plans sind
Vorschläge, und mindestens eine Variante sollte die Form verlassen, die der Plan
vorgemacht hat.

# Was Teilprojekt 2 hinterlässt

Derselbe Zweck wie oben, eine Runde später: gesehen, beurteilt, mit Begründung
verschoben. Gefunden hat das meiste davon Task 14 — der erste Lauf des Ablaufs
`verify_until_green` in einem fremden Projekt (space: Godot 4, GDScript,
headless gdUnit4, Nano Coverage nach LCOV, kein Typechecker). Die Läufe selbst
stehen auf der Ablaufseite `docs/abläufe/verify-until-green.md`; hier steht, was
davon Arbeit bleibt.

## Der Agentenpfad hängt an einer ungepinnten Fremdversion

**ERLEDIGT** (zum Punkt darunter) — das Extra nennt jetzt genau
`claude-agent-sdk==0.2.143`, und `run` prüft vor dem ersten Knoten, ob eine
startbare CLI erreichbar ist. Nachgemessen am 24.08.2026: 0.2.144 ist das
einzige Release seit 0.2.135 **ohne `win_amd64`-Rad** und zugleich `latest` —
auf Windows installiert `uv sync` damit das plattformlose Rad, das keine
`claude.exe` mitbringt. Eine Untergrenze hätte das nicht gefangen, ein Deckel
veraltet bei jedem Release; eine feste Version sagt, wogegen gemessen wurde.
Die Diagnose (`agent_sdk.find_cli`) geht dieselben drei Wege wie das SDK —
konfigurierter Pfad, gebündelte CLI des Rads, `PATH` — und verweigert nur, wenn
alle drei leer ausgehen; strenger zu sein hieße, Läufe zu verbieten, die das SDK
hinbekommen hätte. Sie greift nur, wenn der geladene Graph einen Agent-Knoten
trägt und der Lauf nicht `--no-model` heißt, und sie läuft, bevor Journal und
Marker entstehen. Ein `test_agent_sdk`-Wächter hält fest, dass das installierte
Rad seine CLI dort bündelt, wo ultraloom sucht.

Überholt ist dabei ein Halbsatz des Befunds unten: die Meldung des SDK nennt in
0.2.143 selbst drei Abhilfen, `cli_path` darunter — und das bietet ultraloom
inzwischen an. Was blieb, war der Zeitpunkt.

**`claude-agent-sdk` ist nicht gepinnt, und die Wahl entscheidet über Lauf oder
Nichtlauf.** Das Extra nennt weder Untergrenze noch Deckel. In der Umgebung von
Task 14 löste das auf `0.2.144` auf, und dieses Rad brachte keine
`_bundled/claude.exe` mit. Das SDK fiel auf das `claude.CMD` aus PATH zurück und
**verweigerte den Start**: „Refusing to execute batch script … no reliable
escaping for cmd.exe exists." Das ist kein Randfall und keine Verschlechterung
der Qualität, sondern ein harter Fehlschlag des gesamten Agentenpfads — jeder
Lauf mit Modell endet mit `outcome=error`, bevor ein einziges Token fließt. Mit
`0.2.143`, das die `.exe` mitbringt, läuft es.

Verschoben, weil die richtige Antwort mehr ist als eine Zahl in
`pyproject.toml`: eine Untergrenze schließt genau den Fall nicht aus, der hier
auftrat — die *neuere* Version war die kaputte —, ein Deckel veraltet, und ein
Rad ohne gebündelte CLI ist auf manchen Plattformen die einzige Wahl. Der
Adapter setzt heute kein `cli_path`; dort müsste die Diagnose vermutlich
ansetzen: beim Start prüfen, ob eine ausführbare CLI erreichbar ist, und
andernfalls sagen, welche der drei Abhilfen gemeint ist. Kosten des
Liegenlassens: ein `uv sync` auf einer frischen Maschine kann den Agentenpfad
ohne Zutun stilllegen, und die Meldung zeigt auf cmd.exe statt auf die
Installation.

**ERLEDIGT** (zum Punkt darunter) — umgesetzt in
[2026-08-24-agent-settings-sources-design.md](2026-08-24-agent-settings-sources-design.md).
Die Richtungsfrage ist entschieden: `[agent].settings` benennt die Quellen,
der Standard `["project"]` erbt genau die versionierte Projektdatei, und der
Adapter übergibt das Feld immer statt es dem SDK zu überlassen.

**`setting_sources` bleibt ungesetzt.** Der Adapter (`model/agent_sdk.py`)
übergibt das Feld nicht, also gilt der Standard des SDK. Gemessen in space heißt
das: die `SessionStart`- und `Stop`-Hooks des Projekts liefen im Reparaturlauf
mit, `PostToolUse` nicht. Der `SessionStart`-Hook schrieb dabei eine
`override.cfg` in den Arbeitsbaum — ein Seiteneffekt, den weder der Reparateur
verursacht hat noch die Wache erklären kann, und der nur deshalb harmlos blieb,
weil die Grundlinie ihn vor `guard` abdeckte. Verschoben, weil die Entscheidung
eine Richtungsfrage ist und keine Reparatur: Soll ein ultraloom-Lauf die
Werkzeugumgebung des Projekts *erben* (dann ist er realistisch, aber teurer und
mit fremden Seiteneffekten) oder nicht (dann ist er reproduzierbar, aber
verhält sich anders als eine Sitzung von Hand)? Beides ist vertretbar; still
danebenstehen ist es nicht.

Dazu gehört, dass das SDK dem Reparateur die **global konfigurierten
MCP-Server des Benutzers** anbietet. In Lauf 0005 versuchte er,
`mcp__context-mode__ctx_execute` für einen eigenen gdlint-Lauf zu benutzen, und
wurde von `permission_mode: "dontAsk"` abgewiesen. Die Sperre hält also, und
die verlorene Werkzeugrunde war real — aber nicht aus dem Grund, der hier
zuerst stand. Nachgemessen am 24.08.2026: zwei Läufe, die sich einzig in den
geerbten MCP-Servern unterschieden, hatten einen byte-identischen Prompt. Die
Serverdefinitionen stehen gar nicht im Prompt, weil ultralooms `tools`-Deckel
die eingebauten Werkzeuge erschöpfend aufzählt und nichts anderes durchlässt.
Die verlorene Runde entstand anders: das Modell rief ein Werkzeug, das ihm nie
angeboten worden war, und die Rechteschicht wies es ab — dagegen hilft keine
Ladeliste. `[agent].mcp_servers` sagt heute, was *zusätzlich* erlaubt ist, und
nichts darüber, was ohnehin schon da ist.

## Was die Prüfkette über Werkzeuge annimmt

**ERLEDIGT, soweit er zu erledigen war** (zum Punkt darunter) — die Mechanik
bleibt bewusst, wie sie war (Ausgabe deuten hieße raten); was fehlte, war die
Warnung, und die steht jetzt im README unter *Before you configure a check*.

**Ein Exit-Code ist das ganze Urteil — und manche Prüfwerkzeuge kennen ihn
nicht.** `checks._run` liest `returncode`, sonst nichts. Die Prüfungen von space
sind Claude-Code-Hooks: `coverage_gate.py` meldet seine Befunde über
`hookSpecificOutput` nach stdout und beendet sich **immer** mit 0, weil Exit 2
auf `Stop` dem Agenten das Ende des Zuges verweigern würde. Direkt als
Prüfkommando eingetragen las ein *fehlender* LCOV-Bericht als bestandene
Coverage-Prüfung. Das ist genau der eine Fehlschlag, den `verify-until-green`
nie erzeugen darf, und ultraloom kann ihn strukturell nicht bemerken. In space
steht deshalb eine dünne Hülle davor, die dieselben `findings()` ruft und nur
den Kanal wechselt. Verschoben, weil die Alternative — Ausgabe deuten statt
Exit-Code lesen — eine schlechtere Regel wäre: sie rät. Was fehlt, ist nicht
Mechanik, sondern eine gut sichtbare Warnung in der Dokumentation: **jedes
Prüfkommando, das aus einem Hook-Skript kommt, ist daraufhin anzusehen, ob es
seinen Befund im Exit-Code trägt.** Kosten: ein Projekt, das das übersieht,
bekommt eine grüne Prüfung, die nie etwas geprüft hat.

**ERLEDIGT** (zum Punkt darunter) — umgesetzt in
[2026-08-22-pruefkette-reihenfolge-design.md](2026-08-22-pruefkette-reihenfolge-design.md)
§5 und §6. `lint`, `types` und `test` nehmen drei Gestalten: Zeichenkette,
Liste, Tabelle mit `commands` und `threaded`. Die damaligen Folgefragen sind
beantwortet: **alle** Kommandos laufen, auch nach dem ersten roten (der
Reparateur braucht die vollständige Mängelliste), der Bericht bekommt je
Kommando eine `$`-Überschrift und bleibt bei genau einem Kommando Zeichen für
Zeichen der alte, und `threaded` ist damit ein reiner Geschwindigkeitsschalter.

**Eine Prüfart, ein Kommando.** `config._KINDS` erlaubt je Art genau eine
Zeichenkette. space lintet mit zweien: `gdlint` und `gdformat --check` laufen
über dieselben Verzeichnisse und sind beide „lint". Die Konfiguration von space
fährt heute nur `gdlint`; `gdformat` fehlt damit gegenüber dem, was das Projekt
selbst prüft. Der Ausweg existiert — ein Skript unter `.ultraloom/checks/`, das
beide ruft —, verlagert aber Konfiguration in ausführbaren Code, den niemand
liest. Verschoben, weil die naheliegende Erweiterung, eine Liste statt einer
Zeichenkette, sofort Folgefragen aufmacht: Läuft nach dem ersten roten Kommando
noch das zweite? Wie sieht der Bericht aus, den der Reparateur bekommt? Die
Presets tragen die Antwort in Ansätzen (`measure` plus `argv`), aber für
mehrere gleichrangige Kommandos ist sie nicht gebaut.

**ERLEDIGT** (zum Punkt darunter) — umgesetzt in
[2026-08-22-pruefkette-reihenfolge-design.md](2026-08-22-pruefkette-reihenfolge-design.md)
§3, §4 und §8. Prüfungen laufen in Stufen, nebenläufig nur innerhalb einer
Stufe; die Kanten kommen aus dem Preset und werden von `[verify.after]`
überschrieben. Die zweite Frage — was mit einer Prüfung geschieht, deren
Vorgänger rot ist — ist mit der neuen Quelle `blocked` beantwortet: rot, nicht
übersprungen, aber nicht außer Reichweite. Und die Doppelmessung ist weg:
`test` misst mit, wenn `coverage` im selben Lauf angefordert ist.

**Zwischen Prüfungen gibt es keine Reihenfolge.** Spec 9.4 nimmt an, Prüfungen
seien unabhängig, und lässt sie deshalb nebenläufig laufen. In space ist der
LCOV-Bericht ein Nebenprodukt des Suitenlaufs: das Coverage-Tor liest den
Bericht, den die Suite erst acht Minuten später schreibt, und meldet „no
coverage report". Das Python-Preset löst dasselbe Problem mit einem
`measure`-Schritt, der die Suite ein zweites Mal fährt — für eine Godot-Suite
keine Option, weil das weitere acht Minuten kostet. Das Tor von space löst es
über Reihenfolge und reicht einen Zeitstempel weiter (`measured_after`), damit
ein alter Bericht nicht als neuer durchgeht. Verschoben als Entwurfsfrage: eine
Abhängigkeit „coverage nach test" ist leicht gesagt und berührt die
Nebenläufigkeit, den Bericht und die Frage, ob eine Prüfung ausfällt, wenn die
Prüfung, auf die sie wartet, rot ist. **Dies ist der oberste Punkt dieser
Liste** — er ist der Grund, warum in space kein grüner `precommit`-Lauf steht.

## Kleinere offene Punkte

- **Stille Präzedenz bei Coverage.** `resolve_check` prüft
  `config.coverage_report` **vor** `config.commands`. Wer sowohl
  `[verify.coverage].report` als auch ein `coverage`-Kommando setzt, bekommt
  ohne Warnung das erste. Die Reihenfolge ist jetzt auf der Ablaufseite
  dokumentiert; eine Warnung beim Laden wäre ehrlicher, ist aber eine
  Entscheidung darüber, wie viel `load_config` beurteilen darf.
- **Die Schwelle wird nicht durchgesetzt.** `[verify.coverage].threshold` wird
  gelesen und weitergereicht, aber kein Kommando bekommt sie — durchgesetzt
  wird, was das Coverage-Werkzeug selbst eingestellt hat. `ultraloom check
  coverage` sagt das in einer eigenen Zeile; die Ablaufseite sagt es jetzt
  auch. Was fehlt, ist die Entscheidung, ob der Schlüssel überhaupt bleiben
  soll.
- **Ein Projekt mit Build-Cache braucht einen Einrichtungsschritt, von dem der
  Ablauf nichts weiß.** *Erledigt für Godot:* fehlt
  `.godot/global_script_class_cache.cfg`, sind `test` und `coverage` rot mit der
  Quelle `"unready"`, bevor eine Engine startet. Offen bleibt die allgemeine
  Form — jede andere Sprache mit Build-Cache bringt ihre eigene Markerdatei mit,
  und die Vorbedingung steht heute als Sonderfall im Code statt als Feld neben
  dem Preset. Das Ventil ist `[verify].godot_import = false`, für ein Projekt,
  das den Import selbst fährt oder gar nicht über eine Engine testet; ein
  weiterer Sonderfall, den die allgemeine Form mit auflösen müsste.
- **`_marker` ist first-hit über `PRESETS`.** *Gesehen:* ein Godot-Projekt, das
  zusätzlich eine `pyproject.toml` trägt — für seine Werkzeugkette, für Skripte
  —, wird als Python-Projekt erkannt; die Godot-Presets greifen nicht, und die
  neue Import-Vorbedingung greift ebenfalls nie. *Warum verschoben:*
  vorbestehendes Verhalten der Erkennung, aber die Vorbedingung erbt es still,
  und das ist neu. Eine Reparatur heißt zu entscheiden, was bei mehreren Markern
  gilt — Reihenfolge, Nachfrage, ein expliziter Schlüssel —, und das ist eine
  Entwurfsfrage über die ganze Prüfkette, nicht über diese eine Vorbedingung.
  *Was es kostet:* diese Entscheidung, plus einen Weg, wie ein Projekt seine Art
  selbst benennt.
- **Die Sitzungs-Hook-Falle kann ultraloom nicht prüfen.** *Gesehen:* ein
  Editor- oder Import-Lauf schreibt `project.godot` um, ein Coverage-Addon
  trägt dabei einen zweiten Sitzungs-Hook ein; beide instrumentieren, beide
  leeren den Datenspeicher, und ganze Dateien kommen mit null Treffern aus dem
  Zusammenführen, obwohl ihre Suiten grün liefen. Das Coverage-Tor liest das als
  nicht erreichte Zeilen. *Warum verschoben:* die Prüfung verlangt Wissen, das
  die Prüfkette bewusst nicht hat — welche Addons es gibt, wie ihre Hooks
  heißen, welcher davon einer zu viel ist. Eine Liste von Addon-Namen in
  ultraloom wäre am Tag nach dem nächsten Addon falsch. *Was es kostet:* ein
  Format, in dem ein Projekt seine eigenen Vorbedingungen als Daten beschreibt
  (Datei, Muster, Meldung), plus die Entscheidung, wie viel davon ultraloom
  auswerten darf, ohne zum halben Linter für fremde Konfigurationsdateien zu
  werden. Bis dahin fängt es das Prüf-Tor des Projekts selbst ab, und die
  Ablaufseite benennt die Falle.
- **`uv pip install -e` bricht an einem absoluten Windows-Pfad mit `#`.** Die
  Meldung nennt `C:\Users\micro\Documents` und behauptet, dort liege kein
  Python-Projekt: der Pfad wird an `#` abgeschnitten. Ein relativer Pfad
  funktioniert. Fremdes Werkzeug, aber es kostet beim nächsten Mal wieder eine
  Viertelstunde, und die Installationsanleitung von ultraloom nennt absolute
  Pfade.
- **Die `unavailable`-Übersetzung steht an zwei Stellen**
  (`checks._run_or_report` und `flows/verify_until_green._result_for`). Bewusst
  asymmetrisch: der Ablauf fängt nur `CheckUnavailableError`, `run_all`
  zusätzlich alles andere, damit ein echter Fehler im Ablauf sichtbar bleibt,
  statt still als rote Prüfung zu enden. Als Ermessensfrage festgehalten, nicht
  als Schuld.
- **Der Beweis für die Testsperre fehlt weiterhin.** Fünf Läufe in zwei
  Projekten haben versucht, einen echten Agenten dazu zu bringen, eine
  Testdatei anzufassen; keiner hat es geschafft. Die Wache ist durch Unit-Tests
  abgedeckt und gegen ein echtes Modell unbewiesen.

# Was der Umbau der Prüfkette hinterlässt

Gesehen, beurteilt, mit Begründung verschoben — dritte Runde, aus dem Umfang
[2026-08-22-pruefkette-reihenfolge-design.md](2026-08-22-pruefkette-reihenfolge-design.md)
(Stufen, Prozessgruppe, mehrere Kommandos je Prüfart).

**Ein Journal von vor dem Umbau passt nicht mehr auf seinen `check`-Knoten.**
`VerifyState` hat zwei Felder dazubekommen — `blocked` und `brief` —, und der
Zustand geht in den `input_hash` ein, mit dem das Journal seine Einträge
schlüsselt. Ein Lauf, der vor dem Upgrade begonnen hat, findet nach dem Upgrade
für seinen `check`-Knoten keinen passenden Eintrag mehr: `replay` leitet nicht
mehr dasselbe Ende ab, und eine Wiederaufnahme führt den Knoten neu aus und
wiederholt damit auch die Reparaturrunde dahinter — bezahlte Token für Arbeit,
die schon getan war. Verschoben, weil die Abhilfe eine Versionierung des
Zustands wäre (Migration alter Einträge oder ein Schema-Stempel im Journal), und
das ist eine Entscheidung über das Journalformat, nicht über die Prüfkette. Vor
0.1.0 sind die Kosten gering: die Läufe, die es gibt, sind unsere eigenen. Was
fehlt, ist die Meldung — heute sieht ein solcher Fall aus wie ein Knoten, der
eben nochmal lief, und nicht wie ein unbrauchbar gewordenes Journal.

**Der POSIX-Zweig von `process.py` ist nie ausgeführt worden.** `spawn_kwargs`
und `terminator` sind über eine gewöhnliche Auswahlfunktion testbar, und die
Tests wählen beide Plattformen — geprüft ist damit die *Auswahl*. Der Rumpf von
`_terminate_posix` selbst, also `os.killpg(os.getpgid(pid), SIGKILL)`, trägt ein
`# pragma: no cover  # POSIX-only` und ist auf dieser Maschine (Windows 11) noch
nie gelaufen. Verschoben, weil die einzige ehrliche Prüfung eine POSIX-Maschine
verlangt: ein Kommando, das einen langlebigen Enkel hinterlässt, die Zeitgrenze
darüber, und danach die Frage, ob der Enkel wirklich tot ist. Kosten des
Liegenlassens: die Hälfte der Plattformweiche ist Behauptung. **Dies ist der
oberste Punkt dieser Liste.**

**Das Restfenster bei der PID-Wiederverwendung unter Windows.** Der Baum wird
über ein Job-Objekt getötet, dem der Prozess suspendiert zugewiesen wird — das
schließt das Fenster, in dem ein schnelles Kind schon einen Enkel erzeugt haben
könnte. Der Kehraus über die Eltern-PID-Tabelle daneben hat aber ein eigenes:
Windows behält eine Eltern-PID im Eintrag, nachdem der Elternprozess gestorben
ist, und vergibt PIDs neu. `_could_be_a_descendant` fängt den Normalfall ab —
wer *vor* der Wurzel gestartet ist, kann nicht ihr Nachfahre sein. Ein
Fremdprozess, der *nach* der Wurzel geboren wird und eine recycelte PID aus
unserem Baum als Eltern-PID trägt, bleibt ununterscheidbar und würde
mitgetötet. Verschoben, weil die saubere Antwort Handles statt PIDs über die
ganze Kette wären und die Wahrscheinlichkeit in der Praxis winzig ist: das
Fenster ist der Bruchteil einer Sekunde zwischen dem Tod eines Kindes und dem
Kehraus. Festgehalten, weil ein fremder getöteter Prozess ein Schaden ist, den
niemand mit ultraloom in Verbindung bringen würde.

**Godot hat kein `coverage`-Preset, und das kostet jedes Projekt eine Zeile.**
Der Eintrag für `project.godot` ist bewusst entfallen: GDScript-Coverage ist in
dem Projekt, aus dem der Fall stammt, das Editor-Addon Nano Coverage — es
instrumentiert die Quellen an Ort und Stelle und schreibt `lcov.info` als
Nebenprodukt des Suitenlaufs, und die Schwelle darüber erzwingt ein
projekteigenes Skript. Weder das eine noch das andere ist ein Kommando, das ein
zweites Godot-Projekt aufrufen könnte, und ein erfundenes hätte nach einer
Prüfung ausgesehen, ohne eine zu sein. Die Folge: die Aussage „ein
Godot-Coverage-Bericht folgt immer auf die Suite" ist allgemein wahr, aber
nirgends allgemein hinterlegt — jedes Godot-Projekt schreibt sein
`coverage = "test"` selbst in `[verify.after]`, neben sein Kommando unter
`[verify.coverage].report`. Verschoben, weil die Abhilfe ein Preset wäre, das
eine Reihenfolge ohne ein Kommando trägt, und damit eine neue Sorte
Preset-Eintrag. Kosten: eine Wiederholung je Projekt und ein Godot-Projekt, das
sie vergisst, liest einen Bericht aus dem vorigen Lauf.

**Eine Fehlermeldung nennt ein Kapitel, das in keiner Datei steht.** `_table`
hat den Parameter `label` genau dafür — die verschachtelte Tabelle wird unter
ihrem vollen Kapitelnamen abgelehnt, nicht unter ihrem Blatt. Der Aufruf für
`[verify.coverage]` gibt ihn nicht mit: `coverage = "x"` unter `[verify]`
scheitert mit `[coverage] must be a table`, und `[coverage]` kommt in der Datei
nirgends vor. Einzeilige Reparatur, aber sie ändert eine Meldung, an der Tests
hängen können — deshalb hier und nicht nebenbei in einer Doku-Aufgabe.
Verwandt und ausdrücklich *kein* Fehler: ein Tippfehler **innerhalb** von
`[verify.coverage]` (`reprot` statt `report`) wird still geschluckt, weil einen
unbekannten Schlüssel abzulehnen eine eigene Entscheidung über Vorwärts-
kompatibilität wäre. Der README benennt beides.

**Was der Umbau behauptet hat, ist gemessen — siehe Ablaufseite.** Der
`precommit`-Lauf in space ist wiederholt worden; die Zahlen stehen in
`docs/abläufe/verify-until-green.md` neben den bestehenden Tabellen und nicht
hier, damit sie eine Quelle haben statt zweier. Kurz: `threaded = true` über die
zwei gleichrangigen GDScript-Linter bringt Faktor **1,60** (Median 5,94 s gegen
9,48 s seriell, je drei Messungen), und die eingesparte zweite Suite ist an den
`report_N/`-Verzeichnissen von gdUnit4 abgelesen. Wer die 1,60 anderswo als
1,87 wiederfindet, liest einen überholten Zwischenstand: die erste serielle
Zahl war eine Einzelmessung mit einem Ausreißer von 11,04 s, dessen Ursache
offen bleibt.

**`checks.py` trägt fünf Aufgaben auf gut 800 Zeilen** (815 bei diesem Stand).
Presettabelle, Auflösung und Messentscheidung, Godot-Bereitschaftstor,
Ausführung und Berichtsverschmelzung, Scheduler. Die Nähte sind benannt, aber
der Schnitt ist teurer als er zunächst aussah. `schedule.py` nähme `run_kinds`,
`_gated`, `_stages`, `_blocker`, `_run_or_report`, `run_all`, `BLOCKED` und
`CheckRunner` mit — und dazu, damit das überhaupt übersetzt, `_marker`, `KINDS`,
`CheckResult`, `UNAVAILABLE` und `CheckUnavailableError`. Der Knoten sitzt bei
`run_check`: es gehört fachlich zur Ausführung und bliebe in `checks.py`, wird
aber von `_gated` gerufen, also importierte `schedule.py` aus `checks.py`. Ein
Re-Export von `run_all` in `checks.py`, damit die heutigen Aufrufer nichts
merken, schlösse den Zyklus. Wer den Schnitt macht, muss also zuerst über diese
Richtung entscheiden. `readiness.py` nähme `_unready`, `_import_message`,
`_preset_godot_binary` und die beiden Godot-Konstanten, also die einzige
Godot-Sonderkenntnis außerhalb von `PRESETS`. Kosten: `test_module_boundary.py`
muss die neuen Namen kennen. Eigener Task, kein Merge-Blocker.

**Der eine verbliebene Weg zu Grün über alte Daten.** Ein Projekt mit eigenem
`[verify].test` **und** `[verify.coverage].report` **und**
`[verify.after].coverage = "test"`: `_measures_for` ist dann falsch — das
Testkommando ist fremd, und über fremde Kommandos wird nichts angenommen —,
aber `after in alongside` ist wahr, also gibt es weder einen `measure`-Schritt
noch eine Warnung. Läuft `test` grün, ohne zu messen, berichtet `coverage` über
den vorigen Lauf und kann grün melden; `blocked` schützt hier nicht, weil der
Vorgänger ja grün ist. Das ist die beschlossene Restlücke des Rulings zu Task 8
und steht so im Docstring von `_measures_for`. Der Backlog nannte bisher nur die
umgekehrte Nachlässigkeit — die Warnung, die zu oft käme —, nicht diese.
*Nachtrag 2026-09-11:* Jeder Check-Prozess bekommt jetzt `ULTRALOOM_ALONGSIDE`
mit den Kinds seines Durchgangs. Ein eigener Report kann damit wenigstens
wissen, ob `test` lief; dass es gemessen hat, muss das Projekt weiter selbst
zusichern. ultraloom tut das für sich mit einem messenden `[verify.test]` und
`hooks/coverage-check.py`, das nur dann liest statt misst.

**`brief` ist fast aus `report` ableitbar und geht trotzdem in den
`input_hash`.** Fast, nicht ganz: `_render` klippt **je Befund**, während
`report` der ungeklippte Gesamttext ist — aus dem Gesamttext ließe sich nur noch
global klippen, und das gäbe eine andere Kürzung. Das Ruling bleibt richtig: der
Zustand ist, was der Knoten gesehen hat, und eine gekürzte Fassung, die anders
gekürzt wird, ist ein anderer Zustand. Die Kosten gehören aber danebengeschrieben: jede künftige Änderung an
`clip` invalidiert zusätzlich sämtliche Journale — nicht nur die Einträge, die
den Bericht wirklich gelesen haben. Wer `clip` anfasst, weiß dann, was er
auslöst.
