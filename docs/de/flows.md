# Flows

Ein Flow ist ein Graph aus Knoten, als Daten geschrieben: loomux fährt ihn,
hält ihn an einem Tor an, bis ein Mensch antwortet, setzt ihn fort und spielt
ihn aus seinem Journal nach. Die Befehle sind `loomux flow
run|resume|replay|show|list`; jedes Flag und jeder Exit-Code steht in der
[CLI-Referenz](cli-reference.md#12-flows-loomux-flow).

**Was heute läuft.** Tor- und Ausgangsknoten laufen. Agentenknoten laden,
zeigen Rolle und Modell und laufen im Test des Katalogs gegen ein
Fake-Modell, aber loomux hat noch keinen Modelladapter: Ein Flow mit einem
Agentenknoten lehnt den Start mit `no adapter for provider claude yet` ab
(oder dem Anbieter, auf den seine Rollen auflösen), bevor es einen Lauf gibt.
Die Adapter, Flows über MCP und weitere Knotenarten stehen in der
[Roadmap](../../README.de.md#roadmap).

---

## 1. Was ein Flow ist

Ein Flow ist ein Ordner, und der Name des Ordners ist der Name des Flows:

```
<name>/
  flow.toml            # schema_version = 1
  instructions/*.md    # was die Agentenknoten gesagt bekommen
  questions/*.md       # was die Tore fragen
  README.md            # Pflicht nur im Katalog
  _test/               # nur im Katalog: script.toml und journal.jsonl
```

- **Flow-Namen** folgen `[a-z][a-z0-9-]*`: klein, mit Bindestrich, vorne ein
  Buchstabe (`dev-cycle`, `strict-security-review`). Windows unterscheidet
  `Review` und `review` nicht, und `go:embed` lässt einen Namen mit `_` oder
  `.` vorne weg. Knoten-, Feld-, Parameter- und Rollennamen folgen
  `[A-Za-z_][A-Za-z0-9_]*`; ein Knoten, Feld oder Parameter darf nicht
  `true`, `false` oder `END` heißen.
- **`flow.toml`** trägt `schema_version` (dieser Build liest `1`), `[flow]`
  mit `start` (der erste Knoten, Pflicht) und `role` (die Rolle jedes
  Agentenknotens, der keine nennt), `[params]`, `[state]`, `[[node]]` und
  `[[edge]]`. Ein unbekannter Schlüssel oben, in `[flow]`, in einem Knoten oder
  in einer Kante ist ein Ladefehler, der die bekannten nennt. `[flow] name` ist
  auch einer (den Namen gibt der Ordner), ebenso `model` in `[flow]` oder an
  einem Knoten (ein Flow nennt eine Rolle, siehe Abschnitt 3).
- **Parameter und Zustandsfelder** werden als
  `name = { type = "…", default = … }` deklariert, mit den Typen `string`,
  `int`, `bool` und `list[string]`. Jedes braucht einen Default; ein Name darf
  nicht zugleich Parameter und Feld sein. Parameter sind das, womit ein Lauf
  startet (`--option name=wert`); Felder schreiben die Knoten. Ein Parameter
  darf keinen Namen tragen, den die Marke des Laufs für sich hält —
  `baseline`, `baseline_commit`, `loomux_version`, `origin`, `overlays` —,
  denn seine Option ließe sich dort nie schreiben (`parameter "origin" is a
  name the run's marker keeps for itself; name it otherwise`).
- **Texte.** Eine Anweisung ist eine Datei unter `instructions/`, eine Frage
  eine Datei unter `questions/`, in `flow.toml` relativ zum Ordner genannt, mit
  `/` zwischen Ordnern. Jeder andere Ort, ein absoluter Pfad oder einer, der
  den Ordner verlässt, ist ein Ladefehler: Ein fester Ort je Art erlaubt es
  einem Projekt, Datei für Datei zu überlagern (Abschnitt 4). Ein
  Wagenrücklauf vor einem Zeilenumbruch fällt beim Lesen weg, eine mit CRLF
  gespeicherte Datei rendert und hasht also wie ihre LF-Fassung. `{{name}}`
  rendert ein Feld oder einen Parameter; eine Liste wird zu einer Zeile `- `
  je Eintrag.

### Knotenarten

Jeder Knoten hat `name` und `kind` und darf `max_visits` haben; ein
Agentenknoten darf `role` haben.

| Art | Schlüssel | Was er tut |
|---|---|---|
| `agent` | `instruction` (Pflicht), `reply` (Pflicht: mindestens ein Feld, jedes `string`, `int` oder `bool`), `tools` (`read_only`, die Vorgabe; `edit`; `shell`; `mcp`), `effort` (Text), `role` | Rendert die Anweisung, fragt das Modell, auf das seine Rolle auflöst, und prüft die Antwort gegen `reply`; die Felder der Antwort gehen in den Zustand. |
| `gate` | `question` (Pflicht), `choices` (Pflicht: mindestens zwei Texte ohne Leerraum und `:`, keiner doppelt), `answer` (Pflicht: ein Zustandsfeld) | Hält den Lauf an und fragt. Eine Antwort ist eine Wahl, oder eine Wahl, gefolgt von Leerraum oder `:` und einer Begründung (`no: too thin`); Groß- und Kleinschreibung zählen. Die Wahl kommt in das Feld, das `answer` nennt, die Begründung in `<answer>_text`; `[state]` deklariert beide als `string`. |
| `exit` | `code` (Pflicht: 2 oder 4 bis 255), `message` (Pflicht, Text im Knoten, mit Platzhaltern) | Beendet den Lauf mit eigenem Exit-Code und eigener Meldung. 0, 1 und 3 sind abgelehnt: Die Laufzeit braucht sie für fertig, gescheitert und pausiert. |

`kind = "command"` lehnt der Lader mit `command nodes are planned but not
built` ab; jede andere unbekannte Art nennt die bekannten.

Die Werkzeugprofile: `read_only` ist Glob, Grep und Read; `edit` ist
`read_only` plus Edit und Write; `shell` ist `read_only` plus Bash, ohne Edit
und Write; `mcp` ist `read_only` plus ein `mcp__<server>` für jeden Server in
[`[agent] mcp_servers`](configuration.md#agent-modelle-für-die-rollen-eines-flows).
Jeder andere Name ist ein Ladefehler (`unknown tool profile "bogus"; known
profiles: edit, mcp, read_only, shell`).

### Kanten und Bedingungen

Eine Kante hat `from`, `to` (ein Knoten oder `END`) und höchstens eines von
`when` und `on_error = true`.

- **Die erste Kante, die hält, wird genommen.** Nach einem Knoten liest
  loomux seine Kanten in der Reihenfolge der Datei und nimmt die erste, deren
  `when` hält; eine Kante ohne `when` hält immer. Hält keine, endet der Lauf
  als Fehler (`no edge out of "x" applies to the current state`).
- **Eine Fehlerkante** wird genommen, wenn der Knoten scheitert, trägt kein
  `when` und ist auf dem normalen Weg kein Ausgang: Ein Knoten braucht
  mindestens eine gewöhnliche Kante.
- **Eine Bedingung** ist `<feld> <op> <wert>` mit `==`, `!=`, `<`, `<=`, `>`,
  `>=` (Anordnung nur für `int`), ein Feld `list[string]` nur gegen `[]`;
  mehrere Glieder werden nur mit `|` oder nur mit `&` verbunden, ohne
  Klammern. Jeder Name und jeder Typ wird beim Laden geprüft.
- **Besuche.** `max_visits` ist ohne Angabe 1, sonst eine ganze Zahl ab 1
  oder `"<int-Parameter> + <n>"`. Ein Knoten auf einem Zyklus muss mehr als
  einen Besuch erlauben. Ein Besuch über dem Deckel beendet den Lauf als
  Fehler (`node "draft" exceeded max_visits=6`).

### Die Ladeprüfung

Ein Flow lädt in Stufen: seine Deklarationen; jede Knotenart und ihre
Schlüssel; seine Texte; jeder Name, den ein Text, eine Bedingung oder ein
Deckel nutzt; jedes Feld, das ein Knoten schreibt, gegen `[state]`; dann der
Graph (der Start existiert, jede Kante endet an einem Knoten oder `END`, jeder
Knoten ist erreichbar und hat einen gewöhnlichen Ausgang, kein Knoten auf
einem Zyklus erlaubt nur einen Besuch). Die erste Stufe mit Befunden stoppt
das Laden und meldet alle, eine Zeile je Befund hinter dem Dateinamen.

---

## 2. Einen Flow fahren

```sh
loomux flow run [<flow>] [--option name=wert]... [--root ordner]
loomux flow resume <lauf> [--answer text] [--root ordner]
loomux flow replay <lauf> [--root ordner]
loomux flow show <lauf|flow> [--root ordner]
loomux flow list [--root ordner]
```

- **`run`** startet einen neuen Lauf des Flows, ohne Namen den von
  `[flow] default`. Ohne Default lehnt es ab und nennt die Flows, die es kennt.
- **`resume`** setzt einen pausierten Lauf fort. Mit `--answer` bekommt das
  Tor seine Antwort; ohne fragt das Tor erneut, und nichts wird geschrieben.
  Die Antwort gibt ein Mensch: Der Wächter verweigert sie einem Agenten
  (Abschnitt 6).
- **`replay`** leitet einen beendeten Lauf aus seinem Journal neu her und
  führt nichts aus.
- **`show`** druckt das Journal eines Laufs (eine Laufnummer besteht aus
  Ziffern) oder Knoten, Rollen und Kanten eines Flows (ein Flow-Name beginnt
  mit einem Buchstaben).
- **`list`** nennt jeden Flow, den das Projekt fahren kann, mit Herkunft, und
  einen Flow, der nicht lädt, mit dem Grund, statt ihn wegzulassen.

Das Projekt ist `--root`, sonst die nächste `.loomux/config.toml` oberhalb des
Arbeitsverzeichnisses, sonst das Arbeitsverzeichnis selbst: Auch ein Projekt
ohne Konfiguration hat Flows.

**Exit-Codes:** `0` fertig, `1` Fehler oder Ablehnung, `2` Aufruffehler, `3`
pausiert an einem Tor, und der eigene Code eines Ausgangsknotens. Ein Lauf,
der pausiert, nennt die Frage des Tors auf stdout:

```
$ loomux flow run ship
run 0001 (ship, project): paused
Ship it?
$ echo $?
3
```

**Wo Läufe liegen.** Unter `.loomux/state/runs/` des Projekts, git-ignoriert
mit dem ganzen Zustandsordner; jeder Checkout und jeder Worktree hat also
eigene Läufe. Ein Lauf ist eine vierstellige Nummer, eins mehr als die höchste
dort, und zwei Dateien:

- `<id>.jsonl`, das Journal: eine Zeile je Schritt mit dem Knoten, seiner Art,
  dem Ausgang (`ok`, `paused`, `error`), dem geschriebenen Delta, Tokens und
  Sekunden, dem Modell als `<anbieter>:<modell>` oder `<anbieter>:cli-default`,
  der Rolle und zwei Hashes — einer über das, was der Knoten gesagt bekam
  (`definition_hash`), einer über den Zustand, den er sah (`input_hash`).
- `<id>.flow`, die Marke: in der ersten Zeile der Name des Flows, dann die
  Optionen, woher der Flow kam (`origin`, bei einem Overlay dazu `overlays`),
  der Commit und die geänderten Dateien, von denen der Lauf ausging
  (`baseline_commit`, `baseline`, wenn Git antwortet), und die
  `loomux_version`, die ihn startete.

Fortsetzen und Nachspielen lesen den Flow, den die Marke nennt. Kommt seine
`flow.toml` inzwischen aus der anderen Quelle — dem Projektordner statt dem
Katalog oder umgekehrt, etwa weil sich `[flow] overrides` geändert hat —,
lehnen beide ab und nennen beide Herkünfte, denn das offene Tor gehört
womöglich zu einem anderen Graphen:

```
run 0001 started on project (hides bundled) and example now resolves to bundled, another flow.toml; start a new run with loomux flow run example
```

Eine andere Menge Overlay-Dateien ist derselbe Graph mit anderen Texten: Sie
warnen und laufen weiter, und ein Knoten, dessen Eintrag in `flow.toml` (seine
eigene `role` darin), Anweisung, Modell, Effort oder Werkzeuge sich geändert
haben, wird als auf einer inzwischen geänderten Definition gelaufen genannt.
Die Rolle zählt über das Modell, auf das sie auflöst: Eine geänderte Bindung
ändert die Definition, ebenso die eigene `role` eines Knotens, eine geänderte
`[flow] role` aber nur, wenn sie auf ein anderes Modell auflöst.

Ein Lauf liest kein stdin und hält seinen ganzen Zustand in Journal und
Marke; ein zweiter Aufrufer — heute die Shell eines Agenten, später ein
MCP-Werkzeug — fährt ihn also genauso. Ein Lauf, der zwischen zwei Knoten
abbrach (von Hand, durch einen Absturz, durch die Zeitgrenze eines
Werkzeugs), hat kein offenes Tor und kein Ende, und `resume` lehnt ihn ab.

---

## 3. Rollen und Modelle

Ein Flow nennt Rollen, nie Modelle; das Projekt bindet sie. Für jeden
Agentenknoten gewinnt die erste gesetzte Stufe:

1. `role` am Knoten, sonst `[flow] role`.
2. Die Bindung dieser Rolle in `[agent.roles]` der `.loomux/config.toml`: ein
   Modellname.
3. Ohne Rolle oder ohne Bindung: `[agent] default`, ebenfalls ein Modellname.
4. Nichts gesetzt: Anbieter `claude`, mit dem Vorgabemodell seiner CLI.

Ein Modellname zeigt auf `[agent.models.<name>]` mit `provider` und
wahlweise `model`; ohne `model` wählt die CLI des Anbieters ihre Vorgabe. Die
Schlüssel stehen in der
[Konfigurationsreferenz](configuration.md#agent-modelle-für-die-rollen-eines-flows).
Ein Flow lädt darum in jedem Projekt, und eine Rolle, die ein Projekt bindet,
aber kein Flow nennt, ist kein Fehler: Ein Projekt bindet Rollen für mehrere
Flows.

Ein Lauf hält ein Modell, die Agentenknoten eines Flows müssen also auf einen
Anbieter auflösen; ein Flow, dessen Rollen auf zwei auflösen, wird abgelehnt
(`flow "x" asks agy and claude; a run holds one model, so a flow asks one
provider`).

`loomux flow show <flow>` druckt jeden Agentenknoten mit seiner Rolle, dem
Modell, auf das sie auflöst, und woher:

```
$ loomux flow show example
example (bundled)
nodes:
  draft    agent  writer  claude:cli-default  role writer (node), the CLI's own default
  approve  gate
  stop     exit
edges:
  draft -> approve [verdict == "done"]
  draft -> draft
  approve -> END [answer == "yes"]
  approve -> stop
  stop -> END
```

Mit `[agent.models.w] provider = "agy"`, `model = "m1"` und
`[agent.roles] critic = "w"` liest sich ein Knoten mit der Flow-Rolle
`critic` so:

```
  judge  agent  critic  agy:m1  role critic (flow), bound to w
```

Einer, den `[agent] default` deckt, endet auf `[agent] default <name>`. Das aufgelöste Modell geht in den
Definitions-Hash ein; eine geänderte Bindung zeigt ein Replay also als
geänderte Definition.

---

## 4. Der Katalog und das Überschreiben

Flows kommen von zwei Orten:

- **bundled** — der Katalog unter `flows/catalog/` im loomux-Repository, ins
  Binary kompiliert. Heute hält er einen Flow, `example` (ein Agent entwirft,
  ein Mensch gibt frei, eine Ablehnung endet mit Code 4), die Vorlage zum
  Kopieren.
- **das Projekt** — `.loomux/flows/<name>/`.

Ein Projektordner mit `flow.toml` ist ein **eigener Flow**. Ein Ordner nur mit
`instructions/*.md` und `questions/*.md` ist ein **Overlay** über den
mitgelieferten Flow gleichen Namens: Jede Datei ersetzt die mitgelieferte mit
demselben Pfad. Eine Overlay-Datei, die der mitgelieferte Flow nicht hat, ein
Overlay ohne mitgelieferten Flow seines Namens und jede andere Datei neben
einem Overlay sind Ladefehler: Die ersten beiden sagen `… overlays nothing:
…`, der dritte `… is neither flow.toml nor a file under instructions/ or
questions/`. Ein eigener Flow darf
das `README.md` und `_test/` eines Katalog-Flows tragen, eine kopierte Vorlage
lädt also unverändert. Namen mit `.` vorne werden übergangen.

**Nur ein Mensch lässt einen Projekt-Flow an die Stelle eines mitgelieferten
treten.** Ein Projektordner mit dem Namen eines mitgelieferten Flows zählt nur,
wenn `[flow] overrides` in der `.loomux/config.toml` ihn nennt. Sonst fährt
der mitgelieferte Flow, und `list`, `show`, `run` und der Session-Start sagen
es:

```
warning: .loomux/flows/example is ignored: [flow] overrides does not name it
```

Die Herkunft, die jeder Befehl druckt, und mit ihr der Session-Start:

| Herkunft | Bedeutet |
|---|---|
| `project` | ein eigener Flow des Projekts, unter einem Namen, den der Katalog nicht hat |
| `bundled` | der Flow des Katalogs, wie ausgeliefert |
| `bundled+overlay: <dateien>` | der Flow des Katalogs mit den genannten Dateien aus dem Projektordner |
| `project (hides bundled)` | ein eigener Flow des Projekts unter einem Katalognamen, den `[flow] overrides` nennt |

Bringt ein Release einen mitgelieferten Flow unter einem Namen, den ein
Projekt schon für einen eigenen Flow nutzt, fährt ab dann der mitgelieferte,
mit der Warnung oben, bis `[flow] overrides` den Namen nennt.

`[flow] default` wählt, was `loomux flow run` ohne Namen startet. Ein Default,
der keinen Flow nennt, lässt `list` und ein `run` ohne Namen scheitern
(`[flow] default names "x", which is no flow here`), und `list` warnt vor
einem Namen in `overrides`, den der Katalog nicht ausliefert. Die
Config-Prüfung liest keine Flows, beides zeigt sich also erst dort.

---

## 5. Einen Flow beitragen

Ein Beitrag zum Katalog ist reine Daten:

1. Ein Ordner `flows/catalog/<name>/` mit `flow.toml`, `instructions/`,
   `questions/` und einem `README.md` auf Englisch: was der Flow tut, welche
   Rollen er nennt, welche Parameter er nimmt.
2. `_test/script.toml`: die Optionen des Laufs, dann ein Schritt je Besuch,
   der eine Antwort von außen braucht, in der Reihenfolge, in der der Lauf
   fragt. Ein Agentenschritt nennt seinen Knoten und trägt die Felder der
   Antwort (und `tokens`) oder einen `error`, den das Fake-Modell stattdessen
   wirft; ein Torschritt nennt seinen Knoten und trägt die `answer`.
   Unbekannte Schlüssel lassen den Test scheitern. Das Skript des Flows
   `example`:

   ```toml
   # The run drafts once, asks, and is rejected.
   [[step]]
   node = "draft"
   reply = { verdict = "done", count = 2 }
   tokens = 120

   [[step]]
   node = "approve"
   answer = "no: too thin"
   ```

   Optionen stehen in einer Tabelle `[options]` mit Textwerten, wie
   `--option` sie gäbe.
3. `go test ./flows -update` schreibt `_test/journal.jsonl`, das
   Golden-Journal; lies es. `go test ./flows` lädt dann jeden Katalog-Flow, wie
   das Binary ihn ausliefert, fährt ihn nach seinem Skript mit fester Uhr gegen
   das Fake-Modell und vergleicht das Journal Byte für Byte. Ein fehlendes
   `README.md`, `_test/script.toml` oder `_test/journal.jsonl` lässt den Test
   scheitern.
4. Ein Pull Request mit dem Label `release:minor`: Ein neuer mitgelieferter
   Flow ist eine neue Funktion.

`_test/` kommt nie ins Binary — `go:embed` lässt Namen mit `_` vorne weg. Ein
Flow, der eine Knotenart braucht, die die Laufzeit nicht hat, ist ein eigener
Pull Request mit Go-Code.

---

## 6. Tore gehören einem Menschen

Ein Tor fragt einen Menschen, und der Wächter (`loomux hook pre-tool-use`)
hält die Antwort und alles, was sie ersetzen könnte, von einem Agenten fern:

- **Die Antwort.** `loomux flow resume … --answer` wird einem Agenten in jeder
  Schreibweise verweigert, die die Regel für `loomux config` liest — ein Pfad
  zum Binary, Anführungszeichen, verkettete Befehle, `--answer text` und
  `--answer=text` —, ebenso ein `Start-Process` von loomux, dessen Argumente
  der Wächter nicht sieht:

  ```
  a flow's gate asks a human; the answer is theirs. Ask the user to answer it with `flow resume <run> --answer "…"` themselves
  ```

  `run`, `show`, `list`, `replay` und `resume` ohne `--answer` bleiben einem
  Agenten offen. `resume` und `replay` nehmen nur eine Laufnummer aus
  Ziffern; Journal und Marke, die ein Agent anderswo im Projekt fälscht
  (`resume ../../mine/x`), können keinen Lauf vertreten.
- **Die Laufdateien.** `.loomux/state/runs/**` unter jedem Ordner halten die
  Pfadregeln, für ein schreibendes Werkzeug und eine Shell-Zeile gleichermaßen:
  Ein `answered`-Eintrag, den ein Agent ins Journal schriebe, gäbe dem
  nächsten Fortsetzen ein beantwortetes Tor. Ein Löschen eines Ordners darüber
  und ein Glob, der sie auf der Platte trifft, zählen als Schreiben.
- **Die mitgelieferten Flows.** Ein Schreiben unter `.loomux/flows/<name>/`,
  unter jedem Ordner, verweigern die Pfadregeln, für ein schreibendes Werkzeug
  und eine Shell-Zeile gleichermaßen, wenn `<name>` ein Flow im Katalog dieses
  Binarys ist oder in `[flow] overrides` steht, in jeder Groß- und
  Kleinschreibung des Namens. Ein Glob an der Stelle des Namens zählt, wo er
  auf der Platte einen solchen Ordner trifft (`cp x .loomux/flows/ex*/`), ein Löschen
  dieses Ordners oder von `.loomux/flows` darüber zählt auch; ein Agent, der
  einen eigenen Flow schreibt, schreibt dessen Namen aus. Einen
  Flow unter eigenem Namen darf ein Agent schreiben und fahren; zum Default
  machen kann er ihn nicht, denn `[flow] default` steht in der
  `.loomux/config.toml`.

Die genauen Regeln stehen in
[Hooks](hooks.md#7-der-entscheidungsweg-von-pre-tool-use).

**Wie der Mensch antwortet.** Der Session-Start nennt jeden wartenden Lauf mit
dem Befehl, der ihn beantwortet, gebaut aus dem Pfad des Binarys, aus dem der
Hook läuft, mit Schrägstrichen (loomux liegt nicht auf jedem `PATH`), in
doppelten Anführungszeichen, wenn er Leerraum oder ein anderes Zeichen
enthält, das eine Shell liest:

```
run 0001 (ship, project) is waiting at confirm: Ship it?
  a human answers it with: C:/Users/me/project/bin/loomux.exe flow resume 0001 --answer "your answer"
```

Bei einem Overlay nennt die Herkunft die ersetzten Dateien, etwa `(example,
bundled+overlay: questions/approve.md)`.

Der Mensch führt diese Zeile in einem eigenen Terminal aus, oder in Claude
Code mit dem Präfix `!`, das sie in seiner Shell als seinen Befehl ausführt
statt als Werkzeugaufruf des Agenten. Die Ablehnung nennt keinen eigenen
Pfad: Welcher Befehl beim Menschen funktioniert, hängt von seinem Terminal und
seinem Binary ab, und der Hinweis hat den, den die Hooks dieses Projekts
nutzen.

**Die Grenze, benannt.** Der Wächter liest Werkzeugaufrufe, keine Programme:
Ein Programm, das ein Agent schreibt und dann startet, kann tun, was der
Wächter verweigert. Er erkennt loomux an seinem Dateinamen, `loomux` oder
`loomux.exe` (oder `go run` von `cmd/loomux`): Ein kopiertes oder umbenanntes
Binary kommt an jeder Regel zu loomux' eigenen Befehlen vorbei, die
verweigerte Antwort eingeschlossen. Die Tore halten einen Agenten auf, der
sich an seine Werkzeuge hält; das Journal hält jede Antwort mit Zeit und
Besuch fest.
