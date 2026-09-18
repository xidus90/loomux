# Paritätsliste Code-Graph G2a

**Quelle:** `trailhq/Graft` @ `1e352a3` (MIT), `src/graph/extract.ts`,
`src/graph/resolve.ts`, `src/graph/fingerprint.ts`.
**Spec:** [2026-09-17-loomux-code-g2-design.md](../specs/2026-09-17-loomux-code-g2-design.md)
**Plan:** [2026-09-17-loomux-code-g2a.md](../plans/2026-09-17-loomux-code-g2a.md)
**Ledger:** `.superpowers/sdd/2026-09-17-loomux-code-g2a/progress.md`
**Regel:** Jede Zeile braucht eine Freigabe des Nutzers, bevor die Stufe als
fertig gilt.

**Stand 2026-09-18, freigegeben.** Der Nutzer hat am 2026-09-18 alle 44 Zeilen der Tabelle freigegeben, nachdem sie gegen den Code nachgerechnet, zwei Überlebende mit Tests getötet und die Begründungen berichtigt wurden. Das Tor ist grün auf `HEAD` dieses
Zweigs, Worktree `C:\Users\micro\Documents\#GIT\loomux-code-g2`, Branch
`code-g2`:

```
gofmt -l cmd internal
go vet ./...
go test ./... -count=1 -covermode=set -coverpkg=github.com/xidus90/loomux/... -coverprofile=coverage.out
go run ./cmd/loomux dev covergate --profile coverage.out
```

`gofmt` und `vet` schweigen, alle Pakete grün, `covergate` endet mit 0 — keine
Funktion im Profil liegt unter 100 % (`third_party/toml` fällt durch den vollen
Modulpfad gar nicht hinein).

**Korrektur am Torbefehl selbst.** Task 11s eigener Brief nennt
`-coverpkg=./...`. Damit sammelt `go test` auch `third_party/toml` ein — der
gevendorte, unter der eigenen Regel (`AGENTS.md`) von der Coverage-Pflicht
ausgenommene Baum — und `covergate` bricht mit über 130 ungedeckten Funktionen
fremden Codes ab. Der tatsächliche Gate-Befehl steht in
`ci/gate.sh` (von `.githooks/pre-commit` aufgerufen) und lautet `-coverpkg=github.com/xidus90/loomux/...`
(der volle Modulpfad, nicht `./...`); das ist der Befehl oben und der, den
dieser Lauf und jeder `git commit` auf diesem Zweig tatsächlich fahren. Dieselbe
Art Fehler wie die `Span.Lines`-Korrektur unten: ein Brief, der etwas anderes
behauptet als der Code tut.

Beide Binaries: `go build -o bin/loomux.exe ./cmd/loomux`. Eine zweite
Binary nennt der Brief ("beide Binaries bauen"), aber `.githooks/pre-commit`
kennt nur eine — es baut nach `ci/gate.sh` `bin/loomux.new.exe` und tauscht es
über `dev swap-binary` an die Stelle von `bin/loomux.exe`. Nichts in diesem Repository ruft
ein Binary unter `~/go/bin` (das ist eine Regel aus einem anderen Projekt,
`ultraloom`, die in der globalen `CLAUDE.md` des Nutzers steht und hier nicht
gilt). Eine Binary dieses Namens wurde versehentlich dorthin gebaut und wieder
gelöscht, bevor dieser Lauf endete.

## Die Runde

`go run ./cmd/loomux dev mutants internal/code/extract/golang internal/code/resolve internal/code/freshness`,
in einem Befehl über alle drei Pakete.

**Runde 1** zählte 314 Mutanten (163 über `extract/golang`, 94 über `resolve`,
57 über `freshness`), davon 69 ohne Kompilat (39 + 14 + 16), 201 getötet und
**44 überlebt** (26 + 12 + 6). 24 davon waren fehlende Tests; 19 neue Tests in
den bestehenden schwarzkistigen Testdateien der drei Pakete (keiner fabriziert
einen AST — jeder treibt echten Go-Quelltext oder einen echten Plattenzustand
durch die öffentliche API) erledigen sie. Ein Test, der auf dem Papier einen
Überlebenden hätte töten sollen (`extract.go:352`, siehe Zeile 9 unten),
erwies sich beim Nachrechnen als wirkungslos — er traf den falschen Fall; der
Mutant ist dennoch tötbar, das Kapitel unter der Tabelle rechnet das vor.

**Runde 2** zählte dieselben 314 Mutanten, dieselben 69 ohne Kompilat, **224
getötet und 21 überlebt** (13 + 6 + 2). Die Durchsicht dieser 21 gegen den Code
fand zwei, die doch tötbar sind: `extract.go:352` (a1, `if true`, Zeile 9) und
`fingerprint.go:122` (Zeile 40). Beide haben jetzt einen Test, dessen Wirkung
am jeweils von Hand angelegten Mutanten nachgewiesen ist (Test grün am
Original, rot am Mutanten). **Runde 3** (2026-09-18, derselbe Befehl) bestätigt
das: 226 getötet und **19 überlebt** (12 + 6 + 1), genau die 19 Äquivalenzen
der Tabelle. Diese 19 sind äquivalente
Mutanten: sie ändern den Code, aber nicht sein beobachtbares Ergebnis, und kein
Test kann sie töten. Die Tabelle unten führt alle 44 Fundstellen der Runde 1,
mit der Verfügung, die jede trägt — 25 mit dem nachgezogenen Test benannt, 19
mit der Rechnung, die Äquivalenz zeigt.

## Verfügungen dieser Stufe, die keine Mutation betreffen

Fünf Entscheidungen der Stufe stehen hier, weil sie die Lesart der Tabelle
tragen und nicht aus einer Mutante folgen:

- **Der Paketselektor.** Die eine Erweiterung gegenüber Grafts Go-Pfad, mit
  Verweis auf 6.2 der Spec: Graft lässt einen Selektor `pkg.Fn()` für Go ganz
  fallen, diese Stufe löst ihn innerhalb des benannten Zielpakets auf, und nur
  dann, wenn genau ein Kandidat dort steht ("so a same-named symbol elsewhere
  in the repo cannot become a false edge", Grafts eigene Regel für eine
  Referenz mit Spezifizierer). Gemessen auf diesem Repository (Task 8, erster
  Lauf gegen den echten Baum): von 5020 `calls`-Kanten überqueren 259 eine
  Paketgrenze. Ohne die Erweiterung wären alle 259 abwesend, und der
  Blast-Radius eines Refactorings bliebe über Paketgrenzen hinweg leer.
- **Baubedingungen werden nicht ausgewertet.** Ein `_windows.go`/`_other.go`-Paar
  ist zwei Definitionen, der Aufruf ist mehrdeutig und fällt (`one()` verwirft
  bei mehr als einem Kandidaten). Absichtlich: eine Auswertung würde den Graphen
  je nach Betriebssystem unterschiedlich aussehen lassen, und ein Graph, der
  vom Build-Rechner abhängt, ist kein Graph mehr, den zwei Beitragende
  vergleichen können.
- **Die Typsignatur** folgt Grafts Kommentar statt Grafts Code, mit Verweis auf
  5.2.1 der Spec: `type_spec` beginnt beim NAMEN, der Kopf endet beim
  `struct`-Schlüsselwort, die Signatur für `type Cache struct { ... }` ist also
  das bloße `"type Cache struct"` — ein Wert, den kein Test der Referenz
  festhält, aber der Kommentar der Referenz beschreibt genau das.
- **`testdata` ist der `skipDirs`-Liste beigetreten**
  (`internal/code/sourceset`), und das beantwortet die offene Frage aus 7.1 der
  Spec anders, als beide dort vorgesehenen Antworten es taten. Gemessen (Task
  10, in `docs/en/benchmarks.md` und `docs/de/benchmarks.md` mit Rohausgabe
  belegt): `freshness.Probe` fiel von ~72,5 ms auf ~3,0 ms bei unveränderter
  Dateizahl (254 sowohl vor als auch nach). Der Verzeichnislauf selbst war also
  nie das Problem — 1.910 Verzeichnisse insgesamt, 1.829 davon unter
  `testdata/` (3 Wurzeln plus 1.826 darunter) und keines davon eine Quelle, die
  `sourceset` je einsammelte (`.go.txt`-Fixturen, keine `.go`-Dateien) — und
  `git ls-files` war nicht nötig, um dieses Problem zu lösen, weil es kein
  Dateisystemproblem war.
- **Die Parsing-Untergrenze wurde korrigiert.** Die erste Messung (Task 10,
  Runde 1) zählte 468 Dateien "über 468 Dateien dieses Repositories" und
  behauptete eine Untergrenze von 44-46 ms; tatsächlich enthielt der damalige
  Checkout einen zweiten, vollständigen Checkout desselben Codes unter
  `.claude/worktrees/…` — 468 = 234 + 234, der Baum wurde zweimal geparst. Die
  korrigierte Zahl (`internal/code/extract/golang/parsefloor_bench_test.go`,
  254 Dateien, `SkipObjectResolution`) steht mit Befehl und Rohausgabe in
  beiden Benchmark-Dateien. Aufgenommen hier, weil dies der klarste Fall dieser
  Stufe ist, in dem eine Zahl solide aussah und es nicht war — derselbe Fehler,
  den ein Mutationslauf nicht findet, weil er keine Tests prüft, sondern eine
  Prosa-Behauptung.

## Die Tabelle

44 Fundstellen aus Runde 1, `extract/golang` zuerst, dann `resolve`, dann
`freshness`, in der Reihenfolge, in der der Werkzeuglauf sie meldete.

| # | Ort | Mutation | Verfügung | Freigabe |
|---|---|---|---|---|
| 1 | `extract.go:103` (a2) | `if fn, ok := decl.(*ast.FuncDecl); ok && len(nodes) == 1 {` → `… ok {` | Äquivalent: `declNodes` liefert für jeden `*ast.FuncDecl`-Zweig immer genau `[]model.Node{funcNode(...)}`, also `len(nodes) == 1` immer wahr, wenn `ok` wahr ist. Die Wache prüft eine Bedingung, die an dieser Stelle nie falsch sein kann | freigegeben 2026-09-18 |
| 2 | `extract.go:204` (a1) | `if d.Tok != token.TYPE {` → `if false {` | Äquivalent: Go's Grammatik koppelt `GenDecl.Tok` fest an die Art seiner Specs (`TYPE`→`*ast.TypeSpec`, `CONST`/`VAR`→`*ast.ValueSpec`, `IMPORT`→`*ast.ImportSpec`). Ist `Tok != TYPE`, enthält `d.Specs` nie ein `*ast.TypeSpec`; die nachfolgende Schleife filtert das ohnehin heraus (`ts, ok := spec.(*ast.TypeSpec); if !ok { continue }`) und liefert `nil` — mit oder ohne die frühe Wache | freigegeben 2026-09-18 |
| 3 | `extract.go:232` (a1) | `if owner != "" {` → `if true {` | Fehlender Test: `TestFileLeavesAMethodUnqualifiedWhenItsReceiverTypeCannotBeRead` (`extract_test.go`) — ein Empfängertyp, den `go/parser` zwar akzeptiert (`func (c (Cache)) Get() {}`, ein `*ast.ParenExpr`), den `receiverType` aber nicht liest, liefert `owner == ""`; der Methodenname muss unqualifiziert bleiben. Für die abschließende Durchsicht: `receiverType` hat damit eine Lücke gegenüber Grafts referenzieller Behandlung von Go-Empfängern — kein g2a-Fehler, aber ein Fund, siehe die abgegrenzten Kleinigkeiten unten | freigegeben 2026-09-18 |
| 4 | `extract.go:242` (a1) | `if d.Body != nil {` → `if true {` | Fehlender Test: `TestFileCutsASignatureAtTheHeaderForABodylessDeclaration` — eine körperlose Funktionsdeklaration (`func Stub(x int)`, gültiges Go für einen Assembly-Stub) hat `d.Body == nil`; der Mutant ruft `d.Body.Pos()` auf einem nil-`*ast.BlockStmt` und stürzt ab | freigegeben 2026-09-18 |
| 5 | `extract.go:297` (a2) | `if a < 0 \|\| b > len(source) \|\| a > b {` → `if a > b {` | Äquivalent: `a` und `b` stammen ausschließlich aus `fset.Position(...).Offset` über Positionen desselben geparsten Baums und derselben `source`-Zeichenkette; für jeden echten Aufrufer dieses Codes gilt immer `0 <= a, b <= len(source)`. `a < 0` und `b > len(source)` sind für jeden erreichbaren Aufruf falsch, die verbleibende Wache `a > b` bleibt unverändert | freigegeben 2026-09-18 |
| 6 | `extract.go:297` (a3, `a <= 0`) | `if a < 0 \|\| b > len(source) \|\| a > b {` → `if a <= 0 \|\| … {` | Äquivalent: `a == 0` hieße, ein Knotenspann begänne beim allerersten Byte der Datei — das ist immer das `package`-Schlüsselwort, nie eine Deklaration, deren Kopf oder Körper `slice()` anfordert. `a == 0` ist für keinen Aufruf dieser Funktion erreichbar, also ändert `<=` gegenüber `<` nichts | freigegeben 2026-09-18 |
| 7 | `extract.go:297` (a3, `b >= len(source)`) | `if a < 0 \|\| b > len(source) \|\| a > b {` → `if a < 0 \|\| b >= len(source) \|\| a > b {` | Fehlender Test: `TestFileHandlesASymbolEndingAtTheVeryLastByte` — eine Datei ohne Zeilenumbruch am Ende, deren letzte Deklaration exakt beim letzten Byte endet (`b == len(source)`), muss ihren Text noch bekommen; der Mutant schneidet ihn auf `""` | freigegeben 2026-09-18 |
| 8 | `extract.go:297` (a3, `a >= b`) | `if a < 0 \|\| b > len(source) \|\| a > b {` → `… \|\| a >= b {` | Äquivalent: bei `a == b` liefert das Original `source[a:a]`, also `""`, genau wie der Mutant; ob der Fall erreichbar ist, spielt keine Rolle | freigegeben 2026-09-18 |
| 9 | `extract.go:352` (a1, `if true`) | `if d.Recv == nil {` → `if true {` | Fehlender Test: `TestFileDoesNotLetAMethodNameShadowAnImportedPackage` (`edges_test.go`) — eine Methode, die wie ein importiertes Paket heißt (`func (T) strings() {}`), darf den Paketselektor `strings.ToUpper("x")` nicht verdecken; der Mutant deklariert den Methodennamen mit leerem Typ im Paketbereich, `callEdge` verwirft den Selektor, und die Rohkante mit `Receiver: "strings"` fehlt. Der erste Testentwurf (`TestFileDoesNotDeclareAMethodsBareNameInPackageScope`) traf den falschen Fall, siehe unten | freigegeben 2026-09-18 |
| 10 | `extract.go:352` (a1, `if false`) | `if d.Recv == nil {` → `if false {` | Fehlender Test: `TestFileDeclaresAPlainFunctionsBareNameSoASelectorOnItIsDropped` — eine gewöhnliche Funktion (`Recv == nil`) muss mit ihrem bloßen Namen im Paketbereich stehen, sonst wird ein späterer Selektor auf diesen Namen (`Store.Load()`) nicht verworfen, sondern als unaufgelöster Empfänger weitergereicht | freigegeben 2026-09-18 |
| 11 | `extract.go:352` (a4, Negation) | `if d.Recv == nil {` → `if !(d.Recv == nil) {` | Derselbe fehlende Test wie Zeile 10: die Negation deklariert nur Methoden statt nur Funktionen — dasselbe Symptom (`Store` bleibt undeklariert), derselbe Test tötet sie | freigegeben 2026-09-18 |
| 12 | `extract.go:352` (a3, `!=`) | `if d.Recv == nil {` → `if d.Recv != nil {` | Dieselbe Mutation wie Zeile 11 in anderer Schreibweise, derselbe Test tötet sie | freigegeben 2026-09-18 |
| 13 | `extract.go:385` (a1) | `if fn.Recv != nil {` → `if true {` | Äquivalent: Für eine gewöhnliche Funktion ist `fn.Recv == nil`; `receiverVar(nil)` und `receiverType(nil)` liefern beide `""` (eigene Nil-Wachen in `ident.go`), und `scope.declare("", "")` ist ein No-Op durch die eigene Wache in `scope.go:40` (`if name == "" \|\| name == "_" { return }`). Die zusätzliche Ausführung des Mutanten hat für jede Funktion ohne Empfänger keine Wirkung | freigegeben 2026-09-18 |
| 14 | `extract.go:415` (a1) | `if s.Tok == token.DEFINE {` → `if true {` | Fehlender Test: `TestFileRebindsALocalOnlyOnADefiningAssignment` — `x = Impl{}` (eine schlichte Zuweisung, kein `:=`) darf `x`s deklarierten Typ nicht neu binden; der Mutant behandelt jede Zuweisung wie eine definierende und bindet `x` fälschlich auf `Impl` um, statt `I` zu behalten | freigegeben 2026-09-18 |
| 15 | `extract.go:422` (a1) | `if i < len(s.Rhs) {` → `if true {` | Fehlender Test: `TestFileDoesNotReadPastTheRightHandSideOfAMultiValueDefine` — `a, b := f()` hat eine Rhs-Ausdruck für zwei Lhs-Namen; der Mutant liest `s.Rhs[1]` und stürzt mit Indexüberlauf ab | freigegeben 2026-09-18 |
| 16 | `extract.go:422` (a3, `<=`) | `if i < len(s.Rhs) {` → `if i <= len(s.Rhs) {` | Derselbe fehlende Test wie Zeile 15: bei `len(s.Rhs) == 1` lässt `<=` denselben Index-1-Zugriff zu und stürzt gleich ab | freigegeben 2026-09-18 |
| 17 | `ident.go:46` (a3) | `if i := strings.LastIndex(name, "."); i >= 0 {` → `… i > 0 {` | Äquivalent: `exported` wird ausschließlich mit bloßen Go-Bezeichnern aufgerufen (`extract.go:253,283`: `name`, `ts.Name.Name`; `scope.go:103`: `rest`, der Rest nach dem Abschneiden von `"New"`), von denen keiner je einen Punkt enthält. `strings.LastIndex` liefert für jeden erreichbaren Aufruf `-1`, und `-1 >= 0` ist ebenso falsch wie `-1 > 0` — der Zweig, den die beiden Vergleiche unterscheiden würden (`i == 0`), wird nie erreicht | freigegeben 2026-09-18 |
| 18 | `ident.go:92` (a3) | `if len(out) > maxBodyChars {` → `if len(out) >= maxBodyChars {` | Äquivalent an der Grenze: bei `len(out) == maxBodyChars` (5000) liefert die reale Wache `out` unverändert (bereits 5000 Zeichen lang), der Mutant schneidet auf `out[:5000]` — was für einen bereits genau 5000 Zeichen langen String byteidentisch mit `out` selbst ist. Beide Zweige liefern dasselbe Ergebnis an dieser Grenze | freigegeben 2026-09-18 |
| 19 | `scope.go:40` (a1, `if false`) | `if name == "" \|\| name == "_" {` → `if false {` | Äquivalent, mit Zeile 20/21: `declare("", …)` und `declare("_", …)` werden real erreicht (unbenannter Empfänger, `_`-Empfänger, `_ :=`-Bindungen), aber `lookup()` wird nie mit `name == ""` aufgerufen (kein `*ast.Ident.Name` ist je leer), und mit `name == "_"` nur für nicht übersetzbaren Quelltext: `_.Foo()` parst, scheitert erst an der Typprüfung. Dort verwirft der Mutant die Rohkante, die das Original mit `Receiver: "_"` ausgibt — `resolve` verwirft sie aber ohnehin (der Alias `_` bindet nichts, keine Paketklausel heißt `_`). Äquivalent für jeden übersetzbaren Eingang und für jeden Graphen | freigegeben 2026-09-18 |
| 20 | `scope.go:40` (a2, `name == ""`) | `if name == "" \|\| name == "_" {` → `if name == "" {` | Dieselbe Rechnung wie Zeile 19: `"_"` wird jetzt geschrieben statt gefiltert, aber ebenso nie gelesen | freigegeben 2026-09-18 |
| 21 | `scope.go:40` (a2, `name == "_"`) | `if name == "" \|\| name == "_" {` → `if name == "_" {` | Dieselbe Rechnung wie Zeile 19: `""` wird jetzt geschrieben statt gefiltert, aber ebenso nie gelesen | freigegeben 2026-09-18 |
| 22 | `scope.go:58` (a3) | `for i := len(s.frames) - 1; i >= 0; i-- {` → `… i > 0; i-- {` | Fehlender Test: `TestFileResolvesAPackageLevelReceiverFromTheOutermostScopeFrame` — ein Name, der nur auf Paketebene (Frame 0) steht, muss aus einer Funktion heraus noch auffindbar sein; der Mutant überspringt Frame 0 und findet ihn nicht mehr | freigegeben 2026-09-18 |
| 23 | `scope.go:99` (a1, `if false`) | `if len(fn) <= len(prefix) \|\| fn[:len(prefix)] != prefix {` → `if false {` | Fehlender Test: `TestFileDoesNotReadPastAShortConstructorCandidate` — `Ne()` ist kürzer als `"New"`; der Mutant liest trotzdem `fn[len(prefix):]` und stürzt mit Indexüberlauf ab | freigegeben 2026-09-18 |
| 24 | `scope.go:99` (a2, Längenklausel allein) | `… {` → `if len(fn) <= len(prefix) {` | Fehlender Test: `TestFileBindsAConstructorOnlyWhenTheWholeNewPrefixMatches` — `FooBar()` ist länger als `"New"`, beginnt aber nicht damit; der Mutant bindet trotzdem auf `"Bar"` um, statt die Bindung zu verwerfen | freigegeben 2026-09-18 |
| 25 | `scope.go:99` (a2, Präfixklausel allein) | `… {` → `if fn[:len(prefix)] != prefix {` | Derselbe fehlende Test wie Zeile 23: bei `fn == "Ne"` (kürzer als `"New"`) liest die verbleibende Klausel allein `fn[:3]` und stürzt ab | freigegeben 2026-09-18 |
| 26 | `scope.go:99` (a3, `<`) | `if len(fn) <= len(prefix) \|\| … {` → `if len(fn) < len(prefix) \|\| … {` | Äquivalent: bei `fn == "New"` genau (`len(fn) == len(prefix)`) liefert die reale Wache über die `<=`-Klausel `("", false)`. Der Mutant überspringt diese Klausel (`3 < 3` falsch), fällt auf die Präfixklausel (`"New" != "New"` auch falsch), erreicht also ebenfalls `rest := fn[3:]` — für `fn == "New"` ist das `""`, und `exported("")` ist `false`, also liefert auch der Mutant `("", false)`. Beide Zweige stimmen an dieser Grenze überein | freigegeben 2026-09-18 |
| 27 | `gomod.go:41` (a1, `if false`) | `if path.Base(rel) != "go.mod" {` → `if false {` | Fehlender Test: `TestModulesNeverReadsAFileThatIsNotNamedGoMod` — ein Dateiname, der nicht `go.mod` heißt und nicht auf der Platte liegt, muss ohne Lesen übersprungen werden; der Mutant versucht `os.ReadFile` und liefert einen Fehler statt eines leeren Ergebnisses | freigegeben 2026-09-18 |
| 28 | `gomod.go:60` (a3) | `sort.Slice(out, func(i, j int) bool { return len(out[i].Path) > len(out[j].Path) })` → `… >= …` | Äquivalent (Task 5s abgegrenzte Kleinigkeit, hier bestätigt statt nur behauptet): ein Gleichstand zweier Modulpfade gleicher Länge kann vorkommen, aber zwei so gleichlange, verschiedene Pfade können nie beide auf denselben Importpfad passen — `importDir` verlangt Gleichheit oder ein Präfix mit `/`, und ein Präfix-Treffer erzwingt eine ECHT kürzere Länge für das kürzere Modul. Bei Gleichstand ist also entweder der Importpfad für beide gleich unpassend oder beide Module sind gar nicht betroffen. Einzige Ausnahme: derselbe Modulpfad in zwei Verzeichnissen (etwa ein zweiter Checkout im Baum) — dort entscheidet die Reihenfolge, welches `Dir` gewinnt, aber schon das Original legt sie nicht fest (`sort.Slice` ist nicht stabil); der Mutant ersetzt eine nicht zugesicherte Reihenfolge durch eine andere | freigegeben 2026-09-18 |
| 29 | `resolve.go:27` (a3) | `sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].ID < g.Nodes[j].ID })` → `… <= …` | Äquivalent: jede Knoten-ID ist eindeutig — `mintID` garantiert das innerhalb einer Datei, der Dateipfad als ID-Präfix garantiert es über Dateien hinweg. Zwei verschiedene Elemente der Liste vergleichen sich also nie gleich; `<` und `<=` definieren dieselbe totale Ordnung, wenn kein Gleichstand je auftritt | freigegeben 2026-09-18 |
| 30 | `resolve.go:31` (a3) | `return a.Source < b.Source` → `return a.Source <= b.Source` | Äquivalent: diese Zeile wird nur erreicht, wenn die Wache eine Zeile darüber (`if a.Source != b.Source`) bereits Ungleichheit festgestellt hat. Für ungleiche Werte sind `<` und `<=` dieselbe Aussage | freigegeben 2026-09-18 |
| 31 | `resolve.go:33` (a1, `if false`) | `if a.Relation != b.Relation {` → `if false {` | Fehlender Test: `TestGraphSortsEdgesByRelationBeforeTarget` — zwei Kanten mit gleicher Quelle, verschiedener Relation, deren Ziel-Reihenfolge der GEGENTEILIGEN Richtung der Relation-Reihenfolge folgt (`contains` zu `"z.go#Sym"` vs. `imports` zu `"aaa"`); nur wer vor dem Ziel nach der Relation sortiert, ordnet richtig. Der bereits bestehende Test `TestGraphSortsEdgesBySourceThenRelationThenTarget` deckt diesen Mutanten NICHT — seine beiden Ziele ("a.go#caller" vs. "fmt") liegen zufällig in derselben Richtung wie die Relationen, sortieren also unter beiden Regeln gleich | freigegeben 2026-09-18 |
| 32 | `resolve.go:34` (a3) | `return a.Relation < b.Relation` → `return a.Relation <= b.Relation` | Äquivalent, dieselbe Rechnung wie Zeile 30: erreicht nur bei bereits festgestellter Ungleichheit (`if a.Relation != b.Relation`) | freigegeben 2026-09-18 |
| 33 | `resolve.go:36` (a3) | `return a.Target < b.Target` → `return a.Target <= b.Target` | Äquivalent, aber nicht aus demselben Grund wie G1s `rank.go:131/133` — diese Zeile hat KEINE vorgeschaltete Ungleichheitswache, ein Gleichstand kann hier tatsächlich auftreten (zwei Kanten mit identischer Quelle, Relation UND Ziel, etwa zwei Aufrufe derselben Funktion im selben Funktionskörper). Die Äquivalenz folgt aus einer anderen Eigenschaft: für `RelationContains` gibt es pro Knoten genau eine Kante (jeder Knoten hat genau einen Ursprungsdatei-Container), Duplikate sind unmöglich. Für `RelationImports` liefert `importTarget` für dieselbe Spezifizierer-Zeichenkette immer dasselbe Ziel und dieselbe Konfidenz; zwei identische Importe erzeugen zwei byteidentische Kanten. Für `RelationCalls` ist die Konfidenz eine deterministische Funktion des Auflösungspfads (Selektor/Owner/perFile/global), und dieser Pfad hängt nur von `(raw.File, raw.Receiver, raw.Owner, raw.Name)` ab, nicht vom Aufrufort; zwei Rohkanten mit identischem `(Source, Relation, Target)` müssen also denselben Auflösungspfad genommen haben und tragen dieselbe Konfidenz. Das allein beweist aber nicht, dass gleiches Ziel gleichen Pfad heißt: importiert eine Datei dasselbe Paket zweimal (Punkt- und Normalimport), erreichen `F()` (über `global`, `inferred`) und `p.F()` (Selektor, `extracted`) dasselbe Ziel mit verschiedener Konfidenz; die Reihenfolge solcher Gleichstände legt aber schon das Original nicht fest (`sort.Slice`, nicht stabil). Von dieser Ausnahme abgesehen ist in jedem der drei Fälle ein Gleichstand nach `(Source, Relation, Target)` immer ein Gleichstand der GANZEN `model.Edge`, und die Reihenfolge zwischen zwei identischen Werten ist im JSON nicht beobachtbar | freigegeben 2026-09-18 |
| 34 | `resolve.go:71` (a1, `if true`) | `if !isTestFile(f.Path) && f.Package != "" {` → `if true {` | Fehlender Test: `TestGraphKeepsThePackageClauseFromTheNonTestFileWhateverOrderTheFilesArriveIn` — eine `_test.go`-Datei (`package foo_test`), nach der echten Datei (`package foo`) indiziert, darf die Paketklausel eines Verzeichnisses nicht überschreiben; der Mutant lässt auch Testdateien die Klausel setzen, und da sie später verarbeitet wird, gewinnt sie fälschlich | freigegeben 2026-09-18 |
| 35 | `resolve.go:71` (a2, `!isTestFile(f.Path)` allein) | `… {` → `if !isTestFile(f.Path) {` | Äquivalent: `f.Package` ist für jedes echte `golang.Result` nie leer — `extract.go:92` setzt es unbedingt auf `file.Name.Name`, und ein gültig geparstes Go-File hat immer eine Paketklausel. Die weggelassene Klausel `f.Package != ""` ist für keinen erreichbaren Aufruf falsch | freigegeben 2026-09-18 |
| 36 | `resolve.go:71` (a2, `f.Package != ""` allein) | `… {` → `if f.Package != "" {` | Derselbe fehlende Test wie Zeile 34: ohne die `isTestFile`-Klausel setzt auch die Testdatei die Klausel, dasselbe Symptom | freigegeben 2026-09-18 |
| 37 | `resolve.go:85` (a1) | `if x.perFile[n.Path] == nil {` → `if true {` | Fehlender Test: `TestGraphResolvesASameFileCallToAnEarlierFunctionInTheFile` — drei Funktionen in einer Datei (A, B, C), C ruft A auf; der Mutant setzt die Pro-Datei-Karte bei JEDEM Knoten neu auf, statt nur beim ersten, und verliert A und B, sodass der Treffer auf A nur noch über den (auch eindeutigen) globalen Index als `inferred` statt `extracted` gefunden wird | freigegeben 2026-09-18 |
| 38 | `resolve.go:94` (a1) | `if x.perDir[dir] == nil {` → `if true {` | Fehlender Test: `TestGraphResolvesASelectorToTheFirstIndexedFunctionOfThePackage` — zwei Funktionen in zwei Dateien desselben Pakets (First, Second), ein Paketselektor ruft First; derselbe Fehler wie Zeile 37, aber für die Pro-Verzeichnis-Karte: First wird verloren, der Selektor findet keinen Kandidaten mehr und die Kante fällt ganz weg (kein Fallback wie bei `perFile`) | freigegeben 2026-09-18 |
| 39 | `fingerprint.go:88` (a1) | `if err != nil {` → `if false {` (nach `json.MarshalIndent`) | Äquivalent, dieselbe Begründung wie die bestehende `//coverage:exempt` direkt über `Write` (Task 6, bestätigt von Task 7s Review): `record` besteht ausschließlich aus Strings, `int`s und `map[string]print`, wo `print` wiederum nur `int64`s und einen String hält — keine dieser Formen bringt `json.MarshalIndent` je zum Scheitern. Der Fehlerpfad ist für jeden Aufruf, den dieser Code selbst erzeugt, unerreichbar | freigegeben 2026-09-18 |
| 40 | `fingerprint.go:122` (a1) | `if trusted(p, f) {` → `if false {` | Fehlender Test: `TestProbeTrustsAFileWhoseSizeAndModTimeStillMatch` (`fingerprint_test.go`) — gleich lange, andere Bytes, Mtime per `os.Chtimes` exakt zurückgesetzt: das Original vertraut der Größe und Mtime, liest nicht und meldet nichts (die dokumentierte Grenze des schnellen Pfads); der Mutant hasht jede Datei und meldet `a.go` als `Changed`. Die erste Verfügung ("nur die Kosten ändern sich") übersah, dass sich genau dieser Fall deterministisch herstellen lässt | freigegeben 2026-09-18 |
| 41 | `fingerprint.go:160` (a2, `rec.Extractor != extractor` allein) | `if rec.Version != recordVersion \|\| rec.Files == nil \|\| rec.Extractor != extractor {` → `if rec.Extractor != extractor {` | Fehlender Test: `TestProbeTreatsARecordOfAnotherVersionAsUnknown` — ein Datensatz mit passendem `extractor`-Feld, aber `"version":99`, muss als UNKNOWN gelten; der Mutant akzeptiert ihn und vergleicht die Kartei gegen unsinnige Größen/Mtimes/Hashes einer anderen Schemaversion | freigegeben 2026-09-18 |
| 42 | `fingerprint.go:177` (a3, Size-Klausel geflippt) | `return p.Hash != "" && p.Size == f.Size && p.MTime == f.MTime` → `… p.Size != f.Size …` | Fehlender Test: `TestProbeAlwaysChecksAFileWhoseRecordedSizeDisagrees` — Inhalt länger, Mtime per `os.Chtimes` exakt zurückgesetzt: nur die Größe weicht ab. Real nicht vertraut (Hash-Bestätigung, `Changed`); der Mutant hält es fälschlich für vertrauenswürdig, weil er GENAU dann `true` liefert, wenn die Größe NICHT übereinstimmt | freigegeben 2026-09-18 |
| 43 | `fingerprint.go:177` (a3, MTime-Klausel geflippt) | `… p.Size == f.Size && p.MTime == f.MTime` → `… p.MTime != f.MTime` | Fehlender Test: `TestProbeAlwaysChecksAFileWhoseRecordedModTimeDisagrees` — gleich lange, aber andere Bytes, Mtime vorgerückt: nur die Mtime weicht ab. Dieselbe Art Fehler wie Zeile 42, gespiegelt auf die Mtime-Klausel | freigegeben 2026-09-18 |
| 44 | `fingerprint.go:177` (a3, Hash-Klausel geflippt) | `p.Hash != "" && …` → `p.Hash == "" && …` | Fehlender Test: `TestProbeAlwaysChecksAFileTheLastBuildNeverHashed` — ein Datensatz mit leerem `Hash` (wie `Write` ihn für eine Datei schreibt, die der letzte Build nicht lesen konnte), bei exakt übereinstimmender Größe und Mtime: real nie vertraut (das ist die dokumentierte Absicht der Klausel), der Mutant vertraut GENAU dann, wenn der Hash leer ist — das Gegenteil | freigegeben 2026-09-18 |

**Zusammenfassung.** 44 Zeilen, 25 mit einem benannten, nachgezogenen Test
(Zeilen 3, 4, 7, 9, 10-12 [ein Test für drei Mutanten], 14, 15+16 [ein Test für
zwei], 22, 23+25 [ein Test für zwei], 24, 27, 31, 34+36 [ein Test für zwei], 37,
38, 40, 41-44), 19 mit vorgerechneter Äquivalenz. Kein Überlebender bleibt ohne Verfügung.

## Die eine Fehleinschätzung, aufgeklärt

Zeile 9 der Tabelle (`extract.go:352`, `if true`) wurde zunächst mit einem
eigens dafür geschriebenen Test angegangen: ein Paket-Var `Get` vom Typ
`*Cache`, gefolgt von einer Methode `(c *Cache) Get()`, sollte zeigen, dass der
Mutant den Var-Eintrag mit einem leeren Typ überschreibt. Der Test läuft grün —
aber ein `println`-gestütztes Nachmessen am tatsächlich mutierten Binary zeigte,
dass er den Mutanten NICHT tötet: `walkCalls`s erste Schleife (die diese Zeile
enthält) läuft vollständig, bevor die zweite beginnt, und die zweite Schleife
ruft für JEDE paketglobale `GenDecl` (auch für `var Get *Cache`) `collect()`
auf, das über `ast.Inspect` denselben `*ast.ValueSpec` ein zweites Mal
durchläuft und `Get -> "Cache"` erneut bindet — unabhängig davon, was die erste
Schleife für Methodennamen getan hat. Die zusätzliche Bindung, die der Mutant
für eine Methode erzeugt, wird also immer überschrieben, bevor sie je gelesen
wird, und zwar aus einem Grund, der mit dem Mutanten nichts zu tun hat: jede
`GenDecl` wird zweimal besucht, einmal in jeder Schleife.

Daraus folgte zunächst der Schluss, Zeile 9 sei äquivalent. Das war falsch:
das Nachmessen widerlegte nur den gewählten Fall (Paket-Var), nicht die
Tötbarkeit. Ein Methodenname, dem KEINE `GenDecl` gleichen Namens
gegenübersteht, wird von der zweiten Schleife nicht überschrieben — heißt die
Methode wie ein importiertes Paket (`func (T) strings()`), verdeckt der Mutant
den Selektor `strings.ToUpper()`, und `callEdge` verwirft ihn als "deklariert,
aber ohne Typ". `TestFileDoesNotLetAMethodNameShadowAnImportedPackage` hält
genau das fest; grün am Original, rot am Mutanten. Zeile 9 ist also ein
fehlender Test, kein äquivalenter Mutant. Der erste Test
(`TestFileDoesNotDeclareAMethodsBareNameInPackageScope`) bleibt im Baum: er
dokumentiert korrektes Verhalten (ein Paket-Var bleibt sichtbar, auch wenn eine
spätere Methode denselben Namen trägt), tötet aber diesen Mutanten nicht.

## Abgegrenzte Kleinigkeiten aus dem Ledger

Für die abschließende Durchsicht des ganzen Zweigs, wörtlich aus dem Ledger
übernommen, nicht neu formuliert:

- **Task 1** (`internal/code/pagerank`, `internal/code/blast`): beide bauen
  `Meta{Version: 1}`-Fixturen ohne `BodyHash`. Harmlos, solange keines der
  beiden `Validate` aufruft — beide nehmen einen `*model.Graph` direkt entgegen.
  Wer eines der beiden künftig über `Decode` führt, muss diese Fixturen heben.
- **Task 1** (`atoi`, `internal/code/model`): akzeptiert kein Vorzeichen und
  wird auch nicht danach gefragt; der einzige Aufrufer reicht durchgezählte
  Ziffernteilstrings, also keine erreichbare Lücke — aber ein zweiter Aufrufer
  bräuchte eines.
- **Task 3** (`internal/code/extract/golang`): vier der im weißen Kasten
  hinzugefügten Tests decken defensive Zweige ab, die laut ihrem eigenen
  Kommentar kein Eingang durch `File` je erreichen kann — `markCovered` auf
  einer unparsbaren Spanne, `declNodes` auf einem `BadDecl`, `declNodes` auf
  einem Nicht-`TypeSpec` innerhalb eines `TYPE`-`GenDecl`, `slice` auf einer
  umgekehrten Spanne. Sie beweisen, dass die Funktion an einem fabrizierten AST
  nicht abstürzt, nicht dass das Verhalten richtig ist — bessere Kandidaten für
  `//coverage:exempt` oder Löschung als für einen Test, aber brief-konform (der
  Brief verlangt einen Test vor einer Ausnahme).
- **Task 4** (`internal/code/extract/golang`): dieselbe Art defensiver, am
  fabrizierten AST getesteter Zweig zählt nach Task 4 sieben im ganzen Paket
  (vier aus Task 3, drei aus Task 4: `walkCalls` mit einem `*ast.FuncDecl`, das
  in `owners` fehlt; `collect` mit einer Nicht-`*ast.Ident`-Linkseite in einer
  `:=`-Zuweisung; `scope.lookup` auf einem nie deklarierten Namen). Jeder
  beweist nur, dass die Funktion an einem Eingang, den der Parser nicht
  erzeugen kann, nicht abstürzt.
- **Task 5** (`internal/code/resolve`, `Modules`): sortiert längste
  Modulpfade zuerst mit `sort.Slice` über `len(Path)`, weder stabil noch nach
  Byte-Ordnung. Harmlos, weil Gleichstände zwischen zwei tatsächlich
  konkurrierenden Modulen nicht vorkommen können — siehe Zeile 28 der Tabelle
  oben, wo das jetzt vorgerechnet statt nur behauptet ist.
- **Task 6** (`internal/code/store`, `TestWriteFailsWhenTheTempFileCannotBeWritten`):
  dupliziert production's temp-name formula (`final + "." + pid + ".tmp"`)
  statt sie abzuleiten, und behauptet nur `err != nil` — ein gelöschtes
  `os.Remove(tmp)` in diesem Fehlerpfad ginge unbemerkt. Task 11s
  Mutationsrunde deckt `store` nicht ab, das ist also die letzte Gelegenheit,
  es zu entscheiden: entweder behauptet der Test, dass die Temp-Datei weg ist,
  oder die Aufräumzeile bleibt unbewiesen.
- **Task 6** (Commit-Betreff): der Fix-Commit "Fix Write's coverage exemption:
  name every uncovered arm, cover WriteFile for real" ist ein
  doppelpunktverbundener zusammengesetzter Satz statt des schlichten
  Imperativsatzes, den der übrige Zweig verwendet. Nicht nachträglich
  geändert, weil die Entscheidung mit der Frage nach den zwei
  `Co-Authored-By`-Commits gekoppelt war (siehe Ledger) und diese Frage beim
  Nutzer lag.
- **Task 8** (`internal/cli`, `report`): der `unresolved`-Zähler läuft über
  JEDE Kante, deren Ziel kein Knoten ist, statt auf `imports` zu filtern,
  während sein Label "unresolved import targets" sagt. Auf dem echten Graphen
  (Task 8, erster Lauf) sind beide Mengen identisch — alle 1110 solchen Kanten
  sind `imports` — aber das gilt nur, weil `resolveEdge` `contains`- und
  `calls`-Kanten immer echte Knotenziele gibt und nur `importTarget` ein
  synthetisches Ziel prägen kann; eine künftige Relation könnte das Label
  stillschweigend falsch machen. Eine Zeile (`e.Relation ==
  model.RelationImports`) würde den Code das sagen lassen, was das Label
  behauptet.
- **Task 9** (`internal/cli/graph.go`, `graphCheck`): nichts behauptet, dass
  ein fremder Graph die erneute Extraktion vermeidet, nur dass FOREIGN
  gemeldet wird. Bestätigt als bekannte, black-box-inhärente Grenze, nicht
  als Fehler. (Zeile 40 der Tabelle wurde zunächst mit derselben Art Grenze
  begründet; dort war der Unterschied doch beobachtbar und hat jetzt einen
  Test.)

## Was G2a offen lässt

Für G2b und G4, damit sie es nicht suchen müssen:

- **Die Abfrage** — `lexicon`, `ask`, die Beiakte — ist G2b.
- **`--lsp` gegen `gopls`** ist die optionale Folgestufe von Abschnitt 14 der
  Spec.
- **Die Normalisierung von `Depth` und `Direction`** (ein negativer,
  nicht-`All`-`Depth`-Wert; eine `Direction` außerhalb von `In`/`Out`) gehört
  zu `graph callers`, G4 — dieselbe Verfügung, die G1 für `blast.Reach` schon
  traf, gilt hier unverändert weiter: keine Nutzereingabe erreicht diese Typen,
  bevor die CLI-Verdrahtung dafür steht.
- **Die `check`-Lanes `graph-freshness` und `blast-audit`** und die
  Post-Edit-Hook-Anbindung sind G4.
- **Eine nicht parsbare Datei bricht den ganzen Bau ab** (`buildGraph` in
  `internal/cli/graph.go`). Für G2a ist das richtig: ein Bau ist ausdrücklich,
  und ein halber Graph, still geschrieben, ist schlechter als ein benannter
  Fehler. Für **G4 ist es eine Entscheidung, keine Erbschaft**: im Hook-Pfad
  ist eine Datei mitten in einer Bearbeitung der Normalfall, und dort ließe
  dieses Verhalten sowohl `build` als auch `check` an ihr scheitern. Die Spec
  schweigt dazu. G4 entscheidet bewusst — überspringen und zählen, oder
  abbrechen wie hier.
- **964 doppelte Kantentupel** im echten Graphen dieses Repos (gleiche Quelle,
  Relation, Ziel und Konfidenz, eine je Aufrufstelle). Das ist das Verhalten der
  Referenz und die Spec sagt nichts dazu, aber **G2b erbt es**: `pagerank`
  gewichtet einen mehrfach vorkommenden Aufruf entsprechend stärker, und
  `blast` liefert denselben Treffer mehrfach. Auch das ist dort zu entscheiden
  und nicht zu übernehmen.

Beide letzten Punkte stehen hier, weil sie sonst nur im Ausführungs-Ledger
stünden, und das ist Wegwerf-Arbeitsverzeichnis: eine Verfügung, die mit dem
Arbeitsverzeichnis stirbt, war eine Entscheidung im Verborgenen.
