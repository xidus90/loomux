# Stufe 1b-2 — die Überlebenden der Mutationsrunde

**Bezug:** [Paritätsliste Stufe 1b-2](stufe-1b-2.md), Abschnitt 9. Ein
Überlebender ist kein Fehlschlag; ein undokumentierter ist einer. Die Form ist
die von [`stufe-1b-1-geparkte-mutanten.md`](stufe-1b-1-geparkte-mutanten.md),
mit einem Unterschied: die Pakete hier sind die dieser Stufe, ihre Tests hat
diese Stufe geschrieben, also wird jeder Überlebende **bewertet** und nicht nur
geparkt.

**Stand des Codes:** gelaufen am 2026-09-18 mit `bin/loomux.exe dev mutants
<paket>` gegen Commit `0a4786c` („Give a search without a profile the cheap one,
as both other fronts do"). **Die Datei- und Zeilenangaben gelten für diesen
Commit** und wandern mit jeder späteren Änderung; wer sie auflöst, liest sie
dort (`git show 0a4786c:internal/serve/control.go`).

**`dev mutants` fährt nur die Suite des mutierten Pakets** (`go test
./<paket>/`), nicht die des ganzen Moduls. Wo ein Paket überwiegend durch die
Fälle in `internal/cli` gedeckt ist, überlebt fast alles — siehe
`internal/brain/answer` unten. Das ist keine Aussage über die Deckung, sondern
über die Reichweite dieses Werkzeugs.

## Bewertung in vier Klassen

- **äquivalent** — der Mutant tut dasselbe; kein Test kann ihn töten, und keiner
  sollte es versuchen.
- **ausgenommen** — die mutierte Zeile liegt in einer Funktion mit
  `//coverage:exempt`; der Grund dort nennt, was ohne Eingriff ins
  Betriebssystem unerreichbar ist. Derselbe Grund trägt hier.
- **anderswo gedeckt** — die Entscheidung wird von einer Suite außerhalb des
  Pakets gefahren, die `dev mutants` nicht startet.
- **Lücke** — ein Test kommt dort vorbei, prüft aber nicht, was die Zeile
  entscheidet. Nachrüstbar; hier notiert, damit es niemand für einen
  Äquivalenten hält.

---

## Reichweite dieser Runde — was gelaufen ist und was nicht

| Paket | Mutanten | ohne Mutant | Überlebende | vollständig? |
|---|---:|---:|---:|---|
| `internal/serve` | 193 | 30 | 10 | ja |
| `internal/serve/brain` | 50 | 20 | 1 | ja |
| `internal/lock` | 41 | 9 | 10 | ja (mit Plattformvorbehalt) |
| `internal/mcptools` | 0 | 0 | 0 | ja — das Paket hat keine Entscheidung |
| `internal/brain/answer` | 47 | 3 | 41 | ja |
| `internal/bridge` | 137 | — | 6 in den Mutanten 1–116, alle klassifiziert | **nein** — 21 fehlen; Grund und Abhilfe gleich darunter, die sechs im eigenen Abschnitt weiter unten |

### `internal/bridge` ist unvollständig, und der Grund ist eine Gabelbombe

Drei Läufe über `internal/bridge` endeten mit

```text
loomux dev mutants: fork/exec C:\Program Files\Go\bin\go.exe: Die Auslagerungsdatei ist zu klein, um diesen Vorgang durchzuführen.
```

Die drei Läufe im Einzelnen, damit prüfbar ist, worauf sich der Vergleich unten
stützt:

| # | Lauf | Arbeiter | Aufruf | `internal/bridge` war darin | kam bis | Überlebende in seiner Reichweite |
|---:|---|---:|---|---|---:|---:|
| 1 | A | 8 | alle sechs Pakete in einem Aufruf | das 3. von 6 Paketen | **116** von 137 | 6 |
| 2 | — | 4 | die vier übrigen Pakete | das 1. von 4 | kein Mutant gefahren: „the suite is not green before the round" — der beschädigte Baucache, siehe unten | — |
| 3 | B | 4 | dieselben vier, nach `go clean -cache` | das 1. von 4 | **107** von 137 | 6 |
| 4 | C | 2 | `internal/bridge` allein | das einzige | vor Mutant 107 abgebrochen; kein Bericht | — |

**Die Reihenfolge ist die der Spalte `#`, und sie ist Teil des Befunds.** Lauf A
war der erste auf einer ruhigen Maschine; B lief nach einem geleerten Baucache,
also mit kalten Bauten; C lief unmittelbar nach B, während dessen Waisen noch
ausliefen. **Dieser Vergleich ist damit nicht sauber**, und die Aussage unten
gilt nur in der Richtung, in der sie steht: die Arbeiterzahl **erklärt** den
Abbruch nicht — mehr Arbeiter kamen weiter —, aber sie ist mit der Reihenfolge,
dem kalten Cache und der Paketzahl (sechs gegen vier) verschränkt und deshalb
auch nicht freigesprochen. Was den Abbruch erklärt, ist die Gabelbombe darunter,
und die hängt an keiner der vier Größen.

**Die Arbeiterzahl ist nicht die Ursache:** Lauf A mit acht Arbeitern kam weiter
als Lauf B mit vieren. Gemessen während Lauf C:

```text
(Get-Process -Name "bridge.test").Count  ->  8783
```

Die Ursache ist `connect.go:122`:

```go
func (o Options) spawn() func(string) (bool, error) {
	if o.Spawn != nil {
		return o.Spawn
	}
	return defaultSpawn
}
```

Die Mutanten 89 bis 92 sind genau diese Zeile (a1 `true`, a1 `false`, a4, a3).
Wo sie den Spawner des Tests umgehen, greift `defaultSpawn` →
`serve.Spawn(stateDir, startProcess)` mit dem echten `(*exec.Cmd).Start`, und
`os.Executable()` ist das Testbinary: der Lauf startet `bridge.test.exe serve
--foreground`, entkoppelt und mit Breakaway. Das Testbinary ignoriert die
Argumente und fährt die Suite erneut — die wieder startet. Exponentiell, und
jedes Kind überlebt das getötete `go test`. Alle vier Mutanten werden
**getötet**; die Waisen bleiben und fressen die Commit-Grenze der Maschine auf,
bis `go.exe` nicht mehr startbar ist.

**Kein Fehler im Produktionscode.** `defaultSpawn` ist für den Betrieb richtig:
die Brücke soll den Dienst selbst starten. Es ist eine Grenze dieses Werkzeugs —
ein Mutant auf der Naht, die ein Test gerade dafür eingezogen hat, dass nichts
einen Prozess startet, entfernt genau diesen Schutz.

**Was dennoch gedeckt ist:**

- **Mutanten 1 bis 107** aus **Lauf A und Lauf B** — zwei unabhängige Läufe,
  **Zeichen für Zeichen dieselben sechs Überlebenden** (Tabelle unten).
- **Mutanten 108 bis 116** aus Lauf A allein; alle neun getötet. Zusammen also
  116 von 137 gefahren und sechs Überlebende.
- **Die Familie a2 vollständig**, in einem eigenen Lauf
  (`dev mutants internal/bridge --family a2`, Exit 0): 8 Mutanten, 3 ohne
  Mutant, 1 Überlebender — derselbe wie in A und B. `connect.go:122` hat weder
  `&&` noch `||` und erzeugt in dieser Familie keinen Mutanten, der Lauf ist
  also gefahrlos. In `connect.go` überlebt in a2 nichts.

**Wie verlässlich sind die Urteile 93 bis 116, wo die Maschine schon an ihrer
Grenze stand?** Nachgerechnet, und ausdrücklich nicht weiter, als die Rechnung
trägt. `GoTest` (`internal/dev/mutants/mutants.go:73-93`) kennt drei Ausgänge,
und zwei davon sind hier unbedenklich:

1. **Ein `go`, das gar nicht erst startet**, gibt einen **Fehler** zurück, und
   `Round` bricht die Runde damit ab, statt ihn zu zählen — genau das ist hier
   dreimal passiert.
2. **Ein `go test`, das am Bauen oder Binden scheitert**, druckt eine Zeile mit
   `\n# ` (die Paketüberschrift des go-Werkzeugs); `classify` (`:121-131`) liest
   das als **`BuildFailed`**, also „ohne Mutant", nicht als Tötung. Verschluckt
   wurde so keiner: unter den Mutanten 93 bis 116 steht kein einziges
   „no mutant" — die 24 sind 23-mal `killed` und einmal `SURVIVED`.

**Ein dritter Ausgang ist damit nicht ausgeschlossen, und er wäre der
unangenehme:** ein Testbinary, das **startet** und dann mitten im Lauf an Threads
oder an der Commit-Grenze stirbt, druckt keine `# `-Überschrift und endet mit
einem Status ungleich null. `classify` liest das als `Failed`, also als
**Tötung**. Genau dieser Ausgang ist auf einer Maschine an ihrer Grenze zu
erwarten, und die Beweislage oben deckt ihn nicht ab.

**Was das praktisch bedeutet:** die Mutanten 93 bis 107 sind durch **zwei**
unabhängige Läufe gedeckt, deren Urteile übereinstimmen — ein zufälliger
Ressourcentod müsste beide Male denselben Mutanten treffen. Die neun Mutanten
**108 bis 116 stehen auf Lauf A allein**, alle neun als `killed` verbucht:

```text
[108/137] killed (a1) connect.go:161  if err != nil {  ->  if false {
[109/137] killed (a4) connect.go:161  if err != nil {  ->  if !(err != nil) {
[110/137] killed (a3) connect.go:161  if err != nil {  ->  if err == nil {
[111/137] killed (a1) connect.go:165  if err != nil {  ->  if true {
[112/137] killed (a1) connect.go:165  if err != nil {  ->  if false {
[113/137] killed (a4) connect.go:165  if err != nil {  ->  if !(err != nil) {
[114/137] killed (a3) connect.go:165  if err != nil {  ->  if err == nil {
[115/137] killed (a1) connect.go:174  if state == nil {  ->  if true {
[116/137] killed (a1) connect.go:174  if state == nil {  ->  if false {
```

Wer diese neun nicht auf ein Wort nehmen will, fährt sie nach — sie liegen alle
vor den 21 fehlenden und kommen mit denselben zurück. Die Tötungsquote der 24 ist
im Übrigen keine Auffälligkeit gegen `internal/serve`, wo 153 von 163 wirklichen
Mutanten sterben.

**Was fehlt:** die 21 Mutanten 117 bis 137 der Familien a1, a3 und a4 — der Rest
von `connect.go:174` sowie die Entscheidungen in `held` (`connect.go:189`,
`:192`) und `waitForService` (`connect.go:218`, `:221`).

**Und sie sind mit dem heutigen Werkzeug erreichbar — die frühere Fassung dieser
Datei behauptete das Gegenteil und lag falsch.** Sie nannte nur zwei Wege, ein
`--skip <datei>:<zeile>`, das es nicht gibt, und einen anderen Vorgabe-Spawner,
also die zweite Baustelle, die die Entscheidung des Menschen gerade vermeiden
wollte. Der dritte Weg ist billiger als beide: **ein `TestMain` in
`internal/bridge`, das sofort endet, wenn `os.Args` das Wort `serve` enthält.**
Das Kind stirbt in dem Augenblick, in dem es startet, der Elternlauf scheitert
trotzdem — sein eingesetzter Spawner wurde ja nicht gerufen —, der Mutant wird
getötet, und es gibt keine Bombe. `serve` steht als eigenes `os.Args`-Element
nur da, wo `Spawn` es hingeschrieben hat: ein `-test.run=TestServeSomething` ist
ein Element und nicht gleich `"serve"`. Diese Zeile ist ein Vorschlag an den
Menschen und keine Änderung dieser Aufgabe; sie steht als eigene Zeile in
Abschnitt 8 der [Paritätsliste](stufe-1b-2.md).

## `internal/bridge` — 6 Überlebende in den Mutanten 1 bis 116 von 137

```text
[5/137]  SURVIVED  (a1) bridge.go:47   if o.Channel == privacy.ChannelCloud {  ->  if true {
[18/137] SURVIVED  (a1) bridge.go:148  if ctx.Err() != nil {  ->  if false {
[36/137] SURVIVED  (a3) bridge.go:216  if live := b.session.Load(); live != nil && live != stale {  ->  if live := b.session.Load(); live == nil && live != stale {
[50/137] SURVIVED  (a2) bridge.go:260  if current != nil && current != stale {  ->  if current != stale {
[53/137] SURVIVED  (a1) bridge.go:266  if err == nil {  ->  if true {
[98/137] SURVIVED  (a1) connect.go:144 if err != nil {  ->  if false {
```

| Mutant | Klasse | Grund |
|---|---|---|
| `bridge.go:47` → `if true` | **Lücke** | `Options.endpoint` gäbe immer die cloud-Adresse zurück. Die gemeinsame Vorrichtung der Suite setzt `Local` und `Cloud` auf **dieselbe** `Endpoint` (`bridge_test.go:80-81`), und der eine Fall mit zwei verschiedenen Adressen ist `TestTheCloudBridgeCallsTheCloudAddress` (`:464`) — der prüft die cloud-Seite, die der Mutant gerade richtig lässt. **Ein Spiegelfall `TestTheLocalBridgeCallsTheLocalAddress` tötet ihn**, und er ist die eine Lücke hier, die wirklich weh täte: sie wäre ein local-Wirt, der auf der cloud-Adresse landet — also genau die Trennung, für die es zwei Listener gibt |
| `bridge.go:148` → `if false` | **Lücke, ohne Folge für den Aufrufer** | `forward` würde einen abgebrochenen Kontext nicht mehr als „der Wirt hat die Frage zurückgenommen" erkennen und stattdessen neu verhandeln. Die Neuverhandlung scheitert am selben Kontext, der Aufrufer bekommt denselben `outage`-Fehler; verloren geht nur, dass für nichts ein Handschlag gefahren wird. Tötbar nur mit einem Test, der die Handschläge zählt |
| `bridge.go:216` → `live == nil && …` | **Lücke** | `connect`: nur ein Zustand unterscheidet den Mutanten, nämlich `live == nil` bei einem Aufrufer, der eine tote Sitzung mitbringt (`stale != nil`). Dann gibt der Mutant eine **nil-Sitzung** heraus statt neu zu verhandeln. Kein Fall stellt diesen Zustand her — er entsteht, wenn zwischen dem Leeren des Feldes und dem Wiederverbinden ein zweiter Aufruf durchkommt. In allen übrigen Zuständen fällt der Mutant auf `renew` zurück, und `swap` gibt dort dieselbe Sitzung heraus |
| `bridge.go:260` → `current != stale` | **Lücke, dieselbe** | `swap`, eine Ebene tiefer, exakt derselbe Zustand `current == nil && stale != nil` und dieselbe nil-Sitzung. Ein Fall, der zwei Aufrufe mit einer eben gestorbenen Sitzung verschränkt, tötet beide Zeilen auf einmal |
| `bridge.go:266` → `if true` | **äquivalent** | `swap` legte bei einem gescheiterten Handschlag die nil-Sitzung ins Feld statt nichts. `atomic.Pointer.Store(nil)` und ein Feld, das schon nil ist, sind ununterscheidbar: `b.session.Store(nil)` steht zwei Zeilen darüber bereits. Kein Test kann ihn töten |
| `connect.go:144` → `if false` | **äquivalent** | `ensure`: `state, err := serve.ReadState(...)`, dann `if err != nil { state = nil }`. `ReadState` gibt im Fehlerfall `(nil, err)` zurück (`internal/serve/state.go:55-58`), die Zuweisung setzt also nur noch einmal, was schon dasteht. Der Mutant ist wörtlich wirkungslos |

**Vier der sechs sind Lücken, und zwei davon sind dieselbe Lücke.** Wer hier
etwas nachrüstet, fängt mit `TestTheLocalBridgeCallsTheLocalAddress` an: er ist
einzeilig gegen den vorhandenen cloud-Fall und deckt die einzige der vier, die
eine Kanaltrennung betrifft.

---

## `internal/serve` — 10 Überlebende von 193 Mutanten (30 ohne Mutant)

```text
  (a1) control.go:181  if pid <= 0 {  ->  if false {
  (a3) control.go:181  if pid <= 0 {  ->  if pid < 0 {
  (a1) control.go:185  if err == nil {  ->  if true {
  (a1) serve.go:113  if err != nil {  ->  if false {
  (a1) serve.go:143  if err != nil {  ->  if false {
  (a1) serve.go:270  if !c.started {  ->  if true {
  (a1) state.go:76  if err != nil {  ->  if false {
  (a1) state.go:82  if err != nil {  ->  if false {
  (a1) state.go:109  if err != nil {  ->  if false {
  (a1) state.go:113  if err != nil {  ->  if false {
```

| Mutant | Klasse | Grund |
|---|---|---|
| `control.go:181` → `if false` | **Lücke** | `TestStopWithForceRefusesAPIDThatIsNotOne` schreibt `PID: -2` und prüft nur, dass die Meldung `-2` enthält. Ohne die Wache ruft `kill` `os.FindProcess(-2)`, das unter Windows scheitert, und die Meldung der zweiten Stufe nennt `-2` ebenfalls (`kill process -2, the PID in serve.json: …`). Der Fall bleibt grün, obwohl genau das Verhalten weg ist, für das die Wache existiert: **unter POSIX ist `-2` keine PID, sondern die Prozessgruppe 2**, und `os.Process.Kill` schützt nur 0 und -1. Nachrüstbar mit einer Zusicherung auf `which is not a process` |
| `control.go:181` → `if pid < 0` | **Lücke** | Derselbe Ort, andere Hälfte: kein Fall schreibt `PID: 0`. Ein zweiter Fall mit `PID: 0` schlösse beide Zeilen |
| `control.go:185` → `if true` | **äquivalent** | `os.FindProcess` scheitert unter Windows in dieser Verwendung nicht und ist unter POSIX per Definition unfehlbar; der Zweig ist auf keiner der beiden Plattformen erreichbar |
| `serve.go:113` → `if false` | **Lücke** | `lock.TryAcquire` gibt bei einem Fehler `held == false` zurück, der Lauf endet also ohnehin — aber mit `ErrAlreadyRunning` statt mit dem echten Grund. Kein Fall lässt `TryAcquire` scheitern; erreichbar wäre es mit einem Sperrpfad, der ein Verzeichnis ist. Die Folge wäre eine irreführende Meldung, kein zweiter Dienst |
| `serve.go:143` → `if false` | **ausgenommen** | Der Fehlerzweig von `BuildIdentity`, den die Ausnahme in `serve.go:105` benennt: „BuildIdentity fails only where the operating system cannot name the running program or the binary is gone while it runs" |
| `serve.go:270` → `if true` | **Lücke** | `channel.close` schlösse damit immer nur den Listener, statt einen gestarteten Server geordnet herunterzufahren. Kein Test unterscheidet die beiden: beide geben den Port frei und lassen das Verzeichnis los, und genau das prüfen die Fälle. Verloren ginge das Auslaufenlassen der Anfragen in Flug — messbar nur mit einer Anfrage, die während des Stopps läuft |
| `state.go:76`, `:82`, `:109`, `:113` → `if false` | **ausgenommen** | Die vier Fehlerzweige von `WriteState`, die die Ausnahme in `state.go:70` wörtlich aufzählt: kodieren, `CreateTemp`, `Write`, `Close`. Sie scheitern „only on a denied ACL, a full disk or a withdrawn volume" |

**Ein Nebenbefund dieser Runde, kein Überlebender.** Der Lauf hinterließ
`internal/serve/logs/serve.log` im Quellbaum. Ursache ist
`TestSpawnFailsWhenTheStateDirectoryCannotBeResolved`
(`internal/serve/spawn_internal_test.go:32`): der Test ersetzt `absolutePath`
durch einen Fehler, wechselt aber **nicht** in ein temporäres Verzeichnis.
Der Seam gibt `("", err)` zurück; hebelt ein Mutant die Fehlerprüfung direkt
danach aus, läuft `Spawn` mit `stateDir == ""` weiter, und `LogPath("")` ist
`logs\serve.log` relativ zum Arbeitsverzeichnis des Tests, also im
Paketverzeichnis. **Der beobachtete Ort beweist es:** mit dem unaufgelösten
`"state"` läge die Datei unter `internal/serve/state/logs/`, sie lag aber unter
`internal/serve/logs/`. Der Mutant wird getötet, die Datei bleibt.
**Kein Pfadfehler im Produktionscode:** `LogPath`
hängt korrekt am Zustandsverzeichnis, und der Testfall daneben
(`spawn_test.go:165`) macht es mit `t.Chdir(t.TempDir())` richtig vor. Dasselbe
im internen Test räumt es aus.

## `internal/serve/brain` — 1 Überlebender von 50 Mutanten (20 ohne Mutant)

```text
  (a2) tools.go:100  if command != "search" || len(notes) == 0 {  ->  if command != "search" {
```

| Mutant | Klasse | Grund |
|---|---|---|
| `tools.go:100` | **äquivalent** | `withFindings` gibt bei `search` ohne Befunde den Text unverändert zurück. Mit dem Mutanten läuft derselbe Aufruf in den Anhängeweg — der baut `lines := []string{text}`, hängt nichts an (`notes` ist leer) und gibt `strings.Join([]string{text}, "\n")` zurück, also genau `text`. Gleiche Ausgabe; kein Test kann ihn töten |

## `internal/lock` — 10 Überlebende von 41 Mutanten (9 ohne Mutant)

```text
  (a1) lock.go:54  if err != nil {  ->  if false {
  (a3) lock.go:72  if err := unlock(h.file); err != nil {  ->  if err := unlock(h.file); err == nil {
  (a1) lock.go:83  if err != nil {  ->  if false {
  (a1) lock_other.go:14  if errors.Is(err, unix.EWOULDBLOCK) {  ->  if true {
  (a1) lock_other.go:14  if errors.Is(err, unix.EWOULDBLOCK) {  ->  if false {
  (a4) lock_other.go:14  if errors.Is(err, unix.EWOULDBLOCK) {  ->  if !(errors.Is(err, unix.EWOULDBLOCK)) {
  (a1) lock_other.go:17  if err != nil {  ->  if true {
  (a1) lock_other.go:17  if err != nil {  ->  if false {
  (a4) lock_other.go:17  if err != nil {  ->  if !(err != nil) {
  (a3) lock_other.go:17  if err != nil {  ->  if err == nil {
```

| Mutant | Klasse | Grund |
|---|---|---|
| `lock_other.go:14` und `:17`, sieben Mutanten | **Plattform** | Die Datei trägt `//go:build !windows` und wird auf dieser Maschine gar nicht übersetzt. Jeder Mutant darin ist hier unsichtbar; die Runde sagt über `flock` nichts. Wer sie bewerten will, fährt sie auf POSIX. Das Windows-Gegenstück `lock_windows.go` hat keinen Überlebenden |
| `lock.go:54` → `if false` | **Lücke** | Der Fehlerzweig von `TryAcquire` in `WaitFree`. Kein Fall lässt `TryAcquire` scheitern; dieselbe Lücke wie `serve.go:113`, und mit demselben Mittel zu schließen (ein Sperrpfad, der ein Verzeichnis ist) |
| `lock.go:72` → `err == nil` | **äquivalent auf dem Erfolgsweg, Lücke auf dem Fehlerweg** | `Release`: gelingt `unlock`, schließt der Mutant die Datei im anderen Zweig und gibt `nil` zurück — dasselbe Ergebnis. Scheitert `unlock`, schluckt er den Fehler und gibt das `Close` zurück. Kein Test lässt `unlock` scheitern, also ist der Mutant hier nicht zu töten; töten könnte ihn nur ein Fake für `unlock` |
| `lock.go:83` → `if false` | **Lücke** | Der Fehlerzweig von `os.OpenFile` in `acquire`. Kein Fall gibt einen Pfad, der sich nicht anlegen lässt. Erreichbar mit einem Sperrpfad unter einer Datei statt einem Verzeichnis |

## `internal/mcptools` — 0 Mutanten

Das Paket baut fünf `*mcp.Tool` auf ersten Gebrauch und trifft dabei keine
einzige Entscheidung: kein `if`, also kein Mutant. Die Runde sagt hier nichts,
und es gibt nichts zu sagen.

## `internal/brain/answer` — 41 Überlebende von 47 Mutanten (3 ohne Mutant)

```text
  (a1,a4,a3) answer.go:110  if notice == nil {
  (a1,a4,a3) answer.go:143, :153, :164, :174, :183, :194, :198, :202, :212  if err != nil {
  (a1,a4,a3) answer.go:156  if scope == "all" {
```

**Nachtrag 2026-09-29: erledigt.** Eine Runde über den Stand mit dem
Datenschutz-Fix (`fix(brain): keep a local-only area hidden when an
enclosing area is asked`) ließ in `answer.go` noch 13 Mutanten leben, alle
„anderswo gedeckt“. Das Paket hat seitdem eigene Tests dafür (Commit `test:
kill the surviving mutants of the brain answers and the graph tools`); die
eigene Deckung stieg von 84 % auf 98,8 %, und die Runde über `answer.go`
lässt keinen Mutanten mehr leben. Der Rest dieses Abschnitts beschreibt den
Stand vom 2026-09-21.

**Alle 41 sind „anderswo gedeckt", und das ist die ganze Geschichte.**
`go test ./internal/brain/answer/` deckt **27,7 % der Anweisungen** des Pakets;
der Rest kommt aus `internal/cli` (die 71 Kommandozeilenfälle von 1b-1 und die
54 MCP-Fälle von 1b-2) und aus `internal/serve/brain`. `dev mutants` startet
nur die Suite des mutierten Pakets, also überlebt hier fast alles, was jene
Suiten sonst töten.

**Das ist kein Deckungsloch.** Das Tor ist `dev covergate` gegen ein Profil des
ganzen Moduls, und es ist grün; `answer.go` trägt keine `//coverage:exempt`.
Die Runde misst hier die Reichweite ihres eigenen Werkzeugs und nicht die
Qualität der Tests.

**Wer es sauber haben will**, hat zwei Wege, und beide sind Arbeit für eine
eigene Aufgabe: `dev mutants` ein `--suite <paket>` geben, das eine fremde Suite
als Orakel fährt, oder `internal/brain/answer` eine eigene Tabellensuite
schreiben, die die neun Fehlerweiterreichungen, den `notice`-Vorgabewert und die
`scope == "all"`-Gabel selbst fährt. Der zweite Weg verdoppelt, was der Korpus
bereits prüft; der erste ist die ehrlichere Antwort.
