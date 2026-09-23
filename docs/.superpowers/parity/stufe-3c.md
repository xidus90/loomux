# Paritätsakte Stufe 3c — Brain pflegen

Plan: `plans/2026-09-23-loomux-stufe-3c.md`. Spec: `specs/2026-09-19-loomux-stufe-3-design.md`,
„3c im Einzelnen“. Die Befunde B1–B17 stehen im Plan; hier steht, was die
Prüfung gegen den Tag an ihnen berichtigt hat, dazu B18.

## Vorbedingungen, festgestellt am 2026-09-23

- Arbeitsort: Zweig `claude/mit-3c-fortsetzen-94e9a0`, nicht `master`, sauber.
- Referenz: ultra-brain `loomux-3-source` = `3cc72d2`. Der lokale `master`
  steht auf dem Tag.
- Der Plan wurde gegen `08d985a` gelesen, den Stand von `origin/master`.
  `08d985a` ist ein **Vorfahre** des Tags (`git merge-base --is-ancestor
  08d985a loomux-3-source`), der Tag ist also neuer, und es gibt keinen
  Grund, ihn zu verschieben. Zwischen beiden ändern sich nur
  `cmd/brain/main.go`, `pkg/check/run/run.go` und `run_test.go`; `src/` ist
  gleich.

## B18: `config.ReadManifest` bei unlesbarer Datei

Am Tag verweigert `config.ReadManifest` eine reguläre Datei, die sich nicht
lesen lässt, statt zum nächsten Namen weiterzugehen (Kommentar in
`pkg/check/run/run.go` ab Zeile 346, der Test
`TestADeclarationThatCannotBeOpenedIsReportedToo` in `run_test.go:341-360`).
loomux tut dasselbe schon: `readManifestAmong` gibt bei einer regulären
Datei, die nicht liest, einen Fehler zurück und nie `ErrNoManifest`
(`internal/config/manifest.go:169-174`). Der Test zieht in Task 4 mit, und
nichts ist zu ändern. Er setzt `BRAIN_STATE_DIR`; im Umzug wird daraus
`LOOMUX_STATE_DIR`.

## Entscheidungen, freigegeben am 2026-09-23

| # | Frage | Entscheidung | Freigabe |
|---|---|---|---|
| E1 | Name von `check` | `loomux brain check file\|bundle\|all` | 2026-09-23, wie vorgeschlagen |
| E2 | `check code` | Entfällt; Fusions-Spec, Nachtrag #18 | 2026-09-23, wie vorgeschlagen |
| E3 | Namen der Wiki-Werkzeuge | `loomux wiki init\|types\|retype` | 2026-09-23, wie vorgeschlagen |
| E4 | Referenz für `check` | Go-Binary `brain.exe` vom Tag `loomux-3-source` | 2026-09-23, wie vorgeschlagen |
| E5 | Regeln für `lint --scope` | **Geändert:** die zwölf Regeln von `lint.py` mit deren Texten, Regelnamen und Auslösern, als eigener Regelsatz (`internal/brain/wiki/sweep.go`) mit eigenem Leser (`sweeppage.go`). `house` bleibt die Achse von `brain check` und wird nicht abgebildet. Die Reichweite ist am 2026-09-23 nachentschieden: nur `lint --scope` (siehe unten) | 2026-09-23, geändert gegen den Vorschlag |
| E6 | `loomux lint <datei>` | Bleibt, wie 1a es gebaut hat, samt `wiki-gate`, der Lane `lint/wiki` und dem post-edit-Hook: die fünf Regeln der Go-Form in `lint.go` | 2026-09-23, wie vorgeschlagen |
| E7 | Upkeep-Texte | `loomux reconcile`/`loomux cases` statt `brain …` | 2026-09-23, wie vorgeschlagen |
| E8 | Upkeep und `reindex` | Upkeep ruft nur `reconcile` | 2026-09-23, wie vorgeschlagen |

**Warum E5 geändert wurde.** Die Prüfung der Texte (Abschnitt unten) fand 4
von 13 Regeln mit anderem Text, 2 mit anderer Id, `broken-frontmatter` gar
nicht in `house`, und abweichende Auslöser bei `orphan`, kaputtem
Frontmatter, `no-sources`, `implemented-without-commit`, `missing-type` und
`unlisted-area`. Eine Abbildung bräuchte Textvarianten und eigene Auslöser,
also eine zweite Regelschicht im Verborgenen. Der Preis der Entscheidung:
sieben Regeln stehen in `lint.go` und in `house`.

**Wie weit E5 reicht (nachentschieden am 2026-09-23).** Die Einzeldatei,
`wiki-gate` und die Lane sind in 1a gegen das Go-Binary aufgezeichnet, und
dessen Lint nimmt kleingeschriebene Typen (`type: concept`) an; `lint.py`
kennt nur Katalognamen. Der 1a-Fall `lint/valid-page` trägt genau eine solche
Seite und erwartet Exit 0, und der post-edit-Hook hätte mit den Regeln von
`lint.py` jede Änderung an einer solchen Seite blockiert. Der Mensch hat
entschieden: die Regeln von `lint.py` nur für `lint --scope`. Zwei Regelsätze
stehen damit in `internal/brain/wiki`, der Go-Form-Satz in `lint.go` und der
Referenz-Satz in `sweep.go`.

Der eigene Leser ist nötig, weil `read_page` anders liest als der Go-Leser:
Links sind eine sortierte Menge aus Markdown- und `[[Wiki]]`-Links über den
ganzen Text, Konfliktkästen in umzäuntem Code zählen nicht, ein Wert falscher
Art gilt als fehlend statt als kaputtes Frontmatter, Zeilenenden werden
gefaltet und nicht dekodierbare Bytes ersetzt.

**Orakel.** `stufe-3c-orakel/sweep.py` baut eine Welt mit allen Randfällen
der zwölf Regeln (Prozent-Escapes auch kaputte, `/abs`, `..`, Schemata,
`c:/x`, Query und Fragment, kaputtes Frontmatter, Alias, leerer Typ,
verschachteltes `index.md`, CRLF, ungültiges UTF-8, Wegweiser mit Hub) und
druckt, was `lint_bundle` findet. Am 2026-09-23 stimmte die Go-Form in allen
49 Zeilen überein bis auf den Text eines YAML-Fehlers (siehe Abweichungen).
Die Ausgabe liegt als `internal/brain/wiki/testdata/sweep/python.golden`, und
`TestTheSweepAnswersWhatLintPyAnswers` hält die Go-Form daran. `url.py` und
`unquote.py` sind die Orakel für `pytext.SplitURL` und `pytext.Unquote`.

**Halt in Task 10.** Er entfällt: `lint.go` und die Lane sind unverändert.
`lint_bundle` über `docs/wiki` dieses Repositorys fand am 2026-09-23 nichts
(mit `is_project`).

## Abweichungen

| Fall | Alt | Neu | Begründung |
|---|---|---|---|
| `brain check *` | Referenz Python | Referenz Go-Binary vom Tag | Eine Python-Form von `check` gibt es nicht (B1, E4) |
| `brain check code` | Befehl | entfällt | Fusions-Spec #18 (E2) |
| `lint`, `wiki init`, `wiki types`, `wiki retype` | `--state-dir` | ohne | Der Rekorder setzt den Zustandsort über die Umgebung. `brain check` kennt das Flag schon in Go nicht und ist darum keine Abweichung (B17, berichtigt) |
| `wiki retype` | `write_if_changed` ohne Sperre | `lock.ReplaceText` | Atomar und unter Sperre wie jeder Schreiber seit 3a (B10) |
| Upkeep-Texte | `brain reconcile`, `brain cases` | `loomux reconcile`, `loomux cases` | E7 |
| `brain check *`, Manifest eines Bereichs | `config.ReadManifest` der Go-Form: `.ultra-brain/config.toml`, dann `.brain.toml` | `config.ReadAreaManifestUntilStage4`: davor `.loomux/config.toml` (ohne `[area]` übergangen) | Die Bereiche dieses Rechners tragen noch die alten Namen. Der Leser ist strenger: eine Deklaration mit altem Namen ohne `[area] scope` und eine mit falsch typisierten bekannten Schlüsseln wird zu `house/manifest-unreadable`, wo die Go-Form sie las. Gefunden in Task 4, als `unlisted-area` den `.brain.toml` des Wegweisers nicht mehr sah |
| `brain check *`, Zustandsort | `config.StateDir()` (`BRAIN_STATE_DIR`) | Registry aus `LOOMUX_STATE_DIR`; Manifeste nur lesender Bereiche über `config.ResolvedAreaDir`, neu zuerst, alt als Rückfall | Die Leseregel seit 3a. `run` nimmt den Lookup als Argument und fragt die Umgebung nicht selbst |
| `brain check`, `lint --scope`: Weigerung „Wiki fehlt“ | ``run `brain wiki init --scope …` first`` | ``run `loomux wiki init --scope …` first`` | Der Rat nennt den Befehl, den es hier gibt (Task 9). Die Meldung steht auf stderr, das die Fälle nicht vergleichen |
| `brain check` ohne Breite, Fehlermeldungen | `'brain check' needs file, bundle, all or code` | `'loomux brain check' needs file, bundle or all` | Der Befehlsname von loomux; `code` entfällt (E2). stderr |
| `lint --scope`, Text von `broken-frontmatter` | Fehlertext von PyYAML | Fehlertext von yaml.v3 | Zwei Parser, zwei Texte; der Befund, seine Regel und seine Stufe sind gleich. Der Golden-Test ersetzt genau diesen Text |
| `lint --scope`, `open_conflicts` als Bool | Vergleich als `int`, Ausgabe `True`/`False` | ebenso | Keine Abweichung mehr; gefunden in der Mutationsrunde, siehe „Überlebende Mutanten“ |
| `lint --scope`, Weigerungen | Exit 1 aus `main` | ebenso, `error: …` auf stderr | Keine Abweichung. `lint` ohne Datei war in loomux bisher ein Bedienfehler mit Exit 2 und fegt jetzt über alle Bereiche wie die Referenz |
| Upkeep, Sichtbarkeit | `notes` liest die sichtbaren Bereiche bei jeder Antwort | nur, wenn eine Zeile ansteht | Ohne Fehlschlag und ohne Bericht gibt es nichts zu filtern. Die Referenz machte eine Antwort, die keine Registry braucht, bei fehlender Registry zum Registry-Fehler; gefunden an `TestTheCloudListenerAnswersOnItsOwnChannel` |
| Upkeep, Fehlerarten | `ReconcileError`, `RegistryError`, `OSError` werden zu `failure`, alles andere beendet den Task | jeder Fehler des Durchgangs, auch ein unlesbarer Stempel, wird zu `failure` | Ein abgestürzter Upkeep hielte das Tor für immer zu. In der Referenz ist ein unlesbarer Stempel eine Ausnahme außerhalb der drei Arten |
| Upkeep, Warten | `caught_up` wartet ohne Frist | `CaughtUp` endet mit dem Kontext des Aufrufs, und der Aufruf scheitert | Ein Client, der abbricht, soll keinen Handler hängen lassen |
| Upkeep, Anhang an eine Antwort | eine Ausnahme beim Bauen des Anhangs lässt den Aufruf scheitern | ebenso, als Fehlerantwort mit dem Text der Ursache | Keine Abweichung im Ergebnis; die Form ist die der übrigen Weigerungen dieser Werkzeuge |
| Upkeep, Alterszeile | `brain reconcile` im Text von `core.stale_reconcile` | unverändert (`search.ReconcileAdvice`) | Die 1b-1-Fälle tragen den Text; umgestellt wird mit Stufe 4 (E7) |
| `check.AxisCode`, `LaneBrokenFinding` | im Basispaket | entfallen | Nur `check code` hat sie benutzt (E2) |
| `wiki retype`, doppelter Schlüssel `type:` | PyYAML nimmt den letzten Wert, beide Zeilen werden umbenannt | die Seite bleibt, wie sie ist | Der Leser von loomux (yaml.v3) weist einen doppelten Schlüssel als kaputtes Frontmatter ab, und `retype` überspringt kaputtes Frontmatter wie die Referenz. `loomux lint` meldet die Seite als `broken-frontmatter` |
| Reihenfolge der Seiten (`wiki types`, `wiki retype`, `lint --scope`) | `sorted(rglob)` | ebenso | Keine Abweichung: Python sortiert unter Windows nach Pfadteilen, jeder klein geschrieben, und `markdownBelow` tut dasselbe; gemessen mit Python 3.14 an sechs Namen, Test `TestTheWalkSortsAsPythonSortsPathsOnWindows` |
| `wiki types`, eine Seite, die nicht gelesen werden kann | `read_page` wirft, der Befehl endet mit `error:` und 1 | ebenso | Keine Abweichung; festgehalten, weil die Zählung sonst unvollständig und doch grün wäre |
| `run`, Tests | `inner_parallel_test.go`, `TestCheckCode` | entfallen | Der erste misst ultra-brains eigenes `docs/wiki` und wird nicht ausgeliefert (B5); der zweite gehört zu `check code` (E2). Der Vergleich seriell gegen parallel bleibt in `TestSerialAndParallelRenderTheSameBytes` |

## Fallsuite, aufgezeichnet am 2026-09-23

35 Fälle in `testdata/cases/3c`, aufgezeichnet mit `stufe-3c-orakel/record_all.sh`
über die Welten aus `stufe-3c-orakel/worlds.py`. `brain check` lief gegen
`brain-3c.exe`, das der Mensch am 2026-09-23 vom Tag gebaut hat (`go build -C
ultra-brain -o brain-3c.exe ./cmd/brain`, ultra-brain `HEAD` = `loomux-3-source`
= `3cc72d2`); `lint`, `wiki init`, `types` und `retype` liefen gegen
`brain-mcp.exe` aus dem `.venv` desselben Checkouts. Die Übersetzung steht in
`testdata/cases/3c-map.toml`.

| Befehl | Fälle | Vergleich |
|---|---|---|
| `brain check` | 13 | stdout exakt |
| `lint` | 9 | stdout exakt |
| `wiki init` | 5 | Meldung: Exit und Dateiwelt exakt |
| `types` | 3 | stdout exakt |
| `retype` | 5 | Meldung: Exit und Dateiwelt exakt |

`TestCases3c` spielt sie gegen loomux ab. 34 Fälle gleichen der Referenz ohne
Normalisierung über `{{WORLD}}` hinaus; stderr wird nicht verglichen. Ein
Unterschied ist freigegeben:

| Fall | Referenz | loomux | Grund |
|---|---|---|---|
| `check/code` | Exit 0, keine Ausgabe (kein Manifest erklärt Lanes) | Exit 2, `unknown width "code"` | `check code` entfällt (E2) |

Nicht in der Suite, weil nicht deterministisch aufzuzeichnen: die Altersregeln
(`untouched`, `long-planned`), deren Auslöser die Änderungszeit der Datei ist,
und der Upkeep, dessen Texte die Zeit tragen. Beide halten Go-Tests mit fester
Uhr (`TestLintTakesTheAreasDeclaration`, `internal/serve/upkeep_test.go`).

## Selbstnutzung, 2026-09-23

Gegen die echte Registry dieses Rechners, nur lesend. Damit beide Seiten
dieselben zehn Bereiche sehen, lief loomux mit `LOOMUX_STATE_DIR` und
`LOOMUX_LEGACY_BRAIN_DIR` auf `%LOCALAPPDATA%rain`, dem Zustandsort der
Referenz; die loomux-Registry ist dieselbe plus `project/loomux`.

| Befehl | Referenz | loomux | Ergebnis |
|---|---|---|---|
| `brain check all` | `brain-3c.exe check all` | `loomux brain check all` | beide leer, Exit 0 |
| `brain check all --notes` | ebenso mit `--notes` | ebenso | 25 Zeilen, byte-gleich |
| `lint --scope all` | `brain-mcp lint --scope all` | `loomux lint --scope all` | 21 Zeilen, gleich bis auf die Zeilenenden (Python schreibt auf der Windows-Konsole CRLF, loomux LF), Exit 0 |
| `types` | `brain-mcp types` | `loomux wiki types` | 4 Zeilen, ebenso gleich bis auf die Zeilenenden |

Über die loomux-Registry dazu `project/loomux`: `lint --scope project/loomux`
und `brain check bundle --scope project/loomux` ohne Befund, `wiki types`
zählt die Seiten des Bereichs mit.

Nicht gelaufen, weil sie schreiben: `wiki retype` und `wiki init` (in
`brain-knowledge`, dessen Sicherung offen ist) und `serve` mit einem Stempel
über 24 Stunden, der einen echten `reconcile` fährt. Die drei warten auf die
Entscheidung des Menschen.

## Überlebende Mutanten

**Die Runde mit `loomux dev mutants` (2026-09-23).** Gefahren mit einer Kopie
des Binärs, acht Arbeiter, `LOOMUX_STATE_DIR` und `LOOMUX_LEGACY_BRAIN_DIR` im
Scratchpad der Sitzung, über den neuen Code der Stufe: `pytext` (nur `url`),
`wiki` (`types`, `census`, `retype`, `sweep`), `serve` (nur `upkeep`),
`serve/brain` und die drei umgezogenen Pakete `check/{okf,house,run}`. Die erste
Runde lief gegen `82e8664`, die zweite gegen den Stand mit den neuen Tests.

| Paket | erzeugt (Runde 2) | erste Runde: überlebt | zweite Runde: stehen |
|---|---:|---:|---:|
| `pytext` (`url`) | 77 | 15 | 1 |
| `wiki` (`types`) | 4 | 0 | 0 |
| `wiki` (`census`) | 60 | 3 | 0 |
| `wiki` (`retype`) | 31 | 2 | 2 |
| `wiki` (`sweep`) | 223 | 25 | 8 |
| `serve` (`upkeep`) | 109 | 6 | 2 |
| `serve/brain` | 69 | 1 | 1 |
| `check/run` | 113 | 4 | 4 |
| `check/okf` | 162 | 4 | 2 |
| `check/house` | 213 | 5 | 2 |

**Ein Paritätsfehler, den die Runde gefunden hat.** `open_conflicts: false`
liest Python als 0 und druckt es als `False` (`bool` ist dort ein `int`); die
Go-Form druckte `0`. Das Orakel `stufe-3c-orakel/edges.py` bestätigte
`open_conflicts says False, 1 box(es) found` und für `true` mit zwei Kästen
`says True`. `sweepPage` trägt seitdem die Schreibweise des Wertes mit.

**Getötet, mit Test oder Vereinfachung.**
- `pytext`: die Grenzen eines Schemanamens (`z`, `Z`, `0`, `9`), ein leeres
  `//` vor einem Pfad (`///x`), die erste und letzte Lead-Byte jedes
  UTF-8-Bereichs, alle gegen `stufe-3c-orakel/boundaries.py`. Die Dekodierung
  bestimmt die Länge jeder Sequenz jetzt über `sequence`, auch bei gültigen
  Zeichen; die beiden Abkürzungen (`%` fehlt, Bytes schon gültig) sind
  gestrichen, weil sie nichts entschieden.
- `wiki`: jedes einzelne fehlende Feld einer Quelle, `..` von der
  Bündelwurzel, ein nicht geteilter Bereich, `stale_after` von heute, die
  Realisierungsregeln außerhalb eines Projekts und eine gebaute Seite mit
  Commit, `open_conflicts` als Bool, Zäune am Textanfang und zwei Zäune mit
  einem Kasten dazwischen, eine Nicht-Markdown-Datei und ein Verzeichnis auf
  `.md` im Gang. Gestrichen, weil ohne Wirkung: die Prüfung auf einen leeren
  Wiki-Pfad in `Census` (`os.Stat("")` scheitert), die Vorabprüfung von
  `index.md` in `indexLinks` (das Lesen scheitert ebenso) und die Wahl des
  längsten Scopes in `citedScope` (der Aufrufer fragt nur, ob einer passt).
- `serve`: ein Fehlschlag ohne Bericht, ein leerer Bericht bei kaputter
  Registry, `withinFolded` bei gleichem und bei kürzerem Pfad.
- `okf`: ein Katalogziel, das nur eine Query ist; ein Eintrag genau am Beginn
  des Suchfensters von `entryDescription`.
- `house`: `..` von der Bündelwurzel; `brain:#x` und `brain:?q`.

**Stehen gelassen, äquivalent.**
- `pytext/url.go:23` `i > 0` → `i >= 0`: Bei `i == 0` ist `raw[0]` der
  Doppelpunkt, kein Buchstabe.
- `wiki/retype.go:47` (zwei Formen): Eine Seite mit anderem Typ findet
  `renameTypeField` ohne passende Zeile und lässt sie stehen; die Vorprüfung
  folgt der Referenz und spart das Lesen der Zeilen.
- `wiki/sweep.go:142` und `:303` (`p == ""`): Ein Ziel ohne Pfad löst zum
  Verzeichnis der Seite auf, das es gibt; kein Befund ändert sich.
- `wiki/sweep.go:253` (`joined == scope`): Ohne Treffer kommt der ganze Verweis
  zurück, und der ist dann selbst der geteilte Scope.
- `wiki/sweep.go:335` (`len(ExpectedTargets) == 0`): Ohne erwartete Ziele meldet
  die Schleife nichts; die Prüfung spart das Lesen des Katalogs.
- `wiki/sweeppage.go:151` (`from <= len(text)`): Am Textende findet die Suche
  nichts und bricht ab.
- `wiki/sweeppage.go:187` (zwei Grenzen): Ein Kasten beginnt am Zeilenanfang,
  ein Zaun auch; beide können nicht an derselben Stelle stehen, und das Ende
  eines Zauns liegt mitten in seiner Zeile.
- `serve/upkeep.go:137` und `:236`: Im Fehlerfall ist die Liste leer, beide
  Zweige geben dasselbe zurück.
- `serve/brain/tools.go:141`: Code aus 1b-2; ohne Befunde ist der Text sich
  selbst gleich.
- `check/run/run.go:177` (drei Formen): Seriell und parallel rechnen gleich,
  und `TestSerialAndParallelRenderTheSameBytes` verlangt genau das.
- `check/run/run.go:543` (`area.WikiPath == ""`): `filepath.Rel` mit leerem
  Wurzelpfad scheitert, `under` antwortet dasselbe.
- `check/okf/errors.go:367` (`!p.HasFrontmatter`): Ohne Frontmatter hat die
  Seite keine Schlüssel, die die Regel nennen könnte.
- `check/okf/soft.go:257` (`at < 0` → `false`): Die Prüfung auf das Textende
  zwei Zeilen tiefer fängt denselben Fall.
- `check/house/bundle.go:144` (`p == ""`): wie `sweep.go:142`.
- `check/house/federation.go:161` (`<` → `<=`): `brain:` allein lässt nichts
  übrig, was einen Scope nennen könnte.

## Prüfung gegen den Tag (Task 0, Schritte 3 und 4)

Gelesen am 2026-09-23. ultra-brain `loomux-3-source` = `3cc72d2` (`git show loomux-3-source:<pfad>`),
loomux Worktree `recursing-bartik-b2d7a1` (HEAD `e3cab29` + Plan).
`08d985a` ist ein Vorfahre des Tags. Zwischen beiden ändern sich nur Go-Dateien, `src/` ist gleich.
Für diese Stufe zählen `cmd/brain/main.go` (nach Zeile 691 um −1, nach Zeile 848 um −2 verschoben),
`pkg/check/run/run.go` (+1 ab Zeile 346) und `run_test.go` (+22, neuer Test mit `internal/testlock`).

### Befunde gegen den Tag

| Befund | Stand | Berichtigung |
|---|---|---|
| B1 | Anker verschoben | `pyproject.toml:19-24`, `main.go:93-94` und die Hilfe `:193-196` stimmen. `checkCommand` steht am Tag bei **`:919-978`** (nicht 921-980). `1a-map.toml:16-24` stimmt (`brain guard`, `brain lint`, `brain wiki-gate`). `loomux dev record-case --exe` gibt es (`internal/cli/dev.go:207`). |
| B2 | OK | `check.go:53`, `plan.go:82` (`request == "all"` → `Kinds()`, vier Arten), `schema.go:36`, `README.md:191`, `README.de.md:193` und `brainargs.go:21` stimmen alle. |
| B3 | OK, Anker ungenau | `runner.go:11-17` ist nur der Kommentar zu `laneRank`, die Tabelle steht bei `:19-25` (gofmt, ruff, mypy, pytest, coverage). Den ganzen Block gibt **`:11-25`** an. Die Presets in `presets.toml` stehen bei `:39-79`. `manifest.go:191-215` stimmt (Parsen `:189-205`, Zuweisung `:215`). `Manifest.Lanes` liest außerhalb von `config` niemand. |
| B4 | Anker verschoben, Liste unvollständig | Format und Exit-Codes stimmen (`finding.go:88-115`). Die Ortsangabe ist `scope/relative`; ist der Scope leer (`check file`), steht nur `relative` da (`location`, `finding.go:120-125`). Die Codes sind am Tag um −2 verschoben: `readablePath` steht bei **`:1027-1099`** (Prüfungen `:1059-1097`), `targets` bei **`:1128-1135`** (`return 2` bei `:1132`). Die Liste der Exit-2-Fälle fehlt ein Fall: ein unbekanntes Flag bzw. ein Fehler von `fs.Parse` (`:930-933`). `check code` mit unlesbarem Manifest (`:966-970`) entfällt mit E2. |
| B5 | **FALSCH** (Zahlen und Importe) | Am Tag gezählt (`wc -l`): `okf` **784 / 1.805** (stimmt), `house` **1.301 / 3.385** (stimmt), `run` **666 / 1.905** (nicht 665 / 1.882), zusammen **2.751 / 7.095** (nicht 2.750 / 7.072). Davon ab `inner_parallel_test.go` mit 174 Zeilen, der nicht mitzieht: 2.751 / 6.921. Im Paket `run` liegen außerdem `CheckCode` (`run.go:156-163`) und `TestCheckCode` (`run_test.go:899ff.`), die mit E2 wegfallen. Die Behauptung zu den Importen ist falsch: `run_test.go:11` importiert **`internal/testlock`** (neu seit `08d985a`, genutzt in `TestADeclarationThatCannotBeOpenedIsReportedToo`, `:341-360`), und `bundle_test.go` importiert `time/tzdata`. loomux hat `internal/testlock` mit `Lock(t, path)`, der Umzug geht also. Der Test setzt allerdings `BRAIN_STATE_DIR` und braucht `LOOMUX_STATE_DIR`. Die Helfer `writeBundle`, `vault` und `signpost` gibt es, Fixtures keine. `inner_parallel_test.go:7-11` stimmt. |
| B6 | Anker verschoben | Parallel sind nicht die Zeilen `run.go:182-222` von `CheckAll` selbst: `CheckAll` steht bei `:152-154` und ruft `runAreas` (**`:182-232`**). Die Goroutinen stehen bei `:188-209`, ein gemeinsames `now` bei `:184`, die Föderation erst nach allen Bündeln bei `:231`. `CheckBundle` mit der ganzen Registrierung steht bei **`:113-137`** (Begründung `:122-130`). `:140-151` ist der Kommentar zu `CheckAll`. |
| B7 | OK, ein Anker verschoben | `cli.py:570`, `:754`, `:1669-1677`, `:575-582`, `:876-907`, `:659-663` und `:859-873` stimmen. Unter `wiki` stehen `init` bei `:661` und `gate` bei **`:665`** (nicht 664). Nebenbei: Oben gibt es außerdem `init` (Repo-Onboarding, `:639`) und `wiki-gate` (`:669`). |
| B8 | OK | `scaffold.go:11-14` (Kommentar) und `:96-112` (`InitBundle`, schreibt über `lock.ReplaceText`, `:106`) stimmen. `area.go:330,335` stimmen. |
| B9 | OK | `manifest.go:47-84` (`knownTypes`, bis `:85`) und `:232-242` (`KnowsType`) stimmen. `types.py:30-33` (`ALIASES`) stimmt, die Ränge stehen bei `types.py:36-43`. |
| B10 | OK, Pfad präzisieren | `write_if_changed` liegt in **`src/brain/writer.py:6-23`**, nicht unter `wiki/`. `retype.py:74`, `:79-109` und die Regex `:91` stimmen. Übersprungen werden Gerüstdateien `:48`, kaputtes Frontmatter `:51`, `UnicodeDecodeError` `:59-65`, nur CR `:82-89` sowie gequotet oder gefaltet `:101-108`. |
| B11 | OK, mit Ergänzungen | `wiki.go:22-24` (Exit 2) stimmt, ebenso `cli.py:1562-1593` und `:1655-1666`. Die Weigerung „Wiki fehlt“ steht bei `:1574-1580` und greift **auch im Lauf über alle** (`:1583-1586`): ein Bereich mit `wiki_path`, aber ohne Verzeichnis, bricht den ganzen Lauf ab. Alle Weigerungen enden über `LookupError` in `main` mit `error: …` auf stderr und **Exit 1** (`cli.py:2408-2436`), nicht 2. Python `lint` hat außerdem `--file` (`:566`), und `--scope` hat den Vorgabewert `all` (`:567`). `lint.py:590-611` stimmt. |
| B12 | **FALSCH** in der Aussage „die Go-Form hat sie alle als `house/*`“ | `RULES` `:539-552` enthält 12 Funktionen (stimmt). `missing_type` gibt zwei Regelnamen aus, `broken-frontmatter` und `missing-type`, macht zusammen 13. loomux hat fünf (stimmt). Die Anker `bundle.go:234,284,325,376,415,454` und `federation.go:242,327` stimmen (orphan, dead-link, outside-area, untouched, implemented-without-commit, long-planned; wrong-direction, unlisted-area). In `page.go` steht aber: `:105` = `no-sources`, `:161` = **`source-incomplete`**, `:228` = **`unknown-type`**, `:261` = `conflict-count`, `:298` = `stale`. Daraus folgt: (a) `broken-frontmatter` gibt es in `house` **nicht**. House überspringt solche Seiten (`judgedAsConcept`, `page.go:86-88`), zuständig ist `okf/frontmatter-unparsable` (`okf/errors.go:83-92`). (b) `missing-type` heißt in Go `house/unknown-type`, ohne Alias-Zweig; fehlender oder leerer Typ gehen an `okf/type-missing`. (c) Die zweite Hälfte von `no-sources` ist aufgeteilt auf `house/source-incomplete` (nur doc_id, content_hash, revision), `okf/source-resource-missing` und `okf/source-id-missing` (Warnung). Ein `lint --scope` nur aus `house` fände also kein `broken-frontmatter` und meldete unvollständige Quellen unter anderem Namen. Er braucht dafür `okf`-Befunde oder eine Abbildung (siehe Tabelle unten). |
| B13 | OK | `model.go:100-106` und `lint.go:101` stimmen. Nebenbei: Das Go-`BundleContext` hat kein `is_shared`, `shared_scopes` und `expected_targets`. `house` nimmt dafür die Registrierung (`FederationFor`). |
| B14 | Teilweise **FALSCH**, Anker ungenau | Der Rahmen `server.py:106-310` ist ungenau: `Upkeep` steht bei `:106-286`, `_unreadable_line` bei `:289-317`, `_reconcile_registered` bei `:320-334`. Richtig sind `:508,534`, `:163-167`, `:200-203`, `core.py:27`, `:176-183`, `:43-47`, `:387-400` und `:195-198`. Genauer ist: `CATCH_UP_NOTICE` kommt nur, solange der erste Durchgang nicht fertig ist (`not upkeep.settled`, `:388`), gewartet wird aber vor jedem Aufruf. `failure` wird auch in einem Takt ohne fälligen Durchgang gelöscht (`:188` steht hinter beiden Zweigen), nicht nur nach einem guten Durchgang. **Falsch** ist „Jede Zeile beginnt mit `! `“: `status` bekommt `upkeep.notes(channel)` **ohne** Präfix (`:463-464`), nur die anderen vier bekommen `! ` (`:468`). `_trailer` steht bei `:445-468`. `_with_notes` (`:471-485`) hängt nicht bloß an: Es ersetzt `content` durch **einen einzigen** Textblock aus dem ersten Block und den Zeilen, weitere Blöcke fallen weg, und ist der erste Block kein Text, beginnt die Antwort mit einer Leerzeile. Den Cloud-Text gibt es (`:254-258`). Gefiltert wird in `notes` (`:228,237`) und `_unreadable_line` (`:310-316`). Die Texte nennen `brain cases` (`:235`) und `brain reconcile` (`:261`), dazu die Kopfzeile `` `status` lists them `` (`:286`). |
| B15 | Anker verschoben | `serve.go:14-22` (kein `maintenance`-Import) stimmt, ebenso `Reconcile` `:113`. `ErrNoReviewCentre` steht bei **`:91`** (nicht 89). In `stamp.go` steht `ReconcileInterval` bei **`:16`** (nicht 17); `ReadLastRun` `:33`, `Stale` `:54` und `StaleReconcile` `:60` stimmen. `ReadLastRun` und `StaleReconcile` nehmen `(stateDir, fallbackDir …)`, keinen Lookup. `catchUp` steht bei **`maintenance.go:62-83`** (nicht 60-81), `time.Now().UTC()` bei `:78`. `tools.go:39-99` stimmt (`handler` `:39-76`, `withFindings` `:99-109`). `parity/stufe-1b-2.md:113-123` stimmt. |
| B16 | OK, Liste unvollständig | Die Tests stehen bei `imports_test.go:65` und `:83-103`, die Listen bei `:21-27` (`serve`, `bridge` **und `go-sdk/mcp`**) und `:35-42` (`apply`, `evidence`, `maintenance`, `vcs`). Spec `:147` stimmt. |
| B17 | Teilweise **FALSCH** | `cli-reference.md:411` stimmt als Vorbild, die Zeile gilt aber für `reindex`, `embed`, `reconcile` und `area add` (Stufe 3a); eine gleichlautende Zeile steht bei `:452`. „Jeder Befehl dieser Stufe hat das Flag“ gilt nur für die Python-Befehle `lint` (`cli.py:568`), `types` (`:573`), `retype` (`:582`) und `wiki init` (`:663`). **`brain check` (Go) kennt kein `--state-dir`**: Die Flags sind nur `--notes`, `--scope` und `--lane` (`main.go:926-929`), und der Zustandsort kommt aus `config.StateDir()` bzw. `BRAIN_STATE_DIR`. Für `check` ist das also keine Abweichung. |

### Texte house gegen lint.py

„Go“ meint `pkg/check/house/*.go` am Tag. Wo `house` die Regel nicht hat, steht die Go-Regel der Achse `okf` in Klammern. Python ist `src/brain/wiki/lint.py`. Go `%q` schreibt doppelte Anführungszeichen, Python `!r` einfache.

| Regel (Python) | Id Go | Text Go | Stufe Go | Text Python | Stufe Python | gleich/ungleich |
|---|---|---|---|---|---|---|
| broken-frontmatter | *(nicht in house)* → `okf/frontmatter-unparsable` | `frontmatter is unreadable: <fehler>` (`okf/errors.go:87-88`) | error | `frontmatter is unreadable: {page.broken_frontmatter}` (`:82`) | error | Text gleich, **Id ungleich** (andere Achse). Der Fehlertext selbst kommt von verschiedenen YAML-Parsern und ist deshalb im Einzelfall verschieden |
| no-sources (leer) | `house/no-sources` | `sources[] is empty` (`page.go:105`) | error | `sources[] is empty` (`:128`) | error | gleich |
| no-sources (unvollständig) | `house/source-incomplete` + `okf/source-resource-missing` + `okf/source-id-missing` | `sources[%d] lacks doc_id, content_hash, revision` (`page.go:161-163`); `sources[%d] has no resource` (error); `sources[%d] has no id` (warning) | error / error / warning | `source entries lack id, resource, doc_id, content_hash or revision: <id>, …` (`:144-145`), **eine** Zeile je Seite | error | **ungleich** (Id, Text, Anzahl, Stufe bei id) |
| unlisted-area | `house/unlisted-area` | `the signpost does not link to area "<scope>"; expected one of: <ziele>` (`federation.go:328-329`) | error | `the signpost does not link to area '<scope>'; expected one of: <ziele>` (`:528-529`) | error | **ungleich** (Anführungszeichen; Ziele in Go lexikalisch mit `filepath.Clean`, in Python über `.resolve()`) |
| wrong-direction | `house/wrong-direction` | `a shared area must not cite "<scope>"; promote the page instead (architecture 5.7.4)` (`federation.go:243-244`) | error | `a shared area must not cite '<scope>'; promote the page instead (architecture 5.7.4)` (`:421-422`) | error | **ungleich** (Anführungszeichen) |
| untouched | `house/untouched` | `unchanged for more than %d days` (`bundle.go:376-377`) | warning | `unchanged for more than {n} days` (`:178`) | warning | gleich |
| stale | `house/stale` | `stale_after %s has passed` (YYYY-MM-DD, `page.go:298-300`) | error | `stale_after {isoformat} has passed` (`:159`) | error | gleich |
| implemented-without-commit | `house/implemented-without-commit` | `realization: implemented without implemented_in` (`bundle.go:415-416`) | error | dasselbe (`:203`) | error | gleich |
| long-planned | `house/long-planned` | `planned and untouched since %s` (`bundle.go:454-456`) | warning | `planned and untouched since {mtime.date()}` (`:229`) | warning | gleich |
| missing-type | **`house/unknown-type`** (fehlend/leer: `okf/type-missing`) | `"<typ>" is neither core nor catalogue; add it to `[wiki] types` in the area's manifest, or use a catalogue name` (`page.go:228-231`); fehlt: `no type; every OKF page needs one`; leer: `type is empty; every OKF page needs a non-empty one` (okf, error) | error | `'<typ>' is neither core nor catalogue; …` (`:108-111`); fehlt: `no type; every OKF page needs one` (`:104`); Alias: `'<alt>' is an old name for '<neu>'; rename it` (`:107`); leer: fällt unter „neither core …“ mit `''` | error | **ungleich** (Id, Anführungszeichen, Alias-Zweig fehlt in Go, Leer-Typ anders) |
| conflict-count | `house/conflict-count` | `%d conflict box(es) present, open_conflicts is missing` / `open_conflicts says %d, %d box(es) found` (`page.go:261-267`) | error | dieselben zwei (`:447-448`, `:459-460`) | error | gleich |
| outside-area | `house/outside-area` | `target leaves the area and cannot be checked: <ziel>` (`bundle.go:325-326`) | warning | dasselbe (`:375`) | warning | gleich |
| dead-link | `house/dead-link` | `target does not exist: <ziel> (resolved: <pfad>)` (`bundle.go:284-286`) | error | dasselbe (`:326`) | error | gleich |
| orphan | `house/orphan` | `no page and no catalog entry links here` (`bundle.go:234-235`) | error | dasselbe (`:300`) | error | gleich |

**Ergebnis:** 4 von 13 Regeln haben einen anderen Text: `no-sources` (unvollständig), `unlisted-area`, `wrong-direction` und `missing-type`. Bei zwei Regeln weicht die Id ab, bei `missing-type` (`unknown-type`) und bei `broken-frontmatter` (nur `okf/frontmatter-unparsable`). Die zweite Hälfte von `no-sources` ist auf drei Go-Regeln aufgeteilt. Die Stufen stimmen überall überein, mit einer Ausnahme: `okf/source-id-missing` ist eine Warnung, während der entsprechende Teil in Python ein Fehler ist.

#### Wo die Auslöser abweichen

- **Kaputtes Frontmatter:** Python gibt eine solche Seite an jede Regel weiter. Weil `meta` dann leer ist, meldet `no_sources` zusätzlich `sources[] is empty`, und `conflict_count` meldet Kästen im Rumpf als `open_conflicts is missing`. Go-`house` überspringt die Seite in allen fünf Seitenregeln (`judgedAsConcept`) und meldet dort nichts. In den Bündelregeln (`judgedInBundle`) wird sie wie in Python beurteilt.
- **Gerüstdateien:** `okf/frontmatter-unparsable` beurteilt auch `index.md`, `log.md` usw. (`okf/errors.go:70-82`). Python schließt sie vor jeder Regel aus (`lint.py:582-586`).
- **orphan:** Go zählt jede `index.md` in jeder Tiefe als Linkquelle (`bundle.go:184-199`, bewusst so). Python liest nur das `index.md` an der Bündelwurzel (`:281-282`). Eine Seite, die nur in `topics/index.md` steht, ist in Python verwaist, in Go nicht.
- **no-sources (unvollständig):** Python prüft Wahrheitswerte, Go schneidet vorher Leerraum ab (`strings.TrimSpace`). Python prüft `id` und `resource` mit; Go meldet sie auf der Achse `okf`, und zwar je Eintrag statt einmal je Seite.
- **implemented-without-commit:** Go schneidet Leerraum ab. Ein `implemented_in`, das nur aus Leerraum besteht, gilt in Go als fehlend, in Python nicht (`bundle.go:393-398`).
- **missing-type:** Ein leerer oder nur aus Leerraum bestehender Typ ergibt in Python `missing-type`, in Go `okf/type-missing` und nichts in `house`. Für einen Alias-Typ gibt Go die allgemeine Meldung aus.
- **unlisted-area:** Go überspringt einen Wegweiser, dessen Manifest nicht lesbar ist oder dessen `[layout] hub` aus dem Vault hinausführt (`hubFolder`, `federation.go:334-372`). Python bricht dort mit einem Fehler ab. Go fragt jeden Wegweiser, falls es mehrere gibt. Die Ziele werden in Go lexikalisch aufgelöst, in Python mit `.resolve()`, das Symlinks folgt.
- Keine Abweichung bei den Auslösern von `untouched`, `stale`, `long-planned`, `outside-area`, `dead-link` und `wrong-direction`.
