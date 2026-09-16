# Python nach Go: Entwurf

Stand 2026-09-04. Gegenstand ist die Ablösung der Python-Fassung von
ultra-brain durch die Go-Fassung und die beidseitig optionale Kopplung mit
ultraloom.

## 1. Ausgangslage, gemessen

| Grösse | Wert |
|---|---|
| Go im Baum | 6 727 Zeilen, 7 Unterbefehle |
| Python im Baum | 14 360 Zeilen, 23 Unterbefehle |
| grösste Python-Datei | `src/brain/cli.py`, 2 332 Zeilen |
| schwerster Block | `src/brain/maintenance/`, 4 474 Zeilen |
| Suche | `src/brain/search/`, 1 140 Zeilen, hinter einer Portschnittstelle |

Go kann heute `check`, `guard`, `lint`, `wiki-gate`, `catalog`, `status`,
`version`. Übergesiedelt sind damit der Prüfkatalog mit 31 Regeln auf zwei
Achsen und die Schreibschranke. Das ist rund ein Fünftel des Wegs.

`catalog` gibt es auf beiden Seiten mit **verschiedener Bedeutung**: Go druckt
die Dokumente eines Wikis als JSON, Python die Liste der Bereiche. Fünf Skills
meinen die Python-Bedeutung.

## 2. Zielbild

Die Grenze der Migration ist bewusst **nicht vorab festgelegt**. Scheibe 2
setzt je einen Vertreter aus Wartungsschicht und Suche in Go um und liefert
eine Zahl; erst danach wird entschieden, ob das Ziel „alles" heisst oder vor
einem der beiden schweren Blöcke endet. Der Schnitt der Scheiben ist so
gewählt, dass diese Entscheidung die Scheiben 3 bis 9 umsortieren kann, ohne 0
bis 2 zu entwerten.

## 3. Die Scheiben

Scheiben laufen nacheinander. Innerhalb der Verbscheiben gilt **lesend vor
schreibend**: ein Fehler beim Lesen zeigt eine falsche Antwort, ein Fehler
beim Schreiben beschädigt einen Bestand.

| # | Scheibe | Liefert |
|---|---|---|
| 0 | Prüfbestand | Fallformat, Läufer für beide Sprachen, Erfasser; zwei bestehende Befehle als Beweis umgestellt |
| 1 | `code/`-Achse | Dritte Achse in `pkg/check`, alle fünf Bahnen (`gofmt` zuerst, `pytest` als Prüfstein, dann ruff, mypy, coverage), Befundmodell für eine Bahn, die nicht läuft |
| 2 | Messscheibe | `brain case <id>` und `brain search` in Go; Ergebnis ist die Zahl, die die Grenze zieht |
| 3 | `catalog`, `read` | Die zwei kleinen lesenden Verben; löst die Doppelbedeutung von `catalog` auf |
| 4 | `neighbors`, Graph | `graph.py` (153 Zeilen) samt der zwei Funde aus Anhang B |
| 5 | Suche | Der Rest von `search/` und der qmd-Vertrag über HTTP; ein erster Pfad kam schon in Scheibe 2 |
| 6 | Index | `reindex`, `embed` — schreibt, aber nur Artefakte |
| 7 | MCP | Neu in Go; `scheibe-2c1-daemon` wird verworfen und dient nur als Vorlage |
| 8 | Wartung lesend | `cases` und der Rest der lesenden Wartung; `case` kam schon in Scheibe 2 |
| 9 | Wartung schreibend | `approve`, `reconcile` — der schwerste Block, zuletzt |
| 10 | Aufräumen | `src/brain` fällt, `pyproject.toml`, `uv tool uninstall brain`, Skills zurück auf blankes `brain` |

`convert`, `fetch`, `types`, `retype` und `bench` sind Werkzeuge am Rand und
werden **nach** Scheibe 2 einsortiert; womöglich fallen sie unter die dort
gezogene Grenze.

`cli.py` wird von jeder Verbscheibe angefasst und schrumpft scheibenweise,
statt einmal zu fallen. Das hält jede Scheibe für sich lauffähig.

## 4. Der Nachweis: ein gemeinsamer Prüfbestand

Sprachunabhängig, und er überlebt die Migration.

### 4.1 Form

Ein Verzeichnis je Fall unter `bench/cases/<verb>/<fall>/`:

```
cmd          die Befehlszeile
stdin        optional
world/       Registrierung, Wikis, Manifeste — der Zustand vor dem Lauf
stdout       erwartete Ausgabe
exit         erwarteter Rückgabewert
notes.md     warum dieser Fall existiert und was er festnagelt
```

`world/` ist der Kern: ein Fall bringt seine eigene Welt mit, statt die
Maschine zu befragen. Der heutige Gleichlauftest tut das Gegenteil und zählte
deshalb neun statt acht Bereiche, als ein Checkout den Zweig wechselte.

### 4.2 Läufer

Zwei dünne, einer je Sprache: `tools/cases.py` und `pkg/cases`. Beide bauen
`world/` in ein Wegwerfverzeichnis, setzen `BRAIN_STATE_DIR` darauf, fahren
`cmd`, vergleichen Ausgabe und Rückgabewert. **Kein Normalisierer** —
verglichen wird, was der Nutzer sieht.

Verben mit Nebenwirkungen vergleichen zusätzlich den Zustand **nach** dem
Lauf, also das Verzeichnis und nicht nur die Ausgabe.

### 4.3 Lebenslauf eines Falls

Ein Fall entsteht **vor** der Migration seines Verbs, aus der Python-Fassung:
fahren, Ausgabe erfassen, prüfen ob sie stimmt, festschreiben. Dann ist er rot
für Go. Wird er grün, ist die Aufgabe fertig. Python fällt, der Fall bleibt
und prüft danach Go allein.

### 4.4 Was das gegenüber dem Paarlauf kostet

Der Paarlauf des Prüfkatalogs verglich *jede* Ausgabe beider Fassungen und
fand vier Abweichungen, die kein Einzeltest sah. Ein Bestand prüft nur, woran
jemand gedacht hat.

Gegenmassnahme: Scheibe 0 baut einen Erfasser (`tools/cases.py record <cmd>`),
der einen Fall aus einem echten Lauf schreibt. Damit kostet ein Fall Minuten
statt einer Stunde, und ein Verb bekommt zwanzig Fälle statt fünf.

## 5. Die `code/`-Achse

`brain check` übernimmt die Codequalitätskette; die Bahnen werden Regeln auf
einer dritten Achse neben `okf` und `house`.

### 5.1 Die Abbildung, gemessen

```
ruff:   src\brain\_probe.py:1:8: F401 `os` imported but unused
mypy:   <datei>:<zeile>: error: <text>  [<regel>]
gofmt:  ein Pfad je Zeile
cover:  eine Zeile je Datei, Prozent und fehlende Zeilen
```

Drei von vier sprechen bereits in Befunden — Datei, Zeile, Regelname und
Meldung sind genau die Felder, die `check.Finding` trägt.

| Bahn | Regel | Grad | `Relative` |
|---|---|---|---|
| ruff | `code/ruff-<regel>` | Fehler | die Datei |
| mypy | `code/mypy-<regel>` | Fehler | die Datei |
| gofmt | `code/gofmt` | Fehler | die Datei |
| coverage | `code/uncovered` | Fehler | die Datei |
| pytest | `code/test-failed` | Fehler | die Testdatei |

### 5.2 Der Fall, der nicht passt

Eine Bahn, die gar nicht läuft — Werkzeug fehlt, Suite stürzt beim Sammeln ab
— hat weder Datei noch Zeile. Dafür `code/lane-broken` mit
`Relative: "(lane)"`, nach dem Muster von `house/manifest-unreadable`, das
`"(declaration)"` trägt und sich bewährt hat.

### 5.3 Was hier unbewiesen ist

Ob ein pytest-Lauf sich sauber in Befunde giessen lässt, ist offen: seine
Ausgabe ist ein Bericht, kein Datensatz. Scheibe 1 baut deshalb `gofmt` zuerst
(reine Dateiliste) und nimmt **pytest als Prüfstein**. Trägt es dort nicht,
bleibt die Bahn ein Aufruf mit Rückgabewert und meldet einen einzigen
`code/test-failed` mit der Ausgabe als Meldung. Hässlicher, aber ehrlich, und
es blockiert die Achse nicht.

### 5.4 Wo die Bahnen stehen

Im Manifest unter `[check] lanes` — dieselbe Datei, aus der `[wiki] types` und
`[layout]` kommen. Ein Projekt ohne Bahnen bekommt keine `code/`-Achse und
merkt von ihr nichts. Kein zweites Konfigurationsformat.

## 6. Die Kopplung zu ultraloom

Beide Richtungen sollen optional sein. Sie sind es heute nicht gleichermassen.

**loom ohne brain — fast fertig.** `internal/brainpath` sucht `brain` über
`LookPath`; fehlt es, bleiben die Wiki-Hooks weg und der Rest läuft. Offen:
ein Rest der alten Anheftung (`uv run --directory … brain`) steht noch im
Code, und `ulguard status` meldet „UltraBrain Wiki: Active" auch dort, wo kein
Manifest liegt.

**brain ohne loom — heute unmöglich.** `hooks/git/pre-commit` ruft
`loom check lint|types|test|coverage`. Nach Scheibe 1 ruft es
`brain check code`, und die Abhängigkeit fällt.

**Die Doppelung, und wie sie nicht auseinanderläuft.** Danach können zwei
Werkzeuge Lint fahren. Eine Wahrheit, zwei Leser: die Bahnen stehen im
brain-Manifest unter `[check] lanes`; ultraloom liest sie, wenn ein Manifest
da ist, statt eigene zu erfinden — so wie es heute schon `[area] wiki = true`
liest. Wo kein brain-Manifest liegt, gilt `.ultraloom/config.toml` wie bisher.
Damit gibt es nie zwei gültige Antworten für dasselbe Projekt.

Die Änderungen an ultraloom gehören in dessen Repo; Anhang A führt sie als
drei Aufgaben, damit die dortige Sitzung sie übernehmen kann.

## 7. Arbeitsweise

Ein Worktree je Scheibe (`claude/<scheibe>`), Merge per Fast-Forward, Zweig
und Worktree danach weg. Innerhalb einer Scheibe: ein Umsetzer je Aufgabe,
danach eine Prüfung, Fixrunden bis grün.

**Parallel läuft die Arbeit drumherum**, nicht die Umsetzung: die Prüfung von
Aufgabe *n* neben dem Auftrag für *n+1*, und der Fallerfasser nimmt Fälle für
die **nächste** Scheibe auf, während die laufende baut.

### 7.1 Bindende Vorgaben je Aufgabe

Aus der Prüfkatalog-Arbeit übernommen, weil jede einzelne dort etwas gefunden
hat:

- TDD, zwei Commits je Zyklus, Fehlschlagen gesehen.
- Vier Mutationsdurchgänge. **(a2) über Teilausdrücke ist Pflicht**: er fand
  in sechs Aufgaben hintereinander echte Defekte, die (a) nicht sah.
- **Eine Eingaberaum-Probe neben den Mutanten.** Bei den Wächtern war sie die
  wichtigere Hälfte: 100 % Abdeckung, 554 Mutanten und 79 grüne
  Gleichlauftests standen neben einem Loch, das eine Eingabe fand.
- 100 % Abdeckung, jeder Ausschluss begründet.
- Jeder Zeilenbeleg selbst nachgeschlagen. Keine Zahl in einer
  Commit-Nachricht, die nicht unmittelbar davor gezählt wurde.
- Gemessen statt behauptet, auch und gerade in Kommentaren.

### 7.2 Drei Fallen, die in dieser Arbeit getreten wurden

1. **Nach einem Merge per Ref-Update trägt ein Worktree auf dem bewegten Zweig
   einen neuen HEAD über einem alten Baum.** Der Index enthielt 7 154
   gelöschte Zeilen, unbemerkt. Nach jedem Merge Worktrees prüfen.
2. **Der Haupt-Checkout entscheidet mehr, als er sollte** — `core.hooksPath`,
   die Registrierung und die Schranke hängen an ihm. Steht er auf dem falschen
   Zweig, greift nichts, und alles sieht grün aus.
3. **Die Prüfkette kostet vier Minuten je Commit.** Nach Scheibe 1 messen, ob
   `brain check code` schneller ist.

## 8. Offene Punkte, die dieser Entwurf nicht entscheidet

- **Die Grenze der Migration.** Scheibe 2 liefert die Zahl.
- **pytest als Befunde.** Scheibe 1 liefert die Antwort; der Rückfall ist in
  Abschnitt 5.3 benannt.
- **Nebenwirkungsvergleich für `reconcile`.** Scheibe 0 legt den
  Zustandsvergleich fest, Scheibe 6 übt ihn als erste schreibende Scheibe aus;
  ob er für `reconcile` reicht, ist erst dort beantwortet. Scheibe 2 kann es
  nicht sagen — ihre beiden Vertreter lesen nur.
- **Verteilung der fünf Brain-Skills** — eingebettet in ultraloom oder über
  ein `brain skills install`. Erst nach Scheibe 3 entscheidbar, weil vorher
  nicht messbar ist, ob die Skills ohne ultra-brain-Repo laufen.

## Anhang A — was ultraloom danach ändern sollte

1. Den Rest der alten Anheftung aus `internal/brainpath` entfernen
   (`uv run --directory … brain`).
2. `ulguard status` nur dann „UltraBrain Wiki: Active" sagen lassen, wenn ein
   Manifest es hergibt.
3. `[check] lanes` aus dem brain-Manifest lesen, wo eines liegt, statt eigene
   Bahnen zu führen.

## Anhang B — bekannte Funde, die in Scheiben einfliessen

- `graph.py` schneidet Query und Fragment nicht ab und kennt nur drei
  Schemata: 56 fehlende Kanten, 48 Links unter falschem Grund gebucht
  (Scheibe 4).
- `brain wiki init` prüft `readonly` nicht und legt Bündel an, die der Agent
  nie befüllen darf (Scheibe 6 oder 9, je nach Grenze).
- `wiki-drift` kann für ein Wiki im Projekt nie feuern, weil
  `git status --porcelain` repoweit antwortet (eigener Zug).
- `LintBundle` und `house.orphan` zählen eingehende Links verschieden
  (Quellenmenge und Existenzprüfung).
- `uv tool list` führt einen Eintrittspunkt `brain`, dessen Datei nicht mehr
  existiert (Scheibe 10).
