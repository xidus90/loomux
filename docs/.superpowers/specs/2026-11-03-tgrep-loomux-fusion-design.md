# Fusion tgrep und Code-Graph: Design

Stand 2026-11-03. Basis: Zweig `docs/tgrep-loomux-fusion` (HEAD 1816634d). Die Plugin-Seite ist im zweiten Durchgang gegen `index.js` belegt. Dritter Durchgang: Die Entscheidungen des Menschen sind eingearbeitet (Offene Fragen 1, 2, 3, 10, 11, 13); die Befunde bleiben stehen. Vierter Durchgang: Suchkorpus (alle Textdateien), Ignore-Regeln und Größenlimit sind beschlossen (Offene Fragen 14, 15). Neu: Abschnitt „Suchkorpus und Walk“; Abschnitte „Abdeckung“, „Digest und Gültigkeit“, „Schnittstellen“ und „Privacy“ sind umgeschrieben. Nicht belegt bleiben: tgrep-Standardwerte (64 MiB) und tgrep-Ignore-Verhalten (in C1 per `tgrep --help` zu belegen), die genaue Mechanik von `core.ignorecase` (Test gegen git) und der Inhalt von `iam_backend` (nicht gelesen).
Grundlage: `docs/.superpowers/research/2026-11-03-tgrep-loomux-fusion-brief.md`.

## Anlass und Ziel

`loomux graph grep` durchsucht heute jede Datei des Graphen bei jeder Anfrage
neu (`internal/code/query/grep.go`: `Grep` liest per `os.ReadFile` jeden Pfad
aus `g.Nodes`, `internal/code/grep/grep.go`: `Search`). Der Treffer ist eine
Zeile, gruppiert nach umschließendem Symbol und sortiert nach `inDegree`. Das
Plugin `dsh-tgrep` schattet heute das eingebaute `grep` des Agenten und ruft
dafür die externe Binary `tgrep` auf. Die Binary ist auf diesem Rechner nicht
auf PATH, der Aufruf schlägt mit `spawn tgrep ENOENT` fehl.

Ziel: Die Textsuche bekommt einen Trigramm-Index in loomux, der zusammen mit
dem Graphen gebaut wird. Ein Treffer ist dann ein Graph-Knoten mit Rang und
Kanten, nicht nur eine Zeile; eine Datei ohne Knoten ist ein Dateitreffer (Zeile, ohne Symbol). Das Plugin wird eine dünne Fassade über
`loomux graph grep --json`. tgrep fliegt ganz aus dem Plugin und dem Betriebsweg. Vorher belegt ein Benchmark-Gate, dass loomux mithält (Abschnitte Performance-Ziele und Rollout).

## Entscheidungen des vierten Durchgangs

Vom Menschen beschlossen (diese Spec setzt sie um; die früheren Entscheidungen gelten weiter):

1. **Suchkorpus (Frage 14):** Die reine Suche deckt alle als Text erkannten Dateien ab. Der
   Graph bleibt bei seinen sieben Endungen. Ein Walk, zwei Auswertungen (Abschnitt „Suchkorpus
   und Walk“).
2. **Ignore-Regeln:** `.gitignore`, `.git/info/exclude` und `core.excludesFile` nur im
   Git-Repo; außerhalb keine Auswertung. Die Auswertung ersetzt `skipDirs` für den Suchkorpus.
   Der Graphkorpus bleibt bei `sourceset`.
3. **Größenlimit:** konfigurierbar, Vorgabe 64 MiB (Vorschlag, siehe Abschnitt „Größenlimit“),
   Gegenstück `--no-max-filesize`. Index-Gültigkeit schließt Limit und Ignore-Einstellungen ein.
4. **Zweiter Korpus (Frage 15):** `iam_backend` (Python), in C2 und C3.
5. **Gate-Methode:** gleiche Ignore-Regeln und Größenpolitik auf dem echten Repository, Gleichheit
   der Dateisätze vor der Messung geprüft und im Benchmark-Eintrag festgehalten.

## Korrekturen am Brief

Beim Gegenlesen des Repos haben sich drei Aussagen des Briefs als falsch
erwiesen. Das Design folgt dem Befund, nicht dem Brief.

1. **`sourceset` liest nicht, was git kennt.** `internal/code/sourceset/sourceset.go`
   läuft mit `filepath.WalkDir` über das Dateisystem. Es nimmt genau sieben
   Endungen (`.gd .go .godot .py .tres .tscn .uid`), lässt Dateien über 1 000 000
   Byte weg und überspringt Verzeichnisse mit Punkt sowie `skipDirs`
   (`node_modules`, `dist`, `build`, `vendor`, `testdata`, `venv`, `site-packages`
   u. a.). Git wird dort nicht aufgerufen. Folge: Ein Index auf dieser Dateiliste
   deckt `.md`, `.ts`, `.js`, `.json` und `.c` nicht ab. Ohne Rückfall (Entscheidung, Fragen 3 und 10)
   antwortet das Plugin dafür mit Fehler und Hinweis (siehe Abschnitt Fehlerfälle).
2. **`--json` gibt es bei `graph check` und `graph stats` schon.** `internal/cli/graph.go`
   hat `fs.Bool("json", …)` in `graphCheck` (Zeile 163) und `graphStats` (Zeile 510).
   Der Brief nennt beide ohne `--json`. Gleichlautend stimmt die CLI-Referenz
   (`docs/en/cli-reference.md` §6, Zeilen 915 und 923).
3. **Der Rebuild läuft nicht im Hintergrund.** `query/load.go` ruft
   `ask.EnsureFresh` mit dem synchronen `Build` auf. `ask/refresh.go` nimmt
   eine Sperre per `O_EXCL` (`rebuild.lock`). Hält ein anderer Lauf die Sperre,
   antwortet die Anfrage aus dem Stand auf der Platte und meldet das als Notiz.
   `internal/child` wird nur für die Lebendprüfung einer Sperre gebraucht, nicht
   zum Abspalten. Folge: Ein Plugin-Aufruf mit Refresh kann das 30-s-Timeout
   überschreiten, wenn der Graph neu gebaut werden muss.

Zusätzlich gilt: Der MCP-Dienst hat kein eigenes Tool namens `grep`. Das
Tool `graph_find_all` (`internal/serve/graph/tools.go`) ruft aber dieselbe
`query.Grep` mit `Keep: readable(area)` auf. Der MCP-Weg ist damit schon nach
`[privacy] never` gefiltert. Der CLI-Weg (`loomux graph grep`) ist es nicht.

## Stand der Plugin-Seite (belegt)

Der Abschnitt beschreibt den Ist-Zustand vor den Entscheidungen und bleibt als Beleg stehen.

Quelle: `C:\Users\micro\.dsh\profiles\desktop\node_modules\dsh-tgrep\index.js`
(1135 Zeilen, vollständig gelesen), dazu `package.json` und `cordis.patch.yml`
aus demselben Ordner. Paket `dsh-tgrep` 0.1.13, MIT, `engines.node >= 22.19.0`.
`node_modules\` enthält kein weiteres Paket; `@deepseek-ai/…` fehlt dort.
Umgebung: `tgrep` ist nicht auf PATH. `rg` (ripgrep 15.2.0), `loomux` und
Node v24.22.1 sind auf PATH.

**Ist-Zustand:** Das Plugin ist keine Fassade über `loomux`. Es ruft `tgrep`
direkt auf, baut die Argumente selbst, startet je Workspace einen
`tgrep serve`-Daemon und hat keinen Rückfall.

| Aussage (Entwurf) | Befund | Beleg |
|---|---|---|
| 30 s Timeout | Deklariert als `timeoutMs`. Kein Plugin-Timer. `exec.signal` wird an `spawn` gebunden; die Durchsetzung liegt beim Host (nicht belegt). | `index.js:759`, `:910`, `:959` |
| 300 Zeilen Voreinstellung | Bestätigt. `maxLines` ist Vorgabe und Kürzungsgrenze der Ausgabe, kein Limit der Suche. | `index.js:433`, `:457`; `cordis.patch.yml:9` |
| 20 MiB stdout-Deckel | Bestätigt, gemessen als Zeichenlänge (UTF-16), nicht in Bytes. Bei Überschreitung SIGTERM und **kein Fehler, keine Notiz**: Der Close-Handler lehnt nur bei leerem stdout ab; sonst kommt die Teilmenge als vollständiges Ergebnis zurück. Verstoß gegen A4. | `index.js:977–979`, `:1024–1027`, `:1029–1051` |
| 5000 Treffer Maximum | Bestätigt als Obergrenze der Klammer (`max_results`, `maxLines`). Ein Limit an tgrep gibt es nicht: `maxMatches` wird an `runTgrep` übergeben und dort nicht benutzt. Gekürzt wird erst in `render` und `presentationMeta`, nach dem vollen Lauf. | `index.js:433`, `:810`, `:858`, `:942` |
| Form von `matches` | Falsch. Ist: `{ path, lineNumber, line }`, flach. Kein `text`, keine Symbolfelder. | `index.js:914`, `:1040–1044` |
| Form von `SearchMeta` | Den Typ gibt es nicht. Die Karte heißt `presentationMeta`: `{ shape: 'matches', files, truncated, total }`. | `index.js:724–736`, `:815–819` |
| Tool-Schema | Namen stimmen. `max_results` hat die Klammer 20–5000, nicht 1–5000. Details: Abschnitt „Plugin-Tool-Schema“. | `index.js:746–808` |
| Rückfall | Existiert im Plugin nicht. Details: Abschnitt „Rückfallpfad“. | `index.js:993–999` |
| `spawn tgrep ENOENT` | Bestätigt. `tgrep` fehlt auf diesem Rechner; `loomux` ist dagegen auf PATH. | `index.js:993–999`; Shell-Prüfung |

**Fehlerverhalten (Ist)**

- Eingabeprüfung wirft `Error` mit festem Wortlaut (`pattern must be a non-empty string`, `include must be a positive glob filter …`): `index.js:845–856`, `:922–936`.
- `tgrep` fehlt (`ENOENT`): Ablehnung `Failed to run tgrep: …`, kein Rückfall. Die Daemon-Notiz geht verloren, weil `execute` vor dem `return` wirft (`index.js:908–914`).
- Exit ≥ 2: Ablehnung mit gefiltertem stderr (`index.js:1012–1020`).
- Exit 1: **kein Fehler**, leeres Ergebnis (`(no matches)`). Das ist die ripgrep-Konvention von tgrep. `loomux` nutzt Exit 1 für Fehler (`internal/cli/graph.go:431–433`). Die Klassifikation darf deshalb nicht übernommen werden.
- Signal ohne Ausgabe: Ablehnung `tgrep killed by signal …` (`index.js:1024–1027`).
- Abbruch über `exec.signal`: Ablehnung mit `AbortError` und `tgrep cancelled` (`index.js:944–949`, `:987–991`, `:1004–1008`).
- Die Daemon-Wartezeit läuft vor der Suche und außerhalb von `exec.signal`: bis `daemonReadyTimeoutMs` (Vorgabe 5000, Obergrenze 60000, `index.js:437`) (`index.js:870–873`, `:335–343`).

**Daemon (Ist)**

- Vorgabe `autoStartDaemon: true` (`index.js:436`, `cordis.patch.yml:19`). Je Suche wird `tgrep serve <root>` getrennt gestartet (`index.js:315–322`), `.tgrep/` im Workspace angelegt (`:308–309`) und der Index per `--index-path` gepinnt (`:613–615`, `:894`). Inaktivitätsende nach 30 min (`:45`, `:254–265`).
- Im Repo sichtbar: `.tgrep/serve.log` ist leer, `serve.json` fehlt. Der Start ist am fehlenden Binary gescheitert. Deshalb steht `.tgrep/` als untracked in `git status`.
- Widerspruch zum Ziel: Z3 („kein Server, kein Daemon“). Das Ziel-Plugin startet keinen Daemon.

**Registrierung (Ist)**

- `name` = `dsh-tgrep` (`index.js:17`), `inject = ['tools']` (`:19`), Konfiguration als Standard-Schema-Validator `Config` mit zehn Schlüsseln (`:423–445`).
- `apply(ctx, config)` (`:453`) normalisiert die Konfiguration noch einmal (`:454–465`). `enabled: false` beendet ohne Registrierung (`:467–470`).
- Tool-Ebene: ein Tool-Objekt `grep` (`:753`) wird über `tools.register` je Agent gelegt (`:482`): bei Neuanlage (`agent/created`, `:498`) und für bereits laufende Agenten (`:505–512`). Die Entfernung läuft über `ctx.effect` (`:489`, `:531–548`).
- Host-Ebene: nur wenn `get('grep')` leer ist (`:518–526`).
- `cordis.patch.yml` ist ein `insert`-Eintrag mit `id: tgrep`, `name: dsh-tgrep` und `config` (`cordis.patch.yml:1–36`). Zwei Kommentare dort stimmen nicht: „the call never fails“ (`:20–21`) gilt nur bei vorhandenem `tgrep`. „fall back to one-shot“ (`:6`) meint `tgrep` selbst (Scan ohne Index), nicht das Plugin.
- Schattung: Das Plugin importiert nichts außer Node-Modulen (`index.js:12–15`) und prüft das Stock-Tool nicht. Ob `tools.register` das Host-`grep` ersetzt, liegt im Host und ist nicht belegt.

## Ziele

- **Z1** Eine Textsuche über den Suchkorpus (alle als Text erkannten Dateien
  unter den Ignore-Regeln, Abschnitt „Suchkorpus und Walk“), die schneller ist als
  der Scan über denselben Korpus und dieselben Treffer liefert wie er. Der heutige
  Scan (nur Graph-Knoten) ist nicht mehr der Vergleichsmaßstab.
- **Z2** Jeder Treffer trägt sein umschließendes Symbol, dessen `inDegree` und
  Kanten, sofern der Graph den Pfad kennt. Ein Treffer ohne Symbol bleibt
  gültig und ist ein Dateitreffer.
- **Z3** Ein Binary, CGo-frei, deterministisch. Kein Server, kein Daemon, kein
  Lock außer dem vorhandenen Rebuild-Lock.
- **Z4** Der Index entsteht im selben Durchlauf wie der Graph und liest dieselben
  Bytes wie der Build. Ein Drift in einem der beiden Korpora löst einen vollständigen
  Build aus (Graph und Index). Der Index gilt nur, wenn Suchkorpus, Einstellungen
  und Ignore-Eingaben seines Digests dem aktuellen Record entsprechen (Abschnitt
  „Digest und Gültigkeit“).
- **Z5** Das Plugin ersetzt das `grep`-Tool, ohne dass bestehende Prompts oder
  Karten brechen. Das Tool-Schema bleibt kompatibel.
- **Z6** Kein Treffer aus einer `[privacy] never`-Datei gelangt auf dem
  Modell-Kanal in die Antwort.

## Nicht-Ziele

- Der Graph extrahiert weiterhin nur `.go .py .gd .tscn .tres .godot .uid` (Graphkorpus
  wie bisher über `sourceset`). Die Textsuche deckt dagegen alle als Text erkannten
  Dateien ab (Entscheidung 1). Was dennoch fehlt, nennt der Abschnitt „Abdeckung“.
- Kein Live-Nachführen des Index während einer Sitzung (beschlossen, Frage 1,
  Abschnitt Offene Fragen).
- Keine Symbol-Tabelle als eigene Datenstruktur im Index. Die Knoten des
  Graphen tragen Name, Art, Pfad, Spanne und Signatur bereits (`model.Node`).
- Kein Port von tgrep, keine Einbettung, Keine Abhängigkeit auf tgrep im Plugin:
  weder Laufzeit-Abhängigkeit, noch optionale Binary, noch Rückfall, noch
  tgrep-Flags als Vertrag. tgrep ist nur Entwicklungswerkzeug für das Gate (Plan, Teil C).
- Keine Regex-Engine außer Go `regexp` (RE2). Keine Lookarounds, keine
  Rückbezüge. Die Muster bleiben, was `grep.Search` heute akzeptiert.
- Kein Ersatz für `graph ask`, `callers`, `blast` oder die MCP-Werkzeuge.
  Das Plugin dupliziert keines davon.

## Anti-Ziele

- **A1** Der Index darf keine Dateiliste führen, die von der des Builds
  abweicht, und Build, Probe, Index und Scan verwenden dieselbe Walk-Funktion
  (Abschnitt „Suchkorpus und Walk“). Eine zweite Aufzählung wäre die Fehlerquelle,
  die `sourceset` verhindern soll (`internal/code/sourceset/sourceset.go:1–8`).
  Ausnahme: `goModPaths` (`internal/code/query/build.go:196–219`) bleibt ein eigener
  Walk für `go.mod`-Dateien; er ist keine Quelldatei und nicht Teil des Korpus.
- **A2** Das Plugin filtert `never`-Pfade nicht selbst. Es ruft nur einen
  Schalter auf, dessen Prädikat in Go liegt.
- **A3** Ein leerer Treffer ist nie die Antwort auf eine Frage, die der Index
  nicht beantworten konnte. Der Scan von loomux antwortet dann, mit
  Hinweis. Ist auch der Scan keine Antwort (kein Graph, Abdeckungslücke), gibt es einen Fehler mit Hinweis; ein Rückfall auf ein anderes Werkzeug entfällt (Frage 10).
- **A4** Kein stilles Verschlucken: Jede Abweichung vom Scan-Ergebnis steht im
  Ergebnis (`index`-Feld, Notiz).

## Architektur

### Go-Seite (loomux)

```
internal/code/corpus/       neu: der eine Walk, Ignore-Auswertung (Git-Regeln), Größen- und Binärregel
internal/code/trigram/      neu: Trigramm-Extraktion, Format, Lesen/Schreiben, Kandidaten
internal/code/query/build.go   schreibt den Index nach wiring.json und dem Freshness-Record
internal/code/query/grep.go    nutzt den Index für Kandidaten, sonst Scan wie heute
internal/code/grep/grep.go     Search bleibt die Verifikation; bekommt eine Kandidatenmenge
internal/cli/graph.go          graph grep: --channel, --glob; graph build schreibt den Index
```

`internal/code/trigram` importiert keinen Extraktor. Es ist eine Datenstruktur
über Bytes und Pfade, dieselbe Trennung, die `freshness` und `sourceset`
einhalten (Begründung: der Hook-Pfad darf den Parser nicht mitziehen).

Der Index wird in `graph build` geschrieben. Dort liegen die Bytes schon im
Speicher, die der Build hasht. Ein zweites Lesen entfällt.

### Plugin-Seite (DSH)

**Ziel:** Das Plugin ist eine Fassade. Es übersetzt das Tool-Schema in Flags,
ruft `loomux graph grep --json --channel cloud` auf, übersetzt die Antwort in
`matches` und `presentationMeta` und wählt zwischen Index-Pfad und Fehler
mit Hinweis (kein Rückfall, Frage 10). Es enthält keine Suchlogik und keine Privacy-Logik.

**Ist (belegt):** `dsh-tgrep` ist heute keine Fassade. Es enthält Argumentbau
(`index.js:586–627`), Prozessaufruf und Ausgabe-Parser (`:942–1054`) sowie eine
Daemon-Verwaltung (`:175–186`, `:283–367`). Tool-Ebene, Konfiguration, `inject`
und `apply` stehen im Abschnitt „Registrierung (Ist)“.

## Suchkorpus und Walk

### Wer liefert die Liste, wer wertet sie aus

- **Ein Walk.** `internal/code/corpus` enthält die Funktion `Walk(root, settings)`.
  Sie läuft einmal über den Baum und liefert pro Datei: relativer Pfad, Größe,
  Änderungszeit, ob Git sie ignoriert, ob sie getrackt ist. Sie liest keine Inhalte.
  Basis ist heute `filepath.WalkDir` in `internal/code/sourceset/sourceset.go:120`.
- **Zwei Auswertungen auf derselben Liste.**
  - *Graphkorpus:* das bestehende `sourceset`-Prädikat, unverändert: sieben
    Endungen, Größe bis 1 000 000 Byte (`sourceset.go:32`), kein Verzeichnissegment, das
    ein Punkt-Verzeichnis oder in `skipDirs` ist (`sourceset.go:58–79`, `:172–174`). Geprüft wird nur das Verzeichnissegment, nie der Dateiname: `.x.go` bleibt im Graphen, wie heute (`sourceset.go:133` prüft nur die Endung).
    `sourceset.Stat` wird zu einem Filter über `corpus.Walk`. Die Invariante
    „Graphliste unverändert“ bekommt einen Test (Vergleich mit der Liste vor dem Umbau).
  - *Suchkorpus:* nicht ignoriert, keine Punkt-Datei und kein Punkt-Verzeichnis (rg-Vorgabe; `--hidden` nicht in v1), Größe bis `max_filesize`,
    und nach dem Lesen kein NUL-Byte (Abschnitt „Binärdateien“).
- **Abstieg (Pruning).** Ein Verzeichnis wird nicht betreten, wenn es ein Punkt-Verzeichnis
  ist, oder wenn es in `skipDirs` steht, von Git ignoriert wird **und** kein getrackter
  Pfad darunter liegt. Für alles andere wird abgestiegen. Begründung: Ein ignoriertes
  `node_modules` ohne getrackten Inhalt ist für beide Korpora irrelevant und kostet
  sonst den Walk (`sourceset.go:48–52` belegt die Kosten für `testdata/`).
  Ignorierte Verzeichnisse außerhalb `skipDirs` werden betreten, weil der Graphkorpus
  sie heute nutzt; der Suchkorpus schließt ihre Dateien über das Ignore-Flag aus.
- **Harte Ausschlüsse unabhängig von Ignore-Regeln:** `.git` und alle Punkt-Verzeichnisse,
  auch `.loomux`. Ein Index, der seine eigene Ausgabe liest, wüchse bei jedem Build
  (`sourceset.go:161–163`). Folge: `.github/` ist in v1 nicht durchsuchbar (Frage 19).
- **Außerhalb eines Git-Repos** gibt es keine Ignore-Auswertung (Entscheidung 2). Dann
  liegen `node_modules`, `venv`, `dist` im Suchkorpus, weil `skipDirs` für den Suchkorpus
  entfällt. Das ist eine Folge der Entscheidung und wird gemessen, nicht verschwiegen.
- **Verbraucher:** Build (`query.BuildWith`), Probe (`freshness.Probe`, über `Refresh` und
  `EnsureFresh`, `ask/refresh.go:110`, `:155`) und Scan (`grep.Search`) nutzen dieselbe
  Funktion. Der Probe-Pfad bleibt stat-basiert; Lesen nur für Ignore-Quellen und bei
  Verdacht (Abschnitt „Digest und Gültigkeit“).
- **Ausnahme:** `goModPaths` (`internal/code/query/build.go:196–219`) bleibt eigener Walk
  für `go.mod`. Er liefert keine Quelldatei.

### Ignore-Semantik

- **Quellen:** `.gitignore` in jedem Verzeichnis zwischen Repo-Wurzel und dem Walk-Root
  sowie darunter (nested); `.git/info/exclude`; die globale Datei aus `core.excludesFile`
  (per `git config --path core.excludesFile`). Ist `core.excludesFile` nicht gesetzt,
  gilt die Standardquelle von Git: `$XDG_CONFIG_HOME/git/ignore`, und wenn auch das
  nicht gesetzt ist, `$HOME/.config/git/ignore`. **Belegt am 2026-11-04** mit
  `git check-ignore` in temporären Repositories, in denen `XDG_CONFIG_HOME` und `HOME`
  auf frische Verzeichnisse zeigten: die Datei unter `$XDG_CONFIG_HOME/git/ignore`
  greift, `~/.config/git/ignore` greift bei ungesetztem `XDG_CONFIG_HOME`, eine
  `~/.gitignore_global` ohne `core.excludesFile` greift nicht.
- **Repo-Erkennung:** `git rev-parse --show-toplevel` ab dem Walk-Root. Schlägt das fehl,
  gilt keine Ignore-Auswertung.
- **Muster:** Git-Regeln (Negation `!`, Anker `/`, `**`, Verzeichnismuster mit `/` am Ende,
  Escapes; eine Negation kann eine ausgeschlossene Verzeichnisdatei nicht wieder einschließen).
  Implementierung in reinem Go, keine neue Abhängigkeit. Tests gegen `git check-ignore`
  als Orakel (Tabelle plus Orakel-Vergleich auf dem loomux-Repo, siehe Teststrategie:
  Suchkorpus gleich `git ls-files --cached --others --exclude-standard`, gefiltert).
- **Getrackte Pfade** sind nie ignoriert. Die Tracked-Menge stammt aus `git ls-files -z`
  bei jedem Build. Der Probe-Pfad liest sie nicht neu; er vergleicht stattdessen Größe und
  Änderungszeit von `.git/index` (`git rev-parse --git-path index`). Ändert sich der
  Stand, wird die Menge neu gelesen; das löst ohnehin einen Rebuild aus. Die Menge und die aufgelösten Git-Pfade stehen im Record. `git add -f` und `git rm --cached` ändern so die Liste
  ohne Dateiänderung, und der Probe erkennt sie trotzdem.
- **`core.ignorecase`:** Ist der Wert `true`, passen Muster case-insensitiv; die
  Ausnahme für getrackte Pfade gilt auch dann. **Belegt am 2026-11-04** mit
  `git check-ignore` in einem temporären Repo: bei `core.ignorecase=true` ist `*.LOG`
  ein Treffer für `foo.log`, bei `false` nicht; der Vergleich faltet also Muster und
  Namen. Umgesetzt wird die Faltung auf den Mustervergleich (ASCII wie im Index), der
  Wert gehört in den Digest. Eine Ausnahme: ein Verzeichnis, unter dem noch eine
  getrackte Datei liegt, meldet `git check-ignore` als **nicht** ignoriert, obwohl ein
  Muster es nennt; der Walk entscheidet dort ohnehin über `HasTrackedUnder` und steigt
  ab, die Suchliste enthält die getrackte Datei.
- **Nicht ausgewertet (Grenzen):** `.ignore` und `.rgignore` (ripgrep liest sie),
  Editor- und Betriebssystem-Excludes, die git nicht kennt, Dateien ohne Lesezugriff
  (zählen als `unreadable`), Symlinks (siehe Frage 16).

### Größenlimit und Konfiguration

- **Vorgabe:** 64 MiB = 67 108 864 Byte, als Vorschlag des Menschen. Der tgrep-Wert ist
  **nicht belegt** (in C1 per `tgrep --help` prüfen).
- **Graphkorpus:** bleibt bei 1 000 000 Byte (`sourceset.go:32`). Dieses Design ändert
  das nicht. Ob der Graph ebenfalls konfigurierbar wird, ist Frage 17.
- **Schlüssel:** `max_filesize` (Typ `Int`, Bytes), Modul `Graph`, Vorgabe 67108864.
  Wert `0` heißt unbegrenzt. Schema: `internal/config/schema/schema.go` (Beispiel
  `Kind: Int`, Zeile 71).
- **Abschnitt:** vorgeschlagen `[grep]`. **Konflikt, markiert:** `docs/en/configuration.md`,
  Abschnitt „`[graph]` — there is none“ (ab Zeile 891), sagt, dass kein Graph-Abschnitt
  geplant ist. Der Vorschlag `[grep]` berührt diesen Befund nicht direkt, hebt ihn aber
  nicht auf. Der Mensch entscheidet (Frage 18). Ohne Entscheidung wird nichts geschrieben.
- **Änderung nur als Vorschlag:** `loomux config set grep.max_filesize <n> --propose`
  (`internal/cli/config.go:34`, `:54`). Der Agent schreibt `.loomux/config.toml` nicht selbst.
- **Rangfolge:** Vorgabe, dann Konfiguration, dann CLI-Flag. Das Flag gilt für den einzelnen
  Lauf. Flags gibt es bei `graph build` und `graph grep`.
- **Vergleich gegen den Record (Entscheidung):** Verglichen wird die Einstellung des Aufrufs
  (Flag, sonst Konfiguration, sonst Vorgabe) mit der Einstellung im Record.
  - Ohne Flag und mit Refresh: Weicht sie ab, ist das Drift. Der Rebuild nimmt die Einstellung
    des Aufrufs. Ein Hook oder MCP-Aufruf ohne Flag stellt damit immer den Stand der Konfiguration her.
  - Mit Flag: Weicht der Flag-Wert ab, wird **nicht** neu gebaut, sonst überschriebe eine Messung
    den Index. Die Abfrage läuft als Scan mit dem Flag-Wert und meldet `index: stale` mit Notiz
    „settings differ from index“.
  - Mit `--no-refresh`: nie Rebuild; dieselbe Gleichheitsregel wie oben.
  - Folge für die Messung: Das Gate baut mit `graph build --max-filesize X --no-refresh` und
    misst mit `--no-refresh --max-filesize X`. Zwischen Build und Messung darf kein Lauf ohne
    Flag liegen, weil er den Index auf die Vorgabe zurücksetzt.

### Binärdateien

- **Regel:** Eine Datei gilt als binär, wenn sie ein NUL-Byte enthält. Das ist die
  Heuristik von ripgrep (`rg --help`, Abschnitt `--binary`, Zeilen 555–576). ripgrep
  prüft nur den gelesenen Puffer und bricht beim ersten NUL ab. Hier wird die **ganze**
  Datei geprüft, weil die Bytes im Build ohnehin im Speicher liegen. Das ist eine Abweichung
  und macht das Ergebnis deterministisch.
- **Folge:** Binäre Dateien stehen im Suchkorpus, werden aber nicht indexiert und nicht
  durchsucht. Sie zählen als `skipped_binary`.
- **Grenze:** UTF-16-Dateien enthalten NUL-Bytes und gelten als binär. Sie fehlen dann in der
  Suche (Frage 20).

## Datenmodell und Indexformat

### Ablage

- Datei: `.loomux/state/graph/cache/trigram.idx`.
- Begründung: Derivate liegen seit jeher unter `cache/` (`internal/code/lexicon`
  und `freshness/fingerprint.json` tun das). Der Brief sagte „eigener
  Unterordner unter `graph/`“. Das ist eine **Abweichung**: `cache/` ist der
  bestehende Ort für regenerierbare Sidecars, ein neuer Unterordner würde
  nur eine zweite Konvention einführen.
- Schreiben atomar, wie `store.Write` (temporäre Datei mit PID, dann Umbenennen).

### Bytes

Alle Ganzzahlen little-endian. Das Format hat keine Zeitstempel.

| Abschnitt | Inhalt |
|---|---|
| Kopf (variabel, feste Teile zuerst) | Magic `LTGI` (4), Formatversion u32 (`1`; das Format ist unveröffentlicht, daher kein Versionssprung), Extraktorversion (u16 Länge + Bytes, begrenzt auf 64), Einstellungen (u32 Länge + kanonische Bytes: `max_filesize`, Binärregel-Version, Ignore-Regel-Version, `core.ignorecase`), Digest (32 Byte, s. u.), Dateianzahl u32, Trigrammanzahl u32, Postings-Anzahl u64, Pufferbytes u32 |
| Dateitabelle | je Datei in Byte-Reihenfolge des Pfads: Länge u16, Pfad (Schrägstrich) |
| Trigrammtabelle | je Trigramm, aufsteigend: Trigramm u32 (drei Bytes, Hochbyte 0), Offset der Postings u64, Anzahl u32 (16 Byte je Eintrag) |
| Postings | je Trigramm aufsteigende Datei-IDs als u32 |

Die Dateien sind in Byte-Reihenfolge der Pfade nummeriert. Die ID ist der
Index in der Dateitabelle.

**Empfehlung: Postings auf Dateiebene, ohne Positionen.** Die Verifikation
läuft ohnehin über `grep.Search` auf dem Kandidaten. Positionen würden nur die
Zeilensuche beschleunigen, die `Search` heute schon durch Zeilenzerlegung macht,
und vergrößern den Index. **Alternative:** Postings auf Zeilenebene mit
Zeilennummern. Das lohnt erst, wenn die Verifikation nachweislich der Engpass ist.

### Trigramme und Groß-/Kleinschreibung

- Trigramme über die rohen Bytes jeder Datei, fortlaufend, auch über Zeilenenden
  hinweg. Ein Zeilenende in einem Trigramm verringert die Trefferquote nicht und
  macht den Index nur etwas größer.
- Für Groß-/Kleinschreibung: ASCII-Kleinbuchstaben vor dem Zählen. **Der Index
  ist immer gefaltet.** Die Verifikation mit `(?i)` bleibt Sache von `grep.Search`.
- Nicht-ASCII-Faltung (`ä` gegen `Ä`) wird vom Index nicht abgedeckt. Ein Muster
  mit Nicht-ASCII-Buchstaben und `-i` fällt auf den Scan zurück.

### Digest und Gültigkeit

- **Zwei Records.** Der Graph behält `fingerprint.json` für den Graphkorpus
  (`freshness.Write`, `internal/code/freshness/fingerprint.go:84–97`), unverändert. Der
  Suchkorpus bekommt einen eigenen Record `cache/corpus.json` unter `store.CachePath`.
  Er enthält: Version; die kanonischen Einstellungen (`max_filesize`, Binärregel-Version,
  Ignore-Regel-Version, `core.ignorecase`); je Suchdatei Größe, Änderungszeit, Hash und
  Binär-Flag; die Ignore-Quellen (Pfad, Größe, Änderungszeit, Hash); den Stand von `.git/index`.
- **Digest:** SHA-256 über die kanonischen Einstellungen, die Hashes der Ignore-Quellen und
  die Paare `(rel, hash)` des Suchkorpus in Pfadreihenfolge. Der Index trägt ihn im Kopf.
  Er stammt aus demselben Build, der die Bytes liest. Der Digest hängt nur am Inhalt und an
  den Einstellungen, nicht am Zeitpunkt.
- **Gültig** heißt: Der Index-Digest entspricht dem Digest im Record, Extraktor- und
  Formatversion passen, und die Probe ist sauber. Sauber heißt: kein Drift in Suchdateien,
  Ignore-Quellen, Einstellungen oder `.git/index`. Sonst: Scan mit Notiz, `index: stale`.
  Damit ist die Lücke geschlossen, dass ein Index ohne Drift im Graphen gültig bleibt, obwohl
  eine `.md`-Datei sich geändert hat.
- **Probe bei jeder Abfrage** (`load.go:20–22`): ein Walk, je Datei Größe und Änderungszeit
  vergleichen. Nur bei Abweichung wird gehasht (`fingerprint.go:127–135`, Regel `trusted`).
  Zusätzlich gelesen werden: die kleinen Ignore-Quellen (stat, Hash nur bei Abweichung),
  `.git/index` und `.git/config` (stat) und die Konfiguration (stat, bei Abweichung neu lesen). `git` wird im Probe-Pfad **nur** aufgerufen, wenn sich `.git/index` oder `.git/config` per Stat geändert haben; dann ist ohnehin Drift. Der Matcher wird aus den gespeicherten Quellen und der gespeicherten Tracked-Menge neu gebaut. Der Aufwand
  wird gegen die Baseline gemessen (Performance-Ziele, Punkt 5).
- **Drift in einem der beiden Korpora** löst einen vollständigen Build aus, also Graph und
  Index (Z4). Ein Drift nur im Suchkorpus baut damit auch den Graphen neu. Die Kosten dafür
  sind Lesezeit auf dem Graphkorpus; das Parsen spart der Extract-Cache. Gemessen wird es.
- **Unlesbare Suchdateien** werden übersprungen, gezählt (`unreadable`) und mit leerem Hash
  im Record geführt, wie `fingerprint.go:78–81` es vorgibt. **Abweichung:** Lesefehler bei
  Graphdateien brechen den Build weiter ab (`build.go:133–136`), nur Suchdateien werden
  übersprungen.
- **Ignore-Eingaben, die keine Datei des Korpus sind** (`.gitignore`, `.git/info/exclude`,
  globale Excludes, `.git/index`, Konfiguration) sind Teil des Digests. Ohne sie wäre
  „Einstellungen im Digest“ nur halb wahr.

### Determinismus

- Gleiche Dateiliste mit gleichen Bytes ergibt gleiche Indexbytes. Vierter Durchgang: Die
  Indexbytes hängen zusätzlich an den kanonischen Einstellungen und an den Ignore-Eingaben.
  Die Ignore-Quellen und die Tracked-Menge werden in fester Reihenfolge verarbeitet.
- Test: zweimal bauen, Bytes vergleichen. Zusätzlich ein Golden-Test auf einem
  kleinen Korpus unter `internal/code/trigram/testdata/`.

### Speicher

- Der Aufbau läuft in Speicher: Trigramm → Liste von Datei-IDs, dann sortieren.
- Schätzung, zu messen: Eine Datei enthält typischerweise einige tausend
  verschiedene Trigramme. Der Umfang wächst also mit der Quellmenge, nicht mit
  der Zahl der Zeichen. Diese Schätzung ist **Annahme**, nicht gemessen.
- **Empfehlung:** In-Speicher-Aufbau mit einer dokumentierten Grenze, ab der der
  Build den Index auslässt und eine Notiz schreibt. Die Grenze wird aus der
  Messung des loomux-Repos abgeleitet. Ein Spill auf die Platte entfällt in v1.
  **Alternative:** externes Sortieren (wie tgrep), wenn die Messung die Grenze
  reißt. Das ist mehr Code als der Nutzen rechtfertigt, solange das Repo es nicht
  braucht.

### Symbole

Kein eigener Symbolindex in v1. Die Kandidaten sind Dateien, und für jede
Trefferzeile wird das umschließende Symbol aus dem Graphen bestimmt, wie heute
(`model.Enclosing`). Ein eigener Symbolindex wäre eine zweite Kopie der Knoten.
**Alternative:** Symbolnamen als eigene Trigramm-Postings, wenn der Agent nach
Namen statt nach Text sucht. Das ist ein späterer Schritt mit eigener Messung.

## Kandidatenauswahl

Aus dem Muster wird eine Menge von Trigrammen abgeleitet, die jede
passende Datei enthalten muss. Die Auswahl ist **konservativ**:

- Literale Teile der Konkatenation mit mindestens drei Bytes liefern
  Trigramme. Die Kandidaten sind die Dateien, die alle diese Trigramme enthalten.
- Alternativen, Klassen, Wiederholungen mit null Minimum und Wildcards liefern
  keine Bedingung. Enthält das Muster keine solche Bedingung, sind alle Dateien
  Kandidaten, und der Index wird nicht genutzt.
- `--fixed` ist ein reines Literal.
- Invariante: **Die Kandidatenmenge enthält jede Datei mit einem Treffer.** Sie
  wird per Fuzz-Test gegen die Verifikation geprüft (Abschnitt Teststrategie).

Die Verifikation bleibt `grep.Search` auf jeder Kandidatendatei. Die Sortierung
nach `inDegree` und Pfad bleibt unverändert.

## Schnittstellen

### Go-Pakete

`internal/code/trigram`:

- `func Build(files []File) ([]byte, error)`. `File` trägt Pfad und Inhalt.
- `func Write(root string, digest [32]byte, extractor string, files []File) error`.
- `func Read(root string) (*Index, error)`. Liefert `ErrNoIndex`, `ErrStale`
  (Format oder Extraktor passt nicht) und `ErrCorrupt`.
- `func (ix *Index) Candidates(pattern string, opts Options) ([]string, bool, error)`.
  Das `bool` meldet, ob der Index eine Bedingung hatte. Ohne Bedingung wird die
  ganze Liste ausgegeben und der Scan genutzt.

`internal/code/corpus`:

- `func Walk(root string, m Matcher) (Listing, error)`. `Matcher` ist die Ignore-Auswertung
  (`nil` außerhalb eines Repos). `Listing` trägt die Einträge (Rel, Abs, Size, MTime, Ignored,
  Tracked, Link) und den Repo-Status. Die Größe und die Binärregel werden nicht im Walk
  angewandt, sondern in den Filtern des Graph- und des Suchkorpus.
- `sourceset.Stat` wird zu einem Filter über `corpus.Walk` (Graphkorpus).

`internal/code/query`:

- `GrepOptions` bekommt `Channel` (`local` oder `cloud`), `Glob` und `MaxFilesize`
  (`*int64`, nil = Konfiguration oder Vorgabe). `Keep` entsteht im CLI-Pfad aus dem Channel.
- `GrepAnswer` (= `grep.Result`) bekommt die additiven Felder `index`,
  `files_in_scope`, `candidates`, `skipped_binary`, `skipped_size`, `ignored`. Vorhandene
  Felder bleiben.

### CLI

```
loomux graph grep <pattern> [--root <path>] [-i] [--fixed] [--in <prefix>]
                  [--glob <pattern>] [--max-hits <n>] [--channel local|cloud]
                  [--max-filesize <bytes> | --no-max-filesize]
                  [--no-refresh] [--json]
loomux graph build [--root <path>] [--no-reuse]
                  [--max-filesize <bytes> | --no-max-filesize]
```

- `--glob` (neu): Dateiauswahl über Glob auf dem Pfad, Schrägstrich-Form. Ohne
  dieses Flag gibt es keinen Dateifilter. Das Plugin setzt es für `include`.
- `--max-filesize <bytes>` und `--no-max-filesize` (neu, gleiche Wirkung wie `0`):
  überschreiben Konfiguration und Vorgabe für diesen Lauf (Rangfolge: Abschnitt
  „Größenlimit und Konfiguration“). `graph build` schreibt den Index mit diesem Wert.
  `--no-max-filesize` ist das Gegenstück zur Grenze, nicht zum Graphen.
- **Nicht eingeführt:** `--no-ignore` und `--hidden`. Es gibt keine entschiedene Anforderung
  dafür. `--no-ignore` würde die Ignore-Auswertung optional machen, und `--hidden` würde die
  Dateiliste ändern, die der Gate-Vergleich prüft. Beides wäre eine Entscheidung mit
  eigener Begründung (Frage 19).
- `--channel` (neu): `local` ist die Vorgabe und bleibt das heutige Verhalten.
  `cloud` wendet `[privacy] never` an, mit demselben Prädikat wie der MCP-Weg
  (`privacy.IsReadable(manifest, path) && !area.Conceals(path)`, siehe
  `internal/serve/graph/tools.go`, Funktion `readable`).
- `--max-hits`: Vorgabe 300, wie bisher.
- Exit-Codes: unverändert (`0` auch ohne Treffer, `1` bei fehlendem Graph oder
  ungültigem Regex, `2` bei Aufrufsfehler). Die Dokumentation in §6 wird
  entsprechend ergänzt.
- `graph build` schreibt den Index. Eine Fehlerausgabe auf `stderr` ohne
  Abbruch, wie bei Extract-Cache und Freshness-Record.

### JSON (`--json`, `grep.Result`)

```json
{
  "pattern": "…",
  "files_searched": 0,
  "total_hits": 0,
  "groups": [
    { "symbol": { … model.Node … }, "path": "…", "in_degree": 0,
      "hits": [ { "line": 0, "text": "…" } ] }
  ],
  "truncated_hits": 0,
  "unreadable": 0,
  "skipped_binary": 0,
  "skipped_size": 0,
  "ignored": 0,
  "index": "used | scan | stale | none",
  "files_in_scope": 0,
  "candidates": 0
}
```

- `symbol` fehlt bei Dateitreffern (Datei ohne Graph-Knoten oder außerhalb des Graphen). Das
  Feld ist `omitempty` wie heute (`internal/code/grep/grep.go:22`). Der Dateitreffer ist
  Form, die die Textanzeige schon kennt (`GrepReport`, „file level“, `query/grep.go`).
- `skipped_binary`, `skipped_size`, `ignored` zählen Dateien des Walks, die nicht im Suchkorpus
  sind. Sie sind additiv und stehen dort, damit eine Leere nie still bleibt (Anti-Ziel A4).
- `files_in_scope` ist die Zahl der Dateien im Suchkorpus, `candidates` die Kandidaten nach dem
  Index (gleich `files_in_scope`, wenn kein Literal aus dem Muster ableitbar ist).

- `files_searched` bleibt die Zahl der Dateien, deren Inhalt geprüft wurde. Auf
  dem Index-Pfad ist das die Kandidatenzahl. **Empfehlung:** Das wird im
  Plugin-Text (`render`) benannt, damit niemand die alte Bedeutung voraussetzt.
- `index` nennt den Pfad: `used` (Kandidaten aus dem Index), `scan` (keine
  Bedingung im Muster oder kein Index), `stale` (Index passt nicht, Scan),
  `none` (kein Index geschrieben).
- Eine Freigabe-Anfrage für die Änderung der Form ist nicht nötig: Alle drei
  Felder sind additiv.

### Plugin-Tool-Schema

Belegt gegen `index.js:746–808` (Schema) und `:845–856`, `:922–936` (Prüfung).
Name `grep` (`:753`), einziges Pflichtfeld `pattern` (`:784`), `timeoutMs`
30000 (`:759`). Ausgabeschema deklariert `matches` (Pflicht) und `daemon`
(optional, `:786–808`). Die Beschreibung nennt tgrep (`:755`). Ein Ziel-Text
ändert deshalb den Modell-Prompt, obwohl Z5 Name und Schema festhält.

| Parameter | Typ | Prüfung (Ist) | Wirkung (Ist) | Ziel (Task B1) |
|---|---|---|---|---|
| `pattern` | string, Pflicht | nicht leer, nicht getrimmt (`:848`) | `--regexp=<pattern>` (`:623`), schützt vor Flag-Injektion | Positionsargument, unverändert |
| `path` | string | nach Trim nicht leer (`:851`) | tgrep-Suchpfad relativ zum cwd, nach `--` (`:624`, `:860`), kein Präfix | `--in <prefix>`; Präfix ist nicht gleich Pfad, siehe Routing (B3) |
| `include` | string (Glob) | nicht leer, kein führendes `!`, kein Komma auf Top-Ebene (`:922–936`) | `-g <glob>` (`:597–599`) | `--glob`, fehlt im Quellstand (`internal/cli/graph.go:390–397`); gleiche Prüfung |
| `case_insensitive` | boolean | `Boolean()` (`:901`) | `-i` (`:593–595`) | `-i`, vorhanden (`graph.go:391`) |
| `max_results` | number | `Number(x) \|\| maxLines`, Klammer 20–5000 (`:858`) | nur Kürzung der Ausgabe, kein tgrep-Limit (`:942`) | `--max-hits`, Klammer 20–5000 wie heute (Z5) |

Die Vorgabe „`max_results` 1–5000“ aus dem Brief ist falsch: Die Untergrenze ist
20; 0 oder NaN ergeben `maxLines` (300).

### Plugin-Ergebnisform

**Ist (belegt):**

- `execute` liefert `{ matches: [{ path, lineNumber, line }] }`, optional `daemon` (`index.js:914`, `:881–890`). Pfad relativ, mit `/` (`:1102–1115`). Kein Feld `text`, keine Symbolfelder.
- `render` liefert Text: Dateiköpfe, Zeilen `  N: text`, `(no matches)`, Kürzungsnotiz `... [dsh-tgrep] truncated, showing first N of M matches` und die Daemon-Notiz (`:809–814`, `:1075–1097`, `:702–718`). Laut Kommentar (`:698–701`) ist die Notiz Teil der Tool-Ausgabe.
- `presentationMeta` liefert `{ shape: 'matches', files, truncated, total }` (`:815–819`, `:724–736`). `total` zählt alle gefundenen Treffer. `truncated` ist gesetzt, wenn mehr als `max_results` gefunden wurden oder wenn `capSearchMeta` Gruppen verworfen hat (`:664–669`). Zeilenvorschau höchstens 2000 Byte (`:29`, `:32`, `:641–655`).
- Die 64-KiB-Grenze (`:29`) ist weich: Eine einzelne Dateigruppe bleibt, auch wenn sie allein darüber liegt (`:657–663`, `:667`).
- `presentCall` (`:822–827`): generische Karte `Grep <pattern> in <path> (<include>)`.
- `presentResult` (`:833–843`) baut die Karte nur aus `shape`, `truncated`, `total` und `files` (`narrowSearchFiles`, `:676–693`). Andere Meta-Felder gehen beim Replay verloren.
- Der Entwurf nannte `SearchMeta`; diesen Typ gibt es nicht.

**Ziel (Vorschlag, Task B2):**

- `matches` bleibt `{ path, lineNumber, line }` (Z5). Symbolfelder kommen nur dazu, wenn `grepSearchMeta` und `narrowSearchFiles` sie mitführen. Sonst fallen sie beim Replay weg.
- `index`, die Zähler `ignored`, `skipped_binary`, `skipped_size`, `unreadable` und `notes` gehören in den `render`-Text (`engine` entfällt, es gibt keinen zweiten Suchweg), dem Kanal der Daemon-Notiz, nicht in `presentationMeta`.
- Notizen aus `loomux` (z. B. `another run is rebuilding, answering from the graph on disk`, `internal/code/ask/refresh.go:167`) kommen auf stderr, auch bei Exit 0 (`internal/cli/graph.go:428–430`). Das Plugin muss stderr also auch bei Erfolg lesen.

## Fehlerfälle und Rückfall

Grundsatz: Ein fehlendes Hilfsmittel führt zu einem Fehler mit Hinweis, nie zu einer
leeren Liste. Die Textsuche als Rückfall des Briefs ist verworfen (Abweichung vom Brief,
Fragen 3 und 10).

| Fall | Erkennung | Verhalten |
|---|---|---|
| `loomux` nicht auf PATH | Spawn-Fehler | Fehler mit Hinweis („loomux installieren“) |
| Kein Graph (`ErrNoGraph`) | Exit 1 mit dem Hinweis auf `graph build` | Fehler mit Hinweis auf `loomux graph build`, kein Rückfall |
| Ungültiges Regex | Exit 1, Meldung `invalid regex: …` (`internal/code/grep/grep.go:89`) | Fehler an den Agenten, kein Rückfall. Das Ist-Plugin prüft keine Meldung; dort ist Exit 1 ein leeres Ergebnis (`index.js:1012`). Der Ziel-Plugin unterscheidet am Wortlaut (fragil). **Empfehlung:** Im JSON-Modus ein Feld `error` mit Code. Das ist additiv, aber ein Eingriff in die Fehlerform. |
| `unsupported()`-Form (Lookaround u. ä.) | Exit 1, Meldung `… not supported in RE2 regex: …` (`grep.go:80`) | wie ungültiges Regex |
| Index fehlt oder passt nicht | `index: none` oder `stale` (Digest, Extraktor- oder Formatversion, Probe mit Drift in Suchdateien, Ignore-Quellen, Einstellungen oder `.git/index`) | Scan über denselben Suchkorpus, Notiz. Eine Einstellungsabweichung durch Flag ist Notiz „settings differ from index“, kein Rebuild (Abschnitt „Größenlimit und Konfiguration“). |
| Suchdatei nicht lesbar | Lesefehler beim Build oder Scan | Übersprungen, Zähler `unreadable`; Graphdateien brechen den Build wie bisher ab (`build.go:133–136`). |
| Index kaputt (falsches Magic, Längen außerhalb) | `ErrCorrupt` | Scan, Notiz, Reparatur beim nächsten Build |
| Timeout (30 s) | Host-Deadline über `timeoutMs` (`index.js:759`); das Plugin hat keinen Timer | Fehler mit Hinweis, kein Rückfall, weil der Agent die Suche sonst doppelt zahlt. Frage 7 bleibt bei der Empfehlung (Fehler); ein Rückfall ist nach Frage 10 ausgeschlossen. |
| Rebuild durch andere Sperre | Notiz `another run is rebuilding, answering from the graph on disk` (`internal/code/ask/refresh.go:167`), stderr, Exit 0 | Antwort aus Platte, Notiz (Verhalten heute) |
| Pfad-Präfix nicht im Suchkorpus (`--in`, `path`) | Fehler `prefix not indexed` (Prüfung gegen den Suchkorpus, nicht mehr gegen die Graph-Knoten) | Plugin: Fehler mit Hinweis, wenn der Präfix außerhalb der Repo-Wurzel liegt. Liegt er innerhalb, aber ignoriert, gibt es Zähler und keinen Fehler (Abschnitt „Abdeckung“). |
| Datei nicht im Suchkorpus (ignoriert, binär, zu groß) | Zähler `ignored`, `skipped_binary`, `skipped_size` im Ergebnis | Kein Fehler. Das Plugin nennt die Zähler im `render`-Text, damit eine leere Liste nie ohne Grund bleibt (A3, A4). |
| Exit 1 ohne bekannte Meldung | Exit 1, keine der obigen Meldungen | Fehler mit der Stderr-Zeile, nie ein leeres Ergebnis (A3). |
| stdout-Deckel (nur Ist-Plugin, 20 MiB) | Zeichenlänge über `index.js:977` | Ist: stilles Abschneiden ohne Notiz (`index.js:1024–1051`). Ziel: kein Deckel nötig, weil `--max-hits` die Ausgabe begrenzt. Würde der Ziel-Pfad doch abschneiden, wäre das ein Fehler. |
| `tgrep` fehlt (nur Ist-Plugin) | ENOENT, `index.js:993–999` | Ist: Ablehnung ohne Rückfall; die Daemon-Notiz geht verloren, weil `execute` vor dem `return` wirft (`index.js:908–914`). Entfällt mit tgrep (Frage 3). Ziel: Fehler mit Hinweis, kein Rückfall (Frage 10). |
| Daemon-Start (nur Ist-Plugin) | `ensureDaemonStarted`, Wartezeit bis 60 s | Entfällt im Ziel (Z3). Ist: Die Wartezeit läuft außerhalb von `exec.signal` (`index.js:870–873`, `:335–343`). |

### Abdeckung

Der Suchkorpus deckt alle als Text erkannten Dateien ab, die nach den Ignore-Regeln,
der Größe und der Binärregel übrig bleiben (Entscheidung 1). Die Endungsliste des
Graphen gilt für die Suche nicht mehr. Daraus folgt die Routing-Regel im Plugin:

1. Der Index ist maßgeblich, wenn er gültig ist (Abschnitt „Digest und Gültigkeit“).
   `include` ist ein beliebiger Glob; eine Endung löst keinen Fehler mehr aus. Die
   Fehler auf `.md`, `.ts`, `.js`, `.json` und `.c` aus dem zweiten Durchgang entfallen.
2. `path` außerhalb der Repo-Wurzel des Suchkorpus ist ein Fehler mit Hinweis
   (kein Rückfall, Frage 10). `path` innerhalb ist zulässig, auch wenn die Dateien dort
   nicht im Suchkorpus liegen. Dann zeigt das Ergebnis die Zähler (`ignored`,
   `skipped_binary`, `skipped_size`), und die Notiz nennt sie.
3. Der Index ist ungültig oder fehlt: Scan über denselben Suchkorpus (gleicher Walk, gleiche
   Filter), mit Notiz `index: stale` oder `none`. Ein Scan über nur die Graph-Knoten
   wäre ein anderes Ergebnis und ist deshalb ausgeschlossen (Anti-Ziel A1, A6).

**Nicht abgedeckt (ehrlich benannt):** Punkt-Dateien und Punkt-Verzeichnisse einschließlich `.github/`, `.env` und
`.loomux/` (in v1 bewusst, siehe Abschnitt „Suchkorpus und Walk“); Dateien, die Git
ignoriert; Binärdateien einschließlich UTF-16; Dateien über `max_filesize`; Dateien ohne
Lesezugriff; Symlinks (Frage 16); Ignore-Quellen außerhalb von Git (`.ignore`, `.rgignore`).
Die Zähler im Ergebnis machen jede dieser Lücken sichtbar.

**Entfallen (Stand zweiter Durchgang):** Der Textscan über nicht abgedeckte Endungen (Frage 8)
ist überflüssig, weil es keine Endungslücke mehr gibt. Frage 14 ist entschieden.

### Rückfallpfad (entschieden: entfällt)

Das Ist-Plugin hat keinen Rückfall. Ein fehlendes Binary wird geworfen
(`index.js:993–999`). Verworfen sind `rg` (ripgrep 15.2.0, Kandidat D) und das Stock-Tool über `hostTools.get?.('grep')` (Kandidat E, `index.js:518`); dazu kommen die Kandidaten A (`@deepseek-ai/dsh-tool-fs-search`, nicht belegt, nicht in `node_modules\`) und B (tgrep, entfernt) sowie eine Textsuche im Plugin (C), die entfällt.

**Entscheidung (Mensch, Fragen 3 und 10):** Kein Rückfall, kein tgrep, keine optionale Binary. Die Empfehlung „Fehler mit Hinweis, kein stilles Leer“ bleibt.
Fehlen `loomux`, der Graph oder die Abdeckung, antwortet das Plugin mit einem
Fehler und einem Hinweis (`loomux` installieren, `loomux graph build`). Das ist
keine leere Antwort. A3 ist entsprechend präzisiert (Anti-Ziel A3). Der Scan von
`loomux` läuft nur bei vorhandenem Graphen. Ein zweiter Pfad über `rg` ist verworfen;
`rg` kennt `never` nicht.

## Privacy und Sicherheit

- **Der CLI-Weg ist die Lücke.** `loomux graph` filtert heute keine `never`-
  Pfade (`docs/en/cli-reference.md` §6, Zeile 836; `docs/en/configuration.md`
  Zeile 973). Das Plugin ruft die CLI auf und gibt Ausgaben an das Modell
  weiter. Ohne Änderung wäre das ein neuer Abfluss gegenüber dem MCP-Weg.
- **Empfehlung:** `--channel cloud` als Opt-in, und das Plugin setzt es immer.
  Die Vorgabe `local` bleibt das heutige Verhalten, damit Shell-Aufrufe durch
  den Agenten nicht still ihre Ausgabe ändern.
  **Alternative:** `never` ab Werk auf dem CLI-Weg anwenden. Das ändert
  dokumentiertes Verhalten und ist eine Entscheidung für den Menschen.
- **Fail-closed:** Ist das Manifest nicht lesbar, endet `--channel cloud` mit
  Exit 1 und einer Meldung. Es gibt keine ungefilterte Antwort als Ersatz.
- **Präfix unter `never`:** Mit `--in` auf einen `never`-Pfad liefert `cloud` eine
  leere Antwort mit Notiz, nicht den Fehler `prefix not indexed`. Die Notiz
  verrät dann nicht, ob der Pfad existiert. **Empfehlung**, Alternative: Fehler.
- **Index-Dateien:** Der Index enthält Trigramme aller Dateien, auch der
  `never`-Dateien. Der Extrakt-Cache hält seit jeher Texte von Körpern
  (`docs/en/cli-reference.md` §6, Zeile 843). Der Index ist also kein neuer
  Kanal, sondern ein weiterer Speicherort derselben Art. Der Filter liegt bei der
  Abfrage, nicht beim Schreiben.
- **Kein zweiter Suchweg:** Ohne Rückfall (Frage 10) gibt es keinen Scan über `tgrep`
  oder ein Stock-Tool, der `never` nicht kennt. Das Ist-Plugin enthält keine
  Privacy-Logik (`index.js` vollständig gelesen). Offen bleibt der Shell-Weg des Agenten:
  Er liegt außerhalb des Plugins und kennt `never` heute ebenfalls nicht. Frage 6 ist
  mit dem Entfall des Rückfalls erfüllt; der Shell-Weg ist eine offene Lücke.
- **Mehr Abdeckung, mehr Fläche (neu, vierter Durchgang).** Die Suche liefert jetzt Text aus
  allen Textdateien, nicht nur aus Graphdateien. Dazu gehören `.md`, `.json`, Konfigurationen
  und andere Dateien, die `graph grep` bisher nicht erreichte. `[privacy] never` filtert
  weiter auf der Abfrage (Z6, `--channel cloud`). Git-Ignore ist aber **keine**
  Geheimhaltung: Nicht jede Geheimdatei steht in `.gitignore`. Ob zusätzlich Namensmuster wie
  `*.pem`, `*.key`, `secrets.*` standardmäßig ausgeschlossen werden (Punkt-Dateien wie `.env` sind durch die Punkt-Regel schon draußen), ist eine neue Entscheidung
  und offen (Frage 16). Die Spec wählt sie nicht still.
- **Symlinks** werden in der Suche nicht verfolgt (offen, Frage 16). Der Graphwalk hat sie bisher
  über `Lstat` mitgezählt und dann mit `os.ReadFile` gelesen (`sourceset.go:136–138`). Dieses
  Verhalten bleibt für den Graphen unverändert.
- Keine Schreibrechte für Agenten auf `.loomux/config.toml`. Der Index berührt
  diese Datei nicht.

## Performance-Ziele und Messplan

Vorschläge, keine gemessenen Werte. Die Ziele werden nach der Baseline festgelegt.

1. **Baseline vor jeder Änderung:** der heutige Scan `loomux graph grep` auf dem
   loomux-Repo, kalt und warm, mit mehreren Mustern (literal, Alternation, ohne
   Literal, mit `-i`). Ergebnis in `docs/en/benchmarks.md` und
   `docs/de/benchmarks.md`, chronologisch, mit Datum, Uhrzeit, Basis gegen Änderung.
2. **Benchmark-Gate gegen tgrep (verbindlich, vor der Entfernung):** tgrep wird als
   reines Entwicklungswerkzeug installiert, nicht als Laufzeit-Abhängigkeit (Plan,
   Task C1). Die Messung läuft auf dem loomux-Repo und einem zweiten Korpus, mit
   gleicher Dateiliste und gleichen Treffern (Plan, Task C2). Ergebnis chronologisch
   in `docs/en/benchmarks.md` und `docs/de/benchmarks.md`.
   **Abbruchkriterium:** Ist die loomux-Suche auf dem loomux-Repo und einem zweiten
   Korpus nicht mindestens gleich schnell wie tgrep bei gleichen Treffern, wird das im
   Benchmark-Eintrag festgehalten und die Entfernung von tgrep verschoben, bis die Lücke
   geschlossen ist (oder der Mensch entscheidet anders). Das ist ein Gate, kein weicher
   Wunsch. Die Benchmarks des tgrep-Projekts (24.08.2026) sind nicht unsere Messung.
3. **Vorschläge:** Warme Suche mit Index unter 50 ms auf dem loomux-Repo (Richtwert,
   wie `brain search` < 50 ms in den Vorläufer-Budgets); Indexgröße unter 10 %
   der Suchkorpus-Bytes; Mehrkosten von `graph build` unter 20 %. Jedes Ziel wird
   gegen die Baseline geprüft und nicht vorher behauptet. Das Gate (Punkt 2) nutzt das
   Abbruchkriterium, nicht diese Richtwerte.
5. **Neu (vierter Durchgang): Kosten, die der größere Korpus verursacht.** Zu messen sind, jeweils
   gegen A0 auf dem loomux-Repo. `iam_backend` wird erst in C3 gemessen, und nur mit
   bestätigtem Lesezugriff (Frage 15 ist beschlossen, der Inhalt ist nicht gelesen):
   - Walk plus Probe je Abfrage (kalt und warm), mit Aufschlüsselung nach Walk, Ignore-Auswertung
     und Stat-Vergleich;
   - Build: Zeit für Graph und Index, Anteil Index an der Zeit, Lesezeit der Suchdateien;
   - Indexgröße und Anteil an den Suchkorpus-Bytes, aufgeschlüsselt nach Textdatei-Typ
     (`.md`, `.json`, Quellcode, Sonstiges), damit sichtbar wird, welche Typen den Index treiben;
   - Anzahl und Bytes der Dateien, die binär, zu groß oder ignoriert sind;
   - der Aufwand für `testdata/` und für ignorierte, aber nicht geprunte Verzeichnisse (Abschnitt
     „Suchkorpus und Walk“, Abstieg).
   Die Zahlen werden gemessen und eingetragen, nicht geschätzt. Bis dahin gibt es keine Aussage
   über Mehrkosten.
4. **Korrektheit vor Geschwindigkeit:** Die Ergebnisgleichheit mit dem Scan ist
   Pflicht (Teststrategie), Geschwindigkeit ist zweitrangig.

## Teststrategie

TDD: jede Testzeile läuft rot, bevor der Produktivcode entsteht. Coverage 100 %
je Funktion. Eine Ausnahme nur mit `//coverage:exempt <Grund>` direkt über `func`.
Kein `init()`, eingebettete Daten werden beim ersten Gebrauch geladen.

| Ebene | Test | Prüft |
|---|---|---|
| Literale | Tabelle | Konkatenation, Alternation, Klassen, `-i`, `--fixed`, Nicht-ASCII |
| Format | Round-Trip | Schreiben und Lesen ergibt dieselben Postings |
| Format | Bytes | Zwei Builds ergeben identische Bytes; Golden-Datei |
| Format | Korrupt | falsches Magic, Version, abgeschnittene Datei, Offsets außerhalb, Längen |
| Kandidaten | Fuzz | Zufallsmuster und Zufallstexte: Trefferdateien ⊆ Kandidaten |
| Gleichheit | Differentiell | Index-Pfad und Scan-Pfad liefern gleiche `groups`, `total_hits`, `truncated_hits` auf einem Korpus |
| Build | Integration | `graph build` schreibt Index und Digest; Digest stimmt mit Freshness-Record |
| Gültigkeit | Integration | veralteter Index → `stale` und Scan |
| Privacy | Integration | `--channel cloud` lässt `never`-Pfade weg; unlesbares Manifest → Exit 1; `local` unverändert |
| CLI | Integration | Flags, Exit-Codes, `--json`-Form, `--glob` |
| Plugin | Unit (Node) | Schema → Argumente; Routing; Fehlerpfade; `matches`-Abbildung |
| Walk | Tabelle und Baum | Prune-Regel (Punkt-Verzeichnisse; `skipDirs` ignoriert ohne getrackten Inhalt); harte Ausschlüsse `.git` und `.loomux`; Invariante: Graphliste gleich dem Stand vor dem Umbau |
| Ignore | Tabelle, Orakel `git check-ignore` | Negation, Anker, `**`, Verzeichnismuster, nested `.gitignore` über und unter der Wurzel, `info/exclude`, `core.excludesFile`, Tracked-Ausnahme, `core.ignorecase`. Orakel je besuchtem Pfad: `Ignored` gleich dem Ergebnis von `git check-ignore --stdin` (ohne `--no-index`, damit getrackte Pfade als nicht ignoriert gelten). Auf der Dateiliste: Suchkorpus gleich `git ls-files --cached --others --exclude-standard`, gefiltert mit denselben Binär-, Größen- und Punkt-Regeln. Der Vergleich ist nie „ignorierte Menge gleich git“, weil der Walk geprunte Verzeichnisse nicht betritt |
| Ignore | Integration | außerhalb eines Repos keine Auswertung; Fehler von `rev-parse` → keine Auswertung |
| Binär | Tabelle | NUL-Byte; UTF-16 gilt als binär; NUL nach dem ersten Puffer wird trotzdem erkannt (ganze Datei) |
| Größe | Tabelle | Grenze inklusive; `0` und `--no-max-filesize` heißen unbegrenzt; Rangfolge Vorgabe, Konfiguration, Flag |
| Digest | Integration | Einstellung geändert → `stale`; `.gitignore` geändert → `stale`; `git add -f` ohne Dateiänderung → `stale`; gleiche Eingabe → gleiche Bytes |
| Unlesbar | Integration | Suchdatei mit Lesefehler → Zähler `unreadable`, kein Abbruch; Graphdatei mit Lesefehler → Abbruch wie bisher |
| Routing (Plugin) | Unit | `include: "*.md"` → `loomux` ohne Fehler; `path` außerhalb der Wurzel → Fehler; Zähler im `render`-Text |
| Gate | Messung | Index-Pfad gegen tgrep auf gleicher Dateiliste und gleichen Treffern; Eintrag in beiden Benchmark-Dateien (Plan, C3) |

**Testdaten:** Ein kleiner Korpus unter `internal/code/trigram/testdata/`. Dieser
Ordner wird von `sourceset` übersprungen, das ist für Testdaten gewollt. Für den Suchkorpus
gilt diese Ausnahme nicht: Liegt der Ordner nicht in einer ignorierten Regel, gehört er zum
Suchkorpus des loomux-Repos. Das ist im Gate zu berücksichtigen. Für den
differentiellen Test wird der Graph über `internal/code/query/golden_test.go`
gebaut, damit Index und Scan dieselbe Dateiliste sehen.

## Rollout und Rückbau

- **Phase 1 (loomux):** Index und `--channel`, `--glob`, `--max-filesize`, `--no-max-filesize`
  hinter der Vorgabe `index` aus, bis die Differentialtests grün sind. Dann Vorgabe an.
  Die Ignore-Auswertung ist Teil des Walks und hat keinen eigenen Schalter (Entscheidung 2).
- **Phase 2 (Plugin):** `dsh-loomux` ruft die neuen Flags auf und wird im Profil
  registriert (`insert`-Eintrag, neue `id`). Vor der Veröffentlichung ist ein Lauf
  gegen das reale Plugin-Profil nötig. Während dieser Phase laufen `dsh-loomux` und
  `dsh-tgrep` nebeneinander und registrieren beide das Tool `grep`. Ob das harmlos ist,
  ist nicht belegt (Frage 12).
- **Phase 3 (Gate und Entfernung, Plan Teil C):** Erst das Benchmark-Gate, dann die
  Entfernung. Vorbedingungen: Gate bestanden; `dsh-loomux` registriert und im realen
  Profil geprüft; Frage 12 geklärt, also belegt, welches `grep` antwortet. Dann
  `dsh-tgrep` abschalten bzw. entfernen, laufende `tgrep serve`-Prozesse beenden und
  `.tgrep/` aus dem Workspace entfernen.
- **Konflikt (markiert):** Die frühere Fassung verlangte, `id: tgrep` vor Phase 2
  abzuschalten. Die Entscheidung des Menschen verlangt die umgekehrte Reihenfolge;
  umgesetzt ist sie. Die Doppelregistrierung in Phase 2 ist bis Frage 12 unbelegt.
- **Rückbau:** `trigram.idx` löschen; der Scan läuft weiter. Das Plugin antwortet
  mit Fehler und Hinweis, wenn `loomux` die Flags nicht kennt (Versionsprüfung
  über `loomux version`, Annahme). Ein Rückfall auf tgrep entfällt.
- Ein Index ist regenerierbar. Eine Migration gibt es nicht.

## Dokumentation im selben Zug

- `docs/en/cli-reference.md` und `docs/de/cli-reference.md`, §6: `--channel`,
  `--glob`, `--max-filesize`, `--no-max-filesize`, Felder `index`, `skipped_binary`,
  `skipped_size`, `ignored`, die §6-Anmerkung zu `never` für den Cloud-Kanal.
- `docs/en/configuration.md` und `docs/de/configuration.md`: der Schlüssel `max_filesize` nur,
  wenn Frage 18 entschieden ist. Der Abschnitt „`[graph]` — there is none“ wird danach
  angepasst, nicht stillschweigend umgangen.
- `docs/en/configuration.md` §4 („Where Things Live“): `cache/corpus.json` als Sidecar ergänzen.
- `docs/en/benchmarks.md`, `docs/de/benchmarks.md`: Baseline, Messung, Gate-Eintrag gegen tgrep (Plan, Teil C).
- `README.md`, `README.de.md`: Fähigkeit, Roadmap-Zeile. Die Roadmap-Zeile
  „Fusion tgrep und Code-Graph“ kommt in beide READMEs; Prioritäten setzt der
  Mensch.
- Wiki unter `docs/wiki/`: Seiten nach `docs/wiki/index.md` ermitteln, nicht raten.
- Label: `release:minor` (neues Flag, neue Fähigkeit, keine inkompatible Änderung).

## Offene Fragen

Der Brief stellte drei Fragen, dazu kommen die Fragen aus dem Design. Entschieden sind 1, 2, 3, 10, 11, 13, 14 und 15 (gekennzeichnet mit *Beschlossen* bzw. *Verworfen*); zu den übrigen gibt es Empfehlungen oder sie sind als offen markiert (16 bis 21).

1. **Aktualität.** *Beschlossen (Mensch):* Der Index wird nur im Rebuild neu gebaut, also
   bei Drift, wie der Graph. Das passt zum Freshness-Modell, und Index und Graph
   bleiben ein Ergebnis. *Verworfen:* Live-Nachführung während der Sitzung.
   Das braucht einen Watcher oder Zeitstempel-Vergleich auf jeder Anfrage und ist
   mit der Sperre nicht sauber zu lösen.
2. **Ort des Plugin-Codes.** *Beschlossen (Mensch):* ein neues Paket `dsh-loomux` (nicht `dsh-loomux-graph`).
   Ein Fork von `dsh-tgrep` (MIT, Dritt-Repo) würde Pflege übernehmen, und die
   Abhängigkeit auf tgrep bliebe im Namen. *Verworfen:* Fork von `dsh-tgrep`
   mit eigener Versionierung.
3. **tgrep-Entfernung.** *Beschlossen (Mensch):* tgrep fliegt ganz raus aus Plugin und
   Betriebsweg: kein Fallback, keine optionale Binary, keine Flags als Vertrag. Vorher ein
   verbindliches Benchmark-Gate (Plan, Teil C), danach die Entfernung des tgrep-Erbes.

Weitere Fragen, die sich aus dem Design ergeben:

4. **Plugin-Form.** Beantwortet (Abschnitt Stand der Plugin-Seite): `matches` ist
   `{ path, lineNumber, line }`, die Karte `presentationMeta` ist `{ shape, files, truncated, total }`.
   Offen ist nur, ob Symbolfelder in die Karte kommen (B2).
5. **`never` ab Werk auf dem CLI-Weg?** *Beschlossen (Mensch):* Opt-in mit
   `--channel cloud`, das Plugin setzt es immer.
6. **Rückfall auf `never`-Pfade.** Siehe Privacy. Empfehlung: nicht in v1; mit dem Entfall des Rückfalls erfüllt. Der Shell-Weg des Agenten bleibt offen.
7. **Timeout-Verhalten** des Plugins: Fehler mit Hinweis (siehe Fehlerfälle); Rückfall ist nach Frage 10 ausgeschlossen. Die 30 s sind eine Host-Deadline (`index.js:759`); das Plugin hat keinen Timer.
8. **Abdeckung über Endungen.** *Entfallen (vierter Durchgang, Frage 14):* Es gibt keine Endungslücke mehr im Suchkorpus; zusammenführen ist überflüssig.
9. **Fehlerform im JSON-Modus.** *Beschlossen (Mensch):* ein Feld `error` mit Code;
   das Plugin klassifiziert am Code, nicht am Wortlaut.
10. **Rückfall-Engine.** *Entschieden (Mensch, Frage 3):* tgrep ist keine Option. Die
    Empfehlung „Fehler mit Hinweis, kein stilles Leer“ bleibt; `rg` und das Stock-Tool
    über `hostTools.get('grep')` sind verworfen (Abschnitt Rückfallpfad). Der Mensch
    entscheidet nicht erneut, solange die Empfehlung trägt.
11. **Daemon-Erbe.** *Beschlossen (Mensch):* `dsh-tgrep` wird abgeschaltet bzw. entfernt
    (`id: tgrep`). Laufende `tgrep serve`-Prozesse werden beendet (Pid aus
    `<workspace>/.tgrep/serve.json`, sofern vorhanden); `.tgrep/` wird aus dem Workspace
    entfernt, nicht committet. Reihenfolge: erst nach dem Benchmark-Gate und erst, wenn
    `dsh-loomux` im Profil registriert und geprüft ist (Plan, Task C4). *Konflikt
    (markiert):* Die frühere Fassung (Rollout, Phase 2) verlangte das Abschalten vor der
    Registrierung. Umgesetzt ist die Reihenfolge des Menschen; die Doppelregistrierung
    bleibt bis Frage 12 unbelegt. Offen: ob der Paketordner `dsh-tgrep` bleibt.
12. **Host-Semantik.** Nicht im Profil belegt: ob der Host `exec.signal` nach 30 s setzt
    und ob `tools.register` ein gleichnamiges Stock-`grep` ersetzt. Nur mit dem Host-Code
    oder einem Lauf klärbar. Vorbedingung für das Entfernen von `dsh-tgrep` (Plan, Task C4).
13. **tgrep-Flags.** `serve`, `--index-path`, `--no-max-filesize`, `--no-index`, `-g` und
    `--json` sind ohne Binary nicht belegt. *Beschlossen (Mensch):* kein Vertrag, nur Benchmark-Notiz. Im Gate werden die tatsächlich genutzten Flags aus `tgrep --help` festgehalten.
14. **Abdeckung ohne Rückfall.** *Beschlossen (Mensch, vierter Durchgang):* Die Textsuche deckt alle als Text erkannten Dateien ab; der Graph bleibt bei seinen Endungen. Die Endungslücke entfällt (Abschnitt „Abdeckung“). Was der Suchkorpus nicht enthält (ignoriert, binär, zu groß), bleibt Lücke mit Zählern. Für diese Dateien ist der Weg des Agenten (Shell) nicht Teil dieses Designs.
15. **Zweiter Korpus für das Gate.** *Beschlossen (Mensch, vierter Durchgang):* `iam_backend` (`C:\Users\micro\Documents\#GIT\iam_backend`, Python). **Nicht gelesen.** Dateiliste, Größe und Trefferzahl sind nicht erhoben. Eingetragen in C2 und C3.
16. **Namensmuster und Symlinks.** *Beschlossen (Mensch):* kein stiller Ausschluss von Namensmustern (`*.pem`, `*.key`, `secrets.*`) — dafür sind die Ignore-Regeln da; Symlinks werden im Suchkorpus übersprungen, der Graph bleibt unverändert.
17. **Graph-Größenlimit.** *Beschlossen (Mensch):* Der Graphkorpus bleibt bei
   1 000 000 Byte (`sourceset.go:32`); `max_filesize` gilt nur für den Suchkorpus,
   weil der Graph jede Datei parsen muss (`sourceset.go:28–31`).
18. **Konfigurationsabschnitt.** *Beschlossen (Mensch):* Abschnitt `[grep]` mit dem Schlüssel `max_filesize`; die Begründung (Hooks und MCP-Aufrufe ohne Flag müssen denselben Korpus sehen) kommt in `docs/en/configuration.md` und `docs/de/configuration.md`, und der Abschnitt „`[graph]` — there is none“ wird dort präzisiert. *Verworfen:* nur Flags mit Konstante im Code.
19. **`--hidden` und `--no-ignore`.** *Beschlossen (Mensch):* nicht in v1; `.github/` und andere Punkt-Verzeichnisse bleiben vorerst nicht durchsuchbar. Eine spätere Anforderung wäre eine eigene Entscheidung mit Gate-Auswirkung.
20. **UTF-16-Dateien.** *Beschlossen (Mensch):* UTF-16 gilt als binär und wird gezählt (`skipped_binary`) übersprungen; keine Konvertierung in v1.
21. **Befunde ohne Frage an den Menschen.** *Beschlossen (Mensch):* Die Standardquelle von Git für globale Excludes bei ungesetztem `core.excludesFile` und die Mechanik von `core.ignorecase` werden im Test gegen `git` belegt, nicht vorher festgelegt. *Belegt (2026-11-04, `git check-ignore` in temporären Repos, siehe Abschnitt „Ignore-Semantik"):* Standardquelle ist `$XDG_CONFIG_HOME/git/ignore`, sonst `$HOME/.config/git/ignore`; `core.ignorecase=true` faltet Muster und Namen; ein Verzeichnis mit einer getrackten Datei darunter gilt git als nicht ignoriert.
