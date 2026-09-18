# loomux G2 — Extraktor, Frische, Abfrage

**Datum:** 2026-09-17
**Stand:** **G2a umgesetzt und nach `master` gemerged am 2026-09-18** (Plan
`2026-09-17-loomux-code-g2a.md`, Paritätsakte `parity/code-g2a.md`, 27 Commits).
Damit stehen Schema 2, `sourceset`, `extract/golang`, `resolve`, `store`,
`freshness` und die Befehle `loomux graph build` und `loomux graph check`.
**G2b — die Abfrage (`lexicon`, `ask`, die Beiakte) — ist geplant und offen**
(Plan `2026-09-17-loomux-code-g2b.md`). Die Paritätsakte wartet auf die
Freigabe ihrer 21 Verfügungen.
**Ergänzt:** [`2026-09-14-loomux-code-graph-design.md`](2026-09-14-loomux-code-graph-design.md) und
[`2026-09-16-loomux-code-g1-delta.md`](2026-09-16-loomux-code-g1-delta.md). Diese Datei ersetzt die
Säule-3-Spec nicht, sie berichtigt und verengt sie für die Stufe G2.
**Referenz:** `trailhq/Graft`, Commit `1e352a3` vom 2026-09-16, MIT. Jede Zeilenangabe unten ist
gegen diesen Commit gelesen, nicht aus der Säule-3-Spec übernommen.

## 1. Warum es diese Spec gibt

G1 hat drei reine Rechenpakete gebaut: `internal/code/model` liest das Schema,
`internal/code/pagerank` rangiert, `internal/code/blast` läuft Kanten. Keines davon hat einen
Erzeuger und keines einen Aufrufer. G2 liefert beides: den Extraktor, der einen Graphen schreibt,
und die Befehle, die ihn bauen, prüfen und befragen.

Die Vorgabe für diese Stufe lautet: **wo Graft eine Antwort hat, gilt Grafts Antwort.** Das ist
keine Stilfrage. Beim Abgleich haben sich sechs Behauptungen als falsch erwiesen, und zwei davon
waren Empfehlungen dieses Entwurfs selbst. Abschnitt 3 führt jede mit Belegstelle.

Graft liegt nicht als Arbeitskopie vor; gelesen wurde über `raw.githubusercontent.com` am
festgenagelten Commit.

## 2. Zuschnitt: zwei Pläne, eine Spec

Graft-konform sind es sieben Pakete, ein Schemabruch und drei Befehle. Das ist zu viel für einen
Plan, und die Bruchkante liegt sauber: was Zustand **schreibt**, gegen was ihn **liest**.

### G2a — der Graph entsteht

| Paket | Gegenstand |
|---|---|
| `internal/code/model` (Änderung) | Schema 2: `body_hash`, `body_text`; drei neue Regeln in `Validate` |
| `internal/code/extract/golang` | Go-Definitionen, Signatur, Span, `body_hash`, ID-Prägung, Rohkanten |
| `internal/code/resolve` | Rohkanten → Kanten, Konfidenz, Go-Importe gegen `go.mod` |
| `internal/code/store` | `wiring.json` sortiert, zeitstempelfrei, atomar |
| `internal/code/freshness` | Sonde über `[größe, mtime, hash]`, ohne Extraktor-Import |
| `internal/cli/graph.go` | `loomux graph build`, `loomux graph check` |

**Abnahme:** ein Golden-`wiring.json` über einen kleinen Go-Baum in `testdata`; `build` zweimal
hintereinander auf unverändertem Baum erzeugt byteidentische Dateien; `check` gibt 0 auf frischem
Stand, 1 nach einer inhaltlichen Änderung und **0 nach einem reinen `touch`**.

### G2b — der Graph antwortet

| Paket | Gegenstand |
|---|---|
| `internal/code/lexicon` | `Tokenize`/`Counts`, Beiakte `ask-index.json` |
| `internal/code/ask` | Lexik → Saat → PageRank → Verschmelzung, Ausgabe |
| `internal/cli/graph.go` (Änderung) | `loomux graph ask` mit `--source`, `--full`, `--in`, `--limit`, `--json` |

**Abnahme:** die portierten Vektoren aus Abschnitt 12.3 plus eine echte Abfrage auf loomux selbst.

G2b kommt **hinter** G2a und nicht daneben: `ask` baut bei Drift selbst neu (Abschnitt 10.5) und
hängt damit an Extraktor und Schreiber.

### Was in keinem der beiden Pläne steht

`--lsp` gegen `gopls` (Abschnitt 14), die Pro-Datei-Karten (`FileCard`), der Crux, jede weitere
`graph`-Unterform aus §7 der Säule-3-Spec (`callers`, `blast`, `grep`, `skeleton`, `map`, `stats`,
`viz`), die MCP-Anbindung, die Hook-Anbindung, die `check`-Lanes `graph-freshness` und
`blast-audit`, und Grafts ganze Abfrage-Oberschicht: Multi-Scope-Föderation, File-First-Projektion,
`fileTopLock`, `fuse.ts`, die Abdeckungskennzahlen.

## 3. Berichtigungen

### 3.1 Der Crux ist LLM-gewählt, nicht AST-extrahiert

§2 der Säule-3-Spec führt „Crux-Extraktion — Signatur + Kernlogik-Spans (5–10 Zeilen)
(`src/ask/ask.ts`)" mit der Entscheidung **„Übernehmen: AST-gestützte Span-Extraktion"** und
verbucht sie unter „$0, kein LLM". §4.1 baut darauf das `full`-Flag als „ganzes Definitions-Span
statt ≤ 8 Zeilen Crux-Exzerpt".

Das ist falsch. Der Crux entsteht in Grafts Tier 2, `src/graph/enrich.ts`, aus einem LLM-Aufruf je
Datei; der Kopf der Datei nennt ihn „the LLM meaning layer (`summary` + `crux`)", die gespeicherte
Kappung ist `MAX_CRUX_LINES = 12`, und `ask.ts:1217` schreibt wörtlich „the ≤8-line LLM-chosen
crux". `inlineSource` (`ask.ts:1271–1292`) liest ihn aus dem Graphen über `n.crux?.code`; **fehlt
er, ist der Rückfall `sliceSpan`** — ein reiner Zeilenschnitt des Spans, gekappt bei
`MAX_SPAN_LINES = 80` (`ask.ts:99`, `1249–1268`), mit Restmarker.

**Es gilt:** loomux hat keinen Crux, weil es keinen LLM im Pfad hat. Es hat Grafts eigenen
Rückfall. Das Feld wird nicht erfunden; kommt später ein Erzeuger, greift genau `inlineSource`s
Vorrang, und die Flaggen bleiben.

### 3.2 Quelltext ist eine Flagge, nicht der Standard

`graft ask` gibt ohne `--source` nur Ort und Rang aus (`cli.ts:597`), und `--full` ist ausdrücklich
„with `--source`" (`cli.ts:598`).

**Es gilt:** ebenso. `loomux graph ask` ist ein Ortsangeber; `--source` macht ihn zum Abrufer.

### 3.3 Die Saat ist IDF plus BM25, nicht Substring

Ein früherer Stand dieses Entwurfs hat „Substring-Score auf Symbolnamen, kein BM25" empfohlen. Das
war eine Erfindung. Graft rechnet auf Name und Pfad `Σ qₙ·dₙ·idf` mit `idf = log(1 + n/(1+df))`
(`ask.ts:191–235`) und BM25 auf dem Rumpffeld mit `k1 = 1,2`, `b = 0,75` (`ask.ts:251–272`), und
schreibt `df`, `docCount` und `avgBodyLen` beim Bauen in eine Beiakte (`ask/index-file.ts`).

**Es gilt:** Grafts Rechnung, und die Beiakte mit ihr.

### 3.4 Namensauflösung erzeugt weniger Kanten, nicht mehr

Ein früherer Stand dieses Entwurfs hat gegen den rein syntaktischen Extraktor eingewandt, ein
Methodenaufruf treffe dann „alle gleichnamigen Methoden statt der richtigen — mehr Kanten, keine
falsche Richtung", und das als Überschätzung auf der sicheren Seite verkauft.

Das Gegenteil trifft zu. `resolve.ts` (Kopf) verwirft mehrdeutige dateiübergreifende Treffer statt
zu raten und verlangt für Memberaufrufe einen Empfängertyp **und** einen ownerqualifizierten
Methodentreffer, „because a unique bare method name says nothing about the receiver". Der Dateikopf
von `lsp/enrich.ts` nennt das Namensraten als das, „that halved precision". Und `resolve.ts:38–56`
dokumentiert den Schaden, den Eindeutigkeit anrichtet, wenn sie Sprachgrenzen überschreitet: ein
Go-`make(map[…])` fand einen TypeScript-Helfer `make` in einer Testdatei, ein Symbol sammelte 1040
eingehende Kanten über 476 Dateien, und jeder Pull Request auf diese Testdatei zog das ganze
Go-Backend in seinen Blast-Radius.

**Es gilt:** Mehrdeutig heißt verworfen. Für einen reinen Go-Extraktor ist die
Sprachfamilien-Sperre gegenstandslos, die Regel dahinter nicht.

### 3.5 Die Präzisionsstufe ist LSP, nicht ein Compiler-Frontend

§6.1 der Säule-3-Spec verspricht „dateiübergreifende Type-Resolution via `go/types` … ausschließlich
im Offline-Build" und gleichzeitig „0 externe Abhängigkeiten". Beides zusammen geht nicht, und
Grafts Antwort ist eine dritte: `graft build --lsp` (`src/graph/lsp/{client,enrich,registry}.ts`)
fragt einen Sprachserver nach der Aufrufhierarchie und stempelt die neuen Kanten `lsp_resolved`;
für Go ist der eingetragene Server `gopls` (`lsp/registry.ts:21`). Best-effort: kein Server, ein
Timeout oder ein Fehler lassen den Graphen unverändert.

**Es gilt:** Abschnitt 14.

### 3.6 Die „vier ask-Integrationstests" sind keine abgrenzbare Menge

Das G1-Delta (§5) lässt „die vier `ask`-Integrationstests" draußen und weist sie G2 zu.
`test/ask.test.ts` hat rund dreißig Tests, und die Mehrzahl prüft Schichten, die loomux nicht hat.
**Es gilt:** die benannte Auswahl in Abschnitt 12.3; die Zahl vier im Delta ist gegenstandslos.

### 3.7 Wo der Port bewusst abweicht

Damit die Liste an einer Stelle steht und nicht über die Abschnitte verstreut. Die Trennung ist
die, auf die es ankommt: ändert die Abweichung, **was im Graphen steht**, oder nur, **wie er
zustande kommt**?

**Drei Abweichungen ändern den Inhalt des Graphen** — hier ist zu prüfen, ob man sie will:

| Abweichung | Ort | Grund |
|---|---|---|
| Signatur eines Typs: `type Cache struct` statt `Cache` | 5.2.1 | Grafts Code widerspricht dort Grafts Kommentar, und kein Test der Referenz steht auf dem Wert |
| Paketselektor `pkg.Fn()` wird verdrahtet — Graft verwirft ihn für Go | 6.2 | Grafts eigene Spezifizierer-Regel, auf Gos Importmodell übertragen |
| Repräsentant eines Importziels überspringt Testdateien | 6.3 | ein Importeur sieht die Testdateien des Zielpakets nie; Grafts `sort()[0]` nimmt sie mit |

**Fünf Abweichungen sind der Wechsel des Mediums** — TypeScript zu Go, Node zu einem Binär, Grafts
Kommandozeile zu loomux'. Sie ändern kein Verhalten, das jemand beobachtet:

| Abweichung | Ort | Grund |
|---|---|---|
| Byte-Ordnung statt `localeCompare` | 7 | eine Kollation sortiert je Plattform anders, die Golden-Files wanderten |
| `mtime` als `int64` aus `UnixNano()` | 7 | Grafts `mtimeMs` ist ein Double; in Go wäre das stiller Genauigkeitsverlust |
| Stempel *in* der Frischeakte, nicht in ihrem Dateinamen | 7.2 | Grafts `fingerprint.<stempel>.json` trennt zwei gleichzeitig installierte Grafts (npx gegen lokal); ein einzelnes Binär hat dieses Problem nicht |
| keine Vorfahrensuche nach der Wurzel | 9 | `--root` ist loomux' Hauskonvention, kein Befehl läuft hier ohne Wurzelbegriff |
| Dateimenge über den Verzeichnislauf statt über Git | 7.1 | ein Unterprozess je Sondenaufruf kostet mehr als die Sonde; gemessen und bestätigt — die anfängliche 20-fache Abweichung war eine fehlende Sperrliste, nicht der Verzeichnislauf |
| kein `--only-dir` | 9 | zieht eine Zustandsregel nach, die die Abnahme nicht braucht |

Was Graft **nicht** hat und diese Spec deshalb selbst entscheidet, steht nicht hier, sondern an
seinem Ort: der Extraktor-Stempel (7.2), die Behandlung der Baubedingungen (6.1) und die
Kandidatenmenge des Selektors (6.2).

## 4. Schema 2

`model.Node` bekommt zwei Felder aus Grafts `NodeV1` (`graph/types.ts:54–77`):

```go
BodyHash string `json:"body_hash"`
BodyText string `json:"-"`
```

- **`BodyHash`** ist `sha256` als vollständige Hex-Zeichenkette (`util/id.ts:9–11`, nicht gekürzt)
  über den Text der ganzen Deklaration. Er ist der Auslöser für `check`, das über `id` +
  `body_hash` diffed (Abschnitt 8.2). Ein Dateiknoten hasht die ganze Datei.
- **`BodyText`** ist der durchsuchbare Rumpf: Whitespace zu einzelnen Leerzeichen normalisiert,
  getrimmt, gekappt bei 5000 Zeichen (`searchBody`, `extract.ts:145–148`). Er erreicht die Platte
  **nie**. Graft strippt ihn beim Serialisieren (`stripBodyText`, `graph/write.ts`), weil er „~65 %
  von `wiring.json`s Bytes auf einem großen Graphen" ist, und legt ihn nur tokenisiert in die
  `ask`-Beiakte. Das `json:"-"` setzt diese Entscheidung im Typsystem um: er kann gar nicht
  verschrieben werden.
  Beim **Dateiknoten** ist er das Residuum — die Zeilen, die kein Symbolspan abdeckt, also
  Importkopf, Paketkonstanten, Paketkommentar (`fileResidual`, `extract.ts:150–156`). So findet man
  eine Datei über ein Wort, das in keiner Funktion steht, ohne einen Rumpf doppelt zu speichern.

Dazu kommt ein Feld auf `Meta`:

```go
Extractor string `json:"extractor"`
```

Es trägt den Stempel aus Abschnitt 7.2. `check` vergleicht ihn, **bevor** es Knoten diffed: ein
Graph von einem anderen Extraktor ist nicht veraltet, sondern fremd, und wird als Ganzes verworfen.
`Validate` verlangt ihn nichtleer.

`schemaVersion` geht **1 → 2**. `Decode` verweigert v1 weiterhin — es tut das schon, und der
Kommentar dort trägt bereits die Begründung („a writer that bumps it changed something"). Die Datei
`internal/code/model/testdata/wiring.json` und ihre Tests wandern auf v2 mit.

Die Tier-2-Felder (`summary`, `summary_state`, `crux`) kommen **nicht** mit. Sie hätten in G2
keinen Erzeuger, und das G1-Delta verbietet Vorrat ohne Erzeuger ausdrücklich (§4, zu `FileCard`).

### 4.1 Vier neue Regeln in `Validate`

Zwei davon sind die offenen Lücken aus `docs/.superpowers/parity/code-g1.md`; sie waren harmlos,
solange kein Erzeuger außerhalb der Tests einen Graphen schrieb. G2 schreibt einen.

1. **Doppelte Knoten-IDs werden abgewiesen.** Grund unverändert: `pagerank.Prepare` behält den
   ersten Knoten einer ID, `blast.New` überschreibt, sodass `Hit.Node` auf den letzten zeigt — zwei
   Antworten auf dieselbe Frage aus demselben Graphen.
2. **Die Quelle einer Kante muss ein Knoten sein.** Ein loses Ende ist nur auf der Zielseite einer
   `imports`-Kante vorgesehen; eine erfundene Quelle taucht in `blast` sonst als Treffer mit
   `Node == nil` auf und ist dort von einem echten unaufgelösten Import nicht zu unterscheiden.
3. **`body_hash` darf auf keinem Knoten leer sein**, Dateiknoten eingeschlossen. Ohne diese Regel
   kann `check` einen Knoten nicht beurteilen und müsste ihn stumm für frisch halten.
4. **`meta.extractor` darf nicht leer sein** (siehe oben).

Die Normalisierung von `Depth` und `Direction`, die die G1-Paritätsakte „an die CLI-Verdrahtung in
G2" verweist, gehört **nicht** hierher: weder `build` noch `check` noch `ask` nimmt diese Werte an.
Sie kommen mit `graph callers` in G4, und dorthin wird sie weitergegeben.

## 5. Der Go-Extraktor

Gelesen aus `describeGo` (`extract.ts:1181–1220`) und der Knotenprägung (`extract.ts:566–620`).

### 5.1 Was ein Knoten wird

| Go-Konstrukt | `kind` |
|---|---|
| `func Name(…)` | `function` |
| `func (r Recv) Name(…)` | `method` |
| `type Name struct{…}` | `struct` |
| `type Name interface{…}` | `interface` |
| `type Name <anderes>` | `type` |
| die Datei selbst | `file` |

Der **Dateiknoten** hat eine Form, die die Spec nennen muss, weil Regel 2 aus 4.1 daran hängt
(`extract.ts:395–408`): die `id` ist **der Pfad selbst**, ohne `#`; `name` ist der Basisname;
`span` ist `L1-L<zeilen>`; `signature` ist leer; `exported` ist `true`; der `body_hash` geht über die
ganze Datei. Nur so ist die Quelle einer `imports`-Kante — Graft setzt dort `ctx.rel` — ein Knoten
des Graphen.

`internal/code/model/testdata/wiring.json` hält diese Form schon (`"id": "src/cache.ts"`), setzt für
den Dateiknoten aber `"exported": false`. Das ist eine handgeschriebene Testdatei ohne Erzeuger und
war bisher gleichgültig; mit dem Extraktor wird es `true`, und die Datei wandert beim Schemabruch
ohnehin.

**Keine Konstanten und keine Variablen.** Graft emittiert für Go keine, und ein `const`-Block ist
für den Lauf kein Knoten, sondern Text — er erreicht die Abfrage über das Residuum des
Dateiknotens (Abschnitt 4).

Ein gruppiertes `type ( … )` ergibt **einen Knoten je `TypeSpec`**, nicht einen für die `GenDecl`
(„one `type_spec` per name (grouped `type ( … )` yields several)", `extract.ts:1208`).

### 5.2 Die Felder

- **`Span`** ist `L<start>-L<end>` über die ganze Deklaration, 1-basiert.
- **`Signature`** ist der Quelltext von Deklarationsbeginn bis Header-Ende, Whitespace normalisiert
  (`extract.ts:589`). Bei Funktion und Methode ist das der Beginn des Rumpfs, also
  `func (u *User) Save(ctx context.Context) error` — genau das, was man sehen will. Bei einem Typ
  weicht der Port ab; siehe 5.2.1.
- **`BodyHash`** ist `sha256` über den Bereich `decl.Pos()`–`decl.End()`. Das **schließt den
  Doc-Kommentar aus** und entspricht damit genau tree-sitter, wo der Kommentar ein
  Geschwisterknoten der Deklaration ist. Beim gruppierten `type` wird über den `TypeSpec` gehasht,
  nicht über die `GenDecl` — sonst änderte eine Änderung an einem Typ den Hash aller anderen der
  Gruppe und `check` meldete sie alle als geändert.
- **`Exported`** ist Gos eigene Regel: erster Buchstabe des **eigenen** Namens groß
  (`goExported`, `extract.ts:1971–1978`). Bei einem qualifizierten Namen ist das der Teil nach dem
  letzten Punkt.
- **`Owner`** trägt nur eine Methode, und zwar ihren Empfängertyp ohne Pointer
  (`goReceiverType`, `extract.ts:1961–1970`: `func (u *User)` → `User`). Einziger Verbraucher ist
  der ownerqualifizierte Index der Auflösung.
- **`Path`** ist repowurzel-relativ und **immer** mit Schrägstrichen, also durch `filepath.ToSlash`.
  Graft hält das über `relPosix` als „posix on every platform" fest (`graph/source-files.ts`, zu
  `SourceStat.rel`), und der Grund steht dort: es ist dieselbe Form, aus der die IDs gebaut werden
  und gegen die `check` diffed, „so cache keys and ids can never disagree".

### 5.2.1 Die Signatur eines Typs — wo Grafts Kommentar und Grafts Code auseinandergehen

Der Kommentar in `extract.ts:1215–1216` sagt: „Header ends where the body opens (`{`) for
struct/interface, else the whole node (a one-line alias like `type ID int`)". Die Zeile darunter tut
etwas anderes:

```ts
const headerEnd = type && (kind === "struct" || kind === "interface") ? type.startIndex : node.endIndex;
```

`type.startIndex` ist der Beginn des `struct_type`-Knotens, also das Schlüsselwort `struct` — nicht
die öffnende Klammer. Und der Schnitt beginnt bei `node.startIndex`, wo `node` der `type_spec` ist;
der beginnt in tree-sitter-go beim **Namen**, denn das Schlüsselwort `type` gehört zur
`type_declaration` darüber.

Beide Enden zusammen ergeben für `type Cache struct { … }` den Quelltextbereich vom `C` bis zum `s`
von `struct`, nach `clean()` also die Signatur **`Cache`**. Der bloße Name, ohne `type`, ohne
`struct`. Für den Alias `type ID int` greift der andere Zweig und liefert `ID int`.

Nachgerechnet, nicht gelesen: `test/graph-go.test.ts` prüft für Go `kind`, `exported`, IDs und
Kanten, aber **keine** Signatur — der Wert ist in Graft nirgends festgenagelt. Es ist also kein
Verhalten, auf das sich etwas stützt, sondern ein unbemerkter Randfall.

**Der Port weicht hier ab** — eine von drei Abweichungen, die den Inhalt des Graphen ändern
(die vollständige Liste steht in 3.7):

| Konstrukt | Signatur in loomux |
|---|---|
| `type Cache struct { … }` | `type Cache struct` |
| `type Reader interface { … }` | `type Reader interface` |
| `type ID int` | `type ID int` |

Also `spec.Pos()`–`spec.Type.Pos()` plus das vorangestellte `type`, bei einem Alias die ganze
`TypeSpec`. Begründung: erstens ist das, was Grafts Kommentar beschreibt und offenkundig meint;
zweitens ist eine Signatur, die nur den Namen wiederholt, in der `ask`-Ausgabe wertlos, weil ID und
Name dort schon stehen; drittens kostet die Abweichung nichts, weil kein Verbraucher und kein Test
der Referenz auf dem alten Wert steht. Der `body_hash` und der Span bleiben unberührt — sie hängen
am `hashNode`, nicht am Header.

### 5.3 Die ID

`mintId(pfad#<qualifizierter Name>)` mit einer Schleife, die bei Kollision `~2`, `~3`, … anhängt
(`extract.ts:443–452`). Ausdrücklich eine **Schleife** und kein einmaliges `~2`: ein Quellname kann
selbst auf `~N` enden, und nur die Schleife ist dagegen dicht.

Eine Go-Methode wird über ihren Empfänger qualifiziert —
`internal/code/blast/reach.go#Index.Reach` —, weil Methoden in Go nicht nesten und die ID sonst je
Empfänger kollidierte (`extract.ts:1197–1198`). Der `Name` bleibt dabei der nackte (`Reach`), damit
ein Aufruf `x.Reach(...)` ihn treffen kann; qualifiziert wird nur die ID.

Die Form ist in Graft testfest: `test/graph-go.test.ts:77–93` erwartet `main.go#Foo` für eine
Funktion, `main.go#User.Save` und `main.go#User.name` für Methoden und `main.go#User`,
`main.go#Reader`, `main.go#ID` für Struct, Interface und Alias. Dieselben Erwartungen gelten für den
Port.

### 5.4 Die Rohkanten

`contains` von der Datei zum Symbol (`extract.ts:546`, `617`, `826`), `imports` je
Importspezifizierer (`extract.ts:709`), `calls` je Aufrufstelle. **Kein `extends`, kein
`implements`** — die Erbschaftskanten entstehen nur für `kind: class` und die JVM-/Swift-Typen
(`extract.ts:623–629`), und Go hat kein explizites `implements`. **Kein `references`** — den Sammler
für importierte Symbole gibt es nur für TypeScript und PHP (`extract.ts:859–936`).

Go emittiert damit genau drei Relationen. Das ist Zuschnitt und keine Auslassung; die Spec sagt es,
damit niemand später eine vierte „vervollständigt".

## 6. Auflösung

`internal/code/resolve` macht aus Rohkanten Kanten. Die Konfidenz ist Grafts Zweistufen-Provenienz
(`resolve.ts`, Kopf):

- **`extracted`** — das Ziel ist sicher: ein Treffer in derselben Datei, ein Importspezifizierer,
  strukturelles Containment.
- **`inferred`** — ein nackter Name, dateiübergreifend über **genau einen** Kandidaten aufgelöst.
- **Mehrdeutig heißt verworfen.**

`lsp_resolved` und `lsp_dispatch` erzeugt G2 nicht; sie gehören der Folgestufe.

### 6.1 Aufrufe

| Form in Go | Ausgang |
|---|---|
| `helper()` in derselben Datei definiert | `calls`, `extracted` |
| `helper()`, im Repo eindeutig anderswo definiert | `calls`, `inferred` |
| `helper()`, mehrfach definiert | verworfen |
| `x.Method()` mit lokal gebundenem `x` | `calls`, über den ownerqualifizierten Index |
| `r.Method()` auf der eigenen Empfängervariablen | `calls`, Empfänger ist der eigene Typ |
| **`pkg.Fn()` über Paketgrenze** | siehe 6.2 |

Die vierte und fünfte Zeile sind Grafts `resolveRecvType` (`bindings.ts:200–226`): für Go kennt er
genau zwei Empfänger — die Empfängervariable der umgebenden Methode und eine lokale
Variablenbindung.

**Und „lokale Variablenbindung" ist enger, als es klingt.** `handleGo` (`bindings.ts:692–727`)
erkennt genau vier Formen:

| Form | gebundener Typ |
|---|---|
| `var x T` (auch `*T`) | `T` |
| `x := T{…}` | `T` |
| `x := &T{…}` | `T` |
| `x := NewT(…)` | `T` — der Name nach `New`, eine Konvention, keine Auflösung |

Alles andere bindet nichts: `x := pkg.New(…)` nicht (die aufgerufene Funktion ist ein Selektor und
kein Identifikator), ein Rückgabewert einer gewöhnlichen Funktion nicht, ein Feldzugriff nicht, eine
`range`-Variable nicht. Der Port bildet diese vier Formen ab und nicht mehr — wer sie erweitert,
erweitert damit die Menge der Aufrufkanten und schuldet dafür eine eigene Begründung.

**Baubedingungen werden nicht ausgewertet.** `lock_windows.go` und `lock_other.go` definieren
denselben Namen für verschiedene Plattformen; für den Extraktor sind das zwei Definitionen, der
Aufruf ist mehrdeutig, und die Kante fällt. Das ist nach der Regel richtig und trotzdem
überraschend, deshalb steht es hier: loomux hat diesen Fall in `internal/testlock` selbst, der
Testbaum aus 12.2 deckt ihn ab, und niemand meldet ihn später als Fehler. Ein Auswerten der
Bedingungen hieße, `go/build`s Kontext nachzubilden und sich auf eine Plattform festzulegen — der
Graph würde je Betriebssystem anders aussehen, und die Golden-Files wären hin.

### 6.2 Der Paketselektor — die eine Erweiterung dieser Stufe

**Was Graft tut:** nichts. Ein Paketselektor ist keiner der beiden Empfänger aus 6.1, also bleibt
`recvType` leer (`extract.ts:768`), und `resolve.ts:271–272` verwirft die Kante mit einem nackten
`continue`. Grafts Go-Tier-1 hat **keine paketübergreifenden Aufrufkanten**; die Querstruktur steckt
allein in den Datei→Datei-`imports`-Kanten. Genau das nennt `lsp/enrich.ts` als seinen
Existenzgrund.

Für loomux wäre das dünn: der Graph dieses Repos kennte `internal/cli` → `internal/code/blast` nur
als Import, nie als Aufruf, und der Blast-Radius eines Refactorings an `blast.Reach` wäre leer.

**Was G2a stattdessen tut** — und das ist keine Abweichung von Grafts Logik, sondern deren
Übertragung auf Gos Importmodell. Graft löst eine `references`-Kante mit Spezifizierer so auf
(`resolve.ts:229–237`): Modulpfad plus exportierter Name, aufgelöst **nur innerhalb der Zieldatei**,
und nur bei genau einem Kandidaten — „so a same-named symbol elsewhere in the repo cannot become a
false edge", Konfidenz `extracted`. Go hat für `pkg.Fn()` genau diese beiden Hälften: der
Importblock der Datei bindet `blast` an einen Modulpfad, und `resolveGoImport` bildet den schon
heute auf ein Verzeichnis des Repos ab.

Also: `pkg.Fn()` wird gegen **das Zielpaket allein** aufgelöst, und nur bei Eindeutigkeit dort,
Konfidenz `extracted`. Kein Namensraten, dieselbe Soundness-Regel, ein anderes Importmodell. Zeigt
der Import nach außen (`"fmt"`), entsteht keine Aufrufkante.

Damit die Regel trägt, braucht sie drei Präzisierungen, die alle Go-eigen sind und die Graft nie
stellen musste, weil es den Fall verwirft.

**Erstens: `x` muss überhaupt ein Paketname sein.** Go-Code beschattet Paketnamen ständig:

```go
graph := graph.New(g)   // ab hier ist `graph` eine Variable, kein Paket
model := model.Decode(r)
```

Ein Selektor `x.Fn` ist deshalb nur dann ein Paketselektor, wenn `x` **in keinem umgebenden Gültig­
keitsbereich deklariert** ist — nicht als Empfänger, Parameter, Ergebnis, `var`, `const`, `:=`,
`range`-Variable oder Typschalter-Bindung — **und** einen Importnamen der Datei trifft. Die Prüfung
ist in dieser Reihenfolge zu machen: erst Beschattung, dann Import. Graft macht es genauso
(`resolveRecvType` fragt `bindings.lookup` vor allem anderen) und kommt dabei nur davon, weil sein
Bindungssammler `x := pkg.New(…)` gar nicht kennt und die Kante dann ohnehin fällt.

`go/ast` hätte dafür ein Feld — `File.Unresolved` enthält genau die Identifikatoren, die die Datei
nicht deklariert. Es ist **abgekündigt**: `go doc go/ast.File` sagt zu `Scope` und `Unresolved`
„Deprecated: see Object", und `ast.Object` ist seit Go 1.22 abgekündigt. Ein neues Paket baut nicht
auf einem Feld, das auf dem Weg hinaus ist. Der Extraktor führt deshalb einen eigenen
Gültigkeitsbereich-Stapel, wie Graft es auch tut.

Gemessen, damit die Entscheidung nicht nur Geschmack ist: Parsen mit Objektauflösung kostet auf
diesem Repo warm 26,2 ms gegen 18,8 ms mit `SkipObjectResolution` (255 Dateien, 2026-09-18, AMD
Ryzen 7 9800X3D, als reproduzierbare Go-Bank in
`internal/code/extract/golang/parsefloor_bench_test.go`; Befehl und Rohausgabe in
`docs/de/benchmarks.md`, Eintrag 2026-09-18 11:49). Die ursprüngliche Messung vom 2026-09-17 nannte
468 Dateien und 58–59 ms gegen 44–46 ms — sie zählte einen eingehängten Checkout unter
`.claude/worktrees/recursing-bartik-b2d7a1` mit, siehe die Korrektur in Abschnitt 11. Die ~7 ms
wären so oder so zu verkraften; die Abkündigung ist der Grund, nicht die Zeit.

**Zweitens: der gebundene Name ist die `package`-Klausel, nicht das letzte Pfadsegment.**
`import "gopkg.in/yaml.v3"` bindet `yaml`, `import "github.com/x/go-foo"` kann `foo` binden. Ohne
Alias ist der Selektorname aus der `package`-Zeile der Nicht-Testdateien des Zielverzeichnisses zu
lesen, nie aus dem Pfad. Für Importe nach außen ist die Frage gegenstandslos, weil dort keine
Aufrufkante entsteht.

**Drittens: Kandidaten sind nur `function`.** Ein Aufruf `pkg.T(x)` auf einem Typ ist eine
Konvertierung und keine Aufrufkante; Graft löst nackte Go-Aufrufe ebenfalls nur gegen Funktionen auf
(der Konstruktor-Rückfall in `resolve.ts` gilt Python und Swift, nicht Go). Methoden fallen aus
demselben Grund heraus wie Typen: ein Paket kann `func New()` und `func (x *T) New()` gleichzeitig
haben, und der Selektor `pkg.New` kann nur das erste meinen. Und `_test.go` fällt heraus, weil ein
Importeur die Testdateien des Zielpakets nie sieht — die von `package X` so wenig wie die von
`package X_test`. Eine Typparameterliste ist vorher abzuschälen, damit `pkg.Fn[int](x)` denselben
Weg geht wie `pkg.Fn(x)`.

Erst danach entscheidet Eindeutigkeit — und die ist in Go fast eine Tautologie: ein doppelter
paketweiter Name ist für den Compiler ein Fehler. Was bleibt, ist der Plattformfall aus 6.1, und der
wird verworfen. Das ist die Antwort auf den Einwand, dies sei die einzige Zusage der Spec, die Graft
nicht in Produktion bewiesen hat: Graft braucht für TypeScript eine echte Eindeutigkeitsprüfung,
weil dort zwei Module denselben Namen exportieren können. In Go garantiert der Compiler, was Graft
prüfen muss.

Der Aliasfall gehört dazu: `import b "…/blast"` bindet `b`, und der Selektor `b.New` wird über diese
Bindung aufgelöst, nicht über das letzte Pfadsegment. Ein Punkt-Import (`import . "…"`) und ein
Leer-Import (`import _ "…"`) binden keinen Selektor und erzeugen nur die `imports`-Kante.

### 6.3 Importe

`resolveGoImport` (`resolve.ts:632–657`): Zeigt der Paketpfad in ein Modul des Repos — `go.mod`
irgendwo im Baum, bei mehreren gewinnt das **längste** Präfix —, wird das Ziel ein repräsentativer
Dateiknoten des Paketverzeichnisses, und zwar deterministisch die kleinste ID **unter den
Nicht-Testdateien**. Grafts `sort()[0]` nimmt die kleinste ID überhaupt; in Go führte das dazu, dass
ein Import auf `internal/code/blast/index_test.go` zeigt, wenn dessen Name vorne sortiert —
auf eine Datei, die der Importeur nie sieht. Bleiben nur Testdateien, gibt es keinen Repräsentanten
und der rohe Paketpfad bleibt stehen. Sonst bleibt der rohe
Paketpfad als Zeichenkette stehen, `"fmt"` genauso wie `"golang.org/x/text/unicode/norm"`. Die
Konfidenz ist in beiden Fällen `extracted`: dass ein Import nach außen zeigt, ist eine Tatsache über
den Code und kein Zweifel.

Gelesen wird aus `go.mod` nur die `module`-Direktive, von Hand und ohne `golang.org/x/mod`.

Das unaufgelöste Ziel ist damit derselbe Fall, den G1 schon entschieden hat: `blast.Reach` behält
ihn als Treffer mit `Node == nil`, `pagerank.Prepare` verwirft die Kante. Beide Seiten bleiben, wie
sie sind.

## 7. Ablage und Determinismus

`internal/code/store` schreibt nach `.loomux/state/graph/`:

```
.loomux/state/graph/wiring.json
.loomux/state/graph/cache/fingerprint.json
.loomux/state/graph/cache/ask-index.json
```

`/.loomux/state/` ist in `.gitignore` bereits ausgeschlossen. Die Trennung von `wiring.json` und
`cache/` ist Grafts (`.graph/` gegen `.cache/`) und hat einen Grund, der bleibt: die Beiakten sind
abgeleitet und jederzeit neu erzeugbar, der Graph ist das Ergebnis.

**Sortierung:** Knoten nach ID, Kanten nach Quelle, dann Relation, dann Ziel (`graph/write.ts`,
`edgeOrder`). Verglichen wird in **Byte-Ordnung** (`sort.Strings`), nicht über eine Kollation —
Grafts `localeCompare` ist hier nicht nachzubilden, sondern zu ersetzen, weil eine Kollation je
Plattform anders sortieren kann und die Golden-Files dann wandern.

**Keine Zeitstempel im Graphen.** Zweimal `build` auf unverändertem Baum ergibt eine byteidentische
Datei — das ist die Abnahmebedingung aus Abschnitt 2 und der Grund, warum `Meta` keine Bauzeit
trägt.

**Atomar** über temp + rename, mit der PID im Temp-Namen und Aufräumen im Fehlerfall. Grafts
Begründung gilt unverändert: ein fester Temp-Name lässt einen gleichzeitigen Lauf dieselbe Datei
schreiben und dem Verlierer einen zerschnittenen Graphen unterschieben.

**mtime** wird als `int64` aus `UnixNano()` geschrieben und als JSON-Ganzzahl gelesen, nie über
`float64`. Grafts `mtimeMs` ist eine JavaScript-Zahl und damit ein Double; in Go wäre das ein
stiller Genauigkeitsverlust auf einem Feld, dessen Gleichheit über einen Neubau entscheidet.

### 7.1 Die Dateimenge

Wie Graft (`ingest/fs.ts`, `graph/source-files.ts`), mit einer Ausnahme:

- **Graft fragt Git** nach der Menge, wo es verfügbar ist: verfolgt + unverfolgt − ignoriert.
  **G2 läuft stattdessen das Dateisystem ab** und verlässt sich auf die Sperrliste unten. Grund:
  `git ls-files` ist ein Unterprozess, und die Sonde soll ~3 ms kosten — ein Prozessstart an jedem
  Aufruf ist ein Vielfaches davon. Die Sperrliste plus „jedes Punktverzeichnis" deckt in einem
  Go-Repo praktisch dieselbe Menge ab; wo sie es nicht tut — eine ignorierte, aber nicht gesperrte
  Generatorausgabe —, landet die Datei im Graphen.
  **Das war die eine Entscheidung dieser Spec, die eine Messung umdrehen konnte, und Aufgabe 10 hat
  sie gemessen — mit einer dritten Antwort, die keine der beiden vorgesehenen war.** Die Sonde
  stand auf diesem Repository bei 57,5–72,5 ms statt ~3 ms, weit über der Schwelle, die Git als
  Quelle nahegelegt hätte. Aber die Ursache war nicht der Verzeichnislauf selbst und nicht das
  Fehlen von Git: `internal/code/sourceset.Stat` lief unter `testdata/` hindurch — 3 so benannte
  Verzeichnisse in diesem Repository, unter denen 1.826 weitere von insgesamt 1.910 im ganzen Baum
  liegen — und keine `.go`-Datei darunter ist Teil der Dateimenge, die die Sonde je aufgenommen
  hätte. `testdata` fehlte schlicht auf der Sperrliste — Grafts Liste
  kennt es nicht, weil Graft nicht Go-spezifisch ist, und Gos eigene Werkzeugkette ignoriert
  `testdata/` für Bauten ohnehin. Mit dem Eintrag ergänzt, maß dieselbe Sonde 2,7–4,3 ms: die
  Referenzzahl von ~3 ms, getroffen, ohne dass sich die Dateimenge (254 Dateien) geändert hätte.
  **Der Verzeichnislauf bleibt, Git wird nicht gebraucht, und die Sperrliste bekommt den fehlenden
  Eintrag.** Beide Zahlen, mit Befehl und Rohausgabe, stehen in `docs/de/benchmarks.md`, Eintrag
  vom 2026-09-18 (zweite Runde). Gemessen wird die Drift in beiden Fällen gegen die Bytes im
  Arbeitsbaum — eine nicht eingecheckte Änderung sieht genauso aus wie eine eingecheckte.
- Sperrliste: `node_modules`, `dist`, `build`, `_build`, `out`, `target`, `vendor`, `coverage`,
  `__pycache__`, `venv`, `testdata`; dazu **jedes** Punktverzeichnis ganz. `testdata` ist von
  anderer Art als die übrigen neun — kein Abhängigkeits- oder Bauausgabeverzeichnis, sondern
  Fixturen als Eingabe. Der Satz „keine Datei darunter ist je Quelle" wäre zu allgemein — `go/parser`
  liest eine `.go`-Datei unter `testdata/` genauso wie jede andere, nur `go/build` ignoriert sie
  konventionsgemäß, nicht inhaltlich. Auf **diesem** Repository ist keine der Fixturen unter
  `testdata/` eine `.go`-Datei (sie sind `.go.txt` oder aufgezeichnete Testfall-Korpora), also kostet
  der Eintrag hier nichts. Ein Repository, dessen `testdata/` echten Go-Code enthält, den es indiziert
  haben will, müsste den Eintrag entfernen — dasselbe Argument, mit dem dieser Abschnitt weiter unten
  begründet, warum `_test.go`-Dateien im Graphen bleiben: „wo sind die Tests" ist eine legitime Frage,
  und für eine Fixtur in einem fremden Repository gilt dasselbe für „wo ist das Beispiel".
- Grenze 1 MB je Datei (`MAX_FILE_BYTES`): darüber ist eine Datei in der Praxis generiert oder
  eingelagert, nicht handgeschrieben.
- Die Ausgabemenge selbst (`.loomux/state/`) fällt heraus.

**`_test.go` bleibt im Graphen.** Graft schließt Tests nicht aus, es de-rankt sie bei der Abfrage
(Abschnitt 10.4). Das steht hier, damit die Menge nicht später „hilfsbereit" verengt wird: „wo sind
die Tests" ist eine legitime Frage an den Graphen.

### 7.2 Der Extraktor-Stempel

Grafts Frischeakte ist nach der Identität des Extraktors verschlüsselt, und der Kommentar sagt
warum: ohne sie „an extractor change correctly drops every memo entry, yet the prints still match
the tree byte-for-byte, so the probe would report clean and queries would keep answering from nodes
the old extractor built".

loomux hat dafür kein Vorbild: `cli.Version` ist in jedem Entwicklungsbau `0.0.0-dev` und als
Stempel wertlos. Also eine handgepflegte Konstante in `extract/golang`, die in die Frischeakte und
in `Meta` geht und bei jeder Änderung am Extraktorverhalten hochgezählt wird. Eine Akte mit fremdem
Stempel gilt als nicht vorhanden.

### 7.3 Kein `[graph]` in der Konfiguration

Die G1-Übergabe nennt „der Abschnitt `[graph]` in `.loomux/config.toml`" als Schuld an G2. Sie
entfällt, und das ist Graft-konform: Graft hat keine `languages`- oder `exclude`-Schlüssel, die
Sprachen ergeben sich aus den vorhandenen Parsern, und die Bau-Overrides (`--include-dir`,
`--only-dir`) liegen ausdrücklich im State und **nie** in der Konfiguration des Repos, das indiziert
wird (`cli.ts:419`: „belongs with the graph … never in the source repo"). Dazu passt loomux' eigene
Regel, dass kein Agent `.loomux/config.toml` schreibt: ein Abschnitt, den niemand braucht, muss auch
niemand von Hand eintragen.

G2 führt damit **keinen** neuen Konfigurationsschlüssel ein. Was `build` einschränkt, sind Flaggen.

## 8. Frische: zwei Mechanismen

Die Verwechslung dieser beiden wäre der teuerste Fehler der Stufe.

### 8.1 Die Sonde — `internal/code/freshness`

`probeDrift` (`graph/fingerprint.ts`): je Datei ein Abdruck `[größe, mtime, hash]` in
`cache/fingerprint.json`. Der Lauf statet jede Quelldatei; stimmen Größe und mtime, gilt der Abdruck
ohne Lesen. Nur Verdächtige werden gelesen und gehasht, damit ein `touch` oder ein `git checkout`
identischer Bytes keinen Neubau kostet. Gemessen ~3 ms für 280 Dateien.

Drei Kategorien: `changed`, `added`, `removed`. **Fehlt die Akte, ist das Ergebnis „unbekannt" und
ausdrücklich nicht „frisch"** — Grafts Kommentar sagt das in genau diesen Worten, und der Aufrufer
baut dann neu.

Dieses Paket importiert den Extraktor **nicht**. Das ist die Architekturbedingung, die G4 erlaubt,
die Sonde im Hook-Pfad zu benutzen, ohne das Abhängigkeitstor aus §3.2 der Säule-3-Spec zu brechen.
Grafts Grund ist derselbe, nur anders motiviert (ein Importzyklus): die Dateimenge ist deshalb aus
dem Bauer herausgezogen, damit beide sie identisch aufzählen.

### 8.2 `graph check` — Neuextraktion

`checkGraph` (`graph/check.ts`) prüft **nicht** die Abdrücke. Es extrahiert Tier 1 neu über dieselbe
Dateimenge und diffed die frischen Knoten gegen den geschriebenen Graphen über `id` und `body_hash`:
`added`, `removed`, `changed`.

Und `build` benutzt den stat-Schnellweg **nie**, sondern liest und hasht immer alles. Grafts
Begründung ist scharf und wird übernommen: ein Stat darf entscheiden, ob eine Abfrage überhaupt neu
baut; er darf nicht entscheiden, was der Neubau ansieht — sonst meldet `check` Drift, die der von
ihm empfohlene `build` nicht reparieren kann.

Daraus die Abnahme: `check` gibt 0 auf frischem Stand, 1 nach einer inhaltlichen Änderung, und
**0 nach einem reinen `touch`** — letzteres, weil die Neuextraktion den Inhalt sieht und nicht die
mtime.

## 9. Die Befehle

```
loomux graph build [--root DIR]
loomux graph check [--root DIR] [--json]
loomux graph ask   "<anfrage>" [--root DIR] [--limit N] [--source] [--full]
                   [--in PREFIX] [--json] [--no-refresh]

`--limit` steht auf 8, wie Grafts eigener Standard (`cli.ts:596`).

**Grafts `--only-dir` kommt nicht mit.** Es verlangt, dass die Weißliste in der Frischeakte
mitgeschrieben wird, weil sonst jede ausgeschlossene Datei der Sonde als `added` erscheint — eine
Zustandsregel, die die Abnahme dieser Stufe nicht braucht. Wer einen Teilbaum indizieren will,
zeigt mit `--root` darauf.
```

Die Wurzel kommt über `--root` und sonst aus dem Arbeitsverzeichnis, wie bei jedem anderen
loomux-Befehl (`hook`, `lint`, `worktree`). Grafts Vorfahrensuche `nearestGraftRoot`
(`graph/root.ts`) wird **nicht** portiert: sie löst ein Problem, das loomux nicht hat, weil hier
kein Befehl ohne Wurzelbegriff läuft.

`build` gibt nach dem Schreiben eine Kennzahlenzeile aus — Knoten, Kanten je Relation, unaufgelöste
Importziele, Dateien ohne Symbol, Dauer. Das ist der Abnahmeblick auf den Extraktor und die Zahl,
die die Paritätsakte braucht.

**Exitcodes**, nach loomux' Haus-Disziplin und deckungsgleich mit Graft:

| Fall | Code |
|---|---|
| frisch, bzw. erfolgreich gebaut oder beantwortet | 0 |
| `check` findet Drift | 1 |
| `check` findet keinen Graphen (eine Meldung, die auf `graph build` verweist) | 1 |
| Benutzungsfehler (unbekannte Flagge, fehlendes Argument) | 2 |

Graft hat für den fehlenden Graphen keinen eigenen Code (`cli.ts:687`), und §5.3 der Säule-3-Spec
verlangt nur, dass er „sauber als nicht-initialisiert" gemeldet wird — die Meldung leistet das, ein
dritter Code wäre eine Erfindung.

## 10. G2b: Lexik und Abfrage

### 10.1 `internal/code/lexicon`

`Tokenize`: camelCase aufbrechen, kleinschreiben, an allem Nicht-Alphanumerischen trennen, Länge > 1,
dazu eine Stoppliste von **32** Wörtern (`ask/index-file.ts:28–33`; nachgezählt, nicht aus einem
Kommentar übernommen). Entscheidend ist nicht die Regel,
sondern dass **eine** Funktion sie trägt, die Bauen und Abfragen teilen: die Beiakte kann nur dann
ein korrekter Cache der Abfragerechnung sein, wenn beide Seiten denselben Code rufen. Graft sagt das
im Dateikopf und hat `tokenize` deshalb dort und nicht in `ask.ts`.

Die Regeln werden **byteweise** übernommen, nicht dem Sinn nach: der camelCase-Schnitt greift auf
`[a-z0-9][A-Z]`, getrennt wird an `[^a-z0-9]+`, und beides ist damit reines ASCII. Ein Port mit
`unicode.IsUpper` und `unicode.IsLetter` wäre die schönere Go-Fassung und würde auf einem Bezeichner
mit Umlaut oder griechischem Buchstaben andere Token liefern als die Golden-Werte — also nicht.

Die Stoppliste ist englisch. Das passt, weil Bezeichner und Kommentare in loomux englisch sind.

Die Beiakte `cache/ask-index.json` hält `df`, `docCount`, `avgBodyLen` und je Knoten drei
Token-Beutel für `name`, `path` und `body`. Ihr Grund ist gemessen: Live-Tokenisierung war bei 32k
Knoten ~45 % der Abfragezeit. Geschrieben wird sie von `build`, aus dem Graphen **vor** dem Strippen
von `body_text` (`graph/build.ts`; Graft warnt ausdrücklich davor, sie aus der geschriebenen Datei
zu rekonstruieren).

### 10.2 Die Rechnung

Drei Felder mit drei Gewichten (`ask.ts:917–922`):

```
lex = (3 · score(name) + 2 · score(path) + bm25(body)) · testFaktor
```

- `score` auf Name und Pfad ist `Σ qₙ·dₙ·idf` mit `idf = log(1 + n/(1+df))` — kurze Bezeichner,
  also einfache gewichtete Überdeckung.
- Der Rumpf geht über BM25 mit `k1 = 1,2`, `b = 0,75`, längennormiert über `avgBodyLen`, damit eine
  lange Definition nicht durch Masse gewinnt.
- Die Gewichte **3** und **2** sind nicht zu erfinden und nicht zu runden: sie entscheiden, ob ein
  Namenstreffer einen Pfadtreffer schlägt.

### 10.3 Saat und Verschmelzung

Die lexikalischen Werte **sind** die Saat des Personalized PageRank — „lexical proposes, graph
disposes" (`ask/graphrank.ts`, Kopf). `pagerank.Prepare` und `pagerank.Rank` aus G1 nehmen sie
unverändert.

```
score = (lex/max + 0,5 · pr) · testFaktor
```

`GRAPH_WEIGHT = 0,5` (`ask.ts:414`): Konnektivität darf Fast-Gleichstände umsortieren und einen
verbundenen Treffer von einer isolierten Wortkollision trennen, aber einen klaren lexikalischen
Sieger nicht überstimmen.

`RESCUE_FLOOR = 0,15` (`ask.ts:420`): ein Knoten, den die Anfrage nie wortgetroffen hat, kommt
hinein, wenn der Lauf ihm mindestens 15 % der Masse des Spitzenknotens gibt. Das ist, was die
Hilfsfunktion oder die Konfiguration findet, von der eine Aufgabe abhängt, ohne sie zu benennen.

`--in` verengt den **Lauf** und nicht nur die Saat (`ask.ts:936–943`): ohne das könnte ein über den
Rettungsboden gehobener Nachbar außerhalb des Präfixes erscheinen und den Filter aushebeln. G1 hat
dafür schon die Schnittstelle: `Prepare(g, keep)`.

**Und es verengt auch die Lexik, nicht nur den Lauf.** Graft filtert die Dokumentmenge **vor** dem
Bewerten, also werden `df`, `docCount` und `avgBodyLen` über die gefilterte Menge **neu gerechnet**
— aus den Token-Beuteln der Beiakte, die dafür alles Nötige enthält. Deshalb existiert der Test
`test/ask.test.ts:928` („per-scope idf differs from global — a term's rank flips relative to another
between filtered and unfiltered"): ein Wort, das im ganzen Repo häufig und in einem Unterbaum selten
ist, muss dort diskriminieren. Wer nur den Lauf verengt und die globalen IDF-Werte behält, bekommt
eine andere Rangfolge als die Referenz.

### 10.4 Test-De-Rankung

`TEST_RANK_PENALTY = 0,35`, multiplikativ, und `isTestPath` kennt `_test.go` ausdrücklich
(`ask.ts:196–205`).

Der Faktor wirkt **zweimal**, und das ist Absicht, nicht Versehen. Er geht in den Rohscore
(`ask.ts:917`) — damit sät ein Test von Anfang an weniger Masse in den PageRank — und **erneut** nach
der Normierung (`ask.ts:964`). Grafts Kommentar dort sagt „apply the test penalty after
normalization **too**" und nennt den Grund: ist ein Test trotzdem der stärkste Rohtreffer, stellt die
Division durch `max` ihn wieder auf 1,0 und löscht die De-Rankung. Beobachtet wurde genau das — ein
`ask`, das eine `Test…`-Funktion als Spitzentreffer lieferte und den Agenten in die falsche Datei
schickte.

**Und die Strafe entfällt, wenn die Anfrage selbst nach Tests fragt** (`wantsTests`,
`ask.ts:509–510`): trifft die Anfrage `tests?`, `specs?`, `coverage`, `assert(ion)s?`, `fixtures?`
oder `mocks?`, ist der Faktor 1. Ohne das ist der portierte Vektor `test/ask.test.ts:40` („but not
for a test-seeking one") nicht erfüllbar — „wo sind die Tests für X" muss die Tests liefern.

### 10.5 `ask` baut vorher neu

`cli.ts:605` ruft `refreshBefore` vor jeder Antwort. `refresh.ts` beschreibt, warum die Frische in
den Abfragepfad gehört: vorher hing sie an einem `Stop`-Hook, und jede Abfrage zwischen dem ersten
Edit und dem Ende des Zuges antwortete aus einem Graphen, der die gerade geänderte Datei nicht
kannte; eine Änderung außerhalb des Agenten löste gar nichts aus.

Die Eigenschaften, die mitkommen: **$0 und offline** (nur Tier 1), **nie fatal** (ein gescheiterter
Neubau antwortet aus dem Graphen auf der Platte), **kein Ansturm** (ein Lock), und **es schreibt
nur, was eine Abfrage liest** — Graph, Beiakte, Frischeakte. `--no-refresh` schaltet es ab.

Ein `ask` auf einem frischen Klon baut also, weil „keine Akte" „unbekannt" heißt (Abschnitt 8.1).

### 10.6 Ausgabe

Standardmäßig Score, ID, Pfad, Span, Signatur — ein Ortsangeber. `--source` schneidet die Zeilen des
Spans aus der Datei, gekappt bei 80 Zeilen mit einem Restmarker, der den vollen Ort nennt. `--full`
hebt die Kappung auf. `--json` gibt dasselbe strukturiert.

Kein Crux (Abschnitt 3.1). Kommt später ein Erzeuger, greift `inlineSource`s Vorrang und die
Flaggen bleiben, wie sie sind.

## 11. Startzeit

G2 zieht **keine** Abhängigkeit ein: `go/parser`, `go/token`, `go/ast`, `crypto/sha256`,
`encoding/json` sind Standardbibliothek. Damit ist der Kaltstartboden von G2 kein offener Punkt, und
die Inittrace-Messung bleibt dort, wo das G1-Delta sie hingelegt hat: bei G3, wo `internal/serve`
das MCP-SDK einzieht.

Kein `init()` und keine Paketvariable parst eingebettete Daten — die Projektregel gilt unverändert;
in G2 gibt es nichts Eingebettetes.

Gemessen ist der Extraktorweg schon — aber die erste Messung war falsch, und die Korrektur gehört
hierher, nicht nur ins Benchmark-Journal. Am 2026-09-17 maß ich reines Parsen mit **44–46 ms** warm
mit `SkipObjectResolution`, **58–59 ms** mit Objektauflösung, kalt 914 ms, über „alle 468
Go-Dateien dieses Repos". Der Zähler war falsch: der Hauptcheckout enthielt zu dem Zeitpunkt einen
zweiten, vollständigen Checkout desselben Codes unter `.claude/worktrees/recursing-bartik-b2d7a1`,
und der Lauf zählte ihn mit — 468 `.go`-Dateien insgesamt, davon 234 dieselben Dateien noch einmal
unter diesem Worktree, 234 ohne ihn. Die Messung parste den Baum also zweimal.

Aufgabe 10 hat das am 2026-09-18 zweimal neu gemessen, gegen diesen Worktree, der keinen
eingehängten Checkout enthält. Ein erster Durchgang mit einem Wegwerfwerkzeug maß ~27–32 ms warm
reines Parsen, ~38–45 ms warm mit Objektauflösung; diese Zahl hatte aber selbst keinen Befehl und
keine Rohausgabe, die diese Spec oder das Benchmark-Journal trugen, und wird deshalb hier nicht mehr
zitiert. **Der gültige Boden ist der zweite Durchgang**, als reproduzierbare Go-Bank geschrieben
(`internal/code/extract/golang/parsefloor_bench_test.go`) und mit Befehl und Rohausgabe im
Benchmark-Journal festgehalten (`docs/de/benchmarks.md`, Eintrag 2026-09-18 11:49): **18,8 ms warm**
reines Parsen, **26,2 ms warm** mit Objektauflösung, über 255 Dateien (254 plus die neue Bank-Datei
selbst). Diese Zahl liegt niedriger als der erste Durchgang — die Abweichung ist nicht untersucht,
aber offen benannt statt verschwiegen. Die 44–46 ms und 58–59 ms oben sind falsch und stehen nur noch
als Beleg dafür, woher der Fehler kam. Der Port nimmt weiterhin den schnellen Weg und führt den
Gültigkeitsbereich selbst (6.2); der Grund war immer die Abkündigung von `ast.Object`, nicht die
Zeitersparnis — das gilt mit jedem der beiden Böden unverändert.

## 12. Nachweise und Tor

### 12.1 Das Tor

`.githooks/pre-commit` bleibt unverändert und gilt: `gofmt`, `go vet`,
`go test ./... -count=1 -covermode=set -coverpkg=github.com/xidus90/loomux/...`, `dev covergate` mit 100 % je Funktion, dann
das Pilot-Binary. Ein `//coverage:exempt <grund>` nur mit Begründung.

Das CGo-Freiheitstor aus §9.2 der Säule-3-Spec führt G2 **nicht** ein: es wird keine Abhängigkeit
gezogen, die C brauchen könnte, und ein Tor ohne Gegenstand ist eine Zeile, die niemand pflegt. Es
gehört zu G5 mit `wazero`.

### 12.2 Golden-Files und der Testbaum

Golden-Files liegen unter `internal/code/**/testdata`, nicht unter `testdata/cases/` — das ist laut
eigener README allein die Paritätsakte gegen die ersetzten Werkzeuge, und G2 ersetzt keines. Das
G1-Delta (§5) hat dieselbe Entscheidung schon getroffen und begründet.

Der Extraktor braucht einen kleinen Go-Baum, der jede Entscheidung dieser Spec genau einmal auslöst:

- alle fünf Kinds, dazu eine Datei ohne jedes Symbol;
- ein gruppiertes `type ( … )` mit zwei Namen;
- eine ID-Kollision, die die `~2`-Schleife auslöst, und eine, deren Quellname selbst auf `~2` endet;
- einen Pointer-Empfänger und einen Wert-Empfänger desselben Typs;
- einen Doc-Kommentar über einer Funktion (er darf den `body_hash` nicht verändern);
- einen paketinternen Aufruf innerhalb einer Datei, einen über Dateigrenzen, und einen mehrdeutigen,
  der verworfen wird;
- einen Paketselektor ins eigene Repo, einen mit Alias, einen Punkt-Import und einen Leer-Import;
- ein Zielpaket, in dem derselbe Name als Funktion **und** als Methode vorkommt, sodass der
  Selektor die Methode ausschließen muss;
- ein Zielpaket, dessen alphabetisch erste Datei eine `_test.go` ist, sodass der Repräsentant sie
  überspringen muss, und ein Symbol, das **nur** in einer `_test.go` des Zielpakets steht;
- ein Plattformpaar (`…_windows.go` und `…_other.go`) mit demselben Namen, dessen Aufruf als
  mehrdeutig fällt (6.1);
- einen Import nach `"fmt"`, der als Zeichenkette stehen bleibt;
- eine Datei über 1 MB und eine in einem gesperrten Verzeichnis, die beide nicht vorkommen dürfen;
- eine `_test.go`-Datei, die vorkommen **muss**.

Die Vorlagen liegen unter `internal/code/extract/golang/testdata/` mit der Endung `.go.txt` und
werden vom Test in ein temporäres Verzeichnis geschrieben. Der Grund ist **nicht** `go build` — das
ignoriert jedes `testdata/` ohnehin, und `go vet ./...` ebenso, beides am 2026-09-17 nachgeprüft.
Der Grund ist die erste Bahn des Tors: `gofmt -l cmd internal` steigt in `testdata/` hinein und
meldet eine Datei dort wie jede andere (nachgeprüft mit einer absichtlich krummen Datei unter
`internal/testlock/testdata/`). Eine Vorlage, die absichtlich unformatiert oder syntaktisch krumm
ist, bräche also das Tor. Mit `.go.txt` sieht keine der drei Bahnen sie.

### 12.3 Portierte Vektoren aus Graft

Statt der „vier `ask`-Integrationstests" des Deltas (Abschnitt 3.6) eine benannte Auswahl, jede mit
Herkunftszeile `trailhq/Graft @ 1e352a3 (MIT)`:

| Quelle | Gegenstand |
|---|---|
| `test/ask.test.ts:112` | ohne `--source` nur Zeiger, kein Quelltext |
| `test/ask.test.ts:154` | mit `--source` der Span-Rückfall |
| `test/ask.test.ts:470` | ein Symbol über ein Wort finden, das nur in seinem Rumpf steht |
| `test/ask.test.ts:491` | eine Datei über ein Wort im Modulrumpf finden (das Residuum) |
| `test/ask.test.ts:40`, `:72` | Test-De-Rankung, auch wenn der Test der stärkste Rohtreffer ist |
| `test/ask.test.ts:890` | `--in` segmentbewusst: „widgets" trifft nicht „widgets-extra" |
| `test/ask-index.test.ts` | die Beiakte ist ein exakter Cache des Live-Wegs |

Nicht portiert: alles zu Multi-Scope, File-First-Projektion, `fileTopLock` und Abdeckungskennzahlen
— Schichten, die loomux nicht hat. Die Verschmelzungskonstanten (`0,5`, `0,15`, `0,35`) bekommen
eigene Vektoren auf handgebauten Graphen, weil Grafts Tests sie nur mittelbar prüfen.

### 12.4 Mutationsrunden

`loomux dev mutants internal/code/extract/golang internal/code/resolve internal/code/freshness` für
G2a, `… internal/code/lexicon internal/code/ask` für G2b. Überlebende bekommen ihre Verfügung in
`docs/.superpowers/parity/code-g2a.md` und `code-g2b.md`.

Der Extraktor ist das größte Paket der Stufe; die Runde dort wird lang. Das ist eine Last für den
Plan, nicht für diese Spec.

## 13. Dokumentation und Messungen

Fällig in G2a, teils als Nachholung aus G1:

- `docs/{en,de}/cli-reference.md`: `graph build`, `graph check` (G2b ergänzt `graph ask`).
- `README.md` und `README.de.md`: der Absatz „State (stage G1)" und die Fahrplanzeilen.
- `docs/{en,de}/architecture.md`: die neuen Pakete und die Ablage unter `.loomux/state/graph/`.
- `docs/{en,de}/configuration.md`: dass es **keinen** `[graph]`-Abschnitt gibt und warum
  (Abschnitt 7.3), damit die Frage nicht wiederkehrt.

`docs/{en,de}/benchmarks.md` bekommt chronologisch:

1. Die offene G1-Messung: gepoolte Dangling-Masse ~9 ms gegen ~4,5 s je Dangling-Knoten auf einem
   Graphen mit 20k Knoten, 2026-09-16, AMD Ryzen 7 9800X3D — **nur warm. Die kalte Zahl fehlt und
   ist zu messen, nicht zu übernehmen.**
2. `graph build` auf loomux selbst, kalt und warm, gegen die 18,8 ms des reinen Parsens als
   Untergrenze (Abschnitt 11). Kalt heißt: frischer Prozess und keine Beiakte, und das ist beim
   Eintrag zu vermerken. **Die zuerst genannten 44–46 ms waren falsch** — über einen Baum gemessen,
   der einen eingehängten Checkout doppelt zählte —, und ein zwischenzeitlich genannter
   ~27–32-ms-Boden trug keinen eigenen Befehl; 18,8 ms ist die Zahl mit reproduzierbarer Bank und
   Rohausgabe.
3. Die Sondendauer über die Dateimenge dieses Repos, gegen Grafts ~3 ms für 280 Dateien. **Diese
   Messung hat Abschnitt 7.1 entschieden** — mit einer dritten Antwort: nicht „Verzeichnislauf
   taugt" und nicht „auf `git ls-files` umstellen", sondern „der Sperrliste fehlte `testdata`".
   Gemessen 57,5–72,5 ms vor der Ergänzung, 2,7–4,3 ms danach, bei unveränderter Dateimenge
   (254 Dateien). Der Verzeichnislauf bleibt, die Abweichung entfällt aus einem anderen Grund als
   vorgesehen.
4. Die Antwortzeit von `graph ask`, warm, mit und ohne Beiakte — das ist die Messung, die den
   Existenzgrund der Beiakte für loomux belegt oder widerlegt.

## 14. Die optionale Folgestufe: `--lsp`

Nicht in G2a und nicht in G2b, aber hier festgeschrieben, damit die Lücke aus Abschnitt 6.2 einen
Namen hat.

`loomux graph build --lsp` fragt `gopls` über JSON-RPC auf stdio nach der Aufrufhierarchie jedes
Funktions- und Methodenknotens und trägt die gefundenen Kanten als `calls` mit `lsp_resolved` nach.
Das ist Grafts Weg (`graph/lsp/`, Server für Go ist `gopls`), es zieht **keine** Go-Abhängigkeit,
und es ist best-effort: kein Server, ein Timeout oder ein Fehler lassen den Graphen unverändert.

**Verworfen, mit Zahlen:** `golang.org/x/tools/go/packages` mit `NeedTypes` liefert echte Auflösung,
kostet aber warm 680–1008 ms gegen 44–46 ms fürs reine Parsen — ein Faktor von rund dem 15- bis
23-Fachen —, drei neue Module (`x/tools`, `x/mod`, `x/sync`) und einen Aufruf von `go list` zur
Laufzeit. Gemessen am 2026-09-17, AMD Ryzen 7 9800X3D, warm, Wegwerfmodul im Scratchpad — auf
demselben doppelt gezählten Baum, dessen falschen Zähler Abschnitt 11 korrigiert (468 statt 254
Dateien, weil ein eingehängter Checkout mitgezählt wurde). **Die folgende Zeile ist Schlussfolgerung,
keine erneute Messung**: das Wegwerfmodul zog `x/tools`, `x/mod` und `x/sync` als Abhängigkeiten,
genau die drei, die dieser Abschnitt als einen der Gründe gegen den Weg nennt, und es allein für eine
Neuzählung wiederzubeleben ist den Aufwand nicht wert, wenn die Entscheidung ohnehin an der
Abkündigung von `ast.Object` hängt, nicht an einer Zeitspanne. Die Überlegung: beide Seiten des
Paars — Parsen und `go/packages`-Auflösung — liefen über denselben doppelt gezählten Baum, dieselben
Dateien also zweimal auf beiden Seiten der Division, was das **Verhältnis** ungefähr erhält, auch
wenn keine der beiden absoluten Zahlen diesen Baum in seiner echten Größe (254 Dateien) beschreibt.
Das Verhältnis von rund dem 15- bis 23-Fachen ist deshalb eine plausible Schätzung, keine gemessene
Tatsache für die echte Dateimenge; die Entscheidung selbst braucht sie nicht, weil sie an der
Abkündigung hängt und nicht an der Zeit. Wer die absoluten Zahlen für eine andere Entscheidung
braucht, misst sie neu gegen die echten 254 Dateien, statt diese Zeile weiterzuverwenden.

**Ebenfalls verworfen:** `go/types` mit dem Quell-Importer der Standardbibliothek. Warm 7,5–14,4 s,
und **50 von 76 Paketen scheitern**, weil der Quell-Importer keine Modulauflösung kennt und jede
Fremdabhängigkeit bricht. Gleiche Messung.

## 15. Was G2 offen lässt

- **`--lsp`** (Abschnitt 14) und damit die paketübergreifenden Aufrufkanten, die der Paketselektor
  aus 6.2 nicht erreicht: ein Aufruf über einen Interface-Wert, den erst ein Typprüfer auflöst.
- **`FileCard`, Crux, `confidence`-Rangfolge**: mit ihren Erzeugern, nicht vorher.
- **Die übrigen `graph`-Unterformen** aus §7 der Säule-3-Spec, darunter die Normalisierung von
  `Depth` und `Direction`, die zu `graph callers` gehört (G4).
- **Die Inittrace-Messung** mit dem MCP-SDK: G3.
- **Die `check`-Lanes** `graph-freshness` und `blast-audit` und die Hook-Anbindung: G4.
- **Mehrsprachigkeit** über `wazero`: G5. Mit ihr kommt das CGo-Freiheitstor.
