# Stufe 1b-2: `serve`, MCP über Streamable HTTP, stdio-Brücke

**Stand:** 2026-09-17. Vorgänger: `2026-09-15-loomux-stufe-1b-1-design.md` (die fünf
Datenbefehle, abgeschlossen). Nachfolger: 1b-3 (Wiki- und Doku-Umzug).
Rahmen: `2026-09-14-loomux-fusion-design.md`, Abschnitt `loomux serve`.

1b-1 hat die fünf lesenden Befehle gebaut und mit `brainRun` einen Einstieg
hinterlassen, der Kanal, Text, Hinweise und Fehler schon trennt. 1b-2 stellt
einen MCP-Dienst auf genau diesen Einstieg, ohne ihn neu zu erfinden: einen
langlebigen Prozess `loomux serve` mit zwei HTTP-Listenern und eine stdio-Brücke
`loomux mcp`, die ein MCP-Wirt startet.

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| SDK | `github.com/modelcontextprotocol/go-sdk` **v1.8.0** an beiden Enden — in `serve` als Streamable-HTTP-Handler, in der Brücke als stdio-Server plus Streamable-Client |
| Kanal | **Adresse, nicht Feld der Anfrage.** Zwei Listener, zwei Token |
| qmd | `brain search` **und** `serve` starten ihn, über eine geteilte Sperre |
| Wirtanbindung | **nur über die Brücke**; der HTTP-Endpunkt ist loomux-intern |
| Binary-Drift | **„neuer gewinnt"** — eine ältere Brücke tötet nie ein neueres `serve` |
| Werkzeugnamen | **`brain_*`**, flach; die Zeile der Fusions-Spec „Werkzeuge wie heute" wird verworfen |
| Upkeep | **nicht in 1b-2** — er braucht `reconcile`, und das ist Stufe 3 |
| Parität | **auf der Ebene der Werkzeugergebnisse**, nicht des MCP-Umschlags |

## Messungen vor dem Bau

Alle am 2026-09-17 auf dieser Maschine, warm, gegen `bin/loomux.exe` im Zustand
von `9c10424`. **Nachkorrigiert am 2026-09-17 20:45:** die erste Fassung dieser
Tabelle nannte 33 ms für `--version`. Das war ein Messfehler — die Schleife
maß über zwei `date`-Unterprozesse je Durchlauf und damit überwiegend sich
selbst. Sauber gemessen (`Measure-Command` über 20 Läufe, geteilt) sind es
8,1 ms. Die *Aussage* der Tabelle ändert sich dadurch nicht, die Zahlen schon.

| Messung | Ergebnis |
|---|---|
| `bin/loomux.exe` heute | 13,7 MB, `--version` 8,1 ms |
| dasselbe Binary mit gelinktem `go-sdk` v1.8.0 | 16,0 MB, `--version` 8,2 ms |
| SDK allein, gegen ein leeres `main` | +6,8 MB — davon fast alles `net/http` und `crypto/tls`, die loomux wegen `brain/search/http.go` längst linkt |
| alloc-reichste Paketinitialisierung mit SDK | `encoding/gob` 367–369 und `jsonschema-go` 298 **kommen erst mit dem SDK** (am 2026-09-17 gegen einen ungelinkten Bau gegengemessen), `go-sdk/mcp` 183 — alle unter der Grenze 500 aus `cmd/loomux/start_test.go:23` |
| `go`-Direktive der neuen Abhängigkeiten | `go-sdk` v1.8.0 und `x/time` v0.15.0 verlangen 1.25.0; `x/sys` **v0.48.0** und `x/text` **v0.42.0** verlangen **1.26.0** (`x/sys` v0.47.0 noch 1.25.0) |
| Job-Object-Probe dieses Prozessbaums | in keinem Job |

**Lesart:** der SDK kostet den Hook-Pfad nichts Messbares — 8,1 gegen 8,2 ms
liegt im Rauschen. Der Sprung von 6,8 MB in der Spielzeugmessung war zum größten
Teil Standardbibliothek, die schon drin liegt; am echten Binary bleiben 2,3 MB.
Das Startzeit-Tor hält ohne Nacharbeit, aber **enger als gedacht**: `encoding/gob`
und `jsonschema-go` zieht erst der SDK herein, und damit stehen zwei fremde
Pakete mit 369 und 298 unter der Grenze von 500, wo vorher keines stand. Das ist
ein Beobachtungspunkt für jede weitere Abhängigkeit.

Die Job-Object-Probe sagt über einen von einem MCP-Wirt gestarteten Prozessbaum
**nichts** — sie ist hier nur der Nachweis, dass die Abfrage funktioniert. Die
Annahme, dass ein Wirt uns in ein Job-Object stecken kann, bleibt bestehen und
wird im Verhalten aufgefangen, nicht durch Messung ausgeschlossen.

`x/sys` steigt aus zwei Gründen: der SDK verlangt mindestens v0.41.0, und die
Regel „immer die neueste Fassung" führt auf v0.48.0. `x/text` (v0.38.0 →
v0.42.0) zieht in einem eigenen Commit mit. `yaml.v3` und die gepatchte
TOML-Kopie sind bereits auf dem neuesten Stand.

**Die `go`-Direktive steigt damit von 1.25.0 auf 1.26.0** — nicht wegen des
SDK, der begnügt sich mit 1.25.0, sondern weil die neuesten `x/sys` und `x/text`
es verlangen. Der Preis: Mitbauer brauchen eine Toolchain ab 1.26; die hier
installierte ist 1.27.0. Wer auf 1.25 bleiben will, pinnt `x/sys` auf v0.47.0
und `x/text` entsprechend — dann gilt aber die Regel „neueste Fassung" nicht
mehr. `net/http.CrossOriginProtection` spielt dabei keine Rolle, die gibt es
seit Go 1.25.

## Prozesse, Adressen, Ein-Instanz-Disziplin

Drei langlebige Prozesse: `loomux serve`, je eine Brücke `loomux mcp` pro Wirt
(lebt so lange wie der Wirt), und `qmd mcp --http --daemon` aus 1b-1.

### Zwei Listener, zwei Token

`serve` bindet auf `127.0.0.1` zwei Ports, einen für `local` und einen für
`cloud`, beide über Port 0 vom Betriebssystem gewählt. Je Listener ein Token aus
32 Zufallsbytes (`crypto/rand`, base64url); jede Anfrage trägt ihn.

**Der Kanal ist die Adresse, kein Argument.** Die Referenz begründet das in
`ipc.py`: „ein Kanal, den ein Client nennen kann, ist eine Behauptung, also muss
das Tor die Adresse selbst sein". Ein Aufrufer mit dem cloud-Token kann
`local_only`-Bereiche nicht benennen, auch wenn die Konfiguration eines Wirts
falsch steht. Die Fusions-Spec sah den Kanal als Feld der Anfrage vor; diese
Spec verwirft das.

Die Code-Graph-Spec bestätigt die Form unabhängig (§4.2): zwei Server-Instanzen
mit geteiltem Datenbestand, eine je Profil. Das ist auch spec-konform — die
MCP-Revision 2026-07-28 verlangt, dass `tools/list` nicht pro Verbindung
variiert, erlaubt aber ausdrücklich eine Abhängigkeit von der Autorisierung.

### Zwei Sperren

Ein neues Paket `internal/lock`, Port von `locking.py`: `LockFileEx` unter
Windows, `flock` sonst.

- **`serve.lock`** hält `serve` über seine gesamte Laufzeit. Die
  Ein-Instanz-Garantie ist damit die des Betriebssystems; stirbt der Prozess,
  gibt das Betriebssystem sie frei. Eine PID steht in der Datei, aber nur zur
  Anzeige — „einen toten PID übernehmen" braucht es als eigenen Mechanismus
  nicht.
- **`qmd.lock`** ist ein kurzer kritischer Abschnitt um Probe und Start,
  geteilt von `brain search` und `serve`. Zwei Starter erzeugen nie zwei
  Daemons auf 8765.

`serve.json` unter `%LOCALAPPDATA%\loomux` schreibt nur `serve`, das
`serve.lock` ohnehin hält, atomar über temporäre Datei und Umbenennen. Unter
POSIX `0600`, das Verzeichnis `0700`. Inhalt: beide Adressen mit ihren Token,
die PID, und die Bauidentität (Pfad, Größe, mtime) aus `os.Executable()`.

Die Datei ist nie die Wahrheit, nur ein Hinweis: antwortet der Listener nicht
und ist `serve.lock` frei, gilt sie als verwaist und wird überschrieben.

### Die Brücke

`loomux mcp --channel local|cloud` spricht stdio zum Wirt; ohne das Flag gilt
`local`, wie schon bei `loomux brain`. Beim Start liest sie
`serve.json` und vergleicht die Bauidentität mit dem eigenen Programm.

**„Neuer gewinnt."** Sie startet `serve` nur neu, wenn ihr eigenes Programm
neuer ist als das vermerkte. `serve` ist maschinenweit, `bin/loomux.exe` liegt
pro Checkout, und es gibt mehrere Klone nebeneinander; eine Regel
„anders ⇒ neu starten" ließe zwei Wirte aus zwei Klonen einander bei jedem
Aufruf abschießen. Der Pfad in `serve.json` dient nur der Anzeige.

Der Start läuft **im Hintergrund**, nebenläufig zum `initialize` des Wirts: der
Wirt sieht sofort einen antwortenden Server, auch wenn `serve` und qmd zehn
Sekunden brauchen. Ein gescheiterter Start reißt die Brücke nicht mit — der Wirt
erfährt von der Störung durch den Aufruf, der `serve` braucht, nicht durch einen
Server, der verschwindet (Referenz, Spec 4.2).

Starten zwei Brücken gleichzeitig ein `serve`, rennen nicht sie um `serve.lock`,
sondern die beiden erzeugten Dienste: einer bekommt sie, der andere beendet sich
sofort wieder. Beide Brücken warten derweil im 250-ms-Takt bis 60 s auf ein
`serve.json` mit antwortendem Listener — derselbe Takt und dieselbe Frist wie der
qmd-Handschlag aus 1b-1.

**Der Neustart selbst ist eine Abfolge, kein Sprung.** Ein neu gestartetes
`serve` kann `serve.lock` nicht nehmen, solange das alte es hält. Die Brücke
beendet daher erst das alte: POST auf dessen Stop-Endpunkt mit dem local-Token
aus `serve.json`, dann warten, bis die Sperre frei ist (derselbe 250-ms-Takt,
dieselben 60 s), dann starten. Dafür liest die Brücke **beide** Token aus
`serve.json`, unabhängig von ihrem `--channel` — sonst könnte eine
cloud-Brücke nie neu starten. Das widerspricht dem Kanal-als-Adresse nicht: das
Tor ist die Adresse, die sie *anspricht*, nicht die Datei, die sie lesen darf;
wer `serve.json` liest, ist ohnehin der Nutzer selbst.

**Ein Neustart entwertet die Verbindungen aller anderen Brücken**, weil die
Ports über Port 0 neu vergeben werden. Antwortet ein weitergeleiteter Aufruf mit
„Verbindung abgelehnt", liest die Brücke `serve.json` neu, handelt gegen das
neue `serve` neu aus und wiederholt den Aufruf **genau einmal**; erst dann ein
MCP-Fehler, der `loomux serve status` nennt. Ohne diese Regel bräche jeder
Commit jeden anderen offenen Wirt, weil das Pre-Commit-Tor `bin/loomux.exe` neu
baut.

**`serve` erbt nie die stdio der Brücke.** Der stdout der Brücke ist die
MCP-Leitung des Wirts; ein geerbter Deskriptor zerstört das Framing und hält die
Leitung offen, wenn die Brücke endet. stdin, stdout und stderr zeigen auf NUL
bzw. in `%LOCALAPPDATA%\loomux\logs\serve.log`. `search/daemon.go:57` macht das
für qmd bereits richtig (`cmd.Stdout = nil` ergibt `os.DevNull`).

Gestartet wird entkoppelt: `DETACHED_PROCESS|CREATE_BREAKAWAY_FROM_JOB` unter
Windows, `Setsid` sonst. **Scheitert das Breakaway-Flag** — ein Wirt kann uns in
ein Job-Object ohne `BREAKAWAY_OK` gesteckt haben —, startet die Brücke ohne es,
merkt sich das in `serve.json`, und `loomux serve status` sagt: dieser Dienst
stirbt mit seinem Wirt.

### Lebensdauer

`serve` läuft bis `loomux serve stop` oder bis zur Abmeldung; keine
Leerlauf-Abschaltung. `serve stop` ist ein Endpunkt auf dem local-Listener mit
dem local-Token, kein Signal: die Listener fahren geordnet herunter, die Sperre
wird freigegeben. Antwortet der Endpunkt nicht, bricht `loomux serve stop
--force` den Prozess über die PID aus `serve.json` ab — nur auf dieses Flag hin,
nie von allein.

**`serve stop` beendet qmd nicht.** qmd gehört niemandem allein — `brain search`
startet ihn auch —, und sein Kaltstart kostet einen Modellladevorgang (in 1b-1
mit 5,7 s gemessen). Einem parallel laufenden `brain search` den Daemon unter
den Füßen wegzuziehen, wäre der teurere Fehler.

`loomux serve status` fragt die Listener, statt nur die Sperrdatei zu lesen.
`loomux serve --foreground` läuft im Terminal und schreibt nach stderr — der Weg,
auf dem ein Mensch einen Fehlstart überhaupt sehen kann.

## MCP-Oberfläche, Weiterleitung, Fehlerverhalten

### Die Werkzeuge

Fünf flache Werkzeuge: `brain_search`, `brain_catalog`, `brain_read`,
`brain_neighbors`, `brain_status`. Argumentformen unverändert aus 1b-1:

**Lesart der Tabelle:** „Pflicht" steht als `required` im Schema. Ein `=` nennt
den Wert, den `answer.Run` anwendet, wenn das Feld fehlt — **nicht** ein
`default` im JSON-Schema. Genau ein Feld trägt ein Schema-Default, `n`, und das
ist Parität zur Referenz: `daemon/tools.py:17` gibt `scope` nur `type` und
`description`, Zeile 52 gibt `profile` nur `type` und `enum`, Zeile 53 gibt `n`
sein `"default": 10`. Wer die anderen beiden ins Schema schreibt, bricht die
Parität.

| Werkzeug | Argumente |
|---|---|
| `brain_search` | `query` (Pflicht), `scope` → `all`, `profile` ∈ {`fast`, `full`, `keyword`} → `fast`, `n` = 10 (Schema-Default) |
| `brain_catalog` | `scope` → `all` |
| `brain_read` | `scope` und `relative` (beide Pflicht), `section` |
| `brain_neighbors` | `scope` und `relative` (beide Pflicht) |
| `brain_status` | keine |

Zwei Fallen darin, beide gegen den Code nachgerechnet:

- **Die Profile heißen `fast`, `full`, `keyword`** (`search/port.py:19-41`:
  `keyword` ist BM25, `fast` rein vektoriell, `full` die hybride Kette), und
  genauso stehen sie in `internal/cli/brainargs.go:111`. Das
  `fast|balanced|deep` aus dem Go-Prototyp `ultra-brain/pkg/mcp/tools.go` ist
  veraltet und wird nicht übernommen.
- **`n` ist über MCP 10, auf der Kommandozeile 5.** Das ist kein Versehen der
  Referenz: `daemon/tools.py` setzt 10, `cli.py:483` setzt 5, und 1b-1 hat die 5
  übernommen (`brainResultCount`). Die MCP-Front behält die 10 — hier zählt die
  Parität zur MCP-Front, nicht zur CLI. Die Paritätsliste führt die Abweichung
  ausdrücklich, damit sie niemand später „begradigt".

Die Schemata baut ein Paket auf ersten Gebrauch, nicht in `init()` und nicht in
einer Paketvariable.

**Zwei Zeilen der Fusions-Spec sind damit überholt** und bekommen dort einen
Verweis auf diese Spec, damit der nächste Leser nicht dem älteren Dokument
glaubt: „Werkzeuge wie heute: `search`, `catalog`, …" und „Den Kanal wählt die
Brücke per Flag; er ist Teil der Anfrage, keine eigene Bindung".

**Warum `brain_*` und nicht `search`.** Die MCP-Revision 2026-07-28 kennt keine
geschachtelten Werkzeuge: das `Tool`-Objekt hat `name`, `title`, `description`,
`icons`, `inputSchema`, `outputSchema`, `annotations` und sonst nichts, und die
Namen müssen innerhalb eines Servers eindeutig sein. Die Code-Graph-Spec §4
stellt in denselben Server sechs `graph_*`-Werkzeuge und später
Upstream-Proxies mit eigenem Präfix. Elf und mehr Werkzeuge in einem Server:
dort wäre ein blankes `read` neben `graph_file_api` mehrdeutig — lies was? Das
Präfix ist die einzige Familientrennung, die das Protokoll zulässt.

Die Kollision **zwischen** Servern löst laut Spec der Wirt, und alle drei
loomux-Wirte tun das nachweislich: Claude Code und Codex bilden
`mcp__<server>__<werkzeug>` (`codex-rs/codex-mcp/src/mcp/mod.rs:66-87`), das
Gemini CLI `mcp_<server>_<werkzeug>` mit Kürzung bei 64 Zeichen
(`generateValidName` im gebündelten Paket). Der längste Fall der elf ist
`mcp__loomux__graph_check_freshness` mit 34 Zeichen; Kürzung droht nicht.

Damit ist die Zeile der Fusions-Spec „Werkzeuge wie heute: `search`, `catalog`,
`read`, `neighbors`, `status`" überholt, und die Richtung der 1b-1-Spec
(`brain_*`) bestätigt.

**Kein Dispatcher-Werkzeug.** Gemessen an den echten Schemata kostet die flache
Liste aller elf Werkzeuge 2621 Byte (~655 Token) gegen 1478 Byte (~369 Token)
für zwei Dispatcher `brain`/`graph` — rund 286 Token, einmal je Sitzung und mit
`ttlMs`/`cacheScope` zusätzlich zwischenspeicherbar. Dagegen stehen drei Dinge,
die nicht verhandelbar sind: die Rechtevergabe der Wirte greift **pro
Werkzeugname** (Codex und Gemini schlagen beide mit `policy[toolName]` nach),
`required` je `op` lässt sich in einem flachen Schema nicht ausdrücken, und
Annotationen wie `readOnlyHint` gelten pro Werkzeug — spätestens mit `apply` und
`approve` in Stufe 3 könnte ein gemischter Dispatcher dem Wirt nichts Wahres
mehr über sich sagen. **Schwelle:** wächst der Gateway über etwa 30 Werkzeuge,
wird das neu gerechnet; der Umbau bleibt billig, weil die Namen schon
familienweise geschnitten sind.

**Stufenweises Aufdecken ist ausgeschlossen**, nicht aus Geschmack: die Spec
verlangt, dass die Werkzeugmenge nicht pro Verbindung und nicht als Nebenwirkung
anderer Anfragen variiert.

### Die Brücke ist ein Umleiter

Name auf Name, Argumente auf Argumente, Ergebnis zurück. Tut diese Schicht je
mehr, ist etwas falsch (Referenz, Spec 4.1).

- **`tools/list` beantwortet die Brücke selbst**, aus `internal/mcptools`, das
  sie mit `serve` teilt. Die Beschreibungen sind statisch, und ein Abruf legte
  einen qmd-Kaltstart mitten in den Handschlag. Beide Listen kommen aus
  demselben Paket und sind in fester Reihenfolge sortiert, damit sie
  byteidentisch sind — die Spec verlangt eine deterministische Reihenfolge, und
  eine Liste, die zwischen Brücke und Dienst driftet, wäre der schlimmste
  Fehler dieser Schicht. `ttlMs` und `cacheScope` werden gesetzt; fünf statische
  Werkzeuge kosten das Zwischenspeichern nichts.
- **Zwei unabhängige Aushandlungen.** Wirt↔Brücke und Brücke↔`serve` verhandeln
  je ihre Protokollrevision; keine Stelle im loomux-Code nennt eine
  Versionsnummer. Genau dafür ist der SDK da.
- **Der Progress-Token des Wirts wird nach unten weitergereicht**, sonst stirbt
  der Warm-Hinweis in der Brücke.

### `serve` ruft dieselbe Antwortfunktion wie die CLI

Die Werkzeuge in `serve` rufen `brainRun` mit dem Kanal des Listeners, über den
die Anfrage kam. Kein Argument `channel`, kein zweiter Pfad zu den Daten,
dieselbe Funktion wie `loomux brain search`.

**Dafür zieht `brainRun` aus `internal/cli` um** — nach `internal/brain/answer`,
zusammen mit `brainSearch`, `brainCatalog`, `brainArea`, `brainRead`,
`brainNeighbors` und `brainStatus` (heute `internal/cli/brain.go:79-201`). Sonst
gäbe es einen Importzyklus: `internal/cli` braucht `internal/serve` für den
Einstiegspunkt, und `internal/serve` bräuchte `internal/cli` für die Antwort. In
`internal/cli/brain.go` bleibt, was dort hingehört: Argumente lesen, Nutzungstext
verweigern, stdout und die Hinweiszeilen auf stderr schreiben.

**Der Umzug ändert die Signatur, er ist kein reines Verschieben.** Zwei Gründe:
`brainArgs` ist der Ergebnistyp des CLI-Parsers und bleibt in `internal/cli`, die
umgezogenen Funktionen nehmen ihre Werte einzeln entgegen; und `brainSearch`
nimmt heute ein `stderr io.Writer` (`internal/cli/brain.go:103`), das es nur
braucht, um den Warm-Hinweis hineinzuschreiben. Daraus wird ein
`notice func(string)`. Die Fälle aus 1b-1 laufen unverändert weiter und sind der
Nachweis, dass sich am Verhalten nichts ändert.

**Es sind zwei verschiedene Hinweiswege, nicht einer.** Gegen den Code
nachgerechnet:

- `notes` aus `brainRun` sind `answer.Findings` — die Befunde der Suche. Die CLI
  schreibt sie als `note: …` auf stderr (`internal/cli/brain.go:63`).
- **Der Warm-Hinweis ist kein Note.** `search/mcp.go:62` reicht über `WithNotice`
  einen Rückruf hinein, den `DefaultConnectWith` genau dann auslöst, wenn er den
  Daemon selbst starten musste — **während** des Verbindens, bevor die Suche
  zurückkommt. Die CLI bindet diesen Rückruf an stderr
  (`internal/cli/brain.go:104-106`). Für `serve` gibt es kein stderr, das ein
  Wirt liest, also bindet `serve` ihn an die Fortschrittsmeldung.

Beide Wege enden oben gleich: **Fortschrittsmeldung nach oben**, nie nach
stderr. Ohne Progress-Token des Wirts verfällt der Hinweis in der Sitzung, und
das ist richtig; eine Meldung ohne Empfänger ist keine (Referenz, Spec 4.4).

Der Warm-Hinweis kommt wegen `sync.Once` in `DefaultConnectWith` **einmal je
`ConnectFunc**, in einem langlebigen `serve` also einmal je Prozessleben statt
einmal je Aufruf. Das ist gewollt: er beschreibt einen Kaltstart, und den gibt es
je Daemon nur einmal.

Die Rückgaben werden getrennt gehalten:

- `text` → `CallToolResult` mit `TextContent`.
- `error` → hier trennt sich, was nicht zusammengehören darf (Referenz, Spec
  4.5): **was der Kern verweigert**, ist Inhalt fürs Modell — Ergebnis mit
  `isError: true` und der Begründung; **was der Transport verliert**, ist eine
  Störung — ein MCP-Fehler, der `loomux serve status` nennt. Eine Störung als
  leeres Ergebnis zu melden, brächte dem Modell bei, einen toten Dienst als
  „nichts gefunden" zu lesen. Nie eine leere Trefferliste, wo eine Leitung
  fehlt.

### Zustandslos, und was das für qmd heißt

Der Handler läuft **zustandslos** (`StreamableHTTPOptions.Stateless`): kein
`Mcp-Session-Id`, GET und DELETE antworten mit 405, je Anfrage eine temporäre
Sitzung. Das ist die Richtung der Revision 2026-07-28 (SEP-2567) und nimmt dem
Neustart seinen komplizierten Fall — eine „unbekannte Sitzung" gibt es nicht
mehr, nur eine abgelehnte Verbindung auf einem Port, den es nicht mehr gibt.
Fortschrittsmeldungen bleiben möglich, solange sie im Kontext einer laufenden
Anfrage entstehen, und genau so entstehen unsere.

Zustandslos ist der MCP-Draht, nicht `serve` selbst: seine qmd-Sitzung lebt
Stunden. Stirbt qmd darunter weg, gilt für diesen Sprung dieselbe Regel wie für
Brücke→`serve`: unter `qmd.lock` neu proben, bei Bedarf neu starten, den Aufruf
**genau einmal** wiederholen, sonst ein Fehler — nie eine leere Trefferliste.

### Der Token

`Authorization: Bearer <token>` des jeweiligen Listeners, geprüft von einer
Middleware **vor** dem SDK-Handler. Ein falscher oder fehlender Token endet mit
401, bevor irgendetwas MCP-Ähnliches passiert.

### Cross-Origin-Schutz

Der `StreamableHTTPHandler` bekommt eine `http.CrossOriginProtection`. Ohne sie
könnte eine beliebige Webseite im Browser des Nutzers gegen `127.0.0.1`
schießen; der Token allein hilft nicht, sobald er je in eine URL gerät.

### `serverInfo` ist kein Namensraum

Die Spec sagt ausdrücklich, dass `serverInfo.name` nicht zur Unterscheidung
taugt. Sichtbar wird der Schlüssel aus der `.mcp.json` des Wirts — `loomux` —,
und den setzt der Mensch bzw. ab Stufe 4 `loomux init`, nicht dieser Dienst.

## Pakete

| Paket | Inhalt |
|---|---|
| `internal/serve` | Listener, Lebenszyklus, `serve.json`, `status`, `stop` |
| `internal/serve/brain` | die fünf Werkzeuge, ruft `internal/brain/answer` |
| `internal/brain/answer` | `brainRun` und seine sechs Helfer, aus `internal/cli` umgezogen |
| `internal/bridge` | der Umleiter |
| `internal/mcptools` | die geteilte Werkzeugliste |
| `internal/lock` | Port von `locking.py` |
| `internal/cli` | `serve.go` und `mcp.go` als reine Einstiegspunkte |

**`hooks` importiert nie `serve` oder `bridge`.** Dafür ein Test, der den
Importgraphen liest — ein Satz in einer Spec hält diese Grenze nicht.

## Tests und Paritätsnachweis

**Go-Tests.** Brücke↔`serve` über `mcp.NewInMemoryTransports` — kein Socket,
kein Port, keine Wartezeit. Prozessstart, Uhr und Launcher werden injiziert wie
in `search/daemon.go`. Coverage 100 % je Funktion, jeder Ausschluss mit
`//coverage:exempt <grund>`.

Ausschließlich in Go geprüft, weil ein langlebiger Prozess kein `stdout` hat,
das man vergleicht: zwei Listener und ihre Trennung, Abweisung eines falschen
Tokens, beide Sperren, „neuer gewinnt", die Ein-Wiederholung nach einem
Neustart, der Fehlschlag des Breakaway-Flags.

**Fallkorpus.** Eine neue Familie `testdata/cases/1b-2`: `cmd` ist
`loomux mcp --channel local|cloud`, `stdin` sind JSON-RPC-Zeilen. Verglichen
wird **der Textinhalt der `CallToolResult` und `isError`**, nicht der Umschlag.

Drei Dinge, die das Gerüst dafür braucht:

1. **Isolation über das Zustandsverzeichnis.** `serve.json`, `serve.lock` und
   `qmd.lock` liegen unter `config.StateDir()`, nicht unter einem fest
   verdrahteten `%LOCALAPPDATA%\loomux`. Ein Fall setzt `LOOMUX_STATE_DIR` und
   bekommt damit sein eigenes `serve` — dieselbe Idee wie der Adress-Digest aus
   `ipc.py`, der die Testsitzung vom echten Daemon trennt.
2. **Jeder Fall räumt sein `serve` weg**, sonst lässt ein `go test`-Lauf
   entkoppelte Prozesse liegen. Das Gerüst ruft am Fallende `serve stop` auf dem
   Zustandsverzeichnis des Falls.
3. **Ein neuer Vergleichsmodus** in `internal/cases` und `dev recordcase`: heute
   wird `stdout` byteweise verglichen, hier `result.content[].text` und
   `isError` aus den Antwortzeilen. Das ist eine Erweiterung des Rekorders, kein
   Schalter am vorhandenen Vergleich.

Der Grund für den Modus: die Python-Front spricht über das Python-MCP-SDK, die Brücke über das
Go-SDK. `initialize` liefert schon deshalb andere Bytes — Fähigkeiten,
`serverInfo`, ausgehandelte Revision —, und die Werkzeuge heißen ohnehin anders.
Ein byteweiser Vergleich der Umschläge erzeugte eine Paritätsliste, die
überwiegend aus „Unterschied der Bibliothek" bestünde und bei jeder
SDK-Aktualisierung neu wüchse. Der Umschlag wird gegen die MCP-Spec geprüft,
nicht gegen Python.

Die Abbildungsdatei `1b-2-map.toml` trägt die Umbenennung `search` →
`brain_search` und die vier übrigen; sie ist eine Übersetzungsregel, keine
Abweichung, und steht als solche in der Paritätsliste.

**Mutationsrunde** der Stufe über `loomux dev mutants`, Überlebende dokumentiert
wie in `parity/stufe-1b-1-geparkte-mutanten.md`.

## Messungen nach dem Bau

Chronologisch in `docs/en/benchmarks.md` und `docs/de/benchmarks.md`, je kalt
und warm, Grundlinie gegen Änderung:

1. Binärgröße und Startboden vor und nach dem SDK (Vormessung siehe oben).
2. Der Hook-Pfad, unverändert nachgewiesen — dass `hooks` das SDK nicht
   anfasst, muss messbar bleiben, nicht nur strukturell stimmen.
3. Handschlag der Brücke: Zeit vom Start bis zur Antwort auf `initialize`, mit
   laufendem und mit kaltem `serve`.
4. Ein `brain_search` über die Brücke gegen dasselbe `loomux brain search`
   direkt — der Aufpreis der zwei Sprünge.

## Mensch-Schritte

- Die `.mcp.json`-Einträge der Wirte auf `loomux mcp --channel local` zeigen
  lassen; `loomux init` übernimmt das erst in Stufe 4.
- Freigabe der Paritätsliste.
- Rauchtest: ein `brain_search` aus Claude Code über die Brücke, ein zweiter
  Wirt parallel, ein Commit dazwischen — er zeigt, ob „neuer gewinnt" und die
  Ein-Wiederholung tragen.

## Nicht in 1b-2

- **Upkeep** und `reconcile` — Stufe 3. Der Haken bleibt in `serve` als benannte
  Leerstelle; die 1b-1-Spec führte Upkeep hier, die Stufentabelle der
  Fusions-Spec führt „Upkeep in `serve`" unter Stufe 3, und ohne `reconcile`
  gibt es nichts nachzuholen.
- `graph_*` und die Upstream-Kaskade — Code-Graph G3.
- `/api/…` und die eingebettete Web-App — Folgeprojekt Web-Migration.
- `loomux init` für die MCP-Einträge und die Umstellung der Wirte — Stufe 4.
- Wiki- und Doku-Umzug — 1b-3.
