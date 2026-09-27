# ulflow — die Graph-Laufzeit in Go (M1)

**Datum:** 2026-09-11
**Stand:** freigegeben am 2026-09-11, nicht umgesetzt
**Zweig:** `feature/agent-harness`
**Bezug:** [Harness-Graph](2026-09-10-agent-harness-graph.md) ·
[Go-Hooks für drei Hosts](2026-09-10-go-hooks-drei-hosts-design.md), liegt auf
`master`

## Ziel

`ulflow` ist ein neues Go-Binary, das Flows ausführt: Graph laden und prüfen,
Knoten abarbeiten, jeden Schritt ins Journal schreiben, an Toren anhalten,
fortsetzen und wiedergeben. Es übernimmt die Aufgabe von `ultraloom run`,
`show`, `resume` und `replay`, die heute in Python liegen.

Flows sind **keine Module mehr, sondern Daten**. Ein Flow ist eine TOML-Datei,
die Knoten aus einem festen Vorrat an Bausteinen zusammensetzt. Verhalten, das
über Daten hinausgeht, bleibt Go-Code in diesen Bausteinen.

M1 ist die Grundlage für den MVP `spec_to_board`. Der mitgelieferte
Entwicklungszyklus wird später ein Daten-Flow wie jeder andere, nicht Teil der
Laufzeit.

## Einordnung in die Migration

| # | Teilprojekt | ersetzt |
|---|---|---|
| **M1** | **Graph-Laufzeit in Go** (dieses Dokument) | `graph`, `runner`, `state`, `journal`, `gate`, `discovery`, `tools`, `model/port`, `model/fake`, Flow-Teil von `cli.py` |
| M2 | Modellzugang: Adapter für `claude -p` und `agy -p` | `model/agent_sdk.py` |
| — | MVP `spec_to_board` | — |
| M3 | `verify_until_green` als Daten-Flow | `flows/verify_until_green.py` |
| M4 | Prüfkette und Hooks, Go-Hooks-Stufen 2 bis 5 | `checks`, `process`, `config`, `worktree`, `hooks/` |
| M5 | Commit-Sprachprüfung, gleichwertig | `commit/*` |
| M6 | Schnitt: Python entfernen, `ulflow` wird zu `ultraloom` | alles Übrige |

Bis M6 laufen Python- und Go-Laufzeit nebeneinander. Die Python-Seite wird
dafür **nicht** angepasst.

**Korrektur an der Go-Hooks-Spec, fällig mit M6.** Deren Stufe 5 löscht
`runner.py`, `journal.py`, `state.py` und `gate.py`, während derselbe Text
`verify_until_green` und `model/*` in Python belässt. `runner.py` importieren
aber nur `cli.py` und `verify_until_green.py`. Nach Stufe 5 hätte der
Python-Flow keinen Runner. Mit M1 bis M3 entfällt der Widerspruch, weil der Flow
dann auf `ulflow` läuft. Die Go-Hooks-Spec wird beim Planen von M4 darauf
angepasst.

## Entscheidungen

| Frage | Entscheidung | Grund |
|---|---|---|
| Flows als Code oder Daten | **Mischform:** Bausteine und Prädikate in Go, Zusammensetzung in TOML | Go lädt keinen Code zur Laufzeit (`plugin` läuft unter Windows nicht). Anweisungen, Modelle und Verdrahtung ändern sich am häufigsten und sollen ohne Build änderbar sein |
| Anpassbarkeit | Vier Ebenen: überschreiben, neu verdrahten, eigenes Verhalten über externe Programme, Baustein beitragen | ultraloom wird Open Source |
| Externe Programme als Knoten (`command`) | **Im Format vorgesehen, gebaut nach dem MVP** | Der MVP braucht sie nicht; das Format soll sie später ohne Bruch aufnehmen |
| Zustand | Deklarierte Felder mit Typen; Bausteine nennen, was sie lesen und schreiben; Prüfung beim Laden | Ein Tippfehler fällt beim Laden auf, nicht nach bezahlten Modellaufrufen |
| Bedingungssprache | Vergleiche und Prädikatnamen, nur `\|` oder nur `&`, keine Klammern; einzige Arithmetik `<param> + <zahl>` bei Deckeln | In einem Satz erklärt, vollständig beim Laden prüfbar |
| Heimat | Eigenes Binary `ulflow` aus `cmd/flow` | Ein Neubau tauscht nicht die Schreibschranke `ulguard` aller laufenden Sitzungen aus. `ultraloom` liegt heute zweimal auf dem PATH (`.venv` und `~/.local/bin`) |
| Übergang bis M6 | Ein Laufverzeichnis, eine Nummernregel, Python unverändert | Python wird abgelöst; nur der Go-Leser lernt die neuen Felder |
| Modellwahl | Kette: Knoten → Flow → `[agent] default` → Voreinstellung der CLI | Ohne Angabe gilt, was der Nutzer in seiner CLI eingestellt hat |
| Platzhalter | `{{feld}}` ohne Logik, ein Durchgang, unbekannter Name ist Ladefehler | Anweisungen enthalten JSON und Go-Code; einfache Klammern kollidieren |
| Replay | `definition_hash` je Eintrag; Abweichung wird gemeldet, nie abgelehnt | Beantwortet, welche Fassung der Anweisung ein Ergebnis erzeugt hat, ohne das Verbessern einer Anweisung am Tor zu verbieten |

## Verhaltensvertrag: was `ulflow` von Python übernimmt

Aus dem Code auf `master` gelesen. Jede Zeile wird ein Testfall.

**Graph prüfen** (`graph.py:166–191`), vor dem ersten Knoten, in dieser
Reihenfolge:

1. Der Startknoten existiert.
2. Jede Kante geht von einem bekannten Knoten zu einem bekannten Knoten oder zu
   `END`.
3. Jeder Knoten ist vom Start aus erreichbar.
4. Jeder Knoten hat mindestens eine Kante, die keine Fehlerkante ist.
5. Jeder Knoten auf einem Zyklus erlaubt mehr als einen Besuch.

Eine Fehlerkante trägt keine Bedingung (`graph.py:123–127`).

**Nächster Knoten** (`next_name`): die erste Kante in Dateireihenfolge, deren
Bedingung hält; eine Kante ohne Bedingung hält immer. Hält keine, endet der Lauf
mit Fehler. Fehlerkanten zählen dabei nicht.

**Lauf** (`runner.py:97–110`): Graph prüfen, vom Start aus gehen. Ergebnis ist
`done`, `paused` oder `error`.

**Besuchsdeckel** (`runner.py:368–371`): Besuche zählen im Zustand. Wer seinen
Deckel überschreitet, bekommt einen Journaleintrag `error` mit 0 Tokens und
0 Sekunden, und der Lauf endet mit Fehler.

**Knotenfehler** (`runner.py:245–249`): Eintrag `error` mit der Meldung. Hat der
Knoten eine Fehlerkante, geht der Lauf dort weiter, sonst endet er mit Fehler.
Ein Baustein kann einen eigenen Exit-Code wählen (heute `FlowExit`).

**Tor** (`runner.py:252–282`):
- Beim ersten Erreichen: Eintrag `paused`, `detail` ist die Frage, der Lauf
  endet mit `paused`.
- Ein zweites Erreichen desselben Besuchs ohne Antwort schreibt keinen zweiten
  `paused`-Eintrag.
- Die Antwort wird **dem Besuch** zugeordnet, dessen Eingabe-Hash der offene
  Eintrag trägt, nicht dem Knotennamen. Ein Tor auf einem Zyklus pausiert einmal
  je Durchgang.
- Der Antworteintrag hat `outcome = ok`, 0 Sekunden und
  `detail = "answered: <antwort>"`.

**Fortsetzen** (`runner.py:112–162`, `cli.py:349–360`):
- Ohne Antwort: das Journal ab Start nachgehen, das Tor pausiert erneut.
- Mit Antwort, aber ohne wartendes Tor: Fehler.
- Auf einen Lauf ohne wartendes Tor: abgelehnt, mit Hinweis auf `replay` und
  `run`.

**Nachgehen** (`runner.py:214–230`): Für jeden Knoten gilt der jüngste
Eintrag mit `outcome = ok` zu Name und Eingabe-Hash. Ist keiner da, endet das
Nachgehen und die Arbeit beginnt an dieser Stelle.

**Replay:** führt keinen Knoten aus.
- Ein Knoten ohne Eintrag ist ein Fehler, keine Fehlerkante.
- Eine Antwort wird abgelehnt.
- Ein Lauf, der an einem Tor wartet, wird abgelehnt.

**Journal** (`journal.py`): eine JSON-Zeile je Schritt, Zeilenende LF, Datei
unter `.ultraloom/runs/<lauf>.jsonl`. Ein Eintrag, der sich nicht serialisieren
lässt, wird abgelehnt, bevor die Datei berührt wird.

**Laufnummer** (`cli.py:143–148`): höchste numerische `*.jsonl` im
Laufverzeichnis plus eins, vierstellig.

**Laufmarke** (`cli.py:537–561`): `<lauf>.flow`, erste Zeile Flow-Name, danach
eine Zeile `name=<JSON-Wert>` je Option. Reserviert sind `baseline` und
`baseline_commit`.

**Basis** (`cli.py:586–606`): HEAD-Commit und die zu Laufbeginn geänderten
Dateien. Wird einmal beim Start genommen und in der Marke getragen. Ohne Git ist
keine Basis da; ein Flow, der eine braucht, verweigert dann den Start.

**Eigene Dateien des Laufs** (`cli.py:564–572`): Journal und Marke, einzeln
benannt. Bausteine, die Änderungen im Arbeitsbaum bewerten, ziehen genau diese
zwei ab.

**Anzeige** (`cli.py:445–458`): je Eintrag Knoten, Art, Ergebnis, Tokens,
Sekunden, Werkzeugprofil.

**Exit-Codes** (`cli.py:30–34`): 0 fertig, 1 Fehler, 3 pausiert, sonst der Code,
den ein Baustein gewählt hat.

**Werkzeugprofile** (`tools.py`):

| Profil | Werkzeuge |
|---|---|
| `read_only` | `Glob`, `Grep`, `Read` |
| `edit` | `read_only` plus `Edit`, `Write` |
| `shell` | `read_only` plus `Bash` |
| `mcp` | `read_only` plus `mcp__<server>` je Server aus `[agent].mcp_servers` |

Die Listen sind sortiert und ohne Doppel.

**Bewusst geändert gegenüber Python:**
- Flow-Namen sind ASCII-Bezeichner (`[A-Za-z_][A-Za-z0-9_]*`) statt Pythons
  `isidentifier()`.
- Der Eingabe-Hash ist `sha256` über kanonisches Go-JSON, nicht über Pythons
  `json.dumps`. Keine Seite setzt Läufe der anderen fort.
- Optionen heißen allgemein `--option name=wert`; `--checks` und `--max-rounds`
  entfallen als eigene Schalter.

## Das Flow-Format

```toml
schema_version = 1

[flow]
name  = "example"
start = "draft"
model = "writer"                     # optional, Stufe 2 der Modellkette

[params]
max_rounds = { type = "int", default = 5 }

[state]
verdict = { type = "string", default = "" }
count   = { type = "int",    default = 0 }
notes   = { type = "list[string]", default = [] }
answer  = { type = "string", default = "" }
answer_text = { type = "string", default = "" }

[[node]]
name        = "draft"
kind        = "agent"
instruction = "instructions/draft.md"
effort      = "high"
tools       = "edit"
max_visits  = "max_rounds + 1"
reply       = { verdict = "string", count = "int" }

[[node]]
name     = "approve"
kind     = "gate"
question = "instructions/approve-question.md"
choices  = ["yes", "no"]
answer   = "answer"
max_visits = "max_rounds + 1"

[[node]]
name    = "stop"
kind    = "exit"
code    = 4
message = "rejected after {{count}} rounds"

[[edge]]
from = "draft";   to = "approve"; when = "verdict == \"done\""
[[edge]]
from = "draft";   to = "draft"
[[edge]]
from = "approve"; to = "END";     when = "answer == \"yes\""
[[edge]]
from = "approve"; to = "stop"
[[edge]]
from = "stop";    to = "END"
```

(In echtem TOML steht ein Schlüssel je Zeile. Die `;` dienen hier nur der
Kürze.)

**Sprache:** Flow-Dateien, Anweisungen, Fragen und Meldungen sind englisch,
Knoten- und Feldnamen ebenso. Sie steuern ein Modell oder sind Konfiguration,
und beides fällt unter die Sprachregel in `AGENTS.md`. Die deutschen
Knotennamen im Harness-Graph sind Entwurf, keine Vorlage für Flow-Dateien.

**Kopf:** `schema_version` ist Pflicht. Eine unbekannte Version wird beim Laden
abgelehnt, mit der Version, die `ulflow` kennt.

**Parameter:** Typ und Vorgabe. Auf der Kommandozeile gesetzt mit
`--option name=wert`. Ein unbekannter Name oder ein Wert, der nicht zum Typ
passt, ist ein Fehler vor dem Start.

**Zustand:** Typen `string`, `int`, `bool`, `list[string]`, jedes Feld mit
Vorgabe. Der Startzustand besteht aus den Vorgaben. Er ist unveränderlich:
Bausteine liefern ein Delta, die Laufzeit baut daraus den nächsten Zustand.

**Bedingung** (`when`):
- ein **Vergleich** `<feld> <op> <wert>` mit `==`, `!=`, `<`, `<=`, `>`, `>=`.
  Als Wert gehen Text in Anführungszeichen, Ganzzahl, `true`, `false`, `[]` oder
  ein Parametername. `<` und Verwandte nur für `int`, `[]` nur für
  `list[string]`.
- ein **Prädikat**, also der Name eines Go-Prädikats.
- mehrere davon, verknüpft **nur** mit `|` oder **nur** mit `&`. Gemischt ist ein
  Ladefehler.

**Kante:** `from`, `to`, optional `when`, optional `on_error = true`. Eine
Fehlerkante wird nur genommen, wenn der Knoten mit Fehler endet, und trägt kein
`when`; beides zusammen ist ein Ladefehler (`graph.py:123–127`). Für die Regel
„jeder Knoten hat eine Kante, die keine Fehlerkante ist" zählt sie nicht.

**Deckel** (`max_visits`): Ganzzahl, Parametername oder
`<parameter> + <ganzzahl>`. Vorgabe 1.

**Platzhalter** in Anweisungen, Fragen und Meldungen: `{{name}}` für ein
Zustandsfeld oder einen Parameter.
- Ein Durchgang: Eingesetzte Werte werden nicht erneut ausgewertet.
- Listen erscheinen als eine `- `-Zeile je Eintrag.
- `\{{` schreibt die Klammern wörtlich.
- Ein unbekannter Name ist ein Ladefehler.

**Anweisungsdateien** liegen relativ zur Flow-Datei. Eine fehlende Datei ist ein
Ladefehler.

## Bausteine

Ein Baustein ist eine Go-Implementierung mit festem Vertrag:

- **Name** der Knotenart, z. B. `agent`.
- **Eigene Schlüssel** im Knoteneintrag und deren Prüfung.
- **Gelesene und geschriebene Zustandsfelder mit Typ.** Sie stehen fest oder
  folgen aus dem Knoteneintrag, etwa `reply` beim Agenten.
- **Ausführung:** Zustand und Knoteneintrag rein, Delta oder Fehler raus.
- **Basis:** ob er die Basis des Laufs braucht. Braucht ein Knoten des Flows sie
  und gibt Git keine, verweigert `ulflow` den Start. So ersetzt die Laufzeit
  Pythons `needs_baseline`, das heute der ganze Flow erklärt.

Bausteine und Prädikate liegen in einem **Register**. Die Laufzeit kennt keinen
Baustein beim Namen, und Tests registrieren eigene.

**In M1 gebaut:**

| Art | Aufgabe | liest | schreibt |
|---|---|---|---|
| `agent` | rendert die Anweisung, ruft das Modell über den Port, prüft die Antwort gegen `reply` | die Platzhalter der Anweisung | die Felder aus `reply` |
| `gate` | stellt die Frage, pausiert; beim Fortsetzen muss die Antwort mit einer der `choices` beginnen | die Platzhalter der Frage | `answer` (die Wahl) und `<answer>_text` (der Rest der Antwort) |
| `exit` | beendet den Lauf mit `code` und `message` | die Platzhalter der Meldung | nichts |

**Im Format vorgesehen, nicht gebaut:** `command`. Schlüssel `run` (Programm und
Argumente), `reads`, `writes` (Felder mit Typ), `timeout`. Stdin bekommt die
gelesenen Felder als JSON-Objekt, stdout liefert das Delta als JSON-Objekt. Ein
Flow mit `kind = "command"` wird in M1 beim Laden abgelehnt: „command nodes are
planned but not built".

**Antwort am Tor:** Eine Antwort ist gültig, wenn sie genau einer der
`choices` gleicht oder mit ihr beginnt, gefolgt von `:` oder Leerraum.
Groß- und Kleinschreibung zählt, damit `no` und `No` nicht still dasselbe
heißen. `answer` bekommt die Wahl, `<answer>_text` den Rest ohne führendes `:`
und ohne Leerraum an den Rändern. Aus `offen: Fehlerfälle fehlen` wird also
`answer = "offen"` und `answer_text = "Fehlerfälle fehlen"`. Eine ungültige
Antwort lehnt `resume` ab, nennt die gültigen Wahlen, und das Tor bleibt offen.
Das ersetzt Pythons `apply` am Tor, das freien Code ausführte.

**Agentenantwort:** `reply` erlaubt nur Skalare (`string`, `int`, `bool`), wie
heute (`graph.py:38–44`). Jedes Feld in `reply` muss ein deklariertes
Zustandsfeld gleichen Typs sein. Dasselbe gilt für jedes Feld, das irgendein
Baustein schreibt, auch für `answer` und `<answer>_text` am Tor. Alle Felder sind Pflicht, eine Antwort mit
fehlendem oder zusätzlichem Feld ist ein Modellfehler.

## Prüfung beim Laden

Bevor irgendetwas läuft, prüft `ulflow` in dieser Reihenfolge und meldet
**alle** Befunde einer Stufe, nicht nur den ersten:

1. TOML lesbar, `schema_version` bekannt, Pflichtschlüssel vorhanden.
2. Jede Knotenart ist registriert; `command` wird mit eigener Meldung abgelehnt.
3. Jede Anweisungs- und Fragedatei existiert.
4. Jeder Platzhalter, jede Bedingung und jeder Deckel nennt ein deklariertes
   Feld oder einen deklarierten Parameter, mit passendem Typ. Jedes Prädikat ist
   registriert.
5. Kein Feld wird von zwei Bausteinen mit verschiedenen Typen geschrieben.
6. Die Graphregeln aus dem Verhaltensvertrag.
7. Modellkette: Jeder Name aus Knoten, Flow oder `[agent] default` steht unter
   `[agent.models]`.

Eine Meldung nennt Datei, Knoten und das Bekannte, z. B.
`example.toml: edge draft→approve reads "verdit"; known fields: answer, count,
notes, verdict`.

## Auffindung

- Projekt-Flows: `.ultraloom/flows/<name>.toml`.
- Mitgelieferte Flows werden per `go:embed` ins Binary gelegt. **M1 liefert
  keinen mit**; der erste ist der MVP.
- Ein Projekt-Flow verdeckt einen mitgelieferten gleichen Namens, wie heute
  (`discovery.py:135–142`).
- `ulflow list` zeigt jeden Flow mit Herkunft. Dateien, die sich nicht laden
  lassen, erscheinen mit dem Grund, statt zu fehlen.
- Die Python-Flows unter `.ultraloom/flows/*.py` (heute nur `smoke.py`) sieht
  `ulflow` nicht; sie laufen bis M3 oder M4 weiter über `ultraloom run`.

## Modellwahl und Modell-Port

**Konfiguration** in `.ultraloom/config.toml`:

```toml
[agent]
default = "writer"
mcp_servers = []

[agent.models.writer]
provider = "claude"
model    = "claude-opus-5"
```

Ein Eintrag unter `[agent.models]` braucht `provider`; fehlt der, ist das ein
Ladefehler. Fehlt `model`, gilt die Voreinstellung der CLI dieses Anbieters.

**Kette**, die erste gesetzte Stufe gewinnt:

1. `model` am Knoten
2. `model` in `[flow]`
3. `[agent] default`
4. nichts: Der Adapter übergibt kein Modell, die CLI nimmt ihre Voreinstellung.
   Der Anbieter ist dann `claude`.

**Port:**
- Anfrage: gerenderte Anweisung, Werkzeugliste, Effort, aufgelöstes Modell
  (Anbieter, Modell oder „Voreinstellung"), Antwortfelder mit Typ.
- Antwort: Felder, Tokens und, falls der Adapter es erfährt, das tatsächlich
  genutzte Modell.
- **Fake-Modell** wie `model/fake.py`: Antworten aus einer Warteschlange, jede
  Anfrage wird mitgeschrieben, ein eingereihter Fehler wird ausgelöst.

Das Fake-Modell ist nur aus Tests erreichbar, nicht über die Konfiguration.
`ulflow run` lehnt in M1 deshalb jeden Flow mit Agentenknoten vor dem Start ab:
„no adapter for provider claude yet". Die echten Adapter bringt M2.

## Journal

Die zehn Felder von heute, dazu zwei **optionale**:

| Feld | Inhalt |
|---|---|
| `model` | `null` bei Code- und Torknoten. Sonst `<anbieter>:<modell>` oder `<anbieter>:cli-default`; meldet der Adapter das tatsächliche Modell, steht dieses da |
| `definition_hash` | `sha256` über den kanonischen Knoteneintrag, die rohen Bytes von Anweisung oder Frage, das aufgelöste Modell, Effort und Werkzeugliste |

**Kanonisches JSON** für beide Hashes: Go-`encoding/json` mit sortierten
Schlüsseln und `SetEscapeHTML(false)`.

**`internal/journal` auf Go-Seite** lernt beide Felder als optional. Ein
fehlender Schlüssel ist gültig, damit Journale von Python und von älteren Läufen
lesbar bleiben; unbekannte Schlüssel bleiben ein Fehler. Davon profitiert
`ulguard hook session-start`, das wartende Tore aller Läufe meldet.

**Replay und Resume bei geänderter Definition:** Weicht `definition_hash` eines
nachgegangenen Eintrags von der jetzigen Definition ab, gibt `ulflow` eine
Warnung mit Knotennamen auf stderr aus. Der Lauf geht weiter. Knoten, die nach
dem Fortsetzen neu laufen, nutzen die neue Fassung.

**Laufmarke:** dasselbe Format wie heute, ergänzt um die Zeilen
`runtime="go"` und `ulflow_version="<version>"`. Python liest sie als Optionen
und findet den Flow dann nicht. Die Meldung ist ungenau, verschwindet aber mit
M6.

## Kommandozeile

```
ulflow run <flow> [--option name=wert]... [--root pfad]
ulflow resume <lauf> [--answer text] [--root pfad]
ulflow replay <lauf> [--root pfad]
ulflow show <lauf> [--root pfad]
ulflow list [--root pfad]
```

Exit-Codes wie im Verhaltensvertrag. Jede Ablehnung schreibt ihren Grund auf
stderr und legt kein Journal an, wenn der Lauf noch nicht begonnen hat.

## Pakete

| Paket | Aufgabe |
|---|---|
| `cmd/flow` | Kommandozeile, Exit-Codes |
| `internal/flow` | Verträge (Typen, Zustand, Graph, `Condition`, `Block`, `Registry` und `Catalog`), `Coerce`, `ReadText`/`Render` |
| `internal/flowload` | Format lesen, Prüfung beim Laden, Auffindung |
| `internal/flow/expr` | Bedingungen, Deckel |
| `internal/flow/tmpl` | Platzhalter |
| `internal/blocks` | Bausteine `gate`, `exit`, `agent` |
| `internal/runner` | Gehen, Nachgehen, Tore, Deckel, Fehlerkanten |
| `internal/journal` | vorhandener Leser, dazu Schreiber, kanonisches JSON, Eingabe-Hash und `definition_hash` |
| `internal/model` | Port und Fake |
| `internal/gitwork` | vorhandenes `HeadCommit`, dazu die geänderten Dateien für die Basis |
| `internal/flowcfg` | nur die Tabelle `[agent]` aus `.ultraloom/config.toml`, Modellkette, Werkzeugprofile |
| `internal/runs` | Laufnummer, Laufmarke, eigene Dateien des Laufs |

Platzhalter, Bausteine und Laufdateien haben eigene Pakete, damit parallele
Lanes keine Datei und keine Hilfsfunktion teilen (siehe „Wellen"). Beide Hashes
entstehen nur in `internal/journal`. Die Uhr wird dem Runner übergeben; kein
Paket ruft `time.Now` selbst, sonst ist das Golden-Journal nicht reproduzierbar.

`internal/flowcfg` ist absichtlich klein. Die vollständige Konfiguration liest
`internal/ulconfig` aus Go-Hooks-Stufe 3. Kommt das Paket, geht `flowcfg` darin
auf.

## Installation

`scripts/install.ps1` und `scripts/install.sh` bauen `ulflow` als drittes
Binary mit `go build -o`, wie `ulguard` und `ulinit`. Nicht `go install`, das den
Namen aus dem Verzeichnis nähme (`flow.exe`).

**`ulguard` muss mit neu gebaut werden.** M1 ändert `internal/journal`, und das
liest `ulguard hook session-start`. Ein altes `ulguard` meldete jeden Go-Lauf
mit `definition_hash` als beschädigt („unknown key"). Neu gebaut werden beide
Stände: `~/go/bin` über das Install-Skript und die Binaries im Checkout. Das
eigene Binary schützt die Schreibschranke also vor Änderungen an der Laufzeit,
nicht vor Änderungen an geteilten Paketen.

## Wellen

M1 ist zu groß für einen Plan in einem Zug und wird so geschnitten, dass
möglichst viel parallel und durch Subagenten entsteht.

**Regel:** Eine Lane läuft parallel zu anderen, wenn sie (a) nur Dateien
besitzt, die keine andere Lane anfasst, und (b) nur gegen Verträge baut, die vor
ihrer Welle committet sind. Was beides erfüllt, gehört in dieselbe Welle; alles
andere wartet auf die nächste.

### Welle 0 — Verträge (eine Aufgabe)

Nur Typen und Schnittstellen, keine Funktionsrümpfe, so genau wie Code:

- `internal/flow`: `Type`, `Value`, `State`, `Delta`, `Params`, die
  Strukturen für Knoten, Kante und geprüften Graph, `Condition` mit
  `Holds(State, Params) bool`, `Predicate`, `Block` (Knotenart, Prüfung der
  eigenen Schlüssel, liest/schreibt mit Typ, braucht Basis, Ausführung) und
  die Register-Schnittstelle.
- `internal/model`: `Request`, `Reply`, `Model`.
- je neues Paket eine `doc.go`, damit zwei Lanes im selben Paket keine Datei
  gemeinsam anlegen.

Grün heißt: `go build ./...` und `go vet ./...` laufen, der Coverage-Hook
bleibt grün (Typdeklarationen haben keine Anweisungen).

### Welle 1 — Blattpakete (sieben Lanes, parallel)

| Lane | besitzt | Grün heißt |
|---|---|---|
| `expr` | `internal/flow/expr` | Bedingungen parsen, prüfen, auswerten; Deckel; gemischtes `\|`/`&` ist Fehler |
| `tmpl` | `internal/flow/tmpl` | Platzhalter: Namen, Rendern in einem Durchgang, Listen, `\{{` |
| `journal` | `internal/journal` | Schreiber, kanonisches JSON, beide Hashes, optionale Felder im Leser; `go test ./cmd/guard/...` bleibt grün |
| `model` | `internal/model` (außer den Verträgen) | Fake mit Warteschlange, Mitschrift, eingereihtem Fehler |
| `gitwork` | `internal/gitwork` | geänderte Dateien zu Laufbeginn |
| `flowcfg` | `internal/flowcfg` | `[agent]`, Modellkette, Werkzeugprofile wie `tools.py` |
| `runs` | `internal/runs` | Laufnummer, Laufmarke mit `runtime` und `ulflow_version`, eigene Dateien |

### Welle 2 — Lader, Bausteine, Runner (vier Lanes, parallel)

| Lane | besitzt | Grün heißt |
|---|---|---|
| `loader` | `internal/flowload` | TOML lesen, Register, sieben Prüfstufen mit je einem scheiternden Flow, Graphregeln, Auffindung, Planungs-Flow als Testdatei |
| `gate-exit` | `internal/blocks/gate.go`, `exit.go` | Antwortregel am Tor, Meldung und Code am Ausgang |
| `agent` | `internal/blocks/agent.go` | Anweisung rendern, Port rufen, `reply` prüfen, Modellkette anwenden |
| `runner` | `internal/runner` | Verhaltensvertrag aus Gehen, Nachgehen, Toren, Deckeln, Fehlerkanten und Warnung bei geänderter Definition; getestet mit handgebauten Graphen und Test-Bausteinen, nicht über den Lader |

### Welle 3 — Zusammenbau (sequenziell)

`cmd/flow` mit `run`, `resume`, `replay`, `show`, `list` und `--option`;
Exit-Codes; Ablehnung „no adapter" und fehlende Basis vor dem ersten
Journaleintrag; Golden-Journal von Ende zu Ende; Install-Skripte; Neubau von
`ulflow` und `ulguard` in beiden Ständen. Grün heißt zusätzlich:
`ulguard hook session-start` meldet einen wartenden Go-Lauf. Nicht teilbar,
alles hängt an `cmd/flow/main.go`.

### Arbeitsweise der Lanes

- **Worktree je Lane:** `git worktree add .worktrees/ulflow-<lane> -b
  ulflow/<lane> <Basis>`, abgezweigt vom Stand von `feature/agent-harness` nach
  der vorigen Welle. Nicht über die Worktree-Isolation des Agent-Werkzeugs, die
  unter `.claude/worktrees/` anlegt.
- **Ein Subagent je Lane**, `model: "opus"` ausdrücklich. Einen Effort nimmt der
  Agent-Aufruf nicht an. Er committet in seinem Worktree und pusht nie.
- **Keine Lane ändert `go.mod` oder `go.sum`.** M1 braucht keine neue
  Abhängigkeit. Sprachstand ist `go 1.22` aus `go.mod`, nicht die installierte
  Toolchain.
- **Coverage:** `hooks/coverage-check.py` misst in einem temporären Verzeichnis
  oder in `coverage.out` relativ zur Wurzel. Lanes in eigenen Worktrees
  überschreiben einander das Profil also nicht.
- **Zusammenführen** macht der Orchestrator in `.worktrees/agent-harness`:
  `git merge --no-ff ulflow/<lane>`, danach `go test ./...`. Nach jeder Welle
  `git ls-remote origin` lesen, statt den Berichten zu glauben. Lane-Worktrees
  werden danach entfernt, der Harness-Worktree nicht.
- **Pläne:** ein Plan für Welle 0 und 1. Welle 2 und 3 bekommen ihren Plan erst,
  wenn die vorige Welle zusammengeführt ist, weil sich Verträge bis dahin noch
  bewegen können.

## Außerhalb von M1

- Adapter für `claude -p` und `agy -p` (M2).
- Knotenart `command` (nach dem MVP).
- Paralleler Prüferfächer. Der MVP arbeitet Linsen nacheinander ab.
- Bausteine für Prüfkette und Bereichsprüfung (M3, M4).
- Die Portierung von `verify_until_green` und `smoke.py`.
- Jede Änderung an Python.
- Die Regel „wer schreibt, nimmt nicht ab" (gehört zum Harness).

## Nachweis

- **TDD**, Test vor Code, **100 % Coverage** in jedem neuen Paket.
- **Vorlage sind die Python-Tests** der ersetzten Module, zusammen 1.565 Zeilen auf `master`
  (`test_graph`, `test_runner`, `test_state`, `test_journal`, `test_discovery`,
  `test_tools`, `test_model_fake`). Jeder Fall wird gegen den Python-Code
  nachgerechnet, nicht abgeschrieben.
- **Ein Test je Zeile des Verhaltensvertrags.**
- **Ladeprüfung:** für jede der sieben Stufen mindestens ein Flow, der genau
  dort scheitert, mit der erwarteten Meldung.
- **Planungs-Flow als Testdatei:** die Topologie des Abschnitts „Planen" aus
  dem Harness-Graph als TOML unter den Testdaten von `internal/flow`, mit
  einzeiligen Anweisungen. Er muss die Ladeprüfung bestehen, wird nicht
  eingebettet und nicht ausgeführt. Die echten Anweisungen sind Arbeit des MVP.
- **Golden-Journal:** ein fester Flow gegen das Fake-Modell mit fester Uhr. Das
  erzeugte Journal wird Byte für Byte verglichen.
- **Übergang:** `internal/journal` liest ein Python-Journal ohne die neuen Felder
  und ein Go-Journal mit beiden.
- **Kommandozeile:** `run`, `resume` mit gültiger und ungültiger Wahl, `replay`
  mit geänderter Anweisung (Warnung, kein Abbruch), `show`, `list`, jede
  Ablehnung mit Exit-Code.
