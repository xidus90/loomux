# Paritätsliste Stufe 1b-2 — die MCP-Front

Aufgenommen am 2026-09-18 gegen `brain-mcp mcp` aus dem Referenz-Worktree
`C:/Users/micro/Documents/#GIT/ultra-brain/.claude/worktrees/youthful-jones-ec0e73`,
HEAD `3cc72d23f8701a2c174b88a67e58d668d2654e65` — derselbe Stand, den das Tag
`loomux-1a-source` bezeichnet und gegen den Stufe 1b-1 abgenommen wurde.

Der Korpus liegt unter `testdata/cases/1b-2-source/` (Aufnahme),
`testdata/cases/1b-2/` (übersetzt) und `testdata/cases/1b-2-worlds/` (Welten);
die Suite ist `internal/cli/cases_1b2_test.go` mit 54 Fällen.

---

## 1. Was verglichen wird, und was nicht

**Verglichen wird der Textinhalt der `CallToolResult` und `isError`, nicht der
Umschlag.** Die Referenz spricht über das Python-MCP-SDK, loomux über das
Go-SDK: `initialize` liefert schon deshalb andere `capabilities`, ein anderes
`serverInfo` und eine anders ausgehandelte Revision. Ein byteweiser Vergleich
erzeugte eine Liste, die überwiegend aus „Unterschied der Bibliothek" bestünde
und bei jeder SDK-Aktualisierung neu wüchse. Der Umschlag wird gegen die
Spezifikation geprüft, in den Unittests von `internal/serve` und
`internal/bridge`.

**Kein `world_after`.** Was ein Lauf im Zustandsverzeichnis hinterlässt, gehört
dem Dämon — seine Sperre, seine PID-Datei, seine Abgleichsrunde —, nicht dem
Werkzeug, dessen Antwort der Fall festhält.

**Zwei Noten**, wie in 1b-1: `text` (Text **und** `isError`) und `outcome`
(`isError` allein; der Wortlaut einer Ablehnung ist loomux' eigener). 15 der 54
Fälle tragen `outcome`, jeder mit einer Zeile in Abschnitt 4.

---

## 2. Was gar nicht aufgenommen werden konnte — mit Beleg

### 2.1 `search` mit Treffern: nicht hermetisch aufnehmbar

`brain.search.qmd_mcp` legt den Port der Engine fest:

```python
DEFAULT_PORT = 8765                      # qmd_mcp.py:40
session = _HttpSession(DEFAULT_PORT)     # qmd_mcp.py:238
```

Es gibt keinen Überschreibungsweg — weder Umgebungsvariable noch Argument —,
und `cli._serve_forever` baut den Port ohne `connect`-Naht
(`warm=partial(QmdMcpPort, backbone=...)`). Am 2026-09-18 lauschte auf diesem
Rechner ein echter qmd-Dämon auf `::1:8765` (PID 57268, gemessen mit
`Get-NetTCPConnection -LocalPort 8765`). Jede Suche der Referenz ging dorthin,
also an den Index des Nutzers und nicht an die Welt des Falls. Stünde dort
nichts, startete `_start_daemon` das Fake mit `qmd mcp --http --daemon`, das es
nicht kennt, und die Antwort wäre `SearchUnavailable`. In beiden Fällen hängt
die Aufnahme an der Maschine, nicht an der Welt — **eine Aufnahme wäre keine
Evidenz.**

Aufgenommen sind daher nur die fünf `search`-Fälle, die **vor** dem Zugriff auf
die Engine scheitern oder ohne ihn antworten: `missing-query`,
`invalid-profile`, `unknown-scope`, `missing-registry` (alle vier vor
`port.search`) und `cloud-without-visible-areas` (`core.search` gibt ohne
sichtbare Area sofort `SearchAnswer((), ())` zurück, ohne zu fragen).

Nicht aufgenommen, weil sie die Engine erreichen: `fast`, `keyword`,
`full-hits`, `full-cut-at-n`, `full-one-scope`, `full-no-matches`,
`full-cloud`, `engine-fails`, `broken-register` (in `core.search` liegt
`_ask` **vor** `_assemble`) und `count-below-one` (`core.search` prüft `n`
nicht; die Prüfung ist argparse' Sache).

### 2.2 Die Trefferdarstellung von `search` weicht ab — Entscheidung des Menschen

Aus demselben Grund ist eine Abweichung offen, die **kein Fall dieses Korpus
beweisen kann**:

| | Referenz (`daemon/tools.py:172-181`) | loomux (`internal/brain/search.FormatSearch`) |
|---|---|---|
| Trefferzeile | `{scope}/{relative}:{line}  {title}` | `brain://{scope}/{relative}:{line}  {score}%  {title}` |
| Auszugszeile | `    {snippet}`, eine Zeile, eingerückt | `    {zeile}` je Zeile des Auszugs |
| Trennung | die Treffer mit einem Zeilenumbruch verbunden | nach jedem Treffer eine Leerzeile |

(Die Auszugszeile fehlte in der ersten Fassung dieser Liste; nachgetragen.)

**Entschieden am 2026-09-18:** das Trefferformat **bleibt**, wie es ist — es ist
das von 1b-1 abgenommene, und der Plan verlangt ausdrücklich „dieselbe
Antwortfunktion wie die CLI". Ein zweiter Renderer für die MCP-Front wäre eine
zweite Baustelle für dieselbe Sache. Es ist damit eine **aufgezeichnete
Abweichung** und keine offene Frage mehr.

**Was dagegen behoben wurde, weil es Informationsverlust war und keine
Darstellungsfrage:** die Befunde. Siehe §3.5.

**Aufnehmbar ist das Trefferformat aus zwei Gründen nicht**, und beide müssen
wegfallen, ehe ein Fall es prüfen kann:

1. `qmd_mcp.py:40` legt `DEFAULT_PORT = 8765` ohne Überschreibungsweg fest, und
   auf der Maschine lauscht ein echtes qmd darauf (§2.1).
2. **`internal/dev/fakeqmd` kennt `qmd mcp --http --daemon` nicht.** Selbst auf
   einer Maschine ohne echtes qmd würde `_start_daemon` das Fake mit diesem
   Unterbefehl starten, es antwortete mit `fakeqmd: unknown subcommand "mcp"`
   und Exit 2, und `wait_until_reachable` liefe in `SearchUnavailable`. Ein
   HTTP-MCP-Modus im Fake ist die Vorbedingung, nicht bloß ein freier Port.

### 2.3 Der lokale Kanal

`brain.mcp` setzt `CHANNEL = Channel.CLOUD` fest; die Front der Referenz kennt
keinen anderen Kanal, weil ihre Adresse ihr Kanal ist. Jeder Fall ist deshalb
ein Cloud-Fall, und die Wiedergabe ruft `loomux mcp --channel cloud`. Der lokale
Kanal ist über die Referenzfront **bauartbedingt unerreichbar**; er ist durch
die 71 Kommandozeilenfälle von 1b-1 und durch die Unittests von
`internal/serve` gedeckt.

### 2.4 `status-stamp-stale` und `status-stamp-missing`

`daemon/server.py::Upkeep._catch_up` läuft **vor der ersten Antwort**, sobald
der Stempel fehlt oder älter als 24 h ist, und schreibt ihn dabei neu. Der
Gegenstand beider Fälle wird also vom Dämon zerstört, ehe eine Antwort
entsteht; die Antwort trüge zudem die aktuelle Uhrzeit und wäre nie zweimal
gleich.

Deshalb trägt **jede** Welt unter `1b-2-worlds/` einen Stempel auf
`2999-01-01T00:00:00+00:00`: dann ist nichts fällig, es läuft kein Abgleich,
die Welt bleibt unverändert und der Anhang aus `_trailer` bleibt leer. Was der
Abgleich der Referenz selbst tut, ist keine Parität mit loomux — `loomux serve`
hat gar keinen Upkeep — und gehört in eine eigene Stufe.

---

## 3. Befunde am Go-Code

Alle betreffen `internal/serve/brain/tools.go`, also die MCP-Oberfläche, und
alle sind **in dieser Aufgabe behoben** worden: es sind keine Entwurfsfragen,
sondern Lücken gegenüber dem, was auf der Kommandozeile der Argumentparser
erledigt und was die Referenz in `daemon/tools.py` tut. §3.1 bis §3.4 hat der
Korpus rot gestellt; §3.2a und §3.5 hat der Vergleich der Quelltexte gefunden,
den erst eine Durchsicht angestellt hat.

### 3.1 `scope` hatte keinen Vorgabewert

`brain_catalog` ohne Argumente — der Wurzelkatalog, der gewöhnlichste Aufruf
dieses Servers — fragte nach der Area mit dem leeren Namen und wurde mit
`unknown scope ''` abgelehnt. Die Kommandozeile bekommt `all` von ihrem Parser
(`brainParserFor`), ein Werkzeugaufruf hat keinen. Die Referenz:
`arguments.get("scope", "all")` (`daemon/tools.py::_dispatch`).
**Behoben:** `scope()` in `tools.go`.

### 3.2 Ein fehlendes Pflichtargument wurde beantwortet statt abgelehnt

`brain_search` ohne `query` suchte nach der leeren Zeichenkette und antwortete
`no matches` — die eine Antwort, die ein Modell als „nichts gefunden" auf eine
nie gestellte Frage liest. Nichts auf diesem Weg prüft einen Aufruf gegen sein
Schema. Die Referenz lehnt ab, indem sie `arguments["query"]` liest und den
`KeyError` zum Fehlerergebnis werden lässt (`daemon/tools.py::call`).
**Behoben:** `refuse()` in `tools.go`, für `query` (search) und `relative`
(read, neighbors).

Eine kleine, bewusste Abweichung bleibt darin: `refuse` liest ein Argument, das
als leere Zeichenkette ankommt (`{"query": ""}`), als nicht vorhanden; die
Referenz findet es vorhanden und sucht danach. Kein aufgezeichneter Fall
erreicht das — es ist hier notiert, damit es niemand für einen Fund hält —, und
loomux' Lesart ist die vertretbare: eine Suche nach nichts ist keine Frage, und
`no matches` darauf ist die eine Antwort, die ein Modell missversteht.

### 3.2a `profile` hatte keinen Vorgabewert — und die erste Fassung dieser Liste log darüber

**Diese Liste behauptete hier zunächst, ein fehlendes `profile` sei harmlos,
weil „`answer.Run` seinen eigenen Vorgabewert kennt" und „beide bei `fast`
landen". Beide Hälften waren falsch, und die Behauptung stand damit als
Nicht-Abweichung in genau dem Dokument, an dem die Stufe abgenommen wird.**
Nachgerechnet statt gelesen, so wie es hier immer hätte sein müssen:

- `answer.search` reicht `brainsearch.Profile(req.Profile)` unverändert an den
  Port weiter (`internal/brain/answer/answer.go`). Dort ist kein Vorgabewert.
- Die leere Zeichenkette ist keines der drei bekannten Profile und fällt in den
  `default`-Zweig **beider** Ports: über MCP nach `args["query"]` mit
  `rerank: true` (`internal/brain/search/mcp.go`), auf der Kommandozeile auf den
  Unterbefehl `query` (`internal/brain/search/qmd.go`). Das ist in beiden Fällen
  **`full`**.

`brain_search` ohne `profile` fuhr also eine Rerank-Suche, wo die Referenz
`fast` fährt (`daemon/tools.py`: `arguments.get("profile", Profile.FAST.value)`)
und wo loomux' **eigene Kommandozeile** ebenfalls `fast` fährt
(`internal/cli/brainargs.go`: `fallback: string(search.ProfileFast)`) — das
teuerste der drei Profile, still, auf der Front, die ein Modell am meisten
benutzt. Dieselbe Sorte Lücke wie das fehlende `scope` aus §3.1.

**Behoben:** `profile()` in `tools.go` setzt `fast` ein, wie `scope()` `all`
einsetzt.

**Kein aufgezeichneter Fall kann das prüfen, und keiner wird es können,
solange §2.1 gilt.** Die Unterscheidung zwischen `fast` und `full` ist erst an
der Antwort der Engine sichtbar; ein Fall, der sie zeigte, müsste die Engine
erreichen, und genau das ist nicht hermetisch aufnehmbar. Die fünf
aufgezeichneten `search`-Fälle nennen ihr Profil entweder ausdrücklich
(`full`) oder scheitern vor dem Port, weshalb diese Korrektur **keinen** Fall
bewegt hat. Festgehalten ist sie durch
`TestASearchWithoutAProfileRunsTheCheapOne` in
`internal/serve/brain/tools_test.go`.

### 3.3 Ein unbekanntes Profil wurde durchgereicht

`brain_search` mit `profile: "deep"` suchte trotzdem. Das Schema zählt
`fast, full, keyword` auf, geprüft hat es niemand. Die Referenz scheitert an
`Profile(str(...))`. **Behoben:** ebenfalls in `refuse()`.

### 3.4 Der Zeilenabschluss von stdout stand im Werkzeugergebnis

`status`, `neighbors` und `search` werden von einem Formatierer gebaut, der die
letzte Zeile abschließt, weil stdout dort eine Zeile will. Ein
`CallToolResult` ist kein Zeilenstrom: die Referenz baut dieselben drei durch
Verbinden (`render_neighbors`, `_search`, `"\n".join(core.status(...))`), ihr
Text endet mit dem letzten Zeichen der letzten Zeile. **Behoben:** `forMCP()`
in `tools.go` entfernt genau einen Abschluss, und nur bei diesen dreien —
`read` und `catalog` geben ein Dokument samt seinem eigenen Zeilenende zurück,
auf beiden Fronten.

### 3.5 Die Befunde einer Suche standen nur in den Fortschrittsmeldungen

Ein Befund entwertet eine Antwort, ohne sie ungültig zu machen — der Index ist
alt, die Engine antwortete zweimal leer — und ist die einzige Warnung, dass die
Treffer darunter weniger wert sind, als sie aussehen. Die Referenz schreibt ihn
in den Text, je eine Zeile `! {finding}` unter die Treffer
(`daemon/tools.py:181`).

loomux schickte ihn ausschließlich als Fortschrittsmeldung — und eine
Fortschrittsmeldung **ohne Fortschritts-Token des Wirts hat keinen Empfänger**:
`report()` verwirft sie, was `TestWithoutAProgressTokenTheHintsLapse` festhält.
Ein Wirt, der kein Token schickt — und nichts verpflichtet ihn dazu —, verlor
damit eine Warnung, die die Referenz immer zeigt. **Das ist kein Unterschied in
der Darstellung, das ist eine verlorene Warnung.**

**Behoben:** `withFindings()` in `tools.go` hängt die Befunde einer `search` an
den Text, wie die Referenz. Die Fortschrittsmeldung bleibt zusätzlich: ein Wirt
mit Token sieht den Befund dadurch früh statt erst mit der Antwort, was
**besser** ist als die Referenz, und ein Wirt ohne Token bekommt ihn trotzdem.
Nur `search` — bei den anderen vier sind die Notizen keine Befunde über ihre
eigene Antwort, und was sie dort bedeuteten, rät diese Funktion nicht.

Kein aufgezeichneter Fall deckt das ab: die einzige aufnehmbare Suche
(`cloud-without-visible-areas`) hat keine Befunde. Festgehalten ist es durch
drei Unittests in `internal/serve/brain/tools_test.go`.

Ohne 3.1 bis 3.4 waren 35 der 54 Fälle rot; danach 15, und die 15 sind
Wortlaut (Abschnitt 4). 3.5 hat kein Fall rot gestellt — es kam aus dem
Vergleich des Referenzquelltextes mit dem unsrigen.

### 3.6 Was auf Lektüre ruht und nicht auf einem Fall

Die Hälfte von `refuse`, die `brain_read` und `brain_neighbors` ohne
`relative` ablehnt, ist **durch keinen aufgezeichneten Fall gedeckt**: der
Korpus hat nur `brain-search/missing-query`. Dass die Referenz sich dort
genauso verhält, ist aus `daemon/tools.py::_dispatch` gelesen
(`str(arguments["relative"])` in beiden Zweigen), nicht gemessen. Wer das
absichern will, nimmt zwei Fälle mehr auf; die Unittests
(`TestACallWithoutItsOneRequiredArgumentIsRefused`) halten loomux' Seite fest.

---

## 4. Die 15 Fälle mit der Note `outcome`

Bei allen 15 stimmt `isError` überein, der Text nicht. Zwei Ursachen, beide
zulässig.

**Was loomux dabei sagt, ist nicht ungeprüft.** Eine `outcome`-Note vergleicht
einen einzigen Wahrheitswert, und `brain-read/missing-section` wäre grün, wenn
loomux „die Registry brennt" antwortete. `loomuxWording` in
`internal/cli/cases_1b2_test.go` hält deshalb für jeden der 15 Fälle einen
Teilstring der loomux-Meldung fest — die Zeilen unten, in Code —, und
`TestEveryOutcomeCaseHasItsWordingPinned` schlägt fehl, sobald ein
`outcome`-Fall keine solche Zeile hat oder eine Zeile keinen Fall mehr.

**(a) loomux formuliert eigene Meldungen.** Das ist seit 1a die Regel: der
Wortlaut einer Meldung ist loomux' eigener, und in 1b-1 wurde er gar nicht
verglichen, weil er auf stderr ging. Über MCP **ist** der Wortlaut der Inhalt,
also ist er hier sichtbar — und bleibt trotzdem loomux'.

| Fall | Referenz | loomux |
|---|---|---|
| `brain-catalog/missing-index` | `[Errno 2] No such file or directory: '…/index.md'` | `open …\index.md: <Meldung des Betriebssystems>` |
| `brain-catalog/missing-registry` | `[Errno 2] No such file or directory: '…/registry.toml'` | `open …\registry.toml: <Meldung des Betriebssystems>` |
| `brain-catalog/missing-manifest` | `[Errno 2] … '…/.brain.toml'` | `…/repo-bare: no manifest found (…)` |
| `brain-read/missing-file` | `[Errno 2] No such file or directory: '…/gone.md'` | `open …\gone.md: <Meldung des Betriebssystems>` |
| `brain-read/missing-manifest` | wie oben | wie oben |
| `brain-search/missing-registry` | `[Errno 2] … '…/registry.toml'` | `open …\registry.toml: …` |
| `brain-search/missing-query` | `'query'` (der `KeyError`) | `query is required` |
| `brain-search/invalid-profile` | siehe (b) | `invalid profile "deep"; choose from fast, full, keyword` |

Die Meldung des Betriebssystems ist zudem **sprachabhängig** (auf diesem
Rechner deutsch), was einen Textvergleich ohnehin ausschlösse.

**(b) Die Referenz lässt die Ausnahme aus `tools.call` entkommen, loomux nicht.**
`daemon/tools.py::call` fängt `(AccessDenied, SearchUnavailable, GraphError,
OSError, KeyError)`. Ein `RegistryError`, ein `ManifestError` und der
`ValueError` aus `Profile(...)` stehen nicht darin: sie reißen die Verbindung
ab, und die Front meldet daraufhin einen Ausfall — auf diesem Rechner mit dem
Satz `the brain daemon is running but does not serve cloud; it was started with
--no-cloud`, weil `_why_no_answer` den lebenden lokalen Kanal findet und daraus
den falschen Schluss zieht.

| Fall | Referenz sagt | loomux sagt |
|---|---|---|
| `brain-catalog/broken-registry` | Ausfallsatz | `…/registry.toml: not valid TOML: …` |
| `brain-catalog/manifest-without-scope` | Ausfallsatz | `…/config.toml: [area] scope must be a non-empty string, found ""` |
| `brain-catalog/manifest-wrong-type` | Ausfallsatz | `… found integer` |
| `brain-neighbors/manifest-without-scope` | Ausfallsatz | wie oben |
| `brain-read/missing-section` | Ausfallsatz | `no section titled 'Nowhere'` |
| `brain-search/invalid-profile` | Ausfallsatz | `invalid profile "deep"; …` |
| `brain-status/broken-register` | Ausfallsatz | `…\_identities.tsv: line 2: expected 4 tab-separated fields, found 3` |
| `brain-status/missing-manifest` | Ausfallsatz | `…: no manifest found (…)` |

**loomux ist hier besser, und das ist Absicht:** `internal/serve/brain`
verwandelt jede Ablehnung des Kerns in ein Fehlerergebnis, weil ein Modell auf
„unbekannte Area" handeln kann, während ein Protokollfehler ihm diese Chance
nimmt. Die Referenz verliert bei diesen acht Aufrufen die Sitzung. **Kein
Rückbau:** die Note `outcome` hält fest, dass beide Seiten `isError` melden,
und diese Zeilen halten fest, wodurch.

---

## 5. Die Umbenennung der Werkzeuge

`testdata/cases/1b-2-map.toml` benennt die fünf Werkzeuge um: `search` →
`brain_search` und die vier übrigen ebenso. Das Protokoll kennt keine
verschachtelten Werkzeuge, und der Codegraph wird `graph_*` in denselben Server
legen — das Präfix ist die einzige Familientrennung, die es gibt. **Es ist eine
Übersetzungsregel und keine Abweichung:** an der Antwort ändert der Name
nichts, und `internal/dev/importcases.ImportMCP` schreibt nichts anderes um.

---

## 6. Zwei Dinge, die keine Abweichung sind

- **Die Reihenfolge von `tools/list`.** Das Go-SDK sortiert alphabetisch
  (`go-sdk@v1.8.0/mcp/features.go:76-103`), die Referenz listet in
  Registrierungsreihenfolge. Ein Vergleich dagegen wäre ein Vergleich gegen das
  SDK, nicht gegen uns; `tools/list` ist deshalb kein Fall dieses Korpus.
- **`n` ist über MCP 10 und auf der Kommandozeile 5.** Das ist gewollte Parität
  mit `daemon/tools.py` gegen `cli.py`, kein Fehler.

  **Nachtrag vom 2026-09-18:** Die Referenz trägt die Zehn an **zwei** Stellen,
  und die frühere Fassung dieses Punktes kannte nur die erste. Im Schema
  (`daemon/tools.py:53`, `"default": 10`) und noch einmal im Code
  (`daemon/tools.py:167`, `n=int(str(arguments.get("n", 10)))`). Die zweite ist
  die wirksame: auf diesem Pfad prüft niemand ein Schema und setzt darum auch
  niemand ein `default` ein — beide Registrierungen gehen über die untypisierte
  `AddTool`-Form, und `applySchema` wird nur aus `mcp.AddTool[In,Out]` erreicht.
  loomux hatte nur die erste übernommen; `count()` gab ohne `n` 0 zurück,
  `search.ExecuteSearch` schnitt damit jeden Treffer weg, und `brain_search`
  ohne `n` antwortete `no matches`. Seit dem 2026-09-18 trägt loomux beide:
  das Schema in `internal/mcptools/tools.go`, den Code in
  `internal/serve/brain/tools.go::count` (`mcpResultCount`). Ein ausdrückliches
  `n: 0` bleibt unverändert durchgereicht — das Verhalten der Referenz dort ist
  nicht aufgenommen, siehe `count-below-one` in §2.1.

---

## 7. Eine Eigenheit der Aufnahme, die jeder Nachaufnehmer braucht

Der Rekorder startet den Dämon der Referenz als **`daemon run`** — im
Vordergrund, als Kind des Rekorders — und nicht als `daemon start`.

`daemon start` führt über `client._start_outside_job`, das den Dämon unter
Windows durch WMI (`Win32_Process.Create`) erzeugt. Ein so erzeugter Prozess
bekommt die Standardumgebung des Nutzers, nicht unsere: weder das Verzeichnis,
das die Aufnahme vor `PATH` hängt, noch `LOOMUX_FAKE_QMD_FIXTURE` kommen an.
Der Dämon findet dann das echte qmd der Maschine, und **jede Antwort, die die
Engine fragt, ist der Index des Nutzers statt die Vorrichtung der Welt.**

Gemessen am 2026-09-18: mit `daemon start` antwortete
`brain-status/backlog` mit `project/a: the search engine did not answer (qmd
exited with 1: Collection not found: project-a …)`, mit `daemon run` mit
`7 documents are indexed but not yet searchable; run \`brain embed\`` — genau
dem, was die 1b-1-Aufnahme derselben Welt zeigt. Alle 15 `status`-Fälle hingen
daran.

---

## 8. Die Zeilen zur Freigabe

Nachgetragen am 2026-09-18 mit Task 14. Die Abschnitte oben sind der Beleg; die
Tabelle ist das, was der Mensch abzeichnet. „Alt" ist die Python-Referenz
`brain-mcp mcp`, „Neu" ist `loomux serve` samt `loomux mcp`.

| Fall / Bereich | Alt | Neu | Begründung | Freigabe |
|---|---|---|---|---|
| Werkzeugnamen | `search`, `catalog`, `read`, `neighbors`, `status` | `brain_search`, `brain_catalog`, `brain_read`, `brain_neighbors`, `brain_status` | §5: das Protokoll kennt keine verschachtelten Werkzeuge, und `graph_*` kommt in denselben Server; Übersetzungsregel in `testdata/cases/1b-2-map.toml`, an der Antwort ändert der Name nichts | **offen** |
| `n` über MCP gegen `n` auf der Kommandozeile | `daemon/tools.py`: 10 zweimal — Schema (`:53`) und Code (`:167`); `cli.py`: 5 | ebenso: Schema in `internal/mcptools/tools.go`, Code in `serve/brain/tools.go::count`, und 5 auf der Kommandozeile | §6: gewollte Parität mit der Referenz, kein Fehler. Die wirksame Vorgabe ist die im Code — auf dem Werkzeugpfad setzt nichts ein Schema-`default` ein. Korrigiert am 2026-09-18, nachdem loomux nur die Schemahälfte trug und `brain_search` ohne `n` `no matches` antwortete | **offen** |
| Vergleichsmodus des Korpus | byteweise über den ganzen Umschlag | Textinhalt der `CallToolResult` und `isError`; zwei Noten, `text` und `outcome` | §1: die Fronten sprechen zwei SDKs, ein Umschlagvergleich bestünde überwiegend aus Bibliotheksunterschieden und wüchse mit jeder SDK-Aktualisierung neu. Der Umschlag wird gegen die Spezifikation geprüft, in den Unittests von `internal/serve` und `internal/bridge` | **offen** |
| Trefferdarstellung von `search` | `{scope}/{relative}:{line}  {title}` | `brain://{scope}/{relative}:{line}  {score}%  {title}`, Auszug je Zeile, Leerzeile je Treffer | §2.2: das von 1b-1 abgenommene Format bleibt; ein zweiter Renderer für die MCP-Front wäre eine zweite Baustelle. Durch keinen Fall dieses Korpus prüfbar | **offen** |
| Der lokale Kanal | kennt nur `cloud` | zwei Listener, zwei Token; der Kanal ist die Adresse | §2.3: über die Referenzfront bauartbedingt unerreichbar; gedeckt durch die 71 Kommandozeilenfälle von 1b-1 und die Unittests von `internal/serve` | **offen** |
| Die 15 Fälle mit der Note `outcome` | Meldungen der Referenz bzw. Ausfallsatz nach abgerissener Verbindung | eigene Meldungen, jede Ablehnung des Kerns als Fehlerergebnis statt als Protokollfehler | §4: `isError` stimmt bei allen 15; der Wortlaut ist loomux' eigener und in `loomuxWording` festgenagelt. Bei acht Fällen verliert die Referenz die Sitzung, loomux nicht — kein Rückbau | **offen** |
| Sechs Lücken am Go-Code, in dieser Stufe behoben | — | Vorgabewerte für `scope` und `profile`, Ablehnung fehlender Pflichtargumente und unbekannter Profile, Zeilenabschluss aus dem Werkzeugergebnis, Befunde im Text | §3.1 bis §3.5: Lücken gegenüber dem, was auf der Kommandozeile der Argumentparser erledigt. §3.2a, §3.5 und die Hälfte von §3.6 ruhen auf Lektüre und Unittests, nicht auf einem Fall | **offen** |
| Der Vorgabe-Spawner der Brücke (`bridge.Options.Spawn == nil` → der echte, entkoppelte Start) | — | Entscheidung des Menschen: die Brücke startet den Dienst selbst, statt sich einen Spawner reichen zu lassen | Im Betrieb richtig und hier nicht in Frage gestellt. **Der Preis ist im Test fällig:** ein Mutant auf `connect.go:122` lässt das Testbinary sich selbst als entkoppelten Dienst starten, der die Suite erneut fährt — eine Gabelbombe, 8.783 Waisen am 2026-09-18, drei abgebrochene Mutationsläufe. **Vorschlag, keine Änderung dieser Aufgabe:** ein `TestMain` in `internal/bridge`, das sofort endet, wenn `os.Args` das Element `serve` enthält. Das Kind stirbt beim Start, der Elternlauf scheitert weiterhin (sein Spawner wurde nicht gerufen), der Mutant wird getötet, und die 21 fehlenden Mutanten sind mit dem heutigen Werkzeug erreichbar — **ohne** den zweiten Bauplatz, den die Entscheidung gerade vermeiden wollte. Belege in [`stufe-1b-2-geparkte-mutanten.md`](stufe-1b-2-geparkte-mutanten.md) | **offen** |

## 9. Was zu dieser Stufe sonst noch gehört

- **Die vier Messungen** stehen im Eintrag *2026-09-18 15:30* in
  `docs/en/benchmarks.md` und `docs/de/benchmarks.md`: Größe und Startboden vor
  und nach dem SDK, der Hook-Pfad vorher gegen nachher, der Handschlag mit
  laufendem und kaltem Dienst, und ein `brain_search` über die Brücke gegen die
  Kommandozeile.
- **Die Mutationsrunde** und ihre Überlebenden stehen in
  [`stufe-1b-2-geparkte-mutanten.md`](stufe-1b-2-geparkte-mutanten.md).
  **Sie ist nicht vollständig, und das Fertigkriterium 3 der Stufe ist damit
  nicht erfüllt.** Fünf der sechs Pakete sind ganz gefahren; über
  `internal/bridge` sind es 116 der 137 Mutanten plus die Familie a2 als Ganzes.
  Die 21 fehlenden sind benannt, und der Weg zu ihnen steht als eigene Zeile in
  Abschnitt 8. **Zwei brückeneigene Punkte bleiben damit offen:** (a) ob die 21
  nachgefahren werden, und (b) ob der Spiegelfall
  `TestTheLocalBridgeCallsTheLocalAddress` nachgerüstet wird — er tötet den
  einzigen Überlebenden, der die Kanaltrennung berührt (`bridge.go:47`, immer
  die cloud-Adresse).
- **Die Messvorrichtung ist nicht ortsfest.** `testdata/bench/1b-2-hooks.json`
  und `testdata/bench/edit-readme-1b-2.json` tragen die absoluten Pfade des
  Worktrees, in dem gemessen wurde, wie es `1a-hooks.json` seit Stufe 1a tut.
  Beide Benchmark-Einträge sagen das jetzt und nennen den Bauweg für das
  `vorher`-Binary; wer die Messung anderswo wiederholt, pfadet vorher um.
- **Der Rauchtest ist ein Mensch-Schritt** und steht noch aus: ein
  `brain_search` aus Claude Code über die Brücke, ein zweiter Wirt parallel, ein
  Commit dazwischen. Erst er zeigt, ob „neuer gewinnt" und die Eine-Wiederholung
  im Betrieb tragen; kein Test dieser Stufe misst zwei echte Wirte.
- **Die `.mcp.json` der Wirte** muss ein Mensch auf `loomux mcp --channel local`
  zeigen lassen. `loomux init` übernimmt das erst in Stufe 4.
