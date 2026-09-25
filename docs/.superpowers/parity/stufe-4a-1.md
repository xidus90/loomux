# Stufe 4a-1 — Akte: Schema, `loomux config`, `[modules]`

Plan: `docs/.superpowers/plans/2026-09-24-loomux-stufe-4a-1.md`.
Spec: `docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md`.
Zweig `feat/stage-4-config`. Diese Stufe hat keine Python-Referenz: Sie ist
neu gebaut, die Akte hält darum Messpunkte, Entscheidungen, Selbstnutzung und
die Mutationsrunde fest statt eines Fallkorpus.

## Messungen vor dem Bau

| Messpunkt | Stand | Wer |
|---|---|---|
| Arbeitsverzeichnis eines stdio-MCP-Servers aus dem Nutzerbereich (Task 1: CLI `claude -p` in `…\loomux\internal` und Desktop-App auf einem Worktree) | **offen** | Mensch |
| Oberfläche von Hand in Windows Terminal, conhost und dem Terminal der Claude-App (Task 13, Step 11: Pfeiltasten, `/`, ESC allein, Enter) | **offen** | Mensch |

**Arbeitsverzeichnis.** Nicht gemessen. Die Brücke ist darum nach Zweig A
gebaut (Entscheidung H unten): Sie sucht vom Arbeitsverzeichnis aus nach oben
nach `.loomux/config.toml`, und `--root` nennt das Projekt ausdrücklich.
Startet der Wirt sie außerhalb jedes Projekts, bietet sie jedes Werkzeug an —
das ist Zweig B ohne dessen Hinweiszeile auf stderr. Trägt die Messung die
Annahme nicht, fehlt genau diese Zeile; ein falsches Ergebnis entsteht nicht.

**Drei Terminals.** Nicht geprüft. Die Widgets sind über `tui.Scripted`
getestet; `Open`, `Size` und `enableVT` brauchen eine echte Konsole und sind
von der Abdeckung ausgenommen (siehe Mutationsrunde). Offen dabei vor allem:
ESC allein gegen ESC gefolgt von einer schnell getippten Taste (die Grenze
`Buffered() == 0` im Decoder).

## Entscheidungen beim Bau

Jede Entscheidung des Ledgers, mit ihrem Preis, falls sie falsch ist.

| | Entscheidung | Preis, falls falsch |
|---|---|---|
| A | Task 4 liest das echte Feld, das `post_edit.go` für eine abgeschaltete Lane liest (`parseStack`/`effective.go`), statt des im Plan vermuteten Namens. | Eine Korrekturrunde. |
| B | Task 7 nimmt Auswahl und Vorgabe von `privacy.mode`, `maintenance`, `model` und `untouched_days` aus `declaration.go`, nicht aus dem Plantext; der Test aus Task 8, der alle Vorgaben durch die Leser schickt, ist die Probe. | Eine Korrekturrunde. |
| C | Wo der Plan einen Helfer ruft, dessen Signatur er nicht gelesen hat (`parseInterspersed`, `shellwords.Split`, `mirror.Mirror`), passt der Bauende den Aufruf an die echte Signatur an, ohne das Verhalten zu ändern. | Klein. |
| D | Der Doku-Commit „docs: split setup and intake…“ nennt „the plan for the config command“, also ein Arbeitspapier; `release-pr` formuliert ihn beim Gruppieren vor dem PR um. | Bis zur Umgruppierung verletzt ein Commit auf dem Zweig die Commit-Regel. |
| E | Die Art `schema.Duration` entfällt: Kein Schlüssel braucht sie, `verify.timeout` liest der Leser als ganze Sekunden. Task 9 verliert die Duration-Fälle von `Render`, der Ablehnungstest in Task 14 erwartet „not a whole number“. | Die Art später wieder einführen. |
| F | Ohne `[area]` prüft der Deklarationsleser keinen Brain-Abschnitt; `Validate` nähme also etwa `privacy.mode = "open"` an. `proposeChange` weist darum jeden Brain-Schlüssel außer `area.scope` ab, solange der Text kein `[area]` hat („set area.scope first“). | Ein Schritt mehr für einen Menschen, der Brain-Schlüssel zuerst setzen will. |
| G | Eine Preset-Zeile trägt keinen eigenen Wert; `config list` zeigt dort die Vorgabe des Schlüssels, damit keine Zeile leer aussieht. | Nur die Anzeige. |
| H | Task 1 ist nicht geliefert; Task 5 ist als Zweig A gebaut (Suche nach oben ab dem Arbeitsverzeichnis). Außerhalb jedes Projekts findet die Brücke keine Konfiguration und bietet alle Werkzeuge an. | Eine fehlende einzeilige stderr-Notiz. |
| I | Vorschläge statt Schreiben für Agenten (2026-09-24, Nachtrag zur Spec): `config set`/`unset` mit `--propose` legen die geprüfte Änderung unter `.loomux/state/config/proposals/` ab, statt zu schreiben; `config proposals`, `apply` und `reject` sind die Hälfte des Menschen. Ein Vorschlag hält nur Operation, Schlüssel und Eingabe; `proposals` und `apply` rechnen den Diff jedes Mal neu gegen die aktuelle Datei, ein abgelegter Diff kann darum weder veralten noch gefälscht werden. Schlüssel, Eingabe und Diff gehen maskiert aufs Terminal. Der Wächter lässt `set`/`unset` nur mit `--propose` als eigenem Wort vor jedem `--`, `#` und jeder Umleitung durch; `--propose=…` in jeder Form wird verweigert, weil ein späteres `=false` das Flag wieder abschaltet, und nur auf einer schlichten Zeile: eine Positivliste der Zeichen außerhalb und innerhalb von Anführungszeichen, weil die Liste verbotener Shell-Syntax (`$`, Backtick, `@`, Klammer-Expansion, Globs, PowerShell-Klammern) in jeder Prüfrunde wuchs. Dazu muss der Aufruf direkt sein (loomux als erstes Wort, kein Wrapper wie `cmd /c`, der `^`, `%X%` oder `!X!` in einfachen Anführungszeichen ein zweites Mal liest). Dieselbe Regel gilt für `--dry-run`/`--detect-only` von `init`; `try { loomux init --dry-run }` und `cmd /c loomux init --dry-run` werden darum jetzt verweigert. `config set`/`unset` weist ein `--propose`, das am Ende falsch ist, selbst mit Exit 2 ab. | `proposals` zeigt jeden Diff für sich gegen die aktuelle Datei, nicht das Ergebnis der Folge unter `apply --all`; `apply` zeigt vor jeder Frage den Diff gegen den dann aktuellen Stand. |
| Task 15, geparkt | Der Wächter-Befund der fünften Korrekturrunde ist echt, trägt aber nichts für spätere Aufgaben (niemand baut auf den Interna der Regel auf); die Spec nennt die Regel ein Netz mit bekannten Löchern. Er geht in die Korrekturwelle des Abschlussreviews (siehe „Offene Punkte“). | Die drei Zeilen unter „Offene Punkte“ bleiben bis dahin offen. |

## Abweichungen beim Bau von 4a-1

Dieselben drei wie in der Spec der Stufe 4 unter gleichem Titel:

- **Keine Mehrfachauswahl.** `internal/tui` hat Liste, Eingabe mit Auswahl
  und Bestätigung, aber noch keine Mehrfachauswahl; `config` braucht keine.
  Sie kommt mit 4a-2, wo `init` die Module wählen lässt.
- **Die Tabellen je Stack sind keine Schlüssel des Schemas.**
  `[verify.<stack>.<art>]` ändert ein Mensch von Hand; `loomux check
  precommit --show` zeigt, was sie mit den Presets ergeben. `config list`
  führt sie nicht auf.
- **Ein abgeschaltetes Werkzeug ist kein Werkzeugfehler.** Die Brücke lässt
  es in `tools/list` einfach weg; ein Aufruf trifft darum den Protokollfehler
  des SDK für ein unbekanntes Werkzeug, nicht einen Werkzeugfehler mit
  Hinweis auf `[modules]`. Ein Wirt ruft nur, was gelistet ist.

## Selbstnutzung

Am 2026-09-24 gegen den Zweigstand `58b4a8d`, Binär `bin/loomux.exe`, frisch
gebaut und mit `dev swap-binary --dir bin` getauscht.

**`bin/loomux.exe config list`** (Exit 0):

```text
base   commit.conventional    true                                                                                                                           default
base   commit.language        "en"                                                                                                                           default
base   commit.threshold       2                                                                                                                              default
base   commit.allow           0 entries                                                                                                                      unset
base   modules.brain          true                                                                                                                           default
base   modules.graph          true                                                                                                                           default
base   modules.hooks          true                                                                                                                           default
base   policy.commands.rules  1 entries                                                                                                                      set
base   policy.paths.rules     6 entries                                                                                                                      set
base   worktree.mirror        []                                                                                                                             default
hooks  verify.max_parallel                                                                                                                                   unset
hooks  verify.profiles        { edit = ["lint", "types"], precommit = ["lint", "types", "test", "coverage"], stop = ["lint", "types", "test", "coverage"] }  preset
hooks  verify.timeout         600                                                                                                                            default
brain  area.scope             "project/loomux"                                                                                                               set
brain  index.exclude                                                                                                                                         unset
brain  index.include          ["docs/wiki/**/*.md"]                                                                                                          set
brain  index.unsearched       ["docs/**/**/index.md"]                                                                                                        set
brain  layout.hub                                                                                                                                            unset
brain  layout.inbox                                                                                                                                          unset
brain  layout.review                                                                                                                                         unset
brain  layout.wiki            "docs/wiki"                                                                                                                    set
brain  maintenance.branch                                                                                                                                    unset
brain  maintenance.on_merge                                                                                                                                  unset
brain  model.enabled                                                                                                                                         unset
brain  model.roles                                                                                                                                           unset
brain  privacy.mode           "manual_cloud"                                                                                                                 default
brain  privacy.never          []                                                                                                                             default
brain  wiki.types                                                                                                                                            unset
brain  wiki.untouched_days    180                                                                                                                            default
```

Gegen die Erwartung des Plans: `area.scope = "project/loomux"` (set),
`layout.wiki` (set), `index.include` (set), `policy.paths.rules` mit 6 und
`policy.commands.rules` mit 1 Eintrag, `commit.language` (default) — alles wie
erwartet. Die Tabellen `[verify.go.*]` erscheinen nicht als Schlüssel. Zwei
Beobachtungen: Die Spalte „Wert“ wird von der langen `verify.profiles`-Zeile
auf rund 140 Zeichen gezogen, sodass jede Zeile umbricht, wo das Terminal schmaler
ist; und `config get commit.threshold` antwortet `2`.

**Mensch: `bin/loomux.exe config set commit.threshold 3`, mit `n` bestätigt.**
**Offen** — dieser Schritt gehört dem Menschen. Erwartet: ein Diff mit
`+ threshold = 3` in einer neuen Sektion `[commit]`, die Frage
`write these changes? [y/N]`, dann `loomux config: declined; nothing written`
und Exit 0.

**Agent: `bin/loomux.exe config set commit.threshold 3 --yes` über Bash.**
Vom Wächter verweigert, wie erwartet:

```text
PreToolUse:Bash hook error: ["${CLAUDE_PROJECT_DIR}/bin/loomux.exe" hook pre-tool-use --host claude --root "${CLAUDE_PROJECT_DIR}"]: loomux policy refused this tool call:
  - loomux init, config and area add write the configuration the guard reads; a human runs them
```

`git status` danach sauber, `config get commit.threshold` weiter `2`. Vorher
dieselbe Zeile als Nutzlast direkt an `hook pre-tool-use` gegeben: Exit 2,
dieselbe Begründung.

**Nebenbefunde der Selbstnutzung, alle Fehlverweigerungen, keine Lücke:**

- `bin/loomux.exe config --help` wird verweigert (die Regel liest ein Flag
  nach `config` als interaktive Form).
- Ein Heredoc, dessen Text eine Zeile `loomux config set …` enthält, wird
  verweigert — der Zeilenumbruch ist eine Abschnittsgrenze. Beim Schreiben der
  Befehlsreferenz über `cat >> … <<'EOF'` passiert; die Doku ging danach über
  das Edit-Werkzeug hinein.

**Die Lücken und Fehlverweigerungen, gegen das neue Binär nachgespielt**
(Nutzlast über ein Go-Skript im Scratchpad, damit die Bash-Zeile selbst den
Wächter nicht auslöst; Exit 0 = durchgelassen, 2 = verweigert):

| Exit | Zeile |
|---:|---|
| 0 | `echo "a \" b"; "C:\Program Files (x86)\loomux\loomux.exe" init` |
| 0 | `echo "a \" b"; "C:\R&D Tools\loomux.exe" init` |
| 0 | `sudo -u root loomux init` |
| 0 | `sh -c "loomux init"` |
| 0 | `go run cmd/loomux init` |
| 0 | `go run ./cmd/loomux/main.go init` |
| 2 | `echo "x; loomux init"` |
| 2 | `loomux config --help` |
| 2 | `loomux config --root x list` |
| 0 | `loomux config list` |
| 0 | `loomux config get commit.language` |
| 0 | `loomux init --dry-run` |
| 2 | `loomux area add` |
| 2 | `"C:\Program Files (x86)\loomux\loomux.exe" config set a b` |

Die Befehlsreferenz führt die ersten sechs als bekannte Lücken und die drei
Fehlverweigerungen als solche. Nachtrag aus dem Abschlussfix: Die erste Zeile
wird seit der wieder eingesetzten Lesart über `strings.Fields` verweigert
(festgenagelt in `TestTheGuardRefusesCommandsThatWriteTheConfiguration`, nicht
gegen das Binär nachgespielt); die zweite bleibt ein bekanntes Loch und steht
dort in `holes`.

## Mutationsrunde

**Die Runde mit `loomux dev mutants` (2026-09-24).** Gefahren mit einer
Kopie des Binärs im Scratchpad, `LOOMUX_STATE_DIR` und
`LOOMUX_LEGACY_BRAIN_DIR` im Scratchpad der Sitzung. Umfang: die drei neuen
Pakete ganz, dazu `internal/cli` nur mit `--only config` (`config.go`,
`config_ui.go`) — eine Runde über das ganze `cli` hätte ein Vielfaches der
17,5 Minuten gebraucht, die schon dieser Ausschnitt lief, und der übrige
Code dort ist nicht Teil der Stufe. Die erste Runde lief gegen `58b4a8d`
(`edit`, `schema`, `tui` mit acht Arbeitern in 1:34 min; `cli` mit acht in
17:29 min), die zweite gegen den Stand mit den neuen Tests (`edit`, `schema`,
`tui` mit vier Arbeitern in 2:17 min, parallel zum Ende der ersten
`cli`-Runde; `cli` erneut mit acht). Die Tests für `cli` kamen erst nach dem
Ende der ersten `cli`-Runde in den Baum, damit sie deren Urteile nicht
verunreinigen.

| Paket | erzeugt | nicht übersetzbar | erste Runde: getötet / überlebt | zweite Runde: getötet / stehen |
|---|---:|---:|---:|---:|
| `internal/config/edit` | 202 | 12 | 164 / 26 | 189 / 1 |
| `internal/config/schema` | 79 | 25 | 52 / 2 | 53 / 1 |
| `internal/tui` | 158 | 15 | 104 / 39 | 131 / 12 |
| `internal/cli` (`--only config`) | 225 | 28 | 176 / 21 | 185 / 12 |

**Getötet durch neue Tests.**

- `edit`: `TestSetReadsEscapesOnlyInBasicStrings` (die Maskierung in
  `tripleQuote`, `brackets` und `comment`, `edit.go:146`, `:148`, `:171`,
  `:196` — ein Backslash maskiert nur in `"…"`, in `'…'` ist er ein Zeichen);
  drei neue Fälle in `TestSetRefusesMultiLineStrings` (`:84`, `:99`: eine
  mehrzeilige Zeichenkette, deren Schlusszeile als Kommentar gelesen wird,
  fällt nur an der Eröffnungszeile auf; „after an even string“ für die
  Formen, die in einer Zeichenkette jedes zweite Zeichen überspringen);
  `TestSetSurvivesAListLeftOpenAtTheEnd` (`:104`, sonst ein Index jenseits
  des Endes); „key twice“ in `TestRemoveRefusesWhatItCannotPlace` (`:276`);
  `TestRemoveKeepsTheLineBeforeAnEmptiedSection` (`:296` — der Mutant löschte
  eine Schlüsselzeile des Menschen, der einzige Datenverlust der Runde);
  `TestRemoveTakesATopLevelKey` (`:293`); die exakte Erwartung in
  `TestAppendBlockAddsATableListEntry` (`:307`); `Render(Bool, "true")` in
  `TestRenderMakesALiteralPerKind` (`value.go:25`);
  `TestDiffWhereOneSideIsAPrefixOfTheOther` (`diff.go:12`, `:16`).
- `schema`: `TestValidateReportsAScratchDirectoryThatCannotBeMade` prüft
  jetzt, dass der Fehler der des Scratch-Verzeichnisses ist
  (`validate.go:21`); vorher reichte irgendein Fehler, und hinter einem
  gescheiterten Scratch scheitern die Leser am leeren Pfad aus eigenem Grund.
- `tui`: `TestListShowsTheFilterWhileTypedAndAfterwards` (`list.go:87`, sechs
  Formen), `TestListKeepsTheCursorOnARowWhenTheFilterNarrows` (`:31`),
  `TestListBackspaceOnAnEmptyFilterDoesNothing` (`:45`),
  `TestListShowsAsManyRowsAboveTheCursorAsFit` (`:128`, `:131`),
  `TestFitLeavesALineAloneWhenTheWidthIsUnknown` (`:141`, `width >= 0`),
  `TestInputDrawsHintAndProblemOnlyWhenThereIsOne` (`input.go:15`, `:19`),
  `TestInputIgnoresTabAndBackspaceWithNothingToDo` (`:42` — Division durch
  null —, `:46`), vier neue Fälle in `TestDecodeReadsTheRestOfTheKeys`
  (`keys.go:38`, `:76`, `:79`: die Ränder `0x20`, `0x3f`, `0x40` und ein
  einzelnes Folgebyte `0x80`).
- `cli`: `TestConfigSetRefusesATable` verlangt jetzt `<key> is a table` für
  `verify.profiles` und `commit.allow` (`config.go:194`, drei Formen; vorher
  fing `edit.Render` die Tabelle mit einer anderen Meldung ab);
  `TestConfigUIShowsAListOfTablesByItsCount` (`config_ui.go:61`, vier Formen,
  und `:77`); `TestConfigUIMakesSeveralChangesInOneSession` (`:70` — der
  Mutant beendete die Oberfläche nach der ersten gelungenen Änderung).

**Was steht, mit Begründung.**

| Paket | Mutant | Entscheidung |
|---|---|---|
| `edit` | `comment`, `edit.go:205`: `j > 0` → `j >= 0` | **Äquivalent.** `i` beginnt hinter dem `=`, also steht links vom `#` immer mindestens das `=`; `j` erreicht 0 nie. |
| `schema` | `lookup`, `current.go:74`: `if !ok` → `false` | **Äquivalent.** Ist der Knoten keine Tabelle, ist `table` die leere Map; das Lesen daraus liefert `ok == false`, und die nächste Zeile gibt dasselbe `nil, false` zurück. |
| `tui` | `List`, `list.go:56`: `cursor < len(visible)-1` → `<=` | **Äquivalent.** Der Cursor läuft eins über das Ende; die Klemme am Anfang der nächsten Schleifenrunde (`:31`) holt ihn zurück, bevor gezeichnet oder gewählt wird. |
| `tui` | `fit`, `list.go:141`: `len(rs) > width` → `>=` | **Äquivalent.** Bei gleicher Länge ist `rs[:width]` die Zeichenkette selbst. |
| `tui` | `Size`, `raw.go:25` (zwei Formen); `Open`, `raw.go:37` (vier); `enableVT`, `raw_windows.go:18` (vier) | **Brauchen eine echte Konsole**, von der Abdeckung ausgenommen (`//coverage:exempt`). Geprüft wird das von Hand in drei Terminals (Task 13, Step 11, offen). |
| `cli` | `openConsole`, `config_ui.go:23` (fünf Formen) | **Braucht eine echte Konsole**, ausgenommen; der Test ersetzt `openTerminal`. |
| `cli` | `proposeChange`, `config.go:226`: `if !t.global` → `true` | **Heute nicht erreichbar.** `GlobalKeys()` ist leer, also scheitert jeder globale Schlüssel schon an `lookup`. Wird beobachtbar, sobald `[model]` in die globale Datei zieht (4c). |
| `cli` | `configSet`, `config.go:245`: Lesefehler ignoriert | **Im Exit-Code nicht beobachtbar.** Ein Pfad, der sich nicht lesen lässt (ein Verzeichnis), lässt sich auch nicht schreiben: `lock.ReplaceText` scheitert, Exit 1 bleibt. Nur die Meldung ändert sich. |
| `cli` | `configSet`, `config.go:269`; `changeOne`, `config_ui.go:105`: Fehler von `MkdirAll` ignoriert | **Äquivalent im Ergebnis.** Kann das Verzeichnis nicht angelegt werden, scheitert `lock.ReplaceText` daran. |
| `cli` | `configUI`, `config_ui.go:67`, `:93`, `:101`: `err != nil \|\|` entfernt | **Äquivalent.** `List` gibt bei einem Fehler `-1`, `Input` und `Confirm` geben `false` zurück; der verbleibende Operand trifft denselben Zweig, und `err` wird dort zurückgegeben. |

## Offene Punkte

- **Task 15, geparkt, für die Korrekturwelle des Abschlussreviews:** die
  Lesart über `strings.Fields` mit abgeschnittenen Anführungszeichen wieder
  einsetzen, mit `echo "a \" b"; "C:\Program Files (x86)\loomux\loomux.exe"
  init` als Test festnageln, das Loch im Doc-Kommentar als „jedes frühere
  maskierte `\"` oder `\'` auf der Zeile“ formulieren und
  `echo "a \" b"; "C:\R&D Tools\loomux.exe" init` als bekanntes Loch führen.
  Erledigt im Abschlussfix.
- **Menschenschritte:** Task 1 (Arbeitsverzeichnis), Task 13 Step 11 (drei
  Terminals), Task 16 Step 4 (`config set` mit `n`).
- **Nebenwirkung der Mutationsrunde:** Der Mutant `scratch`,
  `validate.go:34` (`if err != nil` → `false` nach `os.MkdirTemp`) schreibt
  unter `TestValidateReportsAScratchDirectoryThatCannotBeMade` eine
  `.loomux/config.toml` mit `[commit]` in das Paketverzeichnis
  `internal/config/schema`, weil der Pfad dann relativ ist. Der Mutant stirbt,
  die Datei bleibt liegen; nach beiden Runden von Hand entfernt. Abhilfe: der
  Test mit `t.Chdir(t.TempDir())` — erledigt im Abschlussfix.
- **Am 2026-09-25 behoben, zweite Runde** (`plans/2026-09-25-loomux-4a-kleinigkeiten.md`,
  Entscheidungen in `stufe-4a-2.md`, „Offen“): aus der Liste unten `"a,"`,
  gemischte Zeilenenden (schon in der ersten Runde), `Remove` mit
  Kommentaren, die ungetestete Vorgabe in `Current`, Kommata in
  Listeneinträgen (jetzt `"a,b", c`), `config --root DIR list` und fremde
  Flags, die Antwort `yes`, `fit` nach Zellen und die Spalte „Wert“.
  `projectModules` ist nicht behoben, sondern fällt weg (Sortierung in
  `stufe-4a-2.md`); `default` als Wert, `go run` im Wächter und die
  übrigen „Erledigt“-Punkte waren es schon. Offen bleiben nur die
  Menschenschritte oben.
- **Aufgeschobene Kleinigkeiten, die es wert sind, behalten zu werden** (aus
  dem Ledger, Auswahl):
  - `edit.Render(Int)` lässt `"007"` und `"+3"` durch; `strconv.Itoa(n)`
    zurückgeben (Task 9). Erledigt im Abschlussfix.
  - `edit.Render(StringList)` macht aus `"a,"` ein leeres Element (Task 9).
  - Gemischte CRLF/LF-Dateien werden falsch zerlegt; `Diff` behält `\r` (Task 9).
  - `Remove` lässt Kommentare in einem geleerten Abschnitt zurück (Task 9).
  - Eine Vorgabe, die sich nicht dekodieren lässt, wird in `Current` still
    geschluckt; ein Schleifentest über `Keys()` fehlt (Task 10).
  - Kommata in Listeneinträgen überstehen die Eingabeform `a, b` nicht (Task 10).
  - `config --root DIR list` (Flags vor dem Unterbefehl) endet mit Exit 2;
    fremde Flags (`get --yes`) werden still angenommen (Task 11).
  - Ein String- oder Aufzählungsschlüssel kann nicht auf das Wort `default`
    gesetzt werden; die Frage nimmt nur `y`, nicht `yes` (Task 11).
  - `restore()` im interaktiven `config` wird nicht per `defer` gerufen; eine
    Panik ließe die Konsole im Rohmodus (Task 14). Erledigt im Abschlussfix;
    ein Fehler beim Zurücksetzen steht auf stderr und gibt Exit 1.
  - Eine geänderte Datei zwischen Listenbild und Bestätigung wird
    überschrieben (Task 14). Behoben am 2026-09-25: `writeConfig` bekommt
    den gelesenen Text und verweigert, wenn die Datei ihn nicht mehr hält —
    für `set`, `unset`, die interaktive Form und `apply`.
  - `projectModules` nimmt jeden Fehler von `FindRoot` als „kein Projekt“,
    nicht nur `ErrNoRoot` (Task 5).
  - Ältere `TestMcp*`-Tests lesen die eigene `.loomux/config.toml` des Repos;
    `t.Chdir(t.TempDir())` in `stubBridgeRun` (Task 5). Erledigt im
    Abschlussfix.
  - Der Wächter erkennt `go run` mit Flags vor dem Paket,
    `go run cmd/loomux` und `./cmd/loomux/main.go` nicht (Task 15).
  - `fit` in `tui` zählt Runen, keine Anzeigezellen; breite Zeichen können
    umbrechen (Task 13).
  - `config get` auf eine Liste von Tabellen (`commit.allow`,
    `policy.paths.rules`) gibt eine leere Zeile aus, während `list` die Zahl
    der Einträge zeigt — `shownValue` liest `Value`, und `Current` setzt für
    diese Art nur `Count`. Neu, beim Nachprüfen der Befehlsreferenz gegen das
    Binär. Erledigt im Abschlussfix: `Current` füllt `Value` mit einer Zeile
    je Eintrag; `get` und der Dialog der interaktiven Form zeigen sie.
  - Die Spalte „Wert“ von `config list` richtet sich nach der längsten Zeile
    (`verify.profiles`, rund 140 Zeichen) — neu, aus der Selbstnutzung.
