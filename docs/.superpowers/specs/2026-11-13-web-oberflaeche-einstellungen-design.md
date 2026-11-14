# Weboberfläche, Schritt 1: Einstellungen, Projekte, Dashboard

Stand 2026-11-13, Basis da097cab, Zweig `claude/llm-os-visualization-2db6af`.

## Anlass

loomux wird heute über `.loomux/config.toml`, `loomux init` und
`loomux config` im Terminal eingerichtet. Der Nutzer will ein „OS für sein
LLM": eine lokale Weboberfläche, in der er Graphen und Flows sieht und vor
allem Einstellungen setzt, ohne Config-Dateien zu bearbeiten. Die
Einstellungen sollen als Erstes kommen.

## Das ganze Vorhaben und dieser Schritt

Das Vorhaben sind fünf eigene Projekte, jedes mit eigener Spec:

1. **Shell, Einstellungen, Projekte, Dashboard** (diese Spec)
2. Code- und Wiki-Graph, nur lesend, 2D und später 3D
3. Flow-Ansicht: Graph, Lauf, Wiedergabe aus dem Journal
4. Prüfzentrum, Tabellensicht über Wiki-Frontmatter, Kontext-Viewer
5. Flow-Editor und Kanban

Daneben steht als eigenes Projekt direkt nach Schritt 1 die **Lesesperre**
(siehe Nicht-Ziele).

## Ziel und Erfolg

- Ein Mensch öffnet mit `loomux ui` die Oberfläche im Browser.
- Er setzt jede Einstellung eines Projekts und die globalen Einstellungen
  dort, sieht je Wert die Herkunft und vor jedem Schreiben den Diff.
- Globale Sperrregeln gelten in jedem Projekt, in dem der loomux-Hook
  läuft, egal was dessen Config sagt.
- Er fügt ein neues Projekt über einen Assistenten hinzu, der dasselbe tut
  wie `loomux init`.
- Die Startseite ist ein Dashboard aus Kacheln, die er ein- und ausblendet
  und sortiert.
- Erfolg heißt: Für keinen dieser Wege muss er eine TOML-Datei öffnen. Ein
  Agent kommt über die Oberfläche an kein Schreibrecht, das er über
  `loomux config apply` nicht auch hätte.

## Nicht-Ziele

- **Lesesperre.** Der Wächter beurteilt Schreiben und Ausführen, nicht
  Lesen. Eine Sperre, die auch `Read` und lesende Shell-Befehle trifft,
  ist ein eigenes Projekt. Bis dahin hindert eine globale Regel auf `.env*`
  nur das Ändern. Die Oberfläche sagt das an der Regel.
- Graphen, Flows, Prüfzentrum (Schritte 2 bis 5).
- Drag-and-drop im Dashboard, Kachelgrößen, mehrere Dashboards.
- loomux selbst installieren. Der Assistent setzt ein installiertes loomux
  voraus.
- Nutzer, Konten, Rollen, Zugriff aus dem Netz. Gemeint ist der Entwickler
  am eigenen Rechner.
- Übersetzte Oberfläche. Die Texte sind englisch wie die Meldungen der CLI
  (meine Festlegung).
- Globale Vorgaben für andere Schlüssel als `[policy]` (etwa `[verify]`
  oder `[commit]`).

## Anti-Ziele

Gescheitert, auch wenn es läuft, ist eine Lösung, wenn

- ein Geheimnis, das den Schreibweg öffnet, auf der Platte liegt,
- ein Projekt eine globale Sperrregel aufheben oder abschwächen kann,
- eine kaputte globale Datei Regeln still ausfallen lässt, statt dass der
  Wächter verweigert,
- die Oberfläche etwas schreibt, dessen Diff der Mensch nicht gesehen hat,
- sie einen zweiten Einrichtungs- oder Schreibweg neben `internal/setup`
  und `internal/config/edit` einführt.

## Ansätze

**A: Seiten aus dem Go-Binary, JavaScript nur als Insel (gewählt).**
`html/template`, gewöhnliche HTML-Formulare, eingebettet per `embed`.
JavaScript nur dort, wo eine Zeichenfläche nötig ist (ab Schritt 2). Keine
Node-Toolchain, die 100-%-Coverage-Regel greift unverändert. Nachteil:
jede Aktion lädt die Seite neu; der Flow-Editor in Schritt 5 kann eine
andere Wahl brauchen.

**B: JSON-API und statische JS-App ohne Build.** Flüssiger, aber deutlich
mehr Code außerhalb der Go-Tests und eine zweite Coverage-Kette.

**C: React/Vite, wie die Roadmap es nannte.** Reichstes Ökosystem, aber
Node-Build im Tor und viele Abhängigkeiten für Seiten, die Formulare sind.

Der Nutzer hat A gewählt, mit dieser Regel: **Jede Insel holt ihre Daten
über einen JSON-Endpunkt, nie aus dem gerenderten HTML.** So bleibt ein
späterer Wechsel einer Ansicht zu B oder C ein Umbau dieser einen Ansicht.
Schritt 1 hat keine Insel und braucht kein JavaScript.

## Entscheidungen

Vom Nutzer entschieden: Ansatz A mit der Insel-Regel; der Browser schreibt
direkt nach Diff und Klick; globale Sperren; Projekt hinzufügen mit
Assistent; Dashboard mit Ein-/Ausblenden und Sortieren; Lesesperre später.
Alles Weitere in diesem Abschnitt sind meine Festlegungen.

### 1. Ein eigener Prozess `loomux ui`, nicht `loomux serve`

`loomux serve` wird von der MCP-Brücke eines Agenten gestartet, und sein
Token steht in `serve.json`, das jeder Agent lesen kann. Hinge die
Oberfläche daran, läge das Geheimnis des Schreibwegs auf der Platte.

- `loomux ui` startet einen abgelösten Prozess mit einem eigenen Listener
  auf `127.0.0.1`. `internal/serve/spawn.go` löst heute nur
  `serve --foreground` ab, ohne stdin und mit dem Log von `serve`; das
  Ablösen wird so umgebaut, dass beide Befehle es nutzen.
- Der Elternprozess würfelt einen Einmalcode, reicht ihn dem Kind über
  dessen stdin (unter Windows kennt `os/exec` keine `ExtraFiles`), öffnet
  den Browser auf `http://127.0.0.1:<port>/login?code=…` und gibt die
  Adresse ohne Code aus.
- Das Kind tauscht den Code einmal gegen ein Sitzungscookie (`HttpOnly`,
  `SameSite=Strict`) und vergisst ihn. Sitzungen leben nur im Speicher.
- Ein zweites `loomux ui` beendet den laufenden Prozess über dessen PID
  und startet neu; alte Sitzungen sind damit weg. `loomux ui stop` beendet
  ihn. Auf der Platte liegt nur `ui.json` mit PID und Port, kein
  Geheimnis. Beendet wird eine PID nur, wenn hinter ihr wirklich ein
  loomux läuft: `ui.json` ist ein Hinweis, keine Wahrheit, wie
  `serve.json`.
- Port: eine feste Vorgabe, bei belegtem Port ein freier. Welche Nummer,
  entscheidet der Bauer.
- Jede Anfrage prüft den `Host`-Header gegen die eigene Adresse
  (DNS-Rebinding). Jede schreibende Anfrage ist `POST`, läuft durch
  `http.NewCrossOriginProtection` wie in `serve` und trägt ein
  Formular-Token der Sitzung.
- Die Oberfläche ruft die Pakete unter `internal/` direkt; sie spricht
  nicht mit `serve`.

*Verworfen: ein dritter Kanal in `serve` mit eigenem Token.* Der Befehl,
der den Login ausstellt, müsste sich bei `serve` ausweisen, also wieder
mit einem Geheimnis von der Platte.

### 2. Der Wächter

- `loomux ui` kommt in die Liste der Befehle, die nur ein Mensch ausführt
  (`writesConfiguration`, `internal/hooks/guard.go`). Der Schalter
  `llm_config` öffnet ihn nicht.
- **Die globale Datei wird geschützt.** Heute deckt die eingebaute
  Pfadregel nur `**/.loomux/config.toml` (`guard.go:67`); die globale
  `config.toml` im Zustandsverzeichnis trifft sie nicht, und ein Agent
  könnte eine globale Regel mit einer Shell-Zeile löschen. Der Wächter
  verweigert deshalb eingebaut jedes Schreiben, das er erkennt, auf
  `config.toml`, `ui.toml` und `ui.json` im Zustandsverzeichnis, über
  Datei-Werkzeuge wie über die Shell. Das ist die erste Scheibe: ohne sie
  hält das Anti-Ziel zu den globalen Regeln nicht.
- Restrisiko, in der Doku zu nennen: Der Wächter ist ein Netz über
  Befehlszeilen. Ein Agent, der den Browser des Nutzers fernsteuert, in
  dem die Sitzung läuft, handelt als der Nutzer. Das ist dieselbe Grenze
  wie bei `config apply`.

### 3. Globale Sperrregeln

- `<Zustandsverzeichnis>/config.toml` bekommt `[policy]` mit denselben zwei
  Regellisten wie die Projektdatei: `policy.paths.rules` und
  `policy.commands.rules`. `schema.GlobalKeys()` nennt sie, damit
  `loomux config --global` und die Oberfläche sie kennen.
- Der Wächter wendet an: eingebaute Regeln, dann globale, dann die des
  Projekts. Regeln verweigern nur, also ist die Vereinigung die ganze
  Semantik; ein Projekt kann nichts aufheben.
- Globale Pfad-Globs gelten relativ zur Wurzel des jeweiligen Projekts,
  wie die des Projekts.
- `config.ReadPolicy` ist heute an eine Projektwurzel gebunden
  (`internal/config/policy.go:78`), und nichts liest ein globales
  `[policy]`. Ein Leser für die globale Datei ist neu; er prüft Regeln
  genauso wie der des Projekts.
- Eine globale Datei, die nicht lädt (kaputtes TOML, kaputter Glob,
  fehlender `reason`), wirkt wie heute eine kaputte Projektdatei: Der
  Wächter verweigert und nennt Datei und Regel; die Werkzeuge, die heute
  nur einen Hinweis bekommen (`internal/hooks/pretool.go:54`), behalten
  diese Ausnahme. Eine fehlende Datei ist kein Fehler.
- Global wirkt nur, wo der loomux-Hook läuft. Die Oberfläche zeigt je
  Projekt, ob das so ist.

### 4. Einstellungsseite

Eine Seite je Ziel: *Global* und je Projekt eine. Quelle ist
`schema.Keys()` bzw. `schema.GlobalKeys()` mit `schema.CurrentOf`.

**Was ein Projekt ist.** Die Registry kennt nur Bereiche, darunter
schreibgeschützte, Wegweiser und geteilte, die keine Projekte sind; ein
Projekt, das nur Hooks nutzt, steht in keiner Liste. Die Projektliste der
Oberfläche ist deshalb die Vereinigung aus (a) den registrierten Bereichen,
an deren Wurzel eine `.loomux/config.toml` liegt, und (b) den Pfaden unter
`projects` in `ui.toml`. Der Assistent trägt jedes Projekt, das er
einrichtet, in (b) ein; ein Pfad ohne `.loomux/config.toml` erscheint als
„nicht mehr eingerichtet" mit einem Knopf zum Austragen.

- Schlüssel nach Modul gruppiert (`base`, `hooks`, `brain`, `graph`); ein
  ausgeschaltetes Modul zeigt seine Schlüssel als wirkungslos.
- Je Schlüssel: Doku-Satz, Eingabefeld nach `Kind` (Text, Zahl, Schalter,
  Auswahl, Liste), Vorgabewert, Herkunft (`set`, `default`, `preset`,
  `unset`) und „auf Vorgabe zurücksetzen".
- Suche über Schlüsselname und Doku-Satz und ein Filter „nur Geändertes",
  beide als Abfrageparameter, ohne JavaScript.
- Sperrregeln (`TableList`): Liste mit Herkunft je Regel (eingebaut,
  global, Projekt). Eingebaute und, auf einer Projektseite, globale Regeln
  sind nur lesbar. Eigene Regeln lassen sich anlegen und entfernen;
  Ändern ist Entfernen und Anlegen. `internal/config/edit` kann heute nur
  anhängen und bekommt dafür das Entfernen eines Blocks, in beiden
  Schreibweisen (`[[…]]` und Inline-Liste). `commit.allow` ist ebenfalls
  eine `TableList` und bekommt dieselbe Liste ohne Herkunftsspalte.
- `Table`-Schlüssel (`verify.profiles`, `model.roles`) und alles, was das
  Schema nicht kennt, zeigt die Seite an und verweist auf den Rohtext.
- **Rohtext:** ein Reiter mit der ganzen Datei in einem Textfeld. Er ist
  der Weg für alles, wofür es kein Formularfeld gibt, und hält das
  Versprechen „keine Datei öffnen" vollständig.
- **Schreiben in zwei Schritten.** Das Formular geht an eine Vorschau, die
  den neuen Text prüft und den Diff aus `edit.Diff` zeigt; erst deren
  Knopf schreibt, über `edit.Set`/`Remove`/`Append…`, sodass Kommentare
  erhalten bleiben. Geprüft wird wie in der CLI (`validated`,
  `internal/cli/config.go:363`): die Projektdatei mit `schema.Validate`,
  die globale mit den Lesern für `[model]` und `[search]` und dem neuen
  für `[policy]`. Hat sich die Datei seit der Vorschau geändert, schreibt
  die Seite nicht und zeigt die Vorschau neu; diese Prüfung macht heute
  `writeConfig`.
- **Vorschläge von Agenten** (`config set --propose`) stehen oben auf der
  Seite ihres Ziels, je mit Diff gegen den jetzigen Stand, Anwenden und
  Ablehnen. Der Diff neben dem Knopf ist der gesehene Diff; eine zweite
  Bestätigungsseite gibt es nicht. Ist die Datei inzwischen anders,
  schreibt das Anwenden nicht und die Seite lädt neu.
- **Eine Logik für CLI und Oberfläche.** Vorschläge, Prüfung und Schreiben
  hängen heute in `internal/cli` an `configTarget` (`config.go:27`,
  `proposeChange`, `proposeUnset`, `validated`, `writeConfig`,
  `config_proposals.go`). Das zieht in ein Paket, das beide rufen.
- **`llm_config`** ist kein Config-Schlüssel, sondern ein Flag am
  Bereichseintrag in `registry.toml` (`config.SetLLMConfig`). Die
  Projektseite zeigt ihn als Schalter nur bei einem registrierten Bereich,
  sonst mit dem Satz, warum er fehlt. Umlegen geht über eine
  Bestätigungsseite, die sagt, was der Schalter öffnet.

### 5. Projekt hinzufügen

Ein Assistent über `internal/setup`, in denselben Stufen wie `loomux init`:

1. Ordner: ein Textfeld für den Pfad. Kein Ordnerbrowser in diesem
   Schritt.
2. Befund aus `setup.Gather`: erkannte Agenten, Sprachen, Git-Zustand,
   schon registriert oder nicht.
3. Auswahl wie die Fragen von `init`, in derselben Reihenfolge und bis zur
   letzten (`llm_config`): je Modul an oder aus, die Teile, die
   Commit-Sprache. Vorbelegt mit `setup.DefaultChoice`.
4. Plan aus `setup.Build`: jede Änderung als Diff mit einem Haken, dazu
   die Aktionen und die Hinweise.
5. Ausführen über `setup.Apply`; nur Abgehakte werden geschrieben. Der
   Bericht nennt, was geschah und was scheiterte.
6. Weiter zur Einstellungsseite des neuen Projekts.

Ein Teil von `init` liegt heute in `internal/cli/init.go` statt in
`internal/setup`: der Runner der Aktionen (`initRun.action`),
`binaryThere`, `initNotes`, `hooksPathApproved` und die Regel, dass ein
abgelehntes `area-add` den `merge-hook` fallen lässt. Das zieht nach
`internal/setup`, damit CLI und Assistent denselben Weg gehen.

Aktionen, die lange laufen (`binary-install`, `model-pull`,
`graph-build`), laufen im Vordergrund der Anfrage; die Seite wartet. Ein
bereits eingerichtetes Projekt lässt sich mit demselben Assistenten erneut
durchlaufen, wie ein zweites `loomux init`.

### 6. Dashboard

Die Startseite, über alle registrierten Projekte. Kacheln sind eine feste
Liste im Code; jeder spätere Schritt hängt seine dort ein.

| Kachel | Zeigt |
|---|---|
| Projects | registrierte Projekte, je: Hooks eingerichtet, Binary veraltet, `llm_config` |
| Config proposals | offene Vorschläge je Ziel, Link zur Einstellungsseite |
| Waiting flow runs | Läufe, die an einem Tor warten (wie der Hinweis des Sitzungsstarts) |
| Review cases | offene Prüffälle der Wikis |
| Index state | Code-Graph und Suche je Projekt: frisch oder abgewichen |
| Version | laufende Version, Kanal, verfügbares Update |

- Anordnung je Nutzer in `<Zustandsverzeichnis>/ui.toml`, neben der
  Projektliste aus Abschnitt 4: eine geordnete Liste der sichtbaren
  Kacheln. Fehlt die Datei, sind alle sichtbar in der
  Reihenfolge oben. Ein unbekannter Name wird übergangen, eine kaputte
  Datei gilt als fehlend und die Seite sagt es.
- Bearbeiten über eine eigene Ansicht: je Kachel ein Haken und Knöpfe für
  hoch und runter.
- Eine Kachel, deren Quelle scheitert, zeigt den Fehler in sich und lässt
  die anderen stehen.
- Ein Projektumschalter in der Kopfzeile führt zur Einstellungsseite des
  Projekts.

### 7. Ablage und Regeln des Projekts

- Ein Paket für die Oberfläche unter `internal/`, der Befehl in
  `internal/cli`. Vorlagen und CSS per `embed`, beim ersten Gebrauch
  geparst (kein `init()`, keine Paketvariable).
- Tests über `net/http/httptest`; 100 % Coverage je Funktion wie überall.
- Keine neue Abhängigkeit in Schritt 1.

## Doku und Roadmap im selben Pull Request

- `README.md` und `README.de.md`: Funktionen und Roadmap. Die Zeile
  „Web-OS-Shell" verliert „React/Vite" und „von `loomux serve`
  ausgeliefert"; was Schritt 1 baut, verlässt die Roadmap. Neu als Zeilen:
  Lesesperre, Dashboard mit Drag-and-drop; „Graph-Visualizer" nennt 3D.
  Die Priorität der neuen und geänderten Zeilen setzt der Nutzer.
- `docs/en/` und `docs/de/`: `configuration.md` (globale `[policy]`),
  `cli-reference.md` (`loomux ui`), `hooks.md` (Wächter, Reihenfolge der
  Regeln), `getting-started.md` (Assistent als zweiter Weg).
- Die betroffenen Wikiseiten unter `docs/wiki/`.

## Dem Bauer überlassen

- Paketname, Routen, Aufbau der Vorlagen, Aussehen.
- Portnummer, Lebensdauer einer Sitzung, Cookie-Name.
- Wohin die gemeinsame Vorschlagslogik aus `internal/cli` zieht.
- Reihenfolge der Scheiben nach der ersten (Schutz der globalen Datei).
  Naheliegend: globale Regeln im Wächter; `loomux ui` mit Login;
  Einstellungsseite; Assistent; Dashboard.
