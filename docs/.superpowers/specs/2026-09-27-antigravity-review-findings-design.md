# Befunde des Antigravity-Reviews beheben — Design

Datum: 2026-09-27 · Zweig: `fix/antigravity-review-findings` von `origin/master`
(a7805df8, v4.0.0) · Status: Entwurf, Durchsicht offen

## Ausgangslage

Das Code-Review (`/code-review max`) des Zweigs `fix/antigravity-review` gegen
`origin/master` hielt 22 von 24 Befunden (zwei widerlegt: leeres
`LOCALAPPDATA`, der Helfer `writeEdit`). Die sechs Commits des Zweigs liegen
seit v2.14.2 auf master. Eine Nachprüfung auf a7805df8 ergab: alle Befunde
gelten dort unverändert, die Zeilen stimmen mit dem alten Zweigstand
c0b0bb6f überein. Die Befunde mit Belegen stehen im Scratchpad der Sitzung;
diese Spec nennt jeden mit seiner Kennung (C1–C21, D1–D3, E1–E3).

## Ziel

Jeder gehaltene Befund ist behoben oder mit Grund als reiner Kommentar- oder
Dokufix erledigt. Danach gilt:

- Der Guard fällt für jede Antigravity-Eingabe, die er nicht sicher lesen
  kann, geschlossen aus.
- `init` bringt `manage_task` in bestehende Installationen, und die
  versionierte `.agents/hooks.json` entspricht `hostfile.Entries`; beides
  hält ein Test fest.
- post-edit schreibt seinen Kontext über die hosts-Naht, und jeder Hinweis
  auf Übersprungenes erreicht das Modell auch bei Exit 2.
- Kommentare, Nutzerdoku (en/de) und Arbeitspapiere beschreiben, was der
  Code tut.

## Entscheidungen des Nutzers (2026-09-27)

1. **C5 Tipp-Eingaben:** nur ganze Zeilen, zustandslos. Kein Puffer je
   `TaskId`.
2. **C2 Bestand:** `init` hängt einen Block für die fehlenden Werkzeuge an.
3. **C13 Antigravity bei Exit 0:** stdout bleibt verworfen wie heute; ob agy
   `injectSteps` auf PostToolUse liest, bleibt als ungemessen vermerkt.
   Nachtrag 2026-09-28: agy 1.2.12 liest sie; `hosts.Answer` reicht
   post-tool-use-stdout bei Exit 0 seither durch (`parity/stufe-4a-2.md`).
4. **C21 Changelog:** der Fixed-Eintrag berichtigt die Aussage von v2.14.2
   ausdrücklich.

Im Entwurf vorgelegt und ohne Einwand geblieben: C7 `Revive` nur beim ersten
Aufruf (4.1), C8 nur Kommentare (2.3), C11 eine Werkzeugtabelle (1.1), D3
`invocationNum` auch als String (4.2). Neu in dieser Spec und eigens zur
Durchsicht: E1 kehrt eine bewusste Entscheidung um (1.5), und Codex endet in
post-tool-use mit 1 (3.3).

## Nicht-Ziele

- Kein Puffer für Tastenfolgen, keine Nachbildung des Zeileneditors eines
  Terminals.
- Kein unlink für agy (C7 Option c): agy hat kein SessionEnd, und Stop
  feuert nach jedem Zug.
- Keine Änderung an der Schreibschranke `internal/brain/guard` außer einem
  Test (C10); ihre Parität zum Python-Original bleibt.
- Kein Umbau von `formatBlock` für Blöcke mit flachem Befehl und hooks-Liste
  (E2 Option 2): loomux schreibt diese Form nie.
- `CHANGELOG.md` wird nicht von Hand geändert; die Einträge stehen im Block
  `## Changelog` des PR-Rumpfs, den der Release-Lauf überträgt.

## 1 Guard (`internal/hooks/guard.go`, `internal/hooks/pretool.go`)

### 1.1 Eine Werkzeugtabelle (C11)

`commandTools`, `judgedOrRefused` und `quietActions` werden eine Tabelle:

```go
type commandTool struct {
	keys         []string // argument names the line may stand under, matched without case
	lineOptional bool     // a call without a line passes: Claude's shells, whose tool always carries one
	whole        bool     // the value is a whole command line, not keystrokes typed into an open terminal
	quiet        []string // Actions whose calls carry no line
}

var commandTools = map[string]commandTool{
	"Bash":               {keys: []string{"command"}, lineOptional: true, whole: true},
	"PowerShell":         {keys: []string{"command"}, lineOptional: true, whole: true},
	"run_command":        {keys: []string{"CommandLine", "command_line"}, whole: true},
	"send_command_input": {keys: []string{"Input"}},
	"manage_task":        {keys: []string{"Input"}, quiet: []string{"list", "status", "kill"}},
}
```

Jeder Nullwert ist der geschlossene Fall: Ein neues Werkzeug, bei dem
jemand nichts setzt, verweigert einen Aufruf ohne Zeile und wird nach den
Regeln für Tipp-Eingaben (1.4) geprüft. `judgedOrRefused` fällt damit weg,
dessen Nullwert heute offen ausfällt. Die Kommentare an den drei alten Maps
werden ein Kommentar am Typ. Außer `commandLines` und `checkTool` in
`guard.go` liest niemand die drei Maps (grep über `internal/`), der
Typwechsel bleibt also in dieser Datei.

### 1.2 Schlüssel ohne Rücksicht auf die Schreibung (C4)

`commandLines` geht die Schlüssel der Eingabe in sortierter Reihenfolge
durch (damit die Gründe deterministisch sind) und nimmt jeden String-Wert,
dessen Schlüssel per `strings.EqualFold` einem der `keys` gleicht. Jeder
Treffer wird geprüft. Damit ersetzt `EqualFold` die Schreibvarianten der
Tabelle (`input`, `commandLine`); `command_line` bleibt, weil der Unterstrich
keine Schreibvariante ist.

Nachtrag beim Planen (2026-09-27): Ein Wert unter einem dieser Schlüssel,
der kein String ist (`{"Action":"kill","Input":42}`, `null`), könnte die
Zeile sein, die das Werkzeug ausführt; er macht den Aufruf unprüfbar
(`ok false`, ohne `lineOptional` verweigert), die gefundenen Strings werden
trotzdem geprüft. Ein leerer String tippt und startet nichts und zählt nicht
als Zeile, sodass `{"Action":"kill","Input":""}` still bleibt. Für `Bash`
und `PowerShell` (mit `lineOptional`) bleibt ein unlesbarer Wert ohne Grund,
weil ihr Werkzeug immer einen String schickt. Das folgt aus der Regel oben
und bleibt so; Doku und Commit sagen es. Soll es geschlossen werden, braucht
es einen eigenen Nachtrag.

### 1.3 Zeilen zuerst, eine stille Action entschuldigt nur das Fehlen (C3, D1)

Reihenfolge in `commandLines`:

1. Alle Zeilen nach 1.2 sammeln.
2. Gibt es Zeilen, werden sie geprüft, gleich welche `Action` der Aufruf
   nennt.
3. Gibt es keine: Der Aufruf ist still, wenn er mindestens einen Schlüssel
   trägt, der per `EqualFold` `action` gleicht, und der Wert **jedes**
   solchen Schlüssels ein String ist, der exakt in `quiet` steht. Ein Wert,
   der kein String ist (`{"Action":"kill","action":42}`), macht den Aufruf
   nicht still. Still heißt `ok true` ohne Zeilen; sonst `ok false`, und
   ohne `lineOptional` verweigert der Guard.

`{"Action":"kill","Input":"git push origin main\n"}` wird damit als Push
verweigert, `{"Action":"kill","action":"send_input"}` ist nicht still,
`{"Action":"kill","TaskId":"t"}` und `{"Action":"list"}` gehen durch.

### 1.4 Tipp-Werkzeuge: nur ganze Zeilen (C5, D2)

Für ein Werkzeug ohne `whole` gilt für jeden gefundenen Wert:

- Er muss auf `\n` oder `\r` enden.
- Er darf kein Steuerzeichen tragen außer `\n` und `\r`: kein C0
  (U+0000–U+001F), kein DEL (U+007F), kein C1 (U+0080–U+009F, darunter
  U+009B, das 8-Bit-CSI). Auch kein Tab (Nachtrag beim Planen): Eine
  interaktive Shell vervollständigt an ihm ein Wort, aus
  `"git pus\t origin main\n"` wird dort `git push origin main`, nachdem der
  Guard `git pus` gelesen hat.
- Keine seiner Zeilen darf auf `\` oder `` ` `` enden (Nachtrag beim
  Planen): bash setzt eine Zeile nach `\`, PowerShell nach `` ` `` in der
  nächsten fort; `"loomux \\\ninit\n"` läuft dort als `loomux init`, und
  geteilt wären beide Zeilen frei, während der ungeteilte Wert heute an
  `writesConfiguration` scheitert. Verweigert wie ein Bruchstück, mit
  demselben Grund.
- Er wird an `\r\n`, `\r` und `\n` in Zeilen geteilt; jede nicht leere Zeile
  läuft durch die Befehlsregeln und `writesConfiguration`.

Verletzt ein Wert eine der ersten drei Regeln, verweigert der Guard mit
einem Grund, der die Regel nennt, etwa „loomux judges what an agent types
into a task only as whole lines without control characters; this
manage_task input does not end its line, so it refuses“. `"y\n"`,
`"echo hi\r\n"` und `"\n"` gehen durch; `"git pu"`, `"\x03"`, `"\x1b[A\n"`
und `"git pusx\bh origin main\n"` nicht.

Bewusster Preis (Entscheidung 1): Eine Einzeltaste ohne Enter, Ctrl-C,
Ctrl-D, Pfeiltasten und Tab-Vervollständigung kann ein Agent einer offenen
agy-Aufgabe nicht mehr schicken. `kill` beendet eine Aufgabe weiter. Weil
1.3 die Zeilen vor der `Action` prüft, gilt die Regel auch für eine stille
Action: `{"Action":"kill","Input":"x"}` wird als Fragment verweigert, obwohl
`x` keine verbotene Zeile ist.

### 1.5 Ein Aufruf ohne Werkzeugnamen wird verweigert (E1)

`policyReasons` gibt heute für einen dekodierten Aufruf ohne String-Namen
nichts zurück, und die Schranke lässt ihn durch. Das war Absicht und steht
als `TestAPayloadWithoutAToolNameReachesTheBarrier` im Code. Diese Spec kehrt
die Entscheidung um: Ein dekodiertes Objekt ohne Werkzeugnamen verweigert
die Policy mit dem Grund „loomux found no tool name in this call, so it
cannot judge it and refuses“. Ein Payload, der kein Objekt ist, geht
weiter an die Schranke, mit deren Wortlaut.

Soll E1 wegfallen (der Befund ist plausibel, sein Auslöser nicht gezeigt),
entfällt nur dieser Abschnitt samt Commit 2; nichts anderes hängt daran.

Grund: `PreToolUse` sagt selbst „everything that goes wrong on the way
refuses“. Claude (`tool_name`) und agy (`toolCall.name`) senden immer einen
Namen; kein legitimer Aufruf fällt. Ein Host mit anderem Schlüssel, etwa ein
künftig verdrahtetes Codex, liefe heute still an jeder Regel vorbei. Der
alte Test wird umgekehrt und umbenannt.

### 1.6 Matcher gegen die Werkzeuglisten (C10)

Zwei Tests, keine Produktionsänderung:

- In `internal/hooks` (interner Test, importiert `internal/setup/hostfile`;
  `hostfile` importiert nur `hosts` und `shellwords`, kein Zyklus): Jeder
  Schlüssel von `commandTools` steht im PreToolUse-Matcher von
  `hostfile.Entries` für Claude oder für Antigravity (Split an `|`).
- In `internal/brain/guard`: Jedes Werkzeug aus `writingTools` steht im
  PreToolUse- und im PostToolUse-Matcher von Claude oder Antigravity.

Der Kommentar über den Antigravity-Einträgen in `table.go` sagt, dass diese
Tests den Matcher an die Listen binden.

## 2 init und Hookdateien (`internal/setup/hostfile`)

### 2.1 Fehlende Werkzeuge anhängen (C2)

`find` gibt statt des ersten fremden Matchers eines eigenen Blocks für
denselben Hook (`elsewhere`) alle solchen Matcher in Dateireihenfolge
zurück. `Merge` entscheidet für einen Eintrag, der auf seinem eigenen
Matcher keinen eigenen Block hat:

- **Flach** heißt: Der Matcher zerfällt an `|` in nicht leere Namen aus
  `[A-Za-z0-9_]`.
- Sind `entry.Matcher` und alle Matcher aus `elsewhere` flach, ist
  `missing` = Namen des gewünschten Matchers minus der Vereinigung der
  vorhandenen, in der Reihenfolge des gewünschten.
  - `missing` nicht leer: `Merge` hängt `blockFor(entry)` mit dem Matcher
    `strings.Join(missing, "|")` an, verbucht ihn unter `Added` und schreibt
    die Notiz `<Event>: kept an own entry under matcher <A>; added one for
    <missing>`.
  - `missing` leer: wie heute `Kept` mit der Notiz `kept an own entry under
    matcher …`, die jetzt alle Matcher aus `elsewhere` nennt.
- Sonst (ein Regex-Matcher, `.*`, leer): nur die Notiz. Nachtrag beim
  Planen: Für einen eigenen Block ganz ohne `matcher`-Schlüssel ist das neu,
  heute hängt `Merge` dort den gewünschten Eintrag daneben. Behalten ist
  richtig, Claude Code wendet einen Block ohne Matcher auf jedes Werkzeug
  an; die Notiz nennt ihn `(none)` statt mit leerem Namen.

Ein zweiter Lauf auf dem Ergebnis findet beide Blöcke, ihre Vereinigung
deckt den gewünschten Matcher, und `Merge` fügt nichts hinzu. Die Regel gilt
für jeden Host, nicht nur für Antigravity. Bei Claude trifft sie die Einträge
aus ulinit-Zeiten: Deren Matcher `Write|Edit|NotebookEdit|Bash|PowerShell`
ist eine echte Teilmenge des heutigen, also hängt `init` dort einen Block
für `MultiEdit` an, das bisher auf solchen Installationen unbewacht blieb.
Nachtrag beim Planen: Das gilt nur für einen loomux-Befehl unter diesem
Matcher, denn loomux selbst schrieb ihn nie. Ein Eintrag, der noch `ulguard`
ruft, ist nicht `Owned`; neben ihm fügt `init` wie heute den ganzen
loomux-Block ein.
`merge_test.go:574` und `plan_test.go:687` erwarten heute nur die Notiz und
kippen damit auf „angehängt“; der Plan schreibt ihre neuen Erwartungen aus. Die Paketregel „Nothing is
rewritten and nothing removed“ bleibt wahr: `Merge` hängt nur an. Die
Paket-Doku nennt den neuen Fall.

**Vorbedingung (Messung vor dem Bau):** agy lädt eine `hooks.json`, deren
Gruppe `loomux` unter `PreToolUse` zwei Blöcke mit verschiedenen Matchern
trägt, und ruft den Hook für ein Werkzeug des zweiten Blocks auf. Zu messen
mit der dann aktuellen agy-Version (`agy --version` festhalten) und dem Log
auf Parse-Fehler; das Ergebnis kommt datiert in
`docs/.superpowers/parity/stufe-4a-2.md`. agy 1.2.11 verwarf eine Datei
wegen einer Blockform bei Stop still, deshalb wird nicht angenommen, sondern
gemessen. Scheitert die Messung, wird 2.1 nicht gebaut, und der Nutzer
entscheidet neu.

### 2.2 Die eigene `.agents/hooks.json` (C1)

Der PreToolUse-Matcher der versionierten Datei wird von Hand auf den
Matcher aus `Entries` gesetzt (ein Block, keine Anhängung).
`TestTheRepositorysOwnAntigravityHooksNeedNoChange` neben
`TestTheRepositorysOwnSettingsNeedNoChange` liest `../../../.agents/hooks.json`,
führt `Merge` mit `Entries(Antigravity, BinaryOf(…))` aus und verlangt kein
`Added`, keine Notiz und `Merged` Byte für Byte gleich der Datei. Der Test
ist vor der Änderung der Datei rot.

### 2.3 Nur Kommentare (C8, E2)

`commandsOf` und der Testkommentar in `merge_test.go` (~643) behaupten
nicht mehr, ein Host führe bei einem Block mit flachem Befehl und hooks-Liste
beides aus; sie sagen, dass loomux beide liest, weil das Verhalten der Hosts
ungemessen ist. Keine Codeänderung an `find` für C8: die Form schreibt
loomux nie, und der dokumentierte Vertrag von `stale` („when none of them is
entry's command“) ist erfüllt.

## 3 post-edit (`internal/hooks/post_edit.go`, `internal/verify`, `internal/cli/hook.go`)

### 3.1 Ein Satz für das Budget (C12)

`verify.BudgetSkipped(name string) string` baut die Zeile „the edit budget
ran out: <name>“ samt Präfix; `report.go` und `post_edit.go` rufen sie.
`SkipPrefix` wird wieder unexportiert, wenn kein Test außerhalb von
`verify` es braucht. `EditReport` liefert `(red bool, notices []string)`
statt `(int, string)`.

### 3.2 Jeder Skip-Hinweis auch auf stderr (C6)

Ein Helfer in `verify` schreibt einen Skip-Hinweis nach stderr und hängt ihn
an die Hinweise an. `EditReport` nutzt ihn für jede übersprungene Lane
(Budget, fehlendes Werkzeug, nicht bereit), `RunPostEdit` für eine Datei,
die das Budget nicht erreichte. Der Helfer schreibt nie nach stdout; das
bleibt Sache von `RunPostEdit` (3.3), damit nichts doppelt kommt. Bei Exit 2
hören Claude und agy damit jeden Skip; bei Exit 0 steht er für Claude
weiter im Kontext.

### 3.3 Kontext über die hosts-Naht, nichts auf stdout bei Exit ≠ 0 (C13, E3)

- `PostToolUse(stdin, stdout, stderr, root, hostName string, budget)` liest
  den Host wie `SessionStart` (`hosts.ParseHost`; unbekannt → stderr,
  `ExitInternal`) und legt ihn in `EditEnv.Host`. Der Test-Seam
  `postToolUse` in `internal/cli/hook.go` bekommt den Host mitgereicht.
- `RunPostEdit` sammelt Hinweise und Asides als Zeilen. **Nur bei Code 0**
  ruft es `hosts.WriteContext(env.Host, "PostToolUse", stdout, lines)`;
  scheitert das, schreibt es den Fehler nach stderr und endet mit
  `ExitInternal`, wie `SessionStart`. Bei Code ≠ 0 schreibt es nichts nach
  stdout. Damit fällt das Aside einer grünen Datei in einem roten Aufruf weg
  (E3), und der Kommentar „The aside is dropped when a lane is red“ gilt für
  den ganzen Aufruf.
- `verify.WriteNotices` und `TestWriteNotices` entfallen. Der eine
  Claude-Encoder ist `writeClaudeContext` (ohne HTML-Escaping,
  `hookEventName` zuerst); die Beispiele in `cli-reference.md` folgen.
- Antigravity: `WriteContext` schreibt `injectSteps`, `hosts.Answer`
  verwirft post-tool-use-stdout weiter (Entscheidung 3; seit dem
  2026-09-28 überholt, siehe dort). Der Kommentar in
  `answer.go` sagt „the host's context“ statt „Claude's additionalContext“.
- Codex: `WriteContext` antwortet `ErrNoAdapter`, also endet
  `post-tool-use --host codex` bei Code 0 mit 1, wie es `hooks.md` für die
  Codex-Naht schon beschreibt („a seam that refuses with exit 1 rather than
  guessing a shape“). Heute bekommt Codex dort Claudes Envelope; das ist der
  Fehler, nicht die neue Antwort. Ein Payload, der keine Datei nennt, endet
  wie heute vor jedem Schreiben mit 0; die 1 kommt, sobald der Payload eine
  Datei nennt, auch eine, deren Endung die Presets ignorieren (beim Planen
  so festgelegt: eine eigene Zählung geprüfter Dateien nur für einen Host
  ohne Adapter lohnt nicht). Soll die Codex-Änderung wegfallen, genügt es, `WriteContext` für
  Codex nicht zu rufen; der Rest von 3.3 bleibt.

### 3.4 Der Testhelfer liest nach Schlüssel (C14)

`editContextOf` in `post_edit_test.go` dekodiert nach `map[string]any`, geht
exakt `hookSpecificOutput` → `additionalContext` und scheitert bei einem
fehlenden Schlüssel; es verlangt genau ein JSON-Dokument auf stdout. Ein
Gegentest zeigt, dass `{"HookSpecificOutput":{"AdditionalContext":"x"}}`
scheitert. Der Helfer landet vor 3.3, weil 3.3 Schlüsselreihenfolge und
Escaping ändert.

### 3.5 Changelog-Berichtigung (C21)

Keine eigene Codeänderung: Mit 3.2 und 3.3 wird der Mehrdatei-Pfad
beobachtbar (Skips auf stderr bei Exit 2). Der Fixed-Eintrag im PR-Rumpf
sagt ausdrücklich, dass die Hinweise einer Bearbeitung mit mehreren Dateien
seit v2.14.2 keinen Host erreichten (agy liest post-tool-use-stdout nicht,
Claude nennt nur eine Datei) und jetzt über stderr kommen.

## 4 session-start und Payload

### 4.1 `Revive` nur beim ersten Aufruf (C7)

`sessions.Revive` bleibt, wo es steht, **vor** `recordBase`, und wird in ein
eigenes `if !payload.Repeat` gefasst. Es wandert nicht in den späteren Block
mit `staleBinary` und `updateWarnings`: Der Kommentar an Zeile 52–56 legt
fest, dass die Sitzung wieder zählt, bevor die Basis angesehen wird, damit
ein `recordBase`, das mit `ExitInternal` scheitert, sie nicht ungezählt
lässt. Die Kommentare in
`hook_session_start.go` und die Doku von `Payload.Repeat` in `hostio.go`
sagen: Nichts legt eine agy-Unterhaltung zwischen zwei Modellaufrufen still
(nur `WorktreeUnlink` ruft `Retire`, verdrahtet nur für Claude); wird unlink
je für agy verdrahtet, ist das neu zu entscheiden. Der Test, der heute die
Zeile bei einer Wiederholung erwartet (~492), wird umgekehrt; der Fall des
ersten Aufrufs bleibt.

### 4.2 `invocationNum` als Zahl oder Dezimal-String (D3)

`invocationOf(v any) int64` in `antigravity.go`: `float64` → Ganzzahl,
String → `strconv.ParseInt(v, 10, 64)`, sonst oder bei Fehler 0.
`Repeat` = `invocationOf(…) > 1`. Der protojson-Kommentar sagt, dass
64-Bit-Zähler als String kommen können und die Feldbreite ungemessen ist.
Ein falsches `Repeat` false wiederholt nur Ankündigungen.

Nachtrag beim Planen: Ungemessen ist auch, ob `PreInvocation` `invocationNum`
überhaupt trägt. Die einzige aufgezeichnete Nutzlast
(`testdata/cases/2c-payloads/agy-inv.json`, agy 1.2.2) hat nur
`conversationId` und `stepIdx`; fehlt das Feld, ist `Repeat` immer false,
und 4.1 greift nie. Die agy-Messung vor 2.1 zeichnet deshalb im selben Lauf
die `PreInvocation`-Nutzlast mit auf (ein flacher Eintrag, kein Block).

Nachtrag bei der Umsetzung (2026-09-28): Der Vergleich ist
`Repeat` = `invocationOf(…) > 0`, nicht `> 1`, weil die Zählung bei 0
beginnt. Gemessen am 2026-09-27 mit agy 1.2.11 in der Messung vor 2.1:
`PreInvocation` trägt `invocationNum` als JSON-Zahl, 0, 1, 2, 3 bei den vier
Modellaufrufen des Laufs, dazu `initialNumSteps` 1, 3, 5, 7 und kein
`stepIdx`. Das Feld ist also da, und 4.1 greift ab dem zweiten Aufruf.
Ungemessen bleiben nur die String-Form eines 64-Bit-Zählers und die Breite
des Felds. Die Testzeile zu 4.2 in Abschnitt 6 verschiebt sich mit: `"1"` und
`1` sind Repeat, `"0"` und `0` nicht.

## 5 Doku und Arbeitspapiere

Jede Stelle gleich in en und de; Stellen auf a7805df8 (nach dem Rebase auf 9428f0af sind viele Zeilen gewandert; der Plan zitiert die Anker auf dem neuen Stand):

| Thema | Stellen |
|---|---|
| Guard (1.2–1.5) | `docs/*/hooks.md` Antigravity-Absatz (en ~362–371, de ~378–387), `docs/*/cli-reference.md` (en ~1247–1254, de ~1303–1310), `docs/*/migration.md` Zeile 4a-2 (en 38, de 39) |
| init (2.1) | `docs/*/hooks.md` und `docs/*/getting-started.md`: was `init` mit einem älteren Matcher tut; Paket-Doku `merge.go`, Doku von `find` |
| post-edit (3.2, 3.3) | `docs/*/hooks.md` (en ~227–253, de ~233–263), `docs/*/configuration.md` (en ~409–421, de ~417–421), `docs/*/cli-reference.md` (en ~327–328, de ~347–348), `README.md:83`, `README.de.md:83`, Kommentare in `report.go`, `hostio.go:3–4` und `115–120`, `answer.go:38–41` |
| session-start (4.1, 4.2) | `docs/*/hooks.md` (en ~445–449, de ~466–469), `hostio.go:87–93`, `antigravity.go:13–14` |
| Arbeitspapiere (C9) | `parity/stufe-4a-2.md` (~672–685): datierte Nachträge unter den bisherigen Zeilen, auch die Messung aus 2.1 und „agy liest `injectSteps` auf PostToolUse: ungemessen“; Fusions-Spec Nachtrag #23 (~714) an Ort und Stelle, er ist noch „Freigabe offen“; `specs/2026-09-25-antigravity-hooks-integration-design.md` (7, 76–77, 81–86, 109, 131–132, 142–146) und `plans/2026-09-25-antigravity-hooks-integration.md` je mit datiertem Nachtrag |
| Testkommentar (C15) | `internal/setup/helpers_test.go:53–57` verweist auf `cmdSplits` statt die Zeichen aufzuzählen |

Die Arbeitspapiere werden nach 1.3 und 1.4 geschrieben, damit sie die
endgültigen Regeln nennen. `docs/.superpowers/specs-ul/…-messung.md` bleibt
als Messprotokoll unverändert.

## 6 Tests

### 6.1 Vorgehen

- TDD je Befund: erst der rote Test, dann der Code.
- 100 % Coverage je Funktion; keine Ausnahme ist geplant.
- Jeder Test einer Regel wird gegengeprüft: Die Regel im Code wird per
  `go test -overlay` entfernt oder umgekehrt, und genau dieser Test muss rot
  werden. Die Eingabe trennt die Fälle (etwa `input` neben `Input`, ein
  Wert ohne Zeilenende, `\x1b[A\n`, U+009B).
- Tabellengetriebene Tests über `commandTools` statt von Hand gepflegter
  Listen: jedes Werkzeug verweigert einen Push, jedes außer Bash und
  PowerShell verweigert einen Aufruf ohne Zeile. Ein neues Werkzeug ist
  damit ohne Teständerung abgedeckt, und die Schleife prüft, dass sie nicht
  leer ist.
- Das Tor ist `sh ci/gate.sh`; seine Ausgabe geht ganz in eine Datei, bevor
  gefiltert wird.

### 6.2 Tests je Abschnitt

| Abschnitt | Tests |
|---|---|
| 1.2 | `input` neben harmlosem `Input` → Push-Grund; dasselbe für `run_command` mit `COMMANDLINE`; kill mit `Input` 42 oder `null` → Grund; kill mit `Input` "" → keiner; Bash mit `command` 42 neben `COMMAND` Push → Push-Grund |
| 1.3 | kill/status mit Push-Input → Grund; `Action` kill plus `action` send_input → Grund; `Action` kill plus `action` 42 → Grund; kill ohne Input, list → keine Gründe |
| 1.4 | Fragment, `\b`, `\x03`, `\x1b[A\n`, U+009B, Tab, eine Zeile auf `\` oder `` ` `` → Grund; mehrzeilig mit einer zeilenanfangs-verankerten Regel in Zeile 2 → Grund (Nachtrag beim Planen: nicht mit Push, dessen `(^\|\s)` auch ein Zeilenende trifft, sodass der Test ohne Teilen grün bliebe); `"y\n"`, `"echo hi\r\n"`, `"\n"` → keine Gründe; für `manage_task` und `send_command_input` |
| 1.5 | `{"toolCall":{"args":{…}}}`, `{"toolCall":"x"}`, `{"tool_input":{…}}` → Exit 2 mit dem Grund; Nicht-JSON → Wortlaut der Schranke |
| 1.6 | die zwei Paritätstests; der in `internal/hooks` wird rot, wenn `manage_task` aus `table.go` fällt, der in `internal/brain/guard`, wenn ein Schreibwerkzeug wie `write_to_file` fällt (`manage_task` schreibt nicht) |
| 2.1 | alter Matcher → angehängter Block `manage_task`, Notiz nennt ihn; zweiter Lauf → nichts; Regex- oder Obermengen-Matcher → nur die Notiz; zwei alte Blöcke, deren Vereinigung deckt → nichts; Claudes ulinit-Matcher → Block `MultiEdit` (`merge_test.go:574`, `plan_test.go:687` umgestellt) |
| 2.2 | `TestTheRepositorysOwnAntigravityHooksNeedNoChange` |
| 3.1 | `BudgetSkipped` gleich dem Beispiel in `cli-reference.md` |
| 3.2 | rote Lane plus fehlendes Werkzeug → stderr nennt den Skip; `.py` mit ruff rot und mypy fehlt → Exit 2, stderr nennt den Skip |
| 3.3 | `<b>` bleibt wörtlich; rot → stdout leer; zwei Dateien, eine rot, eine grün mit Aside → stdout leer; `--host codex` mit einer grünen Datei oder einer ignorierten (`data.json`) → Exit 1 mit dem Adapterfehler, ohne Datei → 0; der Seam bekommt `antigravity` bei `--host antigravity` |
| 3.4 | Gegentest mit falscher Schreibung scheitert; ein zweites Dokument scheitert |
| 4.1 | Wiederholung mit nicht entfernbarem `.ended` → keine Kontextzeile; erster Aufruf → Zeile; erster Aufruf mit scheiterndem `recordBase` → `Revive` lief vorher (sichtbar an der Änderungszeit, die es der Sitzungsdatei gibt; „zählt“ im engen Sinn prüfen hieße eine neue Naht in `sessions`, die diese Spec nicht will) |
| 4.2 | `"2"` → Repeat, `"1"`, `"x"`, fehlend → nicht, `2` → Repeat |

## 7 Reihenfolge und Commits

Ein Commit je Thema, die Doku im Commit ihrer Änderung:

1. `fix(guard)`: eine Werkzeugtabelle, Schlüssel ohne Schreibung, Zeilen
   zuerst, nur ganze Zeilen (1.1–1.4)
2. `fix(guard)`: Aufruf ohne Werkzeugnamen verweigern (1.5)
3. `test(hooks)`: Matcher an die Werkzeuglisten binden (1.6)
4. `fix(setup)`: fehlende Werkzeuge eines eigenen Matchers anhängen (2.1),
   erst nach der Messung
5. `fix`: die versionierte `.agents/hooks.json` samt Test (2.2)
6. `docs(hostfile)`: Kommentare zu Blöcken mit beiden Formen (2.3)
7. `refactor(verify)`: ein Satz für das Budget (3.1)
8. `test(hooks)`: Kontext nach Schlüssel lesen (3.4)
9. `fix(hooks)`: Skip-Hinweise auch auf stderr (3.2)
10. `fix(hooks)`: post-edit über die hosts-Naht, kein stdout bei Exit ≠ 0
    (3.3)
11. `fix(hooks)`: session-start belebt nur beim ersten Aufruf (4.1)
12. `fix(hosts)`: `invocationNum` als String lesen (4.2)
13. `docs`: Arbeitspapiere und Testkommentar (5, C9, C15)

Spec und Plan liegen auf demselben Zweig und gehen im selben PR.

## 8 Release

`release:patch`: keine Änderung an Befehl, Flag, Konfigurationsformat oder
Exit-Code eines verdrahteten Hosts. Am nächsten dran sind 1.4 (weist
Eingaben ab, die heute durchgehen; Sicherheitsfix, vom Nutzer gewählt) und
3.3 für Codex (Exit 1 statt Claudes Envelope, wie die Doku die Codex-Naht
beschreibt). Changelog im PR-Rumpf unter `Security` (1.2–1.5, 2.1) und
`Fixed` (3.2–3.5, 4.1, 4.2); 2.2, 2.3, 3.1, 3.4, 1.6 und 5 sind für Nutzer
nicht sichtbar und stehen dort nicht.

## 9 Risiken und Messungen

- **Zwei PreToolUse-Blöcke bei agy** (2.1): gemessen, bevor gebaut wird.
- **agy-Version:** vor dem Bau und vor dem PR `agy --version` lesen; weicht
  sie von 1.2.11 ab, werden die Annahmen zu `manage_task` neu gemessen.
- **Legitime Tipp-Eingaben** (1.4): bewusst in Kauf genommen; die Doku nennt
  die Regel und dass `kill` bleibt.
- **Bytegenaue Tests** (3.3): Schlüsselreihenfolge und Escaping ändern
  sich; betroffen sind `report_test.go` (~104, ~129–130, ~146–153,
  ~198–206) und die Beispiele in `cli-reference.md`.
