# Weboberfläche, Schritt 2: Code- und Wiki-Graph mit Verbindungen

Stand 2026-11-15, Zweig `feat/web-graph-view`, abgezweigt bei cf88beaa von
`fix/ui-followups` (Schritt 1 der Weboberfläche). Vorgänger: `2026-11-13-web-oberflaeche-einstellungen-design.md`.
Recherche: `docs/.superpowers/research/2026-11-14-graphify-graphansicht.md`.

## Anlass

Schritt 2 des Vorhabens „OS für das LLM“: eine Ansicht, die Code-Graph und
Wiki-Graph eines Projekts nur lesend zeigt, zuerst in 2D und später in 3D.
Der Nutzer will mit einer **Übersicht** einsteigen und die **Verbindungen
zwischen Wiki und Code** sehen. Heute verweisen Wiki und Code nur in
Fließtext aufeinander, gespeichert ist keiner dieser Verweise.

Gemessen am 2026-11-15 auf dem Haupt-Checkout, mit der Regel aus Abschnitt 1
(Backtick-Span, kein Codeblock, Pfad existiert), aber noch ohne die
Bedingung „mindestens ein `/`“:

| Ordner | Dokumente | Verweise | verschiedene Ziele | ohne `/` | Verzeichnisse |
|---|---|---|---|---|---|
| `docs/wiki` | 24 | 131 | 59 | 13 | 32 |
| `docs/de` | 13 | 353 | 142 | 138 | 114 |
| `docs/.superpowers` | 64 | 1 690 | 494 | 381 | 176 |

Die Treffer ohne `/` sind fast nie Code: `README.md`, `AGENTS.md`,
`.gitignore`, `go.mod`, `docs`, `.`. Darum verlangt die Regel einen `/`.
Mit dieser Bedingung bleiben im Wiki 130 Verweise auf 57 Ziele. Davon stehen
39 Verweise auf 32 Ziele im Code-Graphen und werden Kanten zum Code. Der
Rest nennt vor allem Dokumente (Kanten zwischen Dokumenten) und Dateien ohne
Extraktor, etwa unter `ci/` oder `.toml` (Infofeld).

Annahme, nicht bestätigt: Die Lesesperre, die laut Schritt 1 direkt danach
kommen sollte, stellt der Nutzer zugunsten dieses Schritts zurück.

## Ziel und Erfolg

- Unter `loomux ui` öffnet der Mensch für ein Projekt eine Graphseite. Sie
  zeigt die Code-Übersicht nach Verzeichnissen, die Dokumente des
  Wiki-Bereichs und Kanten von einem Dokument zu dem Code, den es nennt.
- Er klappt ein Verzeichnis auf bis zu Dateien und Symbolen, sucht einen
  Knoten und filtert nach Relation und Konfidenz.
- Der Wiki-Indexer erkennt Codeverweise und schreibt sie in `graph.json`.
  Damit liegen sie auch für Agenten bereit; die Abfrage über MCP ist der
  nächste Schritt (siehe Nicht-Ziele).
- Erfolg heißt: Der Code-Graph von loomux (13 987 Knoten, 47 837 Kanten,
  16 MB `wiring.json`) öffnet als lesbare Übersicht. Jeder Pfad mit `/`, den eine
  Seite in Backticks nennt und den es im Code-Graphen gibt, erscheint als
  Kante. Einer, der nur im Repo existiert, steht im Infofeld der Seite.

## Nicht-Ziele

- 3D. Die Wahl der Bibliothek hält den Weg offen (Abschnitt 5).
- MCP-Abfrage „welche Seiten beschreiben diesen Code“ und ein Hinweis
  „Seite zitiert geänderten Code“. Beide bauen auf dem neuen Feld auf und
  sind eigene Schritte.
- Ein Frontmatter-Feld `code:` (Ansatz B). Es kann später als zweite Quelle
  derselben Kanten dazukommen.
- Verweise aus Code in das Wiki (Kommentare mit `docs/wiki/…`).
- Communities per Clustering-Algorithmus, kürzester Weg zwischen zwei
  Knoten, Bericht und Fragenvorschläge wie bei graphify.
- Bearbeiten von Graph oder Wiki aus der Ansicht.
- Wiki und Code in verschiedenen Bereichen (etwa `project-iam-wiki` neben
  `project-iam-backend`). Die Existenzprüfung läuft gegen die Wurzel des
  eigenen Bereichs; solche Wikis bekommen keine Verweise.
- Verweise, die ein Ding nur beim Namen nennen („der Wächter“, `ReadPolicy`)
  statt bei seinem Pfad.

## Anti-Ziele

Gescheitert, auch wenn es läuft, ist eine Lösung, wenn

- die Seite den ganzen Code-Graphen auf einmal an den Browser schickt oder
  zeichnet,
- die Insel ihre Daten aus dem gerenderten HTML liest statt über einen
  JSON-Endpunkt (Regel aus Schritt 1),
- Logik, die Tests braucht (Zusammenfassen, Auflösen, Filtern der
  Datenmenge), in JavaScript liegt statt in Go,
- eine Bibliothek zur Laufzeit aus dem Netz geladen wird,
- ein alter `graph.json` ohne das neue Feld die Ansicht oder einen
  bestehenden Leser bricht.

## Ansätze

**Woher die Verbindungen kommen.**

- **A: Beim Anzeigen aus dem Text lesen.** Billig, aber nur der Mensch im
  Browser hätte die Verbindungen.
- **B: Frontmatter-Feld `code:`.** Präzise und prüfbar, aber heute auf keiner
  Seite gesetzt; das Bild bliebe leer.
- **C: Der Indexer erkennt die Pfade und schreibt sie in `graph.json`
  (gewählt).** Erkannt wird wie bei A. Gespeichert liegen die Verbindungen
  auch für Agenten bereit, die so beantworten können, welche Seiten einen
  geänderten Code beschreiben. Das stützt die Regel „Code implementiert,
  Wikiseiten im selben Task nachziehen“.

Der Nutzer hat C gewählt, nachdem er nach dem höchsten Mehrwert gefragt hat.

**Zeichenbibliothek.** force-graph (Canvas, 2D) ist gewählt, weil
3d-force-graph dieselbe Schnittstelle hat und der Schritt zu 3D damit ein
Tausch der Bibliothek ist. sigma (WebGL) wäre schneller, hat aber keinen Weg
zu 3D. vis-network, das graphify nutzt, ebenfalls nicht. Für die
zusammengefasste Übersicht reicht Canvas. Das ist meine Festlegung; der
Nutzer hat sie nicht widerrufen.

## Entscheidungen

Vom Nutzer entschieden: Übersicht als Einstieg; Verbindungen zwischen Wiki
und Code; Ansatz C; Verweise für alle Dokumente speichern, aber
standardmäßig nur das Wiki zeigen. Alles Weitere sind meine Festlegungen.

### 1. Codeverweise im Indexer

- `internal/brain/index` reicht dem Renderer neben Titel, Tags und Links den
  Text jedes Dokuments; `graph.Document` bekommt dafür ein Feld.
- Ein Verweis ist ein Inline-Code-Span (Backticks), dessen Inhalt die Form
  `pfad`, `pfad:zeile` oder `pfad:von-bis` hat. Codeblöcke (```` ``` ````)
  zählen nicht mit: Darin stehen Beispiele, keine Verweise. Markdown-Links auf
  Codedateien (`[x](../../internal/a.go)`) zählen mit. Sie werden relativ
  zum Ordner des Dokuments aufgelöst, wie in `ResolveTarget`; Backtick-Pfade
  dagegen relativ zur Wurzel des Bereichs.
- Gilt nur, wenn der Pfad relativ zur Wurzel des Bereichs (`Area.Path`)
  existiert, als Datei oder Verzeichnis, und nicht unter `.git` oder
  `.loomux` liegt. Ein Pfad, der aus der Wurzel herausführt, gilt nicht. So
  bleiben Prosa wie `a/b` und Beispielpfade draußen.
- Ein Verweis enthält mindestens einen `/`. Einzelne Namen wie `go.mod`,
  `docs` oder `internal` zählen nicht (Messung oben).
- Die Existenz wird mit exakter Schreibweise geprüft, Segment für Segment
  über die Einträge des Verzeichnisses, nicht über `os.Stat`. `os.Stat` ist
  unter Windows unabhängig von Groß- und Kleinschreibung, die Pfade des
  Code-Graphen sind es nicht, und `graph.json` soll auf jedem System gleich
  aussehen.
- `graph.json` hängt damit vom Dateibaum ab, nicht nur von den Dokumenten.
  Das ist in Ordnung: `reindex` liest jedes Dokument bei jedem Lauf neu
  (`internal/brain/index/reindex.go`, `readDocFn` über alle Dateien). Ein
  Plan, der eine Datei nennt, bevor es sie gibt, bekommt den Verweis also
  beim ersten `reindex`, nachdem die Datei angelegt ist. Kein Test und kein
  Tor verlangt, dass `graph.json` allein aus den Dokumenten entsteht oder mit
  dem alten Python-Renderer übereinstimmt; `read_python_test.go` prüft nur
  den Leser.

  > Korrektur bei der Umsetzung: Der letzte Satz war falsch.
  > `internal/cli/cases_3a_test.go` und `cases_3b_test.go` spielen
  > Aufzeichnungen des Referenzsystems ab und vergleichen dabei auch
  > `graph.json`. Der Hilfsdekoder `decoded` in `cases_3a_test.go` wirft darum
  > ein leeres `code_refs` vor dem Vergleich weg: Ein leeres Feld sagt nichts,
  > ein gefülltes bleibt stehen und lässt den Vergleich scheitern.
- `graph.json` bekommt ein neues Feld, getrennt von `edges`, denn `edges`
  sind Links zwischen Dokumenten:

  ```json
  "code_refs": [
    {"from": "topics/x.md", "path": "internal/hooks/guard.go", "line": 67, "end": 0}
  ]
  ```

  `line` und `end` sind 0, wo der Verweis keine Zeile nennt. Ist `end` 0,
  meint ein Verweis mit Zeile genau diese eine Zeile. Doppelte Verweise einer
  Seite fallen zusammen; die Reihenfolge ist sortiert, damit der Indexer
  deterministisch bleibt.
- Der Leser (`graph.ReadGraph`, `ParseGraph`) nimmt ein fehlendes Feld als
  leer hin. Bestehende Leser und `brain_neighbors` ändern ihr Verhalten
  nicht.
- Auf ein Symbol löst der Indexer nicht auf: Der Code ändert sich öfter als
  das Wiki. Das tut die Ansicht mit dem aktuellen Code-Graphen (Abschnitt 3).
- Ein schreibgeschützter Bereich oder einer ohne Codepfade bekommt ein leeres
  Feld. Ein Bereich, der kein Repo ist, ebenfalls.

  > Korrektur bei der Umsetzung: Der zweite Satz war falsch. Der Indexer
  > prüft nicht, ob ein Bereich ein Repository ist: `ReadDocument` behält
  > jeden Kandidaten, dessen Pfad unter der Wurzel des Bereichs existiert.
  > Auch ein Bereich, der kein Repo ist, bekommt also Verweise.
- Bekannte Grenzen, in der Doku zu nennen: Eine Zeilenangabe wird gegen den
  heutigen Code aufgelöst. Ist sie veraltet, zeigt sie ohne Warnung auf ein
  anderes Symbol. Und eine Datei, die nach dem letzten `reindex` angelegt
  wurde, bekommt ihre Verweise erst mit dem nächsten; eine reine
  Codeänderung löst kein `reindex` aus.

### 2. Seite und Endpunkt

- Neue Seite `GET /graph/{id}` für ein Projekt der Projektliste aus Schritt 1,
  mit einem Link aus der Kopfzeile und von der Einstellungsseite. Das
  Dashboard bekommt keine neue Kachel; die Kachel „Index state“ verlinkt auf
  die Graphseite.
- Die Seite ist eine Vorlage mit einer Insel. Die Insel holt ihre Daten über
  `GET /graph/{id}/data?open=<verzeichnis>&open=…` als JSON. Gleiche
  Sitzungsprüfung und `Host`-Prüfung wie alle Routen aus Schritt 1; der
  Endpunkt ist nur lesend.
- Quelle des Code-Graphen: `store.Read(root)`. Quelle des Wiki-Graphen:
  `graph.ReadGraph` für den registrierten Bereich des Projekts. Fehlt eines
  von beiden, zeigt die Seite das andere und sagt, was fehlt und welcher
  Befehl es baut (`loomux graph build`, `loomux reindex`). Ein Projekt, das
  kein registrierter Bereich ist, hat keinen Wiki-Graphen.
- `wiring.json` ist groß. Der Server hält den gelesenen Graphen im Speicher
  und liest neu, wenn sich die Änderungszeit der Datei ändert.

### 3. Was der Endpunkt liefert

Der Endpunkt fasst in Go zusammen; die Insel zeichnet nur.

- **Code, zusammengefasst nach Verzeichnis.** Ein Code-Knoten gehört über
  seinen `Path` zu einem Verzeichnis. Die Übersicht zeigt Verzeichnisse als
  Knoten. Ein Verzeichnis in `open` zeigt stattdessen seine Unterverzeichnisse
  und Dateien; eine geöffnete Datei zeigt ihre Symbole.
- **Startebene.** Damit nicht `internal/` als einziger Knoten erscheint,
  klappt der Endpunkt Verzeichnisse von der Wurzel aus auf, solange die
  Übersicht unter einer Obergrenze an Knoten bleibt. Die Zahl wählt der Bauer
  (Größenordnung 100 bis 200).
- **Kanten.** Eine Kante zwischen zwei sichtbaren Knoten zählt die
  Code-Kanten zwischen ihren Mitgliedern, getrennt nach Relation und
  Konfidenz, damit die Insel filtern kann, ohne neu zu laden. `contains`
  zählt nicht als Kante zwischen Gruppen. Kanten innerhalb einer Gruppe
  entfallen. Ziele, die kein Knoten sind (`npm:lodash`), entfallen ebenfalls.
- **Größe** eines Knotens: Summe des PageRank seiner Mitglieder aus
  `internal/code/pagerank`. `Rank` ist personalisiert und gibt ohne Saat
  nichts zurück; die Ansicht sät gleichmäßig über alle Knoten (meine
  Festlegung).
- **Wiki.** Jedes Dokument des Bereichs ist ein eigener Knoten, gefärbt nach
  seinem obersten Ordner. Die Links zwischen Dokumenten sind Kanten.
- **Welche Dokumente (vom Nutzer entschieden).** Der Indexer schreibt
  `code_refs` für alle Dokumente des Bereichs. Die Ansicht zeigt
  standardmäßig nur die Dokumente unter dem Wiki-Ordner des Bereichs
  (`WikiPath`, im Indexer `OwnWikiPrefix`; bei loomux `docs/wiki`), die
  übrigen hinter dem Schalter „weitere Dokumente“. Grund: Die Pläne unter
  `docs/.superpowers` haben mehr als zehnmal so viele Verweise wie das Wiki
  (Messung oben) und würden es im Bild erdrücken. Ein Bereich ohne
  Wiki-Ordner zeigt alle Dokumente.
- **Brücken.** Jeder `code_ref` wird zu einer Kante vom Dokument zum
  sichtbaren Knoten, der den Pfad enthält: ein geschlossenes Verzeichnis,
  eine Datei oder, wenn die Datei offen ist und der Verweis eine Zeile nennt,
  das Symbol, das diese Zeile umschließt (`model.FileSpans`,
  `model.Enclosing`). Nennt ein Verweis ein Verzeichnis, das gerade offen ist, geht
  je eine Kante zu jedem seiner sichtbaren Kinder, die Code enthalten (meine
  Festlegung). Nennt er ein Dokument desselben Bereichs, wird er eine Kante
  zwischen Dokumenten. Jeder andere Verweis, dessen Pfad im Code-Graphen
  fehlt (`.toml`, `.ps1`, eine Sprache ohne Extraktor, eine gelöschte
  Datei), steht im Infofeld des Dokuments als „nicht im Code-Graphen“ und
  ergibt keine Kante.
- **Suche.** `GET /graph/{id}/search?q=` liefert passende Symbole, Dateien
  und Dokumente samt dem `open`, unter dem sie sichtbar werden. Die Insel lädt
  dann diese Ansicht und hebt den Treffer hervor.

### 4. Die Insel

- Zeichenfläche mit Kraftlayout. Sobald es sich beruhigt hat, steht es still.
- Seitenleiste: Suche; Infofeld zum gewählten Knoten (Art, Pfad, Zahl der
  Mitglieder, Nachbarn mit Zählern, bei Dokumenten ihre Codeverweise); Legende
  der Farben mit Haken zum Ausblenden.
- Filter: Haken je Relation und Konfidenz; Wiki ein und aus; „weitere
  Dokumente“ ein und aus (aus als Vorgabe).
- Doppelklick auf ein Verzeichnis oder eine Datei öffnet sie. Ein Pfad über
  der Fläche führt zurück. Der Zustand (`open`, Filter) steht im
  Fragment-Teil der Adresse, damit Neuladen und Zurück funktionieren.
- Ein Klick auf ein Dokument kann es nicht öffnen: Die Oberfläche hat noch
  keinen Wiki-Leser. Das Infofeld zeigt den Pfad.
- Das JavaScript bleibt so dünn, dass ein Fehler darin sichtbar wird statt
  still falsch zu rechnen. Es ruft den Endpunkt, setzt Daten in die
  Bibliothek und schaltet Sichtbarkeit.

### 5. Bibliothek und Ablage

- force-graph in der neuesten Version, exakt gepinnt (Stand 2026-11-14:
  1.52.0, zuletzt geändert am 2026-09-28). Der Bauer prüft Version und
  Pflegestand neu.
- Lizenzen: `internal/notices/NOTICE.md` listet die Lizenz jedes Fremdteils
  im Binary. Erzeugt wird die Datei von `internal/dev/notices` aus
  `go list`, den Grammatiken und dem Zipf-Hinweis, und ein Test prüft, dass
  sie aktuell ist. Das Bündel von force-graph bringt seine eigenen
  Abhängigkeiten mit (d3-Module und weitere). Der Erzeuger bekommt deshalb
  einen Eintrag für das Bündel mit allen darin enthaltenen Lizenzen.
- Das Repo hat kein Muster für eingebettetes JavaScript. Das einzige
  Fremdteil im Quelltext ist `third_party/toml` (Go, Version in
  `AGENTS.md`), und `go:embed` erreicht keine Elternverzeichnisse. Die Datei liegt deshalb
  unter `internal/web/assets/vendor/`, ihre Lizenz daneben. Version und
  SHA-256 stehen in `AGENTS.md` neben der Version von toml, und ein Go-Test
  prüft die Prüfsumme der eingebetteten Datei. Aktualisiert wird von Hand
  (meine Festlegung).
- Die Content-Security-Policy aus Schritt 1 (`middleware.go`,
  `default-src 'self'`) bleibt unverändert. Geprüft am Bündel 1.52.0
  (`force-graph.min.js`, 173 KB): Es ruft kein `eval`. Es enthält einmal
  `Function("return this")()`, die globale Erkennung von lodash. Sie wird im
  Browser nicht erreicht, weil vorher `self` greift. Zweimal fügt es ein
  `<style>`-Element ein, das die CSP blockt. Diese Regeln kommen deshalb in
  das eigene Stylesheet, und das geblockte Einfügen bleibt folgenlos (meine
  Festlegung). Weder `script-src` noch `style-src` wird gelockert. Der Bauer
  prüft das an der gepinnten Version erneut.
- Code der Ansicht im Paket der Oberfläche aus Schritt 1 (`internal/web`).
  Zusammenfassen und Auflösen in einer eigenen Datei oder einem Unterpaket,
  ohne HTTP, damit es für sich getestet wird.
- Tests über `net/http/httptest` und direkte Tests der Zusammenfassung; 100 %
  Coverage je Funktion. Das JavaScript fällt nicht unter die Go-Coverage.
  Geprüft wird es durch einen Test, der die eingebetteten Dateien ausliefert
  und die Seite samt Endpunkt einmal gegen den echten Code-Graphen des Repos
  oder eine Testdatei abruft. Einen Browsertest gibt es nicht; das ist meine
  Festlegung für Schritt 2.

## Doku und Roadmap im selben Pull Request

- `README.md` und `README.de.md`: Funktion „Graphansicht“; die Zeile
  „Graph-Visualizer“ der Roadmap behält nur 3D.
- `docs/en/` und `docs/de/`: Oberfläche (`cli-reference.md` oder die Seite der
  Weboberfläche), das neue Feld `code_refs` dort, wo `graph.json`
  beschrieben ist, und die Regel, was als Codeverweis zählt (Backticks, Pfad
  existiert, keine Codeblöcke). Wer Wikiseiten schreibt, muss diese Regel
  kennen.
- Die betroffenen Wikiseiten unter `docs/wiki/`.

## Dem Bauer überlassen

- Routen im Detail, Aufbau von Vorlage und JSON, Aussehen, Farben.
- Obergrenze der Übersicht, Zeitpunkt des Stillstands im Layout.
- Eine Obergrenze für Dokumentknoten. Ein Bereich ohne Wiki-Ordner, der alle
  Dokumente zeigt, ist sonst unbegrenzt.
- Wie der Graph im Speicher gehalten wird.
- Die Reihenfolge der Scheiben. Naheliegend: `code_refs` im Indexer und Leser;
  Zusammenfassen in Go; Endpunkt; Seite mit Insel; Suche; Doku.
