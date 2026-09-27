# Paritätsakte Stufe 4c-1

**Quelle:** ultra-brain `loomux-3-source` (`3cc72d2`), derselbe Tag wie in
Stufe 3a und 3b. Am 2026-09-25 geprüft: `HEAD` von ultra-brain ist `3cc72d2`,
beide Tags `loomux-1a-source` und `loomux-3-source` zeigen darauf, und
`.venv\Scripts\brain-mcp.exe` ist vorhanden.
**Spec:** `docs/.superpowers/specs/2026-09-23-loomux-stufe-4-design.md`,
Abschnitt „Abweichungen beim Planen von 4c“
**Plan:** `docs/.superpowers/plans/2026-09-25-loomux-stufe-4c-1.md`

`reconcile` fragt für jeden `local_only`-Bereich das lokale Modell nach einem
Vorschlag. Die Referenz tut das in `_proposers` und `_land_case`
(`src/brain/maintenance/reconcile.py`), die Einstellungen liest sie aus
`<zustand>/config.toml` `[model]` (`src/brain/model/settings.py`).

## Fallsatz 4c-1, aufgezeichnet am 2026-09-25

Sieben Fälle, aufgezeichnet mit `stufe-4c-1-orakel/record_all.sh` gegen
`brain-mcp.exe` am Tag `loomux-3-source`, übersetzt mit
`testdata/cases/4c1-map.toml`, abgespielt von `TestCases4c1`.

**Welten.** Je Fall eine eigene Welt unter `testdata/cases/4c1-worlds/`, alle
sieben eine Kopie von `3a-worlds/vault-changed` (der Welt von
`reconcile/changed-source`). Der Plan nannte vier Welten; weil
`config.toml` in der Weltwurzel liegt, die für beide Seiten der Zustand ist,
braucht jeder Fall mit eigenem `[model]` eine eigene Welt. Das Manifest des
Bereichs (`repo-a/.brain.toml`, nach dem Import `.loomux/config.toml`) trägt
`[privacy] mode = "local_only"`, außer in `open-area-not-asked`.

**Attrappe.** `loomux dev fake-ollama` lauscht fest auf `127.0.0.1:11435` und
antwortet jeder Anfrage mit der `ollama-fixture.json` der Welt. Die Fixture
von `proposal-kept` zitiert die hinzugefügte Zeile aus D1 wörtlich, abgelesen
aus `package.md` einer Probeaufzeichnung von `model-off`:
`+The second state of the source, a little longer now.` Die Fixture von
`proposal-invented` zitiert `+erfunden`.

| Fall | `config.toml` | Exit | Aufrufe | Ergebnis der Referenz | Abspielen |
|---|---|---|---|---|---|
| `reconcile/proposal-kept` | `enabled = true`, `endpoint = "http://127.0.0.1:11435"` | 0 | 1 | `proposal.md` neben dem Fall, `prompt_version = "vorschlag-v4"`, kein `manual`, keine Notiz | gleich |
| `reconcile/proposal-invented` | wie oben | 0 | 1 | kein Vorschlag, `manual = true`, zweite Notiz | gleich |
| `reconcile/model-unreachable` | `enabled = true`, `endpoint = "http://127.0.0.1:11436"` (niemand lauscht) | 0 | 0 | `manual = true`, zweite Notiz | gleich |
| `reconcile/model-off` | keine | 0 | 0 | `manual = true`, erste Notiz | gleich |
| `reconcile/open-area-not-asked` | wie `proposal-kept`, Bereich nicht `local_only` | 0 | 0 | Fall ohne `manual`, `local_only`, Notiz und Vorschlag | gleich |
| `reconcile/endpoint-off-loopback` | `enabled = true`, `endpoint = "http://192.0.2.1:11434"` | 1 | 0 | kein Fall, aber `maintenance/project-a/stats.tsv` | gleich |
| `reconcile/broken-model-block` | `model = 5` | 1 | 0 | kein Fall, aber `maintenance/project-a/stats.tsv` | gleich |

`endpoint-off-loopback` trägt `enabled = true`, das die Tabelle des Plans
nicht nennt; die Referenz verweigert den Endpunkt damit wie erwartet mit
Exit 1. `[model]` wird wie in der Referenz erst nach dem Scan aller Bereiche
gelesen (`reconcile.py:205`, `:210`), darum bleibt in beiden Fällen mit
Exit 1 der Stempel-Cache `stats.tsv` liegen, auf beiden Seiten. Die zweite Notiz lautet wörtlich
`manual review: the local proposer returned no usable proposal (slice-6 spec §3)`,
die erste
`manual review: this area is local_only, so no skill path is offered (spec 5)`.

„Gleich“ heißt: null Abweichungen nach `NormalizeState` (Zeitstempel, Datum
in der Fall-ID). stderr vergleicht der Fallsatz nicht (wie 3b). Die
Sperrdatei `areas/<scope>.lock` taucht nicht auf, weil `reconcile` allein sie
nicht nimmt.

### Wie die Aufrufe gezählt werden

Die Attrappe schreibt je Anfrage eine Zeile
`POST /api/generate model=… temperature=… num_ctx=… stream=… think=…`, ohne
den Prompt (er trägt Fall-ID und Tag). Beim Aufzeichnen läuft sie als eigener
Prozess um `record-case`, ihr Log `ollama-calls.log` liegt in `RECORD_DIR`,
außerhalb jeder Welt, und wird vor jedem Fall geleert. `record.sh` gibt
`record-case` eine Notiz mit dem Platzhalter `@CALLS@` und setzt danach mit
`sed -i` die Zeilenzahl des Logs in `notes.md` ein (`ollama calls: N`). Die
Referenz ruft nur `/api/generate` (keine Proben), die Zahl ist also genau.

Beim Abspielen stellt `serveFakeOllama` für jeden Fall mit Fixture einen
eigenen Server mit eigenem Log auf denselben Port; der Server schließt mit dem
Untertest des Falls, damit der nächste den Port bekommt. Das Log ist hinter
einem Mutex, weil `http.Server.Close` nicht auf laufende Handler wartet. Die
Zahl der Zeilen muss `wantOllamaCalls4c1` treffen, die aus den Notizen der
Aufzeichnung stammt: 1 für `proposal-kept` und `proposal-invented`, sonst 0.

## Abweichungen

| Abweichung | Art | Begründung |
|---|---|---|
| `approve --reject` schiebt Quellen und Register vor | freigegeben 2026-09-25 | Heilung #1, Spec „Abweichungen beim Planen von 4c“ |
| Sperre `areas/<scope>.lock` für `reindex` und `approve` | freigegeben 2026-09-25 | Heilung #4; die Datei bleibt liegen wie `registry.lock` |
| Meldungen zu `[model]` in der Form von loomux | Meldungstext | stderr wird nicht verglichen (wie 3b) |
| `temperature` im JSON als `0` statt `0.0` | Format | Ollama liest beides als Zahl |
| Gefragt wird vor dem Anlegen des Fallverzeichnisses, nicht danach | Reihenfolge | Ein abgebrochener Lauf (Ende von `serve`) lässt den stehenden Fall stehen und legt kein leeres Verzeichnis an; die Ausgabe eines vollständigen Laufs ist gleich |

## Überlebende Mutanten

**Die Runde mit `loomux dev mutants` (2026-09-26).** Gefahren mit
`bin/loomux.exe dev mutants`, gebaut aus diesem Baum und per
`dev swap-binary` getauscht, acht Arbeiter, eine Runde nach der anderen:

```
bin/loomux.exe dev mutants ./internal/brain/model
bin/loomux.exe dev mutants -only modelsettings ./internal/config
bin/loomux.exe dev mutants -only arealock ./internal/config
bin/loomux.exe dev mutants -only reconcile ./internal/brain/maintenance
bin/loomux.exe dev mutants -only reject ./internal/brain/apply
bin/loomux.exe dev mutants -only frontmatter ./internal/brain/apply
```

`-only` nimmt jeweils genau eine Datei (`modelsettings.go`, `arealock.go`,
`reconcile.go`, `reject.go`, `frontmatter.go`); `model` läuft ganz
(`client.go`, `propose.go`). Die erste Runde über `model` und
`modelsettings` am 2026-09-26 lief schon über den Stand mit
`test(model): pin the port's upper bound, …` und dem damals nur
vorgemerkten Test der Temperaturgrenzen; ihre Vorher-Spalte kommt darum aus
einer zweiten Fahrt über einen Auszug von `fb0ff992` (`git archive`, ohne
`testdata/`) mit demselben Binär — `fb0ff992` unterscheidet sich in beiden
Paketen nur um diese Tests. Ein Zeitüberlauf zählt als getötet;
`go test ./internal/brain/maintenance` braucht allein 24 s,
`./internal/brain/apply` 8 s, weit unter den 60 s von `goTimeout`.

| Paket (Datei) | erzeugt | nicht übersetzbar | vorher: getötet / überlebt | nachher: getötet / stehen |
|---|---:|---:|---:|---:|
| `internal/brain/model` | 72 | 22 | 44 / 6 (`fb0ff992`) | 50 / 0 |
| `internal/config` (`modelsettings`) | 85 | 22 | 61 / 2 (`fb0ff992`) | 63 / 0 |
| `internal/config` (`arealock`) | 8 | 3 | — | 5 / 0 |
| `internal/brain/maintenance` (`reconcile`) | 362 | 58 | — | 301 / 3 |
| `internal/brain/apply` (`reject`) | 67 | 11 | 55 / 1 | 56 / 0 |
| `internal/brain/apply` (`frontmatter`) | 60 | 15 | 42 / 3 | 44 / 1 |

Die Nachher-Spalte ist je eine eigene, saubere Runde über den Stand mit den
neuen Tests, keine aus Einzelproben abgeleitete Zahl.

**Getötet in `model` (6), `test(model): pin the port's upper bound, the 2xx
range and a short read`.** `client.go:59` `n > 65535` → `>=`: Port 65535
wurde nie angenommen, jetzt `http://localhost:65535` in
`TestGuardEndpointLetsTheLoopbackThrough`. `client.go:131`, vier Formen
(`false`, jeder Operand allein, `> 299` → `>=`): kein abgewiesener Status
trug einen Rumpf, der sonst eine Antwort wäre, und 299 zählte nie als 2xx —
`TestTheStatusDecidesAgainstAnAnswerInTheBody` (200, 299, 300, 500) und
`TestASwitchOfProtocolsIsNoAnswer` (ein 101, der einzige Status unter 200,
den Gos Client als endgültig zurückgibt). `client.go:135` `if err != nil` →
`false`: der kurze Rumpf in `TestABodyCutShortIsNoAnswer` endete vor einem
ganzen JSON-Objekt, der Decoder wies ihn ab und die kurze Lesung blieb
ungeprüft; jetzt endet er nach `{"response":"x"}`.

**Getötet in `modelsettings` (2), `test(config): pin both ends of the model
temperature's range`.** `modelsettings.go:142` `number >= 0` → `>` und
`number <= 2` → `<`: kein Test nahm genau 0 oder genau 2.
`TestModelSettingsTakeBothEndsOfTheTemperatureRange`; beide einzeln per
`go test -overlay` gegen den alten Stand des Testfiles nachgeprüft
(grün) und gegen den neuen (rot).

**Getötet in `apply` (3 von 4), `test(approve): cut the rejection's read
seam to the proposal and pin two source guards`.**
- `reject.go:41` `if err != nil` → `false` (Fehler aus `claimHeadings`):
  `TestRejectPassesOnAProposalThatCannotBeRead` ließ jede Lesung über
  `readBytes` scheitern, also hielt `readPage` die Ablehnung ebenfalls an
  und verdeckte den Mutanten — dieselbe Art zu weiter Naht, die 3b dreimal
  fand. Die Naht scheitert jetzt nur an `proposal.md`.
- `frontmatter.go:99` `block == nil || match == nil` → `block == nil`: der
  Fall „blocks disagree“ deckte nur die eine Richtung (`---\r\n`, der
  Leseblock trifft, der Schnittblock nicht). Die andere — `--- ` mit
  Leerzeichen hinter der schließenden Zeile, der Schnittblock trifft, der
  Leseblock nicht — fehlte; unter dem Mutanten indiziert `AdvanceSources`
  dann einen fehlenden Treffer. Neuer Fall „closing line with a blank“ in
  `TestAdvanceSourcesLeavesAPageWithNothingToAdvance`.
- `frontmatter.go:126` `entry.kind != pyDict` → `false`: ein Eintrag in
  `sources[]`, der kein Mapping ist, liest sein fehlendes `doc_id` wie
  Pythons `str(None)` als `"None"`; nur die Prüfung hält ihn davon ab, eine
  Quelle mit genau dieser ID zu treffen.
  `TestAdvanceSourcesSkipsAnEntryThatIsNoMappingForADocIDOfNone`.

**Stehen, mit Begründung.**

| Paket | Mutant | Entscheidung |
|---|---|---|
| `internal/brain/apply` | `advanceSourceEntries`, `frontmatter.go:121`: `entries == nil \|\| entries.kind != pyList` → `entries == nil` | **Äquivalent, stehengelassen.** Nur eine Liste trägt `items` (`pyyaml.go:174`, sonst nur die Listen von `verified` in `frontmatter.go:81`, `:83`); ein `sources:` als Zeichenkette oder Mapping hat keine, die Schleife läuft nicht, `changed` bleibt `false` — dieselbe Antwort. Die Prüfung sagt, was ein `sources` ohne Liste bedeutet |
| `internal/brain/maintenance` | `byRepository`, `reconcile.go:527`: `if common == ""` → `false` | **Geerbt aus 3a, stehengelassen.** Dieselbe Zeile wie `reconcile.go:475` in `parity/stufe-3a.md`, „Überlebende Mutanten“ („Nicht beobachtbar“); 4c-1 hat sie nicht berührt, nur verschoben |
| `internal/brain/maintenance` | `mergeEvidence`, `reconcile.go:580`: `pathsErr != nil \|\| subjectsErr != nil` → `pathsErr != nil` | **Geerbt aus 3a, stehengelassen.** Der zweite Operand hat keinen Vektor: Ruling vom 2026-09-20 in `parity/stufe-3a.md` („`_merge_evidence`, zwei Lesungen und ein Urteil“) |
| `internal/brain/maintenance` | `isDirectory`, `reconcile.go:734`: `if path == ""` → `false` | **Geerbt aus 3a, stehengelassen.** Dieselbe Zeile wie `reconcile.go:680` in `parity/stufe-3a.md` („Äquivalent“: `os.Stat("")` scheitert) |

Die Zeilen, die 4c-1 in `reconcile.go` geschrieben hat (`proposers`, das
Fragen vor dem Anlegen, `noteFor`, `prompt_version`), haben keinen
Überlebenden; `arealock.go` auch nicht.

**Nebenbefund.** Die Runde über `reconcile` hinterließ ein Fallverzeichnis
`internal/brain/maintenance/project-a/a-2026-09-20-084a/` im Arbeitsbaum
(`case.toml`, `package.md`, `proposal.md`, 12:11 Uhr, mitten in der Runde).
Ein gewöhnlicher `go test ./internal/brain/maintenance` legt es nicht an;
ein Mutant hat einen Test mit relativem Prüfzentrum ins Paketverzeichnis
schreiben lassen. Das Verzeichnis ist entfernt, nicht committet. Welcher
Mutant es war, ist nicht nachgegangen; `dev mutants` isoliert das
Arbeitsverzeichnis der Tests nicht.

## Selbstnutzung — offen (Mensch)

- Eine Wikiseite in obsidian-ai mit Vorschlag des lokalen Modells.
- Der `pktmon`-Mitschnitt: kein Paket verlässt Loopback.
- Die Messung gegen das echte Modell.

Verfahren: Plan `2026-09-25-loomux-stufe-4c-1.md`. Ergebnisse mit Datum hier
eintragen.
