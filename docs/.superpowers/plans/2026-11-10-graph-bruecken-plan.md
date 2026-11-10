# Brücken zwischen Sprachen im Code-Graphen: Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ein sprachneutraler Brückenschritt im Resolver, über den Symbole
verschiedener Sprachen Kanten bekommen; GDScript nutzt ihn schon für Godot →
C#/C++.

**Architecture:** `extract.Result` bekommt `Exports`, `Bridges`, `Embedded`,
`RawEdge` ein Feld `Member`. `resolve.GraphWith` macht die Ergebnisse flach,
löst je Sprache auf und danach alle Brücken in `bridge.go` über eine
Namensraum-Tabelle mit Kardinalität, Realm und Ausweichregeln. Der
GDScript-Resolver meldet Exports und Übergaben.

**Tech Stack:** Go, `github.com/odvcencio/gotreesitter` v0.57.0.

**Spec:** `docs/.superpowers/specs/2026-11-10-graph-bruecken-design.md`

## Global Constraints

- gotreesitter exakt `v0.57.0`; `treesitter.Parser = "gotreesitter/v0.57.0"`.
- GDScript-Extraktor `version = "gdscript/6"`.
- Namensräume in A: `godot.class`, `godot.method`, `godot.signal`.
- GDScript zu GDScript ändert sich nicht: gleiche Kanten, gleiche
  `Unresolved` mit gleichen Gründen.
- Graph byte-gleich bei gleichem Input (deterministische Reihenfolge).
- 100 % Coverage der geänderten Pakete; Ausnahme nur mit
  `//coverage:exempt <Begründung>` wie im Repo üblich.
- Code, Kommentare, Bezeichner, Commits englisch; kein `Co-Authored-By`.
- Kommentare auf anderer Abstraktionsebene als der Code, keine Historie.
- `cacheVersion` in `query/cache.go` bleibt `1`.

## Review Focus

1. Godot-Projekt im Repo-Wurzelverzeichnis (Wurzel `""`) muss ein Realm sein
   → Test in Task 4 (`TestBridgeRealmAtRepoRoot`) und Task 6.
2. Zwei Godot-Projekte mit gleichem `class_name` dürfen sich nicht gegenseitig
   antworten → Task 6 (`TestGodotHandoverStaysInProject`).
3. Ein Export ohne Realm darf eine Brücke ohne Realm nicht treffen → Task 4
   (`TestBridgeWithoutRealmGivesNoEdge`).
4. Cache-Wiederverwendung eines Ergebnisses mit eingebetteten Teilen verliert
   keinen `BodyText` → Task 3 (`TestCacheKeepsEmbeddedBodies`).
5. Eingebettete Teile ändern bei gleichem Input die Reihenfolge der Kanten
   nicht → Task 3 (`TestGraphWithEmbeddedIsDeterministic`).

---

### Task 1: gotreesitter auf v0.57.0

**Files:**
- Modify: `go.mod`, `go.sum`
- Modify: `internal/code/extract/treesitter/doc.go:42`
- Modify: `internal/notices/NOTICE.md` (generiert)

**Interfaces:**
- Produces: `treesitter.Parser == "gotreesitter/v0.57.0"`.

- [ ] **Step 1: Referenz-Binary von `master` bauen** (vor jeder Änderung)

Run: `go build -o "$SCRATCH/loomux-old.exe" ./cmd/loomux`
(`$SCRATCH` = Scratchpad der Session.) Expected: Datei existiert.

- [ ] **Step 2: Abhängigkeit heben**

Run: `go get github.com/odvcencio/gotreesitter@v0.57.0 && go mod tidy`
Danach `Parser` in `doc.go` auf `"gotreesitter/v0.57.0"`.

- [ ] **Step 3: Tests laufen lassen**

Run: `go test ./internal/code/...`
Expected: PASS (`TestParserMatchesGoMod` grün).

- [ ] **Step 4: NOTICE neu erzeugen**

Run: `go run ./cmd/loomux dev notices`, dann `go test ./internal/notices/`.
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum internal/code/extract/treesitter/doc.go internal/notices/NOTICE.md
git commit -m "chore(deps): bump gotreesitter to v0.57.0"
```

### Task 2: Vertrag in `extract`

**Files:**
- Modify: `internal/code/extract/extract.go`
- Test: `internal/code/extract/extract_test.go`

**Interfaces:**
- Produces (alle in Paket `extract`):
  - `type Namespace string`
  - `const NamespaceGodotClass Namespace = "godot.class"`,
    `NamespaceGodotMethod = "godot.method"`,
    `NamespaceGodotSignal = "godot.signal"`
  - `var Namespaces = []Namespace{NamespaceGodotClass, NamespaceGodotMethod, NamespaceGodotSignal}`
  - `type Export struct { Namespace Namespace; Name string; Target model.NodeID }`
    (JSON `namespace`, `name`, `target`)
  - `type Bridge struct { Source model.NodeID; Relation model.Relation; Namespace Namespace; Name string; Scope []string; File string; Line int }`
    (JSON `source`, `relation`, `namespace`, `name`, `scope,omitempty`,
    `file`, `line,omitempty`)
  - `Result` neu: `Exports []Export \`json:"exports,omitempty"\``,
    `Bridges []Bridge \`json:"bridges,omitempty"\``,
    `Embedded []Result \`json:"embedded,omitempty"\``
  - `RawEdge` neu: `Member model.Kind \`json:"member,omitempty"\``
  - `func (r Result) Flatten() []Result` — `r` ohne `Embedded`, gefolgt von
    jedem eingebetteten Teil (ebenfalls ohne `Embedded`), in Reihenfolge.

- [ ] **Step 1: Failing tests**

```go
func TestResultReadsOldJSON(t *testing.T) {
	var r extract.Result
	if err := json.Unmarshal([]byte(`{"path":"a.gd","language":"gdscript","nodes":[]}`), &r); err != nil { t.Fatal(err) }
	if r.Exports != nil || r.Bridges != nil || r.Embedded != nil { t.Fatalf("got %+v", r) }
}
func TestResultOmitsEmptyBridgeFields(t *testing.T) {
	b, _ := json.Marshal(extract.Result{Path: "a", Language: "x", Nodes: []model.Node{}})
	for _, k := range []string{"exports", "bridges", "embedded"} {
		if strings.Contains(string(b), k) { t.Fatalf("%s in %s", k, b) }
	}
}
func TestFlattenPutsPartsAfterTheFile(t *testing.T) {
	r := extract.Result{Path: "p.html", Language: "html", Embedded: []extract.Result{{Path: "p.html", Language: "js"}}}
	got := r.Flatten()
	// len 2; got[0].Language "html" with Embedded nil; got[1].Language "js"
}
func TestNamespacesListsEveryConstant(t *testing.T) {
	// reflect.DeepEqual(extract.Namespaces, []extract.Namespace{"godot.class","godot.method","godot.signal"})
}
```

- [ ] **Step 2: Run** `go test ./internal/code/extract/ -run 'Result|Flatten|Namespaces'` → FAIL (undefined).
- [ ] **Step 3: Typen und `Flatten` implementieren.** Doc-Kommentare nach Spec
  „Vertrag“ und „Eingebettete Teile“ (kein Dateiknoten, höchstens ein Teil je
  Sprache, nie die Sprache der Datei, eine Ebene).
- [ ] **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `feat(graph): add exports, bridges and embedded parts to the extraction result`

### Task 3: Eingebettete Teile in Resolver, Cache und Statistik

**Files:**
- Modify: `internal/code/resolve/resolve.go` (`GraphWith`)
- Modify: `internal/code/query/cache.go` (`readCache`, `writeCache`)
- Modify: `internal/code/query/build.go` (`Extract`)
- Test: `internal/code/resolve/resolve_test.go`, `internal/code/query/cache_test.go`, `internal/code/query/build_test.go`

**Interfaces:**
- Consumes: `extract.Result.Flatten()`.
- Produces: `func tally(per map[string]LangStats, rel string, r extract.Result, parsed bool)`
  in `build.go` — zählt `r` und jeden eingebetteten Teil nach Spec
  („Statistik“); `Extract` ruft sie statt der Inline-Zählung auf.

- [ ] **Step 1: Failing tests**
  - `TestGraphWithFlattensEmbedded` (resolve): Datei `p.html` (Language
    `html`, Dateiknoten `p.html`, Knoten `p.html#body`) mit eingebettetem
    `js`-Teil (Knoten `p.html#script.f`, `contains` von `p.html#body`).
    Erwartet: beide Knoten im Graphen, `contains`-Kante vorhanden,
    `Meta.Languages == []string{"html","js"}`, `model.Validate` ohne Fehler.
  - `TestGraphWithEmbeddedIsDeterministic`: dieselbe Eingabe zweimal →
    `reflect.DeepEqual` der Graphen.
  - `TestGraphWithEmbeddedFileNodeIsRejected`: eingebetteter Teil mit
    eigenem Dateiknoten `p.html` → `model.Validate` des Graphen meldet den
    doppelten Knoten (Regel „kein Dateiknoten“).
  - `TestCacheKeepsEmbeddedBodies` (query): `writeCache` mit eingebettetem
    Knoten `BodyText: "word"`, `readCache` → `BodyText == "word"`.
  - `TestTallyCountsEmbeddedParts`: `tally` mit `html`-Result
    (`ParseErrors: 0`, Knoten 2) und `js`-Teil (`ParseErrors: 1`),
    `parsed=true` → `per["html"] = {Files:1, Parsed:1}`,
    `per["js"] = {Files:1, Parsed:1, ParseErrors:1, ErrorFiles:["p.html"]}`.
    Zweiter Fall `parsed=false` → `Reused` statt `Parsed`.
- [ ] **Step 2: Run** `go test ./internal/code/resolve/ ./internal/code/query/` → FAIL.
- [ ] **Step 3: Implementieren.**
  - `GraphWith`: zuerst `flat := ` alle `Flatten()` hintereinander; danach
    überall `flat` statt `files` (Knoten, Gruppen, `paths`).
  - Cache: Bodies über `Flatten()` sammeln (IDs sind eindeutig, also eine
    Map je Eintrag); zurückschreiben per Index über
    `e.Result.Nodes[j]` und `e.Result.Embedded[i].Nodes[j]`, nicht über die
    Kopien aus `Flatten()`.
  - `tally` aus der Schleife in `Extract` herauslösen; `NoSymbol` bleibt in
    `Extract` und sieht nur `r.Nodes`.
- [ ] **Step 4: Run** → PASS; `go test ./internal/code/...` grün.
- [ ] **Step 5: Commit** `feat(graph): carry embedded parts through resolve, cache and stats`

### Task 4: Brückenschritt

**Files:**
- Create: `internal/code/resolve/bridge.go`
- Create: `internal/code/resolve/bridge_test.go`
- Modify: `internal/code/resolve/resolve.go` (`GraphWith`, `edgesOf`)
- Modify: `internal/code/resolve/gdscript.go` (`project` nutzt `nearestRoot`)

**Interfaces:**
- Consumes: `extract.Export`, `extract.Bridge`, `extract.Namespaces`.
- Produces (Paket `resolve`, unexportiert):
  - `type cardinality int` mit `cardOne`, `cardAll`
  - `type realmKind int` mit `realmRepo` (ein Realm, Wurzel `""`),
    `realmGodot` (nächste `project.godot`)
  - `type fallback int` mit `fallNone` (keine Kante), `fallRealmExtracted`,
    `fallRealmInferred`
  - `type nsRule struct { card cardinality; realm realmKind; unscoped, scopedMiss fallback }`
  - `var builtinRules = map[extract.Namespace]nsRule{` godot.class:
    `{cardOne, realmGodot, fallRealmExtracted, fallNone}`, godot.method und
    godot.signal: `{cardOne, realmGodot, fallNone, fallNone}` `}`
  - `type offer struct { extract.Export; file, lang string }` — ein Export
    samt Datei und Sprache des tragenden `Result`
  - `type handover struct { bridge extract.Bridge; lang string; unresolved int }`
    — `unresolved` ist der Index in der **gesamten** `Unresolved`-Liste von
    `GraphWith` oder `-1`
  - `type langOut struct { edges []model.Edge; unresolved []Unresolved; offers []offer; handovers []handover }`
    — Rückgabe von `edgesOf` (Indizes in `handovers[i].unresolved` relativ
    zu `langOut.unresolved`; `GraphWith` addiert den Versatz)
  - `func nearestRoot(roots []string, file string) (string, bool)` — `roots`
    längste zuerst; `""` passt auf jede Datei
  - `func godotRoots(paths []string) []string`
  - `func mergeLangs(outs []langOut) (edges []model.Edge, un []Unresolved, offers []offer, handovers []handover)`
    — hängt die Ergebnisse der Sprachen in der gegebenen Reihenfolge an und
    verschiebt jeden `handover.unresolved` (≠ `-1`) um die Zahl der
    `Unresolved` aller vorigen Sprachen
  - `func bridgeEdges(rules map[extract.Namespace]nsRule, paths []string, offers []offer, handovers []handover, existing []model.Edge, un []Unresolved) ([]model.Edge, []Unresolved)`
    — liefert die neuen Kanten und die umgeschriebene `Unresolved`-Liste
- `edgesOf` gibt `langOut` zurück; `GraphWith` sammelt Exports aus
  `Result.Exports` (als `offer` mit `Path`/`Language` des Result) und
  Brücken aus `Result.Bridges` (als `handover` mit `lang: ""`,
  `unresolved: -1`) und ruft `bridgeEdges(builtinRules, …)`.

Regeln (Spec „Regeln des Schritts“), in `bridgeEdges`:
1. Realm der Brücke aus `bridge.File`; kein Realm → keine Kante.
2. Kandidaten: Offers desselben Namensraums und Namens, deren Realm gleich
   ist oder die keinen Realm haben; bei `handover.lang != ""` ohne Offers
   dieser Sprache.
3. Mit `Scope`: nur Offers mit `file` in `Scope` → `extracted`; leer →
   `scopedMiss`. Ohne `Scope`: `unscoped`.
4. `cardOne` und ≠1 Treffer → keine Kante; `cardAll` → je Treffer eine.
5. Unbekannter Namensraum, Ziel == Quelle → keine Kante.
6. Zusammenfassen nach (Source, Target, Relation): kleinste Zeile,
   `extracted` vor `inferred`; Schlüssel aus `existing` → entfällt.
7. `Unresolved`: Kante → Eintrag entfernt; >1 Treffer → `ReasonAmbiguous`;
   0 → unverändert. Entfernen erst nach allen Übergaben.

- [ ] **Step 1: Failing tests** in `bridge_test.go`, alle mit eigener
  Testtabelle (`testRules` mit `test.one`, `test.all`, `test.inferred` usw.),
  Results von Hand:
  - `TestBridgeScopedHitIsExtracted`
  - `TestBridgeScopedMissFallsBack` (je `fallNone`, `fallRealmInferred`)
  - `TestBridgeUnscoped` (je `fallNone`, `fallRealmExtracted`, `fallRealmInferred`)
  - `TestBridgeCardOneAmbiguousGivesNoEdge`, `TestBridgeCardAllLinksEveryHit`
  - `TestBridgeUnknownNamespaceGivesNoEdge`, `TestBridgeToItselfGivesNoEdge`
  - `TestBridgeRealmAtRepoRoot` (`project.godot` im Wurzelverzeichnis)
  - `TestBridgeWithoutRealmGivesNoEdge`
  - `TestBridgeExportWithoutRealmCountsEverywhere` (`src/x.cpp`, `game/project.godot`)
  - `TestBridgeOtherRealmIsInvisible`
  - `TestHandoverSkipsOwnLanguage`
  - `TestBridgeMergesAndPrefersExtracted`, `TestBridgeYieldsToLanguageEdge`
  - `TestBridgeRewritesUnresolved` (Kante → entfernt; zwei Treffer →
    `ReasonAmbiguous`; keiner → `ReasonNone`)
  - `TestMergeLangsShiftsUnresolvedIndex`: zwei synthetische `langOut`, die
    erste mit zwei `Unresolved`, die zweite mit einem und einem Handover auf
    Index 0 → nach `mergeLangs` zeigt der Handover auf Index 2; ein Handover
    mit `-1` bleibt `-1`. (Über `GraphWith` ist der Versatz heute immer 0,
    weil nur GDScript `Unresolved` liefert.)
  - `TestBuiltinRulesCoverEveryNamespace`: jede `extract.Namespaces`-
    Konstante ist Schlüssel in `builtinRules` und umgekehrt
  - `TestNearestRoot` (längste zuerst, `""`, keine Wurzel)
- [ ] **Step 2: Run** `go test ./internal/code/resolve/ -run 'Bridge|Handover|Builtin|NearestRoot'` → FAIL.
- [ ] **Step 3: `bridge.go` implementieren**, `edgesOf`/`GraphWith`
  umstellen, `gdIndex.project` auf `nearestRoot(x.roots, file)` umstellen.
  Sortierung der Ergebniskanten bleibt die in `GraphWith`.
- [ ] **Step 4: Run** `go test ./internal/code/...` → PASS, alle
  bestehenden GDScript-, Python- und Go-Tests unverändert grün.
- [ ] **Step 5: Coverage** `go test -cover ./internal/code/resolve/` → 100 %.
- [ ] **Step 6: Commit** `feat(graph): resolve bridges between languages by namespace`

### Task 5: GDScript-Extraktor markiert Connection-Enden

**Files:**
- Modify: `internal/code/extract/gdscript/resource.go:101-108`
- Modify: `internal/code/extract/gdscript/gdscript.go:75`
- Modify: `internal/code/extract/gdscript/gdscript_test.go:187`, `internal/code/extract/all/all_test.go:38`
- Prüfen: jeder Test, der Connection-Rohkanten als ganze Structs vergleicht
  (`rg -n 'connection|RelationReferences' internal/code/extract/gdscript/*_test.go`);
  heute vergleicht keiner `references`-Rohkanten mit `Specifier`, die
  aufgelösten Connection-Kanten in `resolve/gdscript_lines_test.go` ändern
  sich durch `Member` nicht. Findet sich doch einer, bekommt er das Feld.

**Interfaces:**
- Produces: Connection-Rohkanten mit `Member: "method"` (Ende `to`/`method`)
  bzw. `Member: "signal"` (Ende `from`/`signal`); `version = "gdscript/6"`.

- [ ] **Step 1: Failing test** `TestConnectionEndsCarryMember`: Szene mit
  `[connection signal="hit" from="." to="HUD" method="on_hit"]` → zwei
  `references`-Rohkanten, `Name:"on_hit", Member:"method"` und
  `Name:"hit", Member:"signal"`. Versionstests auf `gdscript/6` ändern.
- [ ] **Step 2: Run** `go test ./internal/code/extract/...` → FAIL.
- [ ] **Step 3:** In der Schleife über `{to,method}`, `{from,signal}` das Feld
  `Member: model.Kind(end[1])` setzen; `version` auf `"gdscript/6"`.
- [ ] **Step 4: Run** → PASS; `go test ./internal/code/resolve/` unverändert grün.
- [ ] **Step 5: Commit** `feat(graph): mark which end of a Godot connection an edge is`

### Task 6: GDScript-Resolver meldet Exports und Übergaben

**Files:**
- Modify: `internal/code/resolve/gdscript.go` (`resolveGDScript`, `resolve`, `call`)
- Create: `internal/code/resolve/godot_bridge_test.go`

**Interfaces:**
- Consumes: `langOut`, `offer`, `handover` (Task 4), `RawEdge.Member` (Task 5).
- Produces: `resolveGDScript(files, paths) langOut`.

Verhalten (Spec „GDScript-Seite“):
- Offers: je Eintrag in `x.classes[root][name]` ein `offer{Export{godot.class, name, id}, file, "gdscript"}`.
- `extends`: `raw.Specifier == "" && raw.Receiver == "" && why == ReasonNone`
  → Handover `godot.class raw.Name`, Relation `extends`, mit Index des eben
  angelegten `Unresolved`-Eintrags.
- `calls` mit `raw.Name == "new"`, `raw.Receiver` ohne `.`, `chain` →
  `ReasonNone` → Handover `godot.class raw.Receiver`, Relation `calls`,
  `unresolved: -1`. Die `references`-Rohkante bei `.new()` wird nicht übergeben.
- `references` mit `Specifier`: `r := x.rootScript(p)`; ist `r` keine Datei
  dieses Resolvers (`_, own := x.aliases[r]; !own`) und `raw.Member` ist
  `"method"`/`"signal"` → Handover `godot.method`/`godot.signal raw.Name`,
  `Scope: []string{r}`, Relation `references`. Sonst wie bisher.
- `resolve` und `call` liefern dafür zusätzlich eine optionale Übergabe;
  Signatur nach Wahl, solange `keep` und die Reihenfolge der Kanten
  unverändert bleiben.

- [ ] **Step 1: Failing tests** in `godot_bridge_test.go`, mit
  `GraphWith` über von Hand gebaute `extract.Result`s (C# als Language
  `csharp`, C++ als `cpp`, Exports in `Result.Exports`; GDScript-Dateien
  über `gdscript.Language{}.File`). Ein Test je Fall aus Spec „Test für
  Godot auf C# und C++“, Namen:
  - `TestGodotConnectionSignalOnCSharp` (Signal `Hit` auf C#, Handler
    `on_hit` auf GDScript, beide `extracted`)
  - `TestGodotConnectionHandlerOnCSharp`
  - `TestGodotExtendsCppSameProject`
  - `TestGodotExtendsCppBesideProject` (`src/`, `game/`)
  - `TestGodotExtendsCppAtRepoRoot`
  - `TestGodotExtendsCppOtherProject` (`ReasonNone`)
  - `TestGodotExtendsCppAmbiguous` (ohne Realm + im Projekt; zwei im Projekt)
  - `TestGodotHandoverStaysInProject` (`class_name Only` in `a`, `extends Only` in `b`)
  - `TestGodotLooseClassNameIsNoExport` (`loose/only.gd`)
  - `TestGodotClassExportServesOtherLanguages` (C#-Bridge `godot.class Only`
    aus `Result.Bridges` → Kante auf GDScript-Klasse)
  - `TestGodotExtendsMissingInnerAfterPathIsNoHandover`
  - `TestGodotNewOnCppClass` (`calls`-Kante, kein `Unresolved`)
  - `TestGodotExtendsEngineClass` (`Node2D`, `ReasonNone`)
  - `TestGodotScriptWithoutProject` (keine Kante, `ReasonNone`)
- [ ] **Step 2: Run** `go test ./internal/code/resolve/ -run Godot` → FAIL.
- [ ] **Step 3: Implementieren.**
- [ ] **Step 4: Run** `go test ./internal/code/... ./internal/verify/...` → PASS;
  alle bestehenden `gdscript_test.go`- und `gdscript_lines_test.go`-Fälle
  unverändert grün (Beleg für „GDScript zu GDScript unverändert“).
- [ ] **Step 5: Coverage** `go test -cover ./internal/code/resolve/ ./internal/code/extract/... ./internal/code/query/` → 100 %.
- [ ] **Step 6: Commit** `feat(graph): hand Godot names GDScript cannot bind to the bridge step`

### Task 7: Abnahme, Wiki, README

**Files:**
- Modify: `docs/wiki/topics/code-graph.md`
- Modify: `README.md`, `README.de.md` (Roadmap)

- [ ] **Step 1: Graph vorher/nachher vergleichen.** Neues Binary
  `go build -o "$SCRATCH/loomux-new.exe" ./cmd/loomux`. Ziele: dieses Repo
  (Stand `master`, als `git archive` nach `$SCRATCH`) und
  `C:/Users/micro/Documents/#GIT/space` (drei Godot-Projekte, Szenen mit
  `[connection]`; ohne `.git` nach `$SCRATCH` kopiert, dort nie schreiben).
  Je Ziel **ein** Schnappschuss, zwei Kopien davon; je Kopie ein Binary mit
  `graph build --root <kopie>`. Vergleich von
  `.loomux/state/graph/wiring.json` ohne `meta.extractor` per PowerShell
  (`jq` fehlt in dieser Umgebung):
  `($j = Get-Content a.json -Raw | ConvertFrom-Json).meta.PSObject.Properties.Remove('extractor'); $j | ConvertTo-Json -Depth 100 -Compress`
  für beide Dateien, dann Stringvergleich.
  Expected: gleich. Abweichung → vor dem Merge klären, nicht
  wegdefinieren. `wiring.json` enthält kein `Unresolved`; „gleiche
  `Unresolved`“ belegen die unverändert grünen Resolver-Tests, so steht es
  auch im PR.
- [ ] **Step 2: Wiki** `code-graph.md`: Absatz „Jede Sprache löst in einem
  eigenen Namensindex auf …“ um Brücken, Namensräume, Realm und Übergaben
  erweitern (mit Link auf die Spec); „Was offen ist“ um C# ergänzen.
- [ ] **Step 3: Roadmap** in beiden READMEs: Zeile „C# in the code graph
  (Godot)“ / „C# im Code-Graphen (Godot)“ unter „Coming“, abhängig von den
  Brücken, Priorität offen für den Menschen; Zeile „C++ in the code graph“
  nennt die Brücken als Grundlage.
- [ ] **Step 4: Run** `go run ./cmd/loomux verify` (bzw. das Tor des Repos)
  → grün, `lint/wiki` und `lint/wiki-sources` ok.
- [ ] **Step 5: Commit** `docs: describe bridges between languages in the code graph`

Für alle Tasks: Meldet `lint/wiki-sources` im Hook `pin-stale` oder
`register-stale`, wird im selben Commit nach `AGENTS.md` verfahren (zitierende
Seiten lesen, korrigieren, `loomux wiki repin`). Das Changelog entsteht vor
dem Push über den Skill `release-pr`, nicht von Hand. Während ein
Commit-Hook läuft, keine weiteren Testläufe starten; nie `--no-verify`.
