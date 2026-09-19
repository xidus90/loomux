# Stufe 2b: `check commit-msg` mit `[commit]`, `--calibrate` und `--language`

**Stand:** 2026-09-19, umgesetzt (100 % Testabdeckung, Paritätsfälle 2b, vollständige Dokumentation). Rahmen:
`2026-09-14-loomux-fusion-design.md`, Stufe 2; Geschwister:
`2026-09-19-loomux-stufe-2a-design.md` (umgesetzt), 2c (offen). 2b hängt an
nichts aus 2a außer `child.Run` für `git log`.

2b portiert `ultraloom commit-msg` (`src/ultraloom/commit/*.py`, 1.444 Zeilen
mit Tests, davon `language.py` 1.037) nach Go und ersetzt die heutige
Wortlistenprüfung in `internal/verify/commit/language.go` (70 Zeilen: die
ulinit-Liste von 62 Zeilen aus `ultraloom/internal/commit/language.go` plus
der Conventional-Commits-Header über `internal/conventional`).

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| Ohne `[commit]` | **Englisch als Vorgabe**, Schwelle 2, keine Ausnahmen, Header-Check an. Abweichung von Python, das ohne Sektion gar nicht prüft |
| Kaputtes `[commit]` | Blockiert mit Exit 1 und einer Zeile, die Datei und Schlüssel nennt. Die Vorgabe greift nur bei **fehlender** Sektion |
| Conventional-Header | Schalter `[commit].conventional`, Vorgabe `true`. Nur loomux kennt ihn; weder ulinit noch Python |
| Prüflogik | **B:** die Python-Logik vollständig, alle Zeilen, Schwelle 2, **plus** die Go-Wortliste von heute als vierte Quelle und Umlautwörter als Treffer, beides nur für das Ziel `en` |
| Exit-Codes | loomux' heutige: 0 ok, 1 Ablehnung sowie Konfig- und Lesefehler, 2 falscher Aufruf. Kein Bruch, `release:major` entfällt |
| `--calibrate` ohne Sprache | Flag, sonst `[commit].language`, sonst die Vorgabe `en`. Folgt aus der Vorgabe-Entscheidung; Python verweigert hier |
| Abhängigkeiten | Keine neue. `golang.org/x/text` ist schon direkt eingebunden und liefert NFKD/NFC |

## Befunde, die den Entwurf formen

Gelesen und gemessen am 2026-09-19.

**Niemand nutzt `[commit]`.** Keine `.ultraloom/config.toml` und keine
`.loomux/config.toml` unter `#GIT` enthält die Sektion. Alle
commit-msg-Hooks (iam_backend, iam_frontend, iam_workers, ultraloom, loomux)
rufen die feste englische Form (`ulinit check commit-msg` bzw. `loomux check
commit-msg`). Das Schema lässt sich ohne Migrationslast festlegen.

**Die Python-Wortliste fängt kurze deutsche Betreffzeilen nicht.** Sie enthält
keine Entwicklerverben (`füge`, `hinzu`, `entferne`) und keine Umlautregel. Von
den acht deutschen Proben aus dem heutigen Go-Test lehnt Python bei Schwelle 2
keine ab.

**Messung.** Jede Nachricht ganz gescannt (Span-Übertrag, Trailer ab Zeile 2),
ein Umlautwort zählt höchstens einmal, Schwelle 2, die ganze Historie:

| Variante | deutsche Proben | loomux | ultraloom | ultra-brain |
|---|---|---|---|---|
| A: Python unverändert | 0/8 | 0/286 | 3/464 | 14/1071 |
| **B: gewählt** | **7/8** | **0/286** | 3/464 | 15/1071 |

B kostet gegenüber A eine Ablehnung mehr auf 1.821 Commits. Die Treffer sind
fast alle echte deutsche Body-Absätze in ultra-brain und drei ultraloom-Bodies,
die deutsche Wörter ohne Anführungszeichen erörtern. Die achte Probe
(`entferne ungenutzte importe`) hat einen Treffer und bleibt unter der
Schwelle. Die iam-Repos taugen nicht als Korpus: auch ihre jüngsten Betreffzeilen
sind oft deutsch, der Hook greift dort offenbar nicht.

**Kein Loader lehnt `[commit]` ab.** `verify.ParseConfig` liest nur
`doc["verify"]`, die Policy nur ihre Tabellen. Eine neue Sektion ist ohne
Konflikt möglich.

**Git unterscheidet nur 0 und nicht 0.** Pythons Kommentar zu `EXIT_INTERNAL`
(`commit/cli.py:18–19`), Exit 1 blockiere den Commit nicht, ist falsch: jeder
Exit ungleich 0 bricht `git commit` ab. Die Unterscheidung 1/2 hat keinen
Abnehmer; deshalb bleibt es bei loomux' Codes.

## Pakete

Alles in `internal/verify/commit/`; die CLI-Verdrahtung bleibt in
`internal/cli/check.go`.

| Datei | Inhalt | Vorlage |
|---|---|---|
| `words.go` | Map-Literale: `ordinary` für `en` und `de`, die Quellen German, English, Romance und die Go-Liste von heute. `Stopwords(lang)` bildet Vereinigung minus `ordinary[lang]` beim ersten Aufruf (`sync.OnceValue`), nicht beim Paketstart | `language.py:17–735` |
| `scan.go` | `Scan(text, lang, threshold, allow) []Finding` | `language.py:737–1037` |
| `policy.go` | `ReadPolicy(root) (Policy, error)` liest `.loomux/config.toml` selbst, wie `verify.ReadConfig` | `config.py` |
| `calibrate.go` | `git log -z --format=%B -n N` und die Tabelle | `calibrate.py` |
| `check.go` | `Check(msg, policy) error`: erst Sprache, dann (falls an) der Header. Ersetzt `ValidateCommitMessage` | `commit/cli.py` |

`language.go` und `language_test.go` entfallen; ihre acht Proben wandern als
Regressionsanker in die neuen Tests.

### `Scan`

Wie Python, Zeile für Zeile:

1. Die Scissors-Zeile (`# ---- >8 ----`) beendet den Scan.
2. `#`-Zeilen werden übersprungen und verändern keinen Span-Zustand.
3. Eine Leerzeile schließt jeden offenen Span.
4. Spans: Backtick-Spans tragen über Zeilen, Quotes nur innerhalb einer Zeile.
   Jede Ersetzung setzt ein Leerzeichen. In Zeile 1 wird ein offener Span nach
   dem Scan geschlossen.
5. `[[commit.allow]]`: Eine Zeile, auf die ein Muster passt, wird nach dem
   Span-Schritt übersprungen. Ein Span, den sie öffnet, trägt weiter.
6. Treffer (`hits`): Trailer ab Zeile 2 sind frei. Dann werden Namenspartikel
   und Pfad-Token entfernt. Die Stoppwörter werden auf der gefalteten Zeile
   (`ä→ae`, `ö→oe`, `ü→ue`, `ß→ss`) gesucht, ohne Wörter, die an `-` oder `_`
   hängen. Dazu kommen Läufe fremder Schrift.
7. Eine Zeile mit `len(hits) >= threshold` wird zum `Finding{Line, Text, Hits}`.

**Umlautregel (neu, nur Ziel `en`).** Auf der Zeile nach Schritt 6 ohne
Faltung: Ein Wort, das `ä ö ü Ä Ö Ü ß` enthält, ist ein Treffer. Ein Wort ist
höchstens ein Treffer, `für` zählt also als Stoppwort `fuer` und nicht noch
einmal als Umlautwort. In `hits` erscheinen Stoppwörter gefaltet wie in Python
(`fuer`), reine Umlaut-Treffer wie getippt (`Übersicht`). Bekannte Kosten: zwei
Eigennamen mit Umlaut in einer Zeile (`Müller`, `Zürich`) lehnen ab. Der
Ausweg ist `[[commit.allow]]`; die Konfig-Doku zeigt das Beispiel.

**RE2 kennt kein Lookahead.** `PATH_TOKEN`
(`\S*(?:[/\\]\S*|\.[A-Za-z0-9]{1,5})(?=\s|$)`) wird zu einem Prädikat je
Token, gespalten an `unicode.IsSpace`: Das Token enthält `/` oder `\`, oder es
endet auf `.` gefolgt von 1–5 Zeichen aus `[A-Za-z0-9]`. Das ist äquivalent:
`\S*` zusammen mit dem Lookahead zwingt jeden Treffer, am Token-Anfang zu
beginnen und am Token-Ende zu enden. Ein Suffix eines Tokens erfüllt die
Bedingung nur, wenn das ganze Token sie erfüllt.

**Schrift ohne `unicodedata.name`.** Python liest die Schrift aus dem Präfix
des Zeichennamens. Go nimmt die Tabellen `unicode.Scripts`, geprüft in einer
festen Reihenfolge. Latin, Common und Inherited zählen nicht. Han, Hiragana
und Katakana werden zu `CJK`. Der Dehnungsstrich U+30FC ist in Go `Common`,
in Python über seinen Namen CJK; er bekommt eine eigene Regel. Kombinierende
Zeichen (Kategorie `M`) hängen am laufenden Lauf. Der Lauf wird wie in Python
auf der NFKD-Form gebildet, vor der Ausgabe NFC-normalisiert und auf 12
Zeichen gekappt. Wo Namenspräfix und Script-Tabelle auseinanderlaufen
(`MODIFIER LETTER …`, `FEMININE ORDINAL INDICATOR`), misst der Plan an den
portierten Vektoren; jeder Unterschied ist ein Eintrag der Abweichungsliste.

## Schema `[commit]`

```toml
[commit]
language     = "en"      # "en" | "de"; Vorgabe "en"
threshold    = 2         # Ganzzahl > 0, kein bool; Vorgabe 2
conventional = true      # Conventional-Commits-Header; Vorgabe true

[[commit.allow]]
regex  = '^Quote: '      # Go-RE2, beim Laden kompiliert
reason = "…"             # Pflicht, nicht leer
```

- Unbekannte Schlüssel in `[commit]` und in `[[commit.allow]]` werden
  abgelehnt; die Meldung nennt die bekannten.
- `match` in `[[commit.allow]]` wird mit eigener Meldung abgelehnt: Ein Glob
  hat an einer Textzeile keine klare Bedeutung, `regex` ist der Schlüssel.
  Diese Prüfung läuft vor der auf unbekannte Schlüssel, wie in Python
  (`config.py:117–139`).
- Ein ungültiger Ausdruck scheitert beim Laden, nicht beim ersten Treffer.
- `[commit]` ist keine Tabelle, `language` hat einen unbekannten Wert,
  `threshold` ist kein Integer oder ≤ 0, `allow` ist keine Tabellenliste:
  jeweils eine Zeile `<pfad>: <was>` und Exit 1.

## Aufrufe

| Form | Verhalten |
|---|---|
| `loomux check commit-msg <datei>` | Der Hook. `--language` ist verboten (Exit 2): ein Commit darf die Regel nicht wählen, nach der er beurteilt wird |
| `loomux check commit-msg --calibrate N [--language en\|de]` | Keine Datei (sonst Exit 2); `N ≥ 1` (sonst Exit 2; `git log -n -1` hieße unbegrenzt). Sprache: Flag, sonst `[commit]`, sonst `en`. `allow` aus der Konfig gilt beim Messen mit, `threshold` nicht: die Tabelle zeigt die Schwellen 1–4 |

**Ausgabe der Ablehnung** (stderr, Exit 1), Aufbau wie Python:

```
loomux check commit-msg: this message reads as German, and commits here are English.
  line 1: fix: Fehler beim Laden der Datei behoben
          hits: fehler, beim, der, datei
Rewrite it, or use `git commit --no-verify` if this cannot wait. The next
commit runs this check again.
```

Alle abgelehnten Zeilen erscheinen in einem Durchlauf; der Einzug folgt der
Länge des Labels. Ein Header-Fehler folgt als eigene Zeile, damit beide
Befunde auf einmal sichtbar sind.

**`--calibrate`** (stdout, Exit 0, sobald die Tabelle steht):

```
286 messages, checked as en
  threshold 1: 1 refused
    #260  fix(bench-hooks): validate before measuring and port the start-floor correction
  threshold 2: 0 refused
  …
```

`git log` läuft über `child.Run` mit 60 s Frist. Der Aufruf sitzt hinter
einer Paketvariablen wie `checkStart` in `cli/check.go`, damit Zeitüberschreitung
und Git-Fehler ohne echtes Repo testbar sind. Eine Zeitüberschreitung wird vor
dem Rückgabecode geprüft. `-z` trennt die Nachrichten, denn ein Body darf
Leerzeilen enthalten.

`.githooks/commit-msg` bleibt unverändert.

## Parität

### Aufzeichnen

`testdata/cases/2b-source/check-commit-msg/<fall>/` zeichnet die
**Python**-Form auf (`uv run --project <ultraloom> ultraloom commit-msg …`),
Welt mit `.ultraloom/config.toml`. Aufgezeichnet werden nur Fälle, in denen B
und Python gleich urteilen sollen:

- Spans über Zeilen, Quotes je Zeile, Absatzgrenze, Scissors, `#`-Zeilen
- Trailer ab Zeile 2, Trailer-Form in Zeile 1
- Pfade, Namenspartikel, Joiner
- Fremdschrift-Läufe, CJK zusammengelegt
- Schwellen, `[[commit.allow]]`
- Konfigfehler: unbekannter Schlüssel, `match`, fehlendes `reason`, kaputter
  Ausdruck
- `--calibrate` über eine kleine Probe-Historie in der Welt

**Voraussetzung:** Ein Mensch trägt die Policy-Regel für
`testdata/cases/2b-source/**` in `.loomux/config.toml` ein, wie für `2a-source`.

### Übersetzen

`testdata/cases/2b/check-commit-msg/<fall>/` hält das loomux-Gegenstück. Die
Fall-Läufer vergleichen Exit und stdout, nie stderr (`internal/cases/case.go:24`):

- Ablehnungsfälle: `compare = message`, Exit 1 statt Pythons 2, dazu
  `notes.md` mit Verweis auf den Abweichungseintrag.
- `--calibrate`: `compare = data`, die Tabelle ist byte-genau gleich.
- Welt mit `.loomux/config.toml` statt `.ultraloom/config.toml`.

Den Wortlaut auf stderr (`hits:`, Einzug, Pfad in Konfigfehlern) sichern
Unit-Tests.

### Abweichungsliste `parity/stufe-2b.md`

1. Ohne `[commit]` wird englisch geprüft (Python: keine Prüfung).
2. Go-Wortliste als vierte Quelle für das Ziel `en`.
3. Umlautwörter sind Treffer für das Ziel `en`.
4. `[commit].conventional` und der Header-Check.
5. Exit 1 statt 2 bei Ablehnung.
6. `--calibrate` ohne Sprache misst gegen `en` (Python: Exit 1).
7. Ungültiges UTF-8 in der Nachricht wird gelesen (Python: Exit 1).
8. Schrift-Randfälle aus der Script-Tabelle, einzeln benannt nach der Messung.
9. `allow`-Ausdrücke sind RE2, nicht Pythons `re`.
10. Pythons falscher Kommentar zu Exit 1 wird nicht übernommen.

## Tests und Tore

- **Portierung der Python-Tests** (`test_language.py` 794, `test_config.py`
  152, `test_calibrate.py` 155 Zeilen) als Go-Tabellentests. **Vorher** sortiert
  der Plan jeden Vektor ein: Parität (übernehmen) oder Abweichung (unter B
  gekippt: Go-Wort, Umlaut; invertieren und in die Abweichungsliste). Erst die
  sortierte Liste darf delegiert werden, sonst passt ein Subagent die Tests an
  den Code an.
- Die acht Proben aus `language_test.go`: sieben lehnen ab,
  `entferne ungenutzte importe` nicht (ein Treffer).
- Die heutigen Header-Tests (`TestValidateCommitMessageHeader`) gegen `Check`.
- Regressionsanker: `--calibrate` über die loomux-Historie ergibt bei
  Schwelle 2 null Ablehnungen (Integrationstest nur, wenn `git` da ist, sonst
  übersprungen; die Coverage trägt die Naht).
- 100 % je Funktion, TDD.

## Doku

- `README.md`, `README.de.md`: commit-msg-Absatz, `[commit]`.
- `docs/en|de/cli-reference.md`: `check commit-msg`, `--calibrate`,
  `--language`, Exit-Codes.
- `docs/en|de/configuration.md`: `[commit]` mit dem `allow`-Beispiel für
  Eigennamen.
- `docs/en|de/hooks.md`: der commit-msg-Hook, was er prüft.
- Fusions-Spec: 2b auf ✅; Z. 99–101 ergänzen, dass loomux' Kopie 70 Zeilen
  plus Header-Check hat (die 62 gelten für ulinit und stimmen).

## Release

`feat(check)` → `release:minor`. Das Konfigformat wird nur erweitert, die
Exit-Codes bleiben. Changelog:

- `### Added`: `[commit]` mit `language`, `threshold`, `conventional`,
  `[[commit.allow]]`; `check commit-msg --calibrate N [--language]`.
- `### Changed`: commit-msg prüft jetzt alle Zeilen der Nachricht, nicht nur
  den Betreff; ein einzelnes deutsches Wort genügt nicht mehr zur Ablehnung
  (Schwelle 2); die Ablehnung nennt jede Zeile mit ihren Treffern.

## Fertig, wenn

- `loomux check commit-msg` urteilt nach B; loomux' eigene Historie ergibt mit
  `--calibrate 300` bei Schwelle 2 null Ablehnungen.
- Alle Fälle unter `testdata/cases/2b` grün, die Abweichungsliste vollständig.
- `sh ci/gate.sh` grün, 100 % Coverage.
- Doku in beiden Sprachen nachgezogen.

## Reihenfolge für den Plan

1. `words.go` mit `Stopwords` und dem Filter.
2. `scan.go` ohne Umlautregel, Paritätsvektoren grün.
3. Umlautregel und Go-Liste, Abweichungsvektoren grün.
4. `policy.go`.
5. `check.go`, Ersatz von `ValidateCommitMessage`, CLI-Verdrahtung.
6. `calibrate.go` und `--calibrate`.
7. Aufzeichnung `2b-source` (nach der Policy-Regel), Gegenstücke `2b`.
8. Doku, Fusions-Spec, Abweichungsliste.

## Nachträge

Was Umsetzung und Review gegenüber dem Entwurf geändert haben. Wo Text und
Nachtrag sich widersprechen, gilt der Nachtrag.

1. **Der Exit-Code wird übersetzt, nicht von Hand gesetzt.** Der Entwurf wollte
   im Gegenstück Exit 1 eintragen und die Abweichung in `notes.md` erklären.
   Stattdessen kennt die Übersetzungstabelle jetzt `[[exit]]`
   (`testdata/cases/2b-map.toml`, `internal/dev/importcases`): Die Aufzeichnung
   behält Pythons 2, das Gegenstück bekommt beim Import 1. So bleibt die
   Übersetzung mechanisch, und **jeder** 2b-Fall muss bestehen. Eine Liste
   „genehmigter Abweichungen“, in der ein Fall bloß irgendwie scheitern muss,
   gibt es nicht mehr; sie hätte auch einen falschen Code durchgelassen.
   Die Regel gilt unbedingt: Aufruffehler (auch bei Python Exit 2) werden
   deshalb nicht aufgezeichnet.
2. **Unicode gegen ASCII war im Entwurf nicht vorgesehen.** Pythons `\b`, `\s`,
   `\w` und `str.splitlines` sind Unicode, Gos sind ASCII. Ohne Nachbau lehnte
   `Credit to von Müller` eine englische Zeile ab. Nachgebaut sind daher die
   Wortgrenze des Namenspartikels, das Wortmuster `[\p{L}\p{Nl}\p{No}]+` und
   die Zeilentrennung (auch in `Subject`). Zusätzlich faltet `foldGerman` das
   große `ẞ` wie Python zu `ss`.
3. **Drei Aufzeichnungen mehr als geplant:** `allow-match-key`,
   `allow-broken-regex` und `paragraph-break-span`. Damit sind es 19 Fälle.
4. **Die Abweichungsliste hat neun Einträge, nicht zehn.** Der Unicode-Nachbau
   (Nachtrag 2) hinterlässt keinen Unterschied und steht deshalb in der Fußnote
   der Liste, nicht als Eintrag. Pythons falscher Kommentar zu Exit 1
   (`commit/cli.py:18–19`) ist ebenfalls keine Verhaltensabweichung und steht
   dort.
5. **Offen, nur von Hand zu erledigen:** die Policy-Regel für
   `testdata/cases/2b-source/**` in `.loomux/config.toml`. Sie fehlt noch, und
   deshalb schützt die Schranke die Aufzeichnungen heute nicht.
