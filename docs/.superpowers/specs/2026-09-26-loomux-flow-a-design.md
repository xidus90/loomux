# Flow A — die Flow-Laufzeit zieht nach loomux

**Datum:** 2026-09-26
**Stand:** freigegeben am 2026-09-26; beim Planen nachgezogen: Katalog unter
`flows/catalog/`, Tests unter `_test/`, Lader ohne Import von `flows`,
Namenskollision nach einem Release folgt `[flow] overrides`, `resume` lehnt
einen Lauf ab, dessen `flow.toml` inzwischen aus einer anderen Quelle kommt
**Plan:** [2026-09-26-loomux-flow-a.md](../plans/2026-09-26-loomux-flow-a.md)
**Zweig:** `feat/flow-runtime`
**Quelle:** ulflow M1 auf ultraloom-`feature/agent-harness` (`d7041e5`, 61
Commits, nie gepusht)
**Bezug:** [Fusions-Spec](2026-09-14-loomux-fusion-design.md) (Flow als
Folgeprojekt, Lückenzeilen #14 und #22) ·
[ulflow-Laufzeit M1](../specs-ul/2026-09-11-ulflow-laufzeit-design.md) ·
[Harness-Graph](../specs-ul/2026-09-10-agent-harness-graph.md) ·
[Knotenkatalog](../specs-ul/2026-09-10-agent-harness-knotenkatalog.md)

## Ziel

Am Ende steht ein Flow, der von der Planung bis zum Pull Request führt:
Klärung, Spec und Plan mit Prüferfächern, je Aufgabe Recherche, Test, Bau,
Prüfkette, Codereview und Nacharbeit, dann Doku, Abschlussreview und Commit;
ein Mensch antwortet an festen Toren und pusht. Der Harness-Graph auf dem
ultraloom-Zweig zeichnet diesen Flow schon.

Dieser Flow ist **nur die Vorgabe**. Ein Projekt fährt eigene Flows, heute als
Dateien und später als Graph im Web-OS (W5). Flows aus der Community kommen
per Pull Request als Beispiele ins Repository, werden mit dem Binary
ausgeliefert und über `.loomux/config.toml` gewählt.

Teilprojekt A legt dafür den Grund: Es holt die Laufzeit aus M1 nach loomux
und gibt ihr das Ordnerformat, den Katalog, den Beitragsweg, die Rollen und die
Auswahl per Config. Echte Modellaufrufe gibt es mit A noch nicht.

## Zerlegung

| Teil | Inhalt | Hängt ab von |
|---|---|---|
| **A** (diese Spec) | Umzug von M1, Ordnerformat, Katalog und Beitragsweg, Rollen, Overlay, Auswahl per Config, Läufe, Session-Start, Wächterregel | — |
| A2 | Flows über MCP: `flow_list`, `flow_show`, `flow_run` (startet einen Lauf in einem von `serve` abgelösten Kindprozess und gibt sofort die Laufnummer zurück), `flow_status` (laufender Knoten, fertig, Fehler, wartendes Tor mit Frage); nur auf dem lokalen Kanal; kein Werkzeug zum Beantworten eines Tors. Dazu der Lebenslauf eines Laufs: Statusdatei, Erkennen und Fortsetzen eines unterbrochenen Laufs. Vorher gemessen: ob Claude Code und agy MCP-Elicitation können, damit ein Tor den Menschen direkt über den Wirt fragt. Pflichtfragen, beide Windows: `internal/child` setzt `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`, ein Kind in diesem Job stirbt mit `serve` (Ausbruch nur mit `BREAKAWAY`); und das Self-Update tauscht die exe, aus der ein Kindlauf gerade läuft | A; parallel zu B, gegen das Fake-Modell vollständig testbar |
| B | Modellzugang: Adapter für Claude und Gemini, Werkzeugprofile je Anbieter; entscheidet Lückenzeile #22 (CLI, API oder beides). Pflichtfrage: Ein Kindlauf von `claude -p` lädt die Hooks des Projekts; der Wächter soll dort greifen, das Stop-Tor und die Subagent-Hooks vermutlich nicht | A |
| C | Bausteine: Prüferfächer, `command`, Prüfkette, Rot-Prüfer, Commit, Plan → Board, Picker und Booker; dazu `verify-until-green` als mitgelieferter Daten-Flow, der erste, der Prüfkette und Nacharbeit verbindet | A, für echte Läufe B |
| D | Der Entwicklungszyklus als mitgelieferter Default-Flow, mit den Anweisungen je Rolle | B, C |

Jeder Teil hat eine eigene Spec, einen eigenen Plan und einen eigenen Pull
Request.

`smoke.py`, der einzige Python-Flow unter ultraloom-`.ultraloom/flows/`, zieht
nicht um: Er prüfte ultraloom selbst, und kein Wirt nutzt Flows
(Fusions-Spec, „Befunde“).

## Entscheidungen

| Frage | Entscheidung | Grund |
|---|---|---|
| Was heißt „über die Config auswählen“ | Der ganze Flow per `[flow] default`, dazu überschreibt ein Projekt einzelne Anweisungen und Modelle | Ein Vertrag, in dem Community-Flows dem Default gleichgestellt sind. Steckplätze für Unterflows bräuchten Unterflows in der Laufzeit |
| Modelle in einem mitgelieferten Flow | **Rollen:** der Knoten nennt eine Rolle, das Projekt bindet sie | Ladestufe 7 von M1 verlangte jeden Modellnamen unter `[agent.models]` des Projekts; ein mitgelieferter Flow wäre ohne diesen Eintrag unladbar |
| Anweisungen überschreiben | **Overlay** per Ordner, Datei für Datei | Eine Anweisung ändern, ohne den Flow zu kopieren; `definition_hash` enthält die Bytes, Replay warnt also weiter |
| Wer einen mitgelieferten Flow verdecken oder überlagern darf | Nur ein Mensch: freigegeben per `[flow] overrides` in der Config, der Wächter sperrt Agenten zusätzlich; ein Agent schreibt Flows unter eigenem Namen. Herkunft und Overlay stehen in jedem Hinweis und jeder Torfrage | Sonst ließe sich ein Pflichttor abschaffen, statt es zu beantworten; die Config hängt weder vom Katalog noch vom Binary ab |
| Teile einer `flow.toml` überschreiben | Nein | Die Ladeprüfung müsste einen zusammengesetzten Graphen erklären, jede Meldung nennte zwei Dateien |
| Reihenfolge | **Flow ist eine eigene Spur neben der Fusion**, wie der Code-Graph | A zieht nichts aus 4c, 4d oder 4e ein; 4e wartet ohnehin auf einen Remote für `brain-knowledge`. Die übrigen Prioritäten bleiben |
| Historie aus ultraloom | Nicht übernommen; Code und Tests ziehen um, die Commits werden nach Themen neu geschnitten | 61 Commits mit Wellen- und Task-Verweisen verstießen gegen die Commit-Regel in `AGENTS.md` |
| Python-Koexistenz | Entfällt | loomux setzt keine Läufe von ultraloom fort; ulflow und `ultraloom run` bleiben bis zur Umstellung in ihren Repos |
| Parität | Portierte Tests und Golden-Journal, keine Aufzeichnung unter `testdata/cases/` | Die Quelle ist Go und hat dieselben Tests schon; eine Aufzeichnung maße Go gegen Go |
| Arbeitspapiere von ulflow | Unverändert nach `specs-ul/` und `plans-ul/` | Sie liegen nur auf einem ungepushten Zweig; D baut auf Graph und Knotenkatalog auf |

## Ablage

| Heute in ultraloom | In loomux |
|---|---|
| `internal/flow`, `internal/flow/expr`, `internal/flow/tmpl` | gleich |
| `internal/flowload` | `internal/flow/load` (Format, Ladeprüfung, Auffindung, Overlay, Parameter) |
| `internal/blocks` | `internal/flow/blocks` (`agent`, `gate`, `exit`) |
| `internal/runner` | `internal/flow/runner` |
| `internal/journal` (Leser auf `master`, Schreiber und Hashes auf dem Zweig) | `internal/flow/journal` |
| `internal/runs` | `internal/flow/runs` |
| `internal/model` | `internal/flow/model` (Port, Fake, Rollen- und Modellauflösung, Werkzeugprofile) |
| `internal/flowcfg` | geht auf: die Schlüssel liest `internal/config`, siehe „Konfiguration“; Modellkette und Werkzeugprofile liegen in `internal/flow/model` |
| `internal/gitwork` (geänderte Dateien zu Laufbeginn) | in das vorhandene `internal/gitwork` eingefügt, nichts überschrieben |
| `cmd/flow` | `loomux flow …` in `internal/cli` |
| Session-Start in `cmd/guard` | `internal/hooks/hook_session_start.go` |
| — | `flows/` im Repowurzelverzeichnis: ein Go-Paket `flows`, das `flows/catalog/` mit `go:embed` einbettet |

Der Katalog wird beim ersten Gebrauch gelesen. Kein `init()` und keine
Paketvariable parst ihn (Regel in `AGENTS.md`). Der Test im Tor, der jedes
Paket-Init über 500 Allokationen scheitern lässt, gilt weiter.

`AGENTS.md` nennt unter „Where things live“, dass `flows/` der Katalog ist und
ein Beitrag reine Daten bleibt.

## Das Flow-Format

Die Sprache der M1-Spec gilt unverändert: Knoten, Kanten, Bedingungen, Deckel,
Platzhalter, Zustand, Parameter und die sieben Ladestufen. Was A ändert:

**Ein Flow ist ein Ordner.**

```
<name>/
  flow.toml            # schema_version = 1
  instructions/*.md    # Anweisungen der Agentenknoten, englisch
  questions/*.md       # Fragen der Tore, englisch
  README.md            # was der Flow tut und wofür; Pflicht nur im Katalog
  _test/               # nur im Katalog, siehe „Katalog und Beitrag“
```

Wie der Editor (W5) Lage und Aussehen ablegt, entscheidet W5; eine Datei mit
`_` vorne bliebe wie `_test/` aus dem Binary (siehe „Katalog und Beitrag“).

Pfade zu Anweisungen und Fragen stehen in `flow.toml` relativ zum Ordner:
eine Anweisung unter `instructions/`, eine Frage unter `questions/`. Ein Pfad
anderswohin, ein absoluter Pfad oder einer, der den Ordner mit `..` verlässt,
ist ein Ladefehler; nur so hat ein Overlay feste Stellen, an denen es ersetzen
kann.

**Zeilenenden.** Der Lader liest Anweisungen und Fragen mit LF: ein CRLF wird
beim Lesen zu LF, bevor gerendert und `definition_hash` gebildet wird. Sonst
hinge der Hash und damit jedes Golden-Journal davon ab, mit welchem Editor und
welchem `core.autocrlf` eine Datei entstand; `* text=auto eol=lf` in
`.gitattributes` normalisiert erst beim Commit, nicht eine lokal neu angelegte
Datei. Journale unter `flows/catalog/*/_test/` bekommen eine eigene Zeile `text eol=lf`
in `.gitattributes`, wie M1 sie für seine Testdaten hatte.

Der Name des Ordners ist der Name des Flows. **Flow-Namen folgen einer eigenen
Regel:** `[a-z][a-z0-9-]*`, also klein, mit Bindestrich, mit einem Buchstaben
vorne (`dev-cycle`, `strict-security-review`). Gründe: Die Community wählt die
Namen, und Bindestriche sind dort üblich; ein Flow-Name steht nie in einem
Platzhalter oder einer Bedingung, braucht also keine Bezeichnerform; Windows
unterscheidet `Review` und `review` im Dateisystem nicht, darum nur
Kleinbuchstaben; und `go:embed` lässt ohne `all:` Namen mit führendem `_` oder
`.` still weg, darum ein Buchstabe vorne. Knoten-, Feld-, Parameter- und
Rollennamen behalten die Bezeichnerregel von M1 (`[A-Za-z_][A-Za-z0-9_]*`).
`[flow] name` entfällt. Steht es trotzdem
da, ist es ein Ladefehler, damit zwei Namen nicht auseinanderlaufen.

**`schema_version` bleibt 1.** M1 hat kein Format ausgeliefert; kein Flow in
einem Projekt nutzt es (Fusions-Spec, „Befunde“).

**Unbekannte Schlüssel** auf oberster Ebene, in `[flow]`, in einem Knoten und
in einer Kante sind ein Ladefehler, mit den bekannten Schlüsseln in der
Meldung. Ob M1 das schon so hält, prüft der Plan an `decl.go` und schließt die
Lücke, falls nicht.

**Rollen.** Ein Knoten trägt `role = "<name>"` statt `model`; `[flow] role`
setzt eine Vorgaberolle für den Flow. Ein Rollenname folgt der Namensregel.
`model` am Knoten und in `[flow]` sind Ladefehler mit dem Hinweis auf `role`.

**Für den Editor festgehalten, gebaut mit W5:**
- Die Reihenfolge der Kanten trägt Bedeutung (die erste Kante, deren Bedingung
  hält). Ein Editor schreibt sie in der Reihenfolge zurück, die er zeigt.
- Nichts, was der Lader braucht, steht nur in einem Kommentar; ein Editor
  schreibt die TOML neu und verliert Kommentare.
- Lage und Aussehen stehen in `layout.toml`. Der Lader liest die Datei nicht,
  und sie geht in keinen `definition_hash` ein.

## Rollen und Modelle

**Auflösung**, die erste gesetzte Stufe gewinnt:

1. `role` am Knoten, sonst `[flow] role`.
2. Die Bindung dieser Rolle in `[agent.roles]` des Projekts: ein Modellname.
3. Ohne Rolle oder ohne Bindung: `[agent] default`, ebenfalls ein Modellname.
4. Nichts gesetzt: Anbieter `claude`, Modell „Voreinstellung der CLI“.

Ein Modellname zeigt auf `[agent.models.<name>]` mit `provider` und optional
`model`; fehlt `model`, gilt die Voreinstellung der CLI des Anbieters.

**Ladestufe 7 entfällt für Flows.** Ein Flow bringt keine Bindung mit und lädt
in jedem Projekt. Dass jede Bindung und `[agent] default` auf einen Eintrag
unter `[agent.models]` zeigen, prüft die Config-Prüfung (siehe
„Konfiguration“). Eine Rolle, die ein Projekt bindet, die aber kein Flow nennt,
ist kein Fehler: Ein Projekt bindet Rollen für mehrere Flows.

**Journal:** Das Feld `model` bleibt `<anbieter>:<modell>` oder
`<anbieter>:cli-default`. Neu ist das Feld `role`, damit das Journal sagt,
welche Rolle ein Modell gespielt hat; wie `model` ist es `null` bei Code- und
Torknoten und bei einem Agentenknoten ohne Rolle. Der `definition_hash` nimmt das
aufgelöste Modell auf wie in M1; eine geänderte Bindung meldet Replay also als
geänderte Definition.

`loomux flow show <flow>` nennt je Knoten Rolle, aufgelöstes Modell und woher
die Auflösung kam (Knoten, Flow, Bindung, Default, Voreinstellung).

## Auffindung, Overlay und Auswahl

**Zwei Quellen:** der Katalog im Binary und `.loomux/flows/<name>/` im Projekt.

- Liegt im Projektordner eine `flow.toml`, ist es ein **eigener Flow**. Er
  verdeckt einen mitgelieferten gleichen Namens ganz, wie in M1.
- Liegen dort nur `instructions/*.md` oder `questions/*.md`, ist es ein
  **Overlay** über den mitgelieferten Flow gleichen Namens: Jede Datei ersetzt
  die gleichnamige. Eine Datei, die der mitgelieferte Flow nicht hat, ist ein
  Ladefehler („overlays nothing“). Ein Overlay ohne mitgelieferten Flow
  gleichen Namens ist ebenfalls ein Ladefehler.
- Andere Dateien im Projektordner neben einem Overlay sind ein Ladefehler.
- Beides, Verdecken und Overlay, gilt nur für Namen, die `[flow] overrides`
  freigibt (siehe „Wächter“). Ohne Freigabe fährt der mitgelieferte Flow, und
  `list`, `show` und der Session-Start warnen.

**Anzeige.** `loomux flow list` nennt jeden Flow mit Herkunft (`project`,
`bundled`, `bundled+overlay`), markiert den Default und führt einen Flow, der
nicht lädt, mit dem Grund, statt ihn wegzulassen.

**Auswahl.** `[flow] default = "<name>"` in `.loomux/config.toml` bestimmt, was
`loomux flow run` ohne Namen startet. Ohne `default` lehnt `run` ohne Namen ab
und nennt die bekannten Flows und den Schlüssel. Ein `default`, der keinen Flow
nennt, ist ein Fehler bei `run` und `list`, nicht bei der Config-Prüfung: Die
Config-Prüfung liest keine Flows.

## Konfiguration

Neue Schlüssel in `.loomux/config.toml`, eingetragen in
`internal/config/schema` und gelesen von einem Leser in `internal/config`, den
`schema.Validate` über `readAll` mitfragt:

| Schlüssel | Art | Bedeutung |
|---|---|---|
| `[agent] default` | Text | Modellname für jede ungebundene Rolle |
| `[agent] mcp_servers` | Liste von Texten | Server für das Werkzeugprofil `mcp` (Lückenzeile #14, halb) |
| `[agent.models.<name>] provider` | Text, Pflicht | Anbieter |
| `[agent.models.<name>] model` | Text | Modell; fehlt es, gilt die Voreinstellung der CLI |
| `[agent.roles] <rolle>` | Text | Modellname, an den die Rolle gebunden ist |
| `[flow] default` | Text | der Flow, den `loomux flow run` ohne Namen startet |
| `[flow] overrides` | Liste von Texten | mitgelieferte Flows, die ein Projekt-Flow gleichen Namens verdecken oder überlagern darf (siehe „Wächter“) |

Unbekannte Schlüssel unter `[agent]`, `[agent.models.<name>]` und `[flow]` sind
Fehler, mit den bekannten in der Meldung. Im Schema tragen alle neuen Schlüssel
das Modul `Base`: Ein eigenes Modul `flow` liegt außerhalb von A, und `config`
und `init` filtern nach dem Modul. `[agent] settings` gehört zu B
(Lückenzeile #14, zweite Hälfte) und ist bis dahin unbekannt. `cli_path`
entfällt, wie es die Spec von 2a schon festhält.

Der Anbieter wird in A nicht gegen eine Liste geprüft; die Liste bringt B mit
den Adaptern.

`[agent]` ist nicht `[model]`: `[model]` ist das lokale Modell der Stufe 4c
mit seinen eigenen Rollen `describe`, `place` und `propose`. Die
Konfigurationsreferenz stellt beide nebeneinander, damit niemand sie
verwechselt.

Die Datei schreibt weiterhin kein Agent. Eine Bindung schlägt ein Agent mit
`loomux config set agent.roles.reviewer gemini --propose` vor.

**Dafür braucht das Schema benannte Schlüssel, und das ist Arbeit in A.** Heute
kennt `schema.Lookup` nur feste IDs, und die Art `Table` ist „shown, not
edited“. A führt Schlüssel mit einem Namenssegment ein: `agent.roles.<rolle>`
(Text) sowie `agent.models.<name>.provider` und `agent.models.<name>.model`
(Text), wobei `<…>` der Namensregel folgt. `Lookup` löst eine ID gegen diese
Muster auf; `config list` zeigt jeden vorhandenen Eintrag einzeln, `config
set`, `unset` und `--propose` schreiben ihn über `internal/config/edit`, das
eine fehlende Tabelle schon heute am Ende anlegt. Ob `edit.Set` mit einer
Tabelle `[agent.models.<name>]` zurechtkommt, die es noch nicht gibt, belegt
der Plan mit einem Test, bevor der Schlüssel eingetragen wird.

## Katalog und Beitrag

Der Katalog liegt unter `flows/catalog/<name>/`, das Paket `flows` bettet
das Verzeichnis `catalog` mit `//go:embed catalog` ein. Go läuft ein
eingebettetes Verzeichnis ab und lässt dabei jeden Namen mit `.` oder `_`
vorne weg; `_test/` bleibt so aus dem Binary, ohne dass ein Muster je
Unterordner nötig wäre (ein Muster wie `*/questions` bräche den Build, sobald
ein Flow keine Fragen hat, und `*/*` nähme `_test/` wieder mit). Ein Test
belegt, dass jeder Ordner unter `flows/catalog/` im Binary ankommt und kein
`_test/`. Ein Katalog-Flow braucht zusätzlich:

- `README.md`, englisch: was der Flow tut, welche Rollen er nennt, welche
  Parameter er nimmt.
- `_test/script.toml`: die Optionen des Laufs, die Antworten des Fake-Modells
  je Agentenbesuch in Reihenfolge und die Antworten an den Toren.
- `_test/journal.jsonl`: das erwartete Journal.

**Ein Test im Tor** (externes Testpaket `flows_test`) geht über jeden Ordner
unter `flows/catalog/`, lädt den Flow, fährt ihn mit fester Uhr gegen das
Fake-Modell nach `script.toml` und vergleicht das Journal Byte für Byte mit
`journal.jsonl`. `go test ./flows -update` erzeugt die Journale neu. Ein
fehlendes `README.md` oder `_test/` ist ein Fehler dieses Tests.
`internal/flow/load` importiert `flows` nicht: Der Lader bekommt den Katalog
als `fs.FS` übergeben, die Kommandozeile reicht `flows.FS()` hinein, und Tests
reichen eigene Kataloge. Ein angenommener Beitrag lädt und läuft also nachweislich.

Das Format von `script.toml` legt der Plan fest. Es muss ausdrücken können:
eine Antwort je Agentenbesuch mit den Feldern aus `reply`, einen eingereihten
Modellfehler, eine Antwort je Torbesuch und die Optionen des Laufs.

**Ein Beitrag ist reine Daten.** Braucht ein Flow einen Baustein, den es nicht
gibt, ist das ein eigener Pull Request mit Go-Code. `docs/{en,de}/flows.md`
beschreibt den Weg: Ordner anlegen, Skript und Journal erzeugen, `go test
./flows`, Pull Request mit `release:minor` (ein neuer mitgelieferter Flow ist
eine neue Funktion).

**A liefert einen Katalog-Flow:** `example`, der Beispiel-Flow der M1-Spec
(Entwurf → Tor → Ausgang), als Vorlage zum Kopieren. Bis B lehnt `loomux flow
run example` mit „no adapter for provider claude yet“ ab; er läuft nur im
Test. Das steht in seinem `README.md`.

## Läufe

- Verzeichnis `.loomux/state/runs/`, git-ignoriert wie der ganze
  Zustandsordner. Jeder Checkout und jeder Worktree hat seine eigenen Läufe.
- Journal `<id>.jsonl`, Marke `<id>.flow`, vierstellige Nummer, Claim mit
  `O_EXCL` — wie M1.
- Die Marke trägt den Flow-Namen, die Herkunft (`project`, `bundled`,
  `bundled+overlay`), die Optionen, die Basis und `loomux_version`. Die Zeile
  `runtime` entfällt.
- `resume` und `replay` vergleichen die Herkunft der Marke mit der heutigen.
  Kommt die `flow.toml` inzwischen aus einer anderen Quelle (Projektordner
  gegen Katalog, etwa weil `[flow] overrides` sich geändert hat), lehnen sie
  ab und nennen beide Herkünfte: das offene Tor gehört womöglich zu einem
  anderen Graphen. Ändert sich nur die Menge der Overlay-Dateien, warnen sie
  und laufen weiter.
- Die Basis bleibt: HEAD und die geänderten Dateien zu Laufbeginn; ohne Git
  verweigert ein Flow, der eine Basis braucht, den Start. Die eigenen Dateien
  des Laufs liegen jetzt unter einem git-ignorierten Ordner und tauchen in der
  Basis ohnehin nicht auf; die Abzugsregel aus M1 bleibt trotzdem, weil ein
  Projekt `.loomux/state/` in seiner `.gitignore` fehlen lassen kann.

**Kommandozeile:**

```
loomux flow run [<flow>] [--option name=wert]... [--root pfad]
loomux flow resume <lauf> [--answer text] [--root pfad]
loomux flow replay <lauf> [--root pfad]
loomux flow show <lauf|flow> [--root pfad]
loomux flow list [--root pfad]
```

Exit-Codes wie M1: 0 fertig, 1 Fehler, 2 Aufruffehler, 3 pausiert, sonst der
Code eines Bausteins.

**Aus einer Sitzung** ruft ein Agent in A die Befehle über die Shell; das
Werkzeug begrenzt einen Aufruf auf zehn Minuten. Der bequeme Weg mit Status und
ohne diese Grenze ist A2. Damit A2 nichts umbauen muss, hält A zwei Dinge:
Ein Lauf hängt an keinem Terminal (er liest kein stdin und schreibt seinen
ganzen Zustand in Journal und Marke), und die Kommandozeile ist eine dünne
Schicht über einer Funktion, die ein zweiter Aufrufer ebenso rufen kann.

Was A2 dennoch dazubaut, steht hier, damit niemand es A zuschreibt: Das Journal
schreibt einen Eintrag erst **nach** einem Knoten. „Läuft gerade Knoten X“ und
„der Prozess lebt nicht mehr“ lassen sich daraus nicht lesen. A2 legt dafür eine
eigene Statusdatei neben Journal und Marke (Knoten, Prozess, Startzeit); das
Journalformat bleibt unverändert, und die Pfadregel für
`.loomux/state/runs/**` deckt die Datei schon ab.

**Ein unterbrochener Lauf** (Abbruch von Hand, Absturz, Zeitgrenze eines
Werkzeugs) hat kein wartendes Tor und keinen Endeintrag. Nach dem M1-Vertrag
lehnt `resume` ihn ab. In A ist das kaum zu treffen, weil ohne Adapter kein
Knoten lange läuft. Das Fortsetzen eines solchen Laufs per Nachgehen und seine
Erkennung gehören deshalb zu A2, wo der Lebenslauf eines Laufs ohnehin
entworfen wird; B stößt als Erstes darauf und wartet auf A2. `show` mit einem Flow-Namen statt einer Laufnummer zeigt
den Flow: Knoten, Kanten, Rollen und ihre Auflösung. Verwechseln lassen sich
beide nicht: Eine Laufnummer besteht aus Ziffern, ein Flow-Name beginnt mit
einem Buchstaben.

## Session-Start

`loomux hook session-start` meldet jeden wartenden Lauf des Projekts:

```
run 0001 is waiting at confirm: Ship it?
a human answers it with: <binary> flow resume 0001 --answer "your answer"
```

„a human“ steht da, weil der Hinweis auch beim Modell ankommt und die Antwort
ein Mensch gibt (siehe „Wächter“). `<binary>` ist der Pfad, aus dem der Hook
selbst läuft (`os.Executable`), mit Schrägstrichen: `loomux` liegt nicht auf
jedem PATH — in Git Bash auf diesem Rechner nicht, gemessen am 2026-09-26 —,
und je nach Projekt läuft der Hook aus `bin/loomux.exe` oder aus
`%LOCALAPPDATA%\loomux\bin\loomux.exe`. Ein kaputtes Journal oder eine kaputte
Marke nennt der Hook mit Datei und Grund, ohne zu blockieren.

**Zwei Binaries, ein Lauf.** Einen Lauf kann ein anderes Binary geschrieben
haben als das, aus dem der Hook läuft; beide Stände driften (siehe
`AGENTS.md`, Session-Start-Warnung). Scheitert das Lesen und trägt die Marke
eine andere `loomux_version` als das lesende Binary, nennt der Hook beide statt
„kaputt“: „run 0001 was written by loomux 0.0.0-dev, this is 3.0.0 (beta)“. Er
behauptet nicht „neuer“: Ein Checkout-Build meldet `0.0.0-dev` (gemessen am
2026-09-26 an `bin/loomux.exe`, das maschinenweite Binary `3.0.0 (beta)`), ein
Versionsvergleich wäre also Raten.

`internal/hooks` bindet dafür nur `internal/flow/runs` und den Leser aus
`internal/flow/journal`. `internal/cli/imports_test.go` bekommt eine Liste wie
`forbiddenForHooks`: `internal/flow/runner`, `internal/flow/blocks`,
`internal/flow/load` und `internal/flow/model` bleiben vom Hook-Pfad fern.

Gemessen wird vor und nach dem Umzug: `loomux hook session-start` gegen ein
Projekt ohne Läufe und gegen eines mit einem wartenden Lauf, kalt und warm, mit
Binärgröße. Der Eintrag geht nach `docs/{en,de}/benchmarks.md`. M1 maß
+2,1 ms warm, innerhalb der Streuung.

Ebenso gemessen wird `loomux hook pre-tool-use`, weil A dort neue eingebaute
Regeln einführt: ein Edit außerhalb von `.loomux/`, ein Edit unter
`.loomux/flows/` und eine Shell-Zeile, kalt und warm, vor und nach dem Umzug.

## Wächter

**Die Antwort an einem Tor gibt ein Mensch.** `writesConfiguration` in
`internal/hooks/guard.go` (oder eine Schwester daneben) verweigert einem Agenten
`loomux flow resume … --answer`, mit eigenem Grund:

> A flow's gate asks a human; the answer is theirs. Ask the user to answer it
> with `flow resume <run> --answer "…"` themselves.

Der Grund nennt kein `!` und keinen Pfad: Welcher Befehl beim Menschen
funktioniert, hängt von Terminal und Binary ab; den genauen Pfad nennt der
Session-Start-Hinweis.

`run`, `show`, `list`, `replay` und `resume` ohne `--answer` bleiben erlaubt.
Ohne die Regel könnte ein Agent die Pflichttore des späteren
Entwicklungszyklus selbst beantworten, darunter die Freigabe des Plans und des
Pushs. Die Regel greift in jeder Schreibweise, die `writesConfiguration` heute
erkennt (Pfade zum Binary, Anführungszeichen, verkettete Befehle), für
`--answer text` wie für `--answer=text`, und bekommt dieselben Tests.

**Die Laufdateien schützt der Wächter ebenso.** Ein Agent, der einen
`answered`-Eintrag selbst ins Journal schreibt, bekäme beim nächsten erlaubten
`resume` ohne Antwort ein beantwortetes Tor. Deshalb:

- eine eingebaute Pfadregel `.loomux/state/runs/**` in `builtinPathRules`, wie
  `.loomux/state/hooks/**` für das Stop-Tor, mit dem Grund „a flow's journal
  and marker are written by loomux, not by the party the gates ask“;
- eine eingebaute Befehlsregel für Shell-Schreibzugriffe auf diesen Ordner
  (`>`, `tee`, `sed -i`, `Set-Content`, `cp`/`mv` darauf, dazu Löschen mit
  `rm`, `Remove-Item`, `git rm`: ein gelöschtes Journal verliert die Antworten
  eines Menschen), nach dem Muster der Regel für `.loomux/config.toml`.

**Mitgelieferte Flows ändert nur ein Mensch.** Ohne diese Regel ließe sich ein
Pflichttor abschaffen, statt es zu beantworten: Ein Agent legte
`.loomux/flows/dev-cycle/flow.toml` ohne Plan- und Push-Tor an, das verdeckte
den mitgelieferten Flow, oder er weichte per Overlay die Anweisung des
Reviewers auf, und startete danach erlaubt `flow run`. Deshalb:

- **Die Freigabe steht in der Datei, die nur ein Mensch schreibt.**
  `[flow] overrides = ["dev-cycle"]` in `.loomux/config.toml` nennt die
  mitgelieferten Flows, die ein Projekt-Flow gleichen Namens verdecken oder
  überlagern darf. Der Lader erkennt ein Verdecken oder Overlay nur für einen
  Namen aus dieser Liste an; sonst fährt der mitgelieferte Flow, und `list`,
  `show` und der Session-Start warnen („.loomux/flows/dev-cycle is ignored:
  [flow] overrides does not name it“). Ein Agent schlägt eine Freigabe mit
  `loomux config set flow.overrides … --propose` vor.

  Warum in der Config und nicht nur im Wächter: Der Wächter kennt nur die
  Namen im Katalog *seines* Binarys. `dev-cycle` kommt erst mit Teil D; bis
  dahin könnte ein Agent den Namen vorab besetzen, und nach dem Release
  verdeckte seine Fassung den mitgelieferten Flow. Und Wächter und Lauf können
  aus verschiedenen Binaries mit verschiedenen Katalogen kommen (gemessen:
  `0.0.0-dev` gegen `3.0.0 (beta)`). Die Liste in der Config hängt von keinem
  der beiden ab.
- **Der Wächter gibt früh Rückmeldung** und schützt freigegebene Inhalte vor
  späteren Änderungen: Er verweigert einem Agenten jeden Schreibzugriff auf
  `.loomux/flows/<name>/**`, wenn `<name>` ein Flow des Katalogs im laufenden
  Binary ist oder in `[flow] overrides` steht, per Write/Edit wie per Shell,
  Löschen eingeschlossen (`rm`, `Remove-Item`, `git rm`: sonst verschwände ein
  strengeres Overlay des Menschen). Grund: „a bundled flow's gates and
  instructions are a human's to change; give your flow a name of its own, or
  ask the user to hide or overlay `<name>`“.
- Die Katalognamen liest der Wächter aus dem eingebetteten Katalog, und die
  Regel entsteht erst, wenn ein Pfad oder eine Zeile `.loomux/flows/` berührt,
  nach dem Muster von `builtinCommands` (`sync.OnceValue`). Das Paket `flows`
  enthält nur Daten und darf auf dem Hook-Pfad liegen; der Importgraph-Test
  oben lässt es zu.
- Verbleibende Grenze, benannt: Nach der Freigabe eines Namens schützt nur noch
  der Wächter dessen Inhalt; ein Wächter-Binary ohne diese Regel ließe eine
  Änderung durch. Stärker wäre ein Hash des freigegebenen Inhalts in der
  Config; das bleibt bis zu einem Anlass offen.
- Einen Flow mit eigenem Namen darf ein Agent schreiben und starten. Zum
  Default machen kann er ihn nicht, denn `[flow] default` steht in
  `.loomux/config.toml`. Ein torloser Flow unter eigenem Namen gibt einem Agenten
  nichts, was er nicht auch ohne Flow täte; das ist die benannte Grenze.
- **Sichtbarkeit:** Der Session-Start-Hinweis und jede Torfrage nennen Flow
  und Herkunft, bei einem Overlay mit den ersetzten Dateien:
  `run 0001 (dev-cycle, bundled+overlay: instructions/review.md) is waiting at
  approve_plan: …`. Die Marke hält Herkunft und ersetzte Dateien fest.
- Bringt ein neues Release einen Katalog-Flow mit einem Namen, den ein Projekt
  schon für einen eigenen Flow nutzt, fährt ab dann der mitgelieferte, bis
  `[flow] overrides` den Namen freigibt; `list`, `show` und der Session-Start
  warnen bis dahin („.loomux/flows/<name> is ignored: [flow] overrides does
  not name it“). Mit Freigabe zeigt `flow list` den Projekt-Flow als
  `project (hides bundled)`.

Die Grenze ist dieselbe wie bei `config.toml` und wird so genannt: Ein
Programm, das ein Agent erst schreibt und dann startet, sieht der Wächter
nicht. Die Tore halten einen Agenten auf, der sich an die Werkzeuge hält,
keinen, der sie gezielt umgeht; dafür gibt es das Journal, in dem jede Antwort
mit Zeit und Besuch steht.

## Übernahme aus M1

**Unverändert übernommen**, samt Tests: der Verhaltensvertrag der M1-Spec
(Graph prüfen, nächster Knoten, Lauf, Deckel, Knotenfehler, Tor, Fortsetzen,
Nachgehen, Replay), die Antwortregel am Tor, die Bausteine `agent`, `gate`,
`exit`, Bedingungssprache, Platzhalter, kanonisches JSON und beide Hashes,
Fake-Modell, die Uhr als Eingabe des Runners, Ablehnung von `command` beim
Laden („planned but not built“).

**Entfällt mit Grund:**
- Lesen von Python-Journalen und die optionalen Felder dafür: loomux setzt
  keine Läufe von ultraloom fort. `model`, `definition_hash` und `role` sind in
  jedem Eintrag, den loomux schreibt, vorhanden (`null`, wo sie nichts sagen).
- Die Markenzeile `runtime` und die Kollisionskommentare gegen Python in
  `claim.go` und `marker.go` sowie der Satz „neither runtime resumes the
  other's runs“ im Session-Start: Es gibt keine zweite Laufzeit.
- Die Punkte 1 und 2 aus „Für M2“ der Wave-3-Befunde (Nummernkollision mit
  Python): aus demselben Grund.

**Bleibt offen, mit Zeile im Plan:** Die Punkte 3 und 4 aus „Für M2“
(`modelFor` kennt nur `kind = "agent"`; `resume` ohne Antwort fragt zuerst die
Modellfabrik) gehören zu B und C. Die Punkte unter „Kleinere Befunde, bewusst
offen“ ordnet der Plan einzeln ein: behoben, wo der Umzug die Stelle ohnehin
anfasst, sonst mit Grund offen.

**Neu gegenüber M1:** Ordnerformat, Rollen, Overlay, Katalog mit Beitragstest,
`[flow] default`, Config über `internal/config` und das Schema samt benannter
Schlüssel, LF beim Lesen von Anweisungen, feste Orte für Anweisungen und Fragen,
die Wächterregeln für Antwort und Laufdateien, `show <flow>`.

## Doku im selben Pull Request

- Fusions-Spec: Flow als eigene Spur neben der Fusion (Nachtrag unter
  „Reihenfolge der offenen Stufen“), Lückenzeile #14 zur Hälfte in A, #22 an B.
  Das steht schon im Commit dieser Spec.
- Roadmap in `README.md` und `README.de.md`: Die Zeile „Flow-Laufzeit“ nennt,
  was A gebaut hat und was B bringt; eine neue Zeile „Flows über MCP“ (A2). Hängt am Pull Request der Roadmap; ist er
  bis dahin nicht gemergt, rebased dieser Zweig darauf.
- Neue Seite `docs/{en,de}/flows.md`: Format, Rollen, Overlay, Auswahl,
  Katalog, Beitragsweg.
- `docs/{en,de}/cli-reference.md` (`loomux flow`),
  `docs/{en,de}/configuration.md` (`[agent]`, `[flow]`),
  `docs/{en,de}/hooks.md` (Session-Start).
- `AGENTS.md`: wo `flows/` liegt.
- `docs/{en,de}/benchmarks.md`: die Messung des Session-Starts.
- Label `release:minor`: neue Befehle und Schlüssel, nichts Bestehendes bricht.

## Nachweis

- TDD, 100 % je Funktion; eine Ausnahme nur mit `//coverage:exempt <grund>`.
- Jede portierte Datei behält ihre Tests; ein Test, der Python-Verhalten
  belegte, entfällt mit dem Verhalten und wird im Plan genannt.
- Neu getestet: Rollenauflösung in jeder Stufe der Kette, Overlay (ersetzt,
  „overlays nothing“, Overlay ohne Original, fremde Datei), Auffindung mit
  Herkunft, `[flow] default` (gesetzt, fehlt, nennt keinen Flow), die neuen
  Config-Schlüssel über `schema.Validate`, die benannten Schlüssel in
  `Lookup`, `config list`, `set`, `unset` und `--propose`, der Katalog-Test
  über `flows/`, CRLF in einer Anweisung (gleicher Hash wie LF), Pfade außerhalb
  von `instructions/` und `questions/`, die Wächterregeln (Antwort am Tor in
  allen Schreibweisen, Pfad- und Shell-Schutz der Laufdateien, Schutz eines
  Katalognamens und eines freigegebenen Namens unter `.loomux/flows/` bei
  freiem eigenem Namen, Löschen als Schreibform), `[flow] overrides` (Verdecken
  und Overlay nur mit Freigabe, sonst Warnung und mitgelieferter Flow), die
  Herkunft in Hinweis und Torfrage, der
  Session-Start mit dem Pfad des Binarys und einem Lauf eines anderen Binarys,
  die Namensregel für Flows (Bindestrich ja, Großbuchstabe, `_` und `.` vorne
  nein), die Einbettung jedes Katalog-Ordners ohne `_test/`, der Importgraph.
- Golden-Journal: der Katalog-Test über `example` ersetzt den Golden-Test aus
  `cmd/flow`.
- Kommandozeile: jeder Befehl, jede Ablehnung mit Exit-Code, wie M1.

## Außerhalb von A

- Die MCP-Werkzeuge für Flows und der abgelöste Kindprozess (A2).
- Echte Adapter und Werkzeugprofile je Anbieter, `[agent] settings` (B).
- Knotenarten `command` und Fächer, Prüfkette, Rot-Prüfer, Commit und Board als
  Bausteine (C).
- Der Entwicklungszyklus und seine Anweisungen (D).
- Ein Modul `flow` in `[modules]` und die Einrichtung von Flows durch
  `loomux init`.
- Der Editor und `layout.toml` über das Festhalten hinaus (W5).
- Unterflows und Steckplätze.
