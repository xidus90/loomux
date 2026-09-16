# Installer-Kern (P1) — Entwurf

Stand: 2026-08-28. Status: entworfen, nicht umgesetzt.

## Warum

Sechs Repos unter `#GIT` erzwingen heute dieselben Dinge mit verschiedenen
Mitteln: Geheimnisse und generierte Dateien schützen, Tests grün halten, Typen
prüfen, das Wiki pflegen. Jedes Repo trägt seine eigene Fassung, und die
Fassungen laufen auseinander — `wiki_gate.py` liegt trotz dokumentiertem Kanon
und Drift-Test in zwei von drei iam-Repos veraltet, gemessen am 2026-08-27.

Ein neues Projekt einzurichten heißt heute: `settings.json` von Hand schreiben,
Hook-Programme kopieren, Werkzeuge installieren, Konventionen abtippen. Eine
Änderung an einer dieser Sachen heißt: dieselbe Arbeit sechsmal, oder Drift.

`ultraloom init` soll beides lösen — eine Quelle für die Erzwingung, und ein
Befehl, der sie in ein Projekt bringt.

## Messgrundlage

Alle Zahlen am 2026-08-27/28 auf Windows 11 gemessen, jeweils abwechselnd
A-B-A-B statt blockweise. Ein Blockvergleich hat bei genau diesen Fragen zweimal
das Gegenteil gesagt, weil die Maschinenlast über den Effekt hinweg streut.

| Was | Zeit |
|---|---|
| `ultraloom policy hook` (feuert bei **jedem** Write/Edit/Bash/PowerShell) | 299 ms |
| `ultraloom hook post-edit` | 685 ms |
| dieselben zwei Prüfungen direkt aufgerufen | 267 ms sequentiell, 232 ms parallel |
| nackter Python-Start dieser Maschine | 160 ms |
| Import von `ultraloom.policy.hook` obendrauf | +75 ms |
| `ruff check .` über 33 Dateien, kompiliertes Binary | 20–27 ms |

Daraus die zwei Sätze, die den Entwurf tragen:

**Der Boden jeder Python-Lösung liegt bei ~160 ms plus Arbeit.** Eine
Import-Diät holt höchstens ~40 ms. Nur ein kompiliertes Binary kommt darunter.

**Latenzkritisch ist nur, was je Werkzeugaufruf feuert.** Der Stop-Gate läuft
ein- bis dreimal je Sitzung mit 300-Sekunden-Budget; dort sind 400 ms egal. Der
Policy-Hook feuert bei jedem Aufruf — bei 200 Aufrufen je Sitzung ist er allein
eine Minute.

## Zerlegung

Was gewünscht wurde, zerfällt in fünf Teilprojekte. Die Abhängigkeit läuft in
eine Richtung: P2 bis P5 sind Inhalte, die P1 verteilt.

| | Teilprojekt | Inhalt |
|---|---|---|
| **P1** | Installer-Kern | **dieser Entwurf** |
| P2 | Hook-Bestand | die Hooks selbst, aus sechs Repos konsolidiert |
| P3 | Dokumentvorlagen | AGENTS.md, CLAUDE.md, der ausgebaute OKF-Richtlinientext |
| P4 | `.claude`-Ausstattung | Skills, Subagenten, Slash-Befehle |
| P5 | `ultraloom-guard` | der Policy-Hook nativ: 299 ms auf ~15 ms |

## Was gebaut wird

`ultraloom init` — ein Go-Binary, das in einem Projektverzeichnis läuft, den
Stack erkennt, ein Interview führt und daraus eine Ausstattung erzeugt.

**Go, nicht Rust oder C++.** Kriterium war schnellster Start *und* Windows,
macOS, Linux, BSD. Der Startunterschied zwischen Go und Rust liegt bei ~2 ms von
285 ms Ersparnis, also praktisch bei null; der Unterschied bei BSD ist real — Go
hat offizielle Ports für FreeBSD, OpenBSD, NetBSD und DragonFly, Rust führt
OpenBSD und NetBSD als Tier 3 ohne Zusage.

**Es lebt im ultraloom-Repo.** Dort liegt schon, was es verteilen soll. Ein
zweites Repo verteilte die Wahrheit sofort auf zwei Orte — der Zustand, aus dem
dieses Vorhaben herausführen soll.

```
ultraloom/
  src/ultraloom/     Python: Prüfkette, post-edit, stop   (bleibt)
  cmd/init/          Go: der Installer                    (neu)
  templates/         Vorlagen, per go:embed eingebettet   (neu)
```

Die Vorlagen werden ins Binary gebacken. Ein Release ist damit eine Datei je
Plattform, ohne Archiv und ohne Nachladen, und der Vorlagenstand ist untrennbar
mit der Binärversion verbunden.

## Erkennung

Die Erkennung ist eine reine Funktion: Verzeichnisbaum hinein, Faktenmenge
heraus. Kein Schreiben, kein Netz, keine Unterprozesse außer
`git config --get core.hooksPath`. Damit ist sie ohne Dateisystem testbar, und
`--dry-run` fällt gratis ab.

Gesucht wird im Wurzelverzeichnis plus eine Ebene tiefer für Arbeitsbereiche.

| Signal | Schluss | Zieht nach sich |
|---|---|---|
| `pyproject.toml`, `uv.lock` | Python, uv-verwaltet | ruff, mypy, pytest, Coverage |
| `requirements.txt` ohne `uv.lock` | Python, nicht uv-verwaltet | dieselben Werkzeuge ohne `uv run`-Annahme |
| `manage.py` | Django | Migrationen als geschützte Pfade *vorschlagen* |
| `project.godot` | Godot | gdlint, gdformat, Suite headless, `override.cfg` |
| `project.godot` mit `[dotnet]` | Godot mit C# | zusätzlich zur GDScript-Kette, nicht statt ihr |
| `*.csproj`, `*.sln` | .NET | `dotnet format --verify-no-changes`, `build`, `test` |
| NuGet `gdUnit4.api` / `gdUnit4.test.adapter` | Godot-Tests in C# | `dotnet test`, dessen coverlet-LCOV in den Merge geht |
| `package.json` + `tsconfig.json` | TypeScript | eslint, tsc, vitest |
| `Cargo.toml` / `go.mod` | Rust / Go | fmt, clippy bzw. vet, test |
| `.git` mit gesetztem `core.hooksPath` | vorhandene Git-Hooks | melden, nicht überschreiben |
| Nachbarverzeichnis `*_wiki` mit `.git` | Wiki als Nachbar-Repo | Commit-Pflicht anbieten |
| `wiki/` mit OKF-Struktur | Wiki im Repo | brain anbieten |

### Stacks überlagern sich

`{godot, gdscript, csharp}` ist ein gültiges Ergebnis. space wird für die Dauer
seiner C#-Migration beides sein; die Kette wird zusammengesetzt, nicht
ausgewählt.

**Coverage ist die einzige Prüfart, die über Sprachgrenzen hinweg eine Aussage
macht.** Findet die Erkennung mehr als eine Sprache mit Coverage, fragt das
Interview nach *einer* Schwelle und sieht einen Merge-Schritt vor. space hat mit
`lcov_merge.py` bereits einen.

### Zwei Regeln aus dem Entwurf zu generierten Dateien

**Was überall dasselbe bedeutet, wird eingebaut. Was projektabhängig ist, wird
gefragt.** `uv.lock` heißt überall dasselbe. `migrations/0001_x.py` heißt bei
Django "generiert" und bei handgeschriebenem SQL "gewollt" — also erkennt das
Werkzeug Django und fragt, statt eine Regel zu setzen.

**Ein Fehlalarm kostet mehr als eine fehlende Regel.** Bei mehrdeutigen Funden
meldet die Erkennung beide Möglichkeiten und lässt das Interview entscheiden.

## Relevanz

Jede erzeugte Kette trägt eine Abbildung von Pfadmustern auf Prüfarten. Eine
Markdown-Änderung soll keine Testsuite starten.

```
*.md, docs/**, *.txt        →  (nichts)
*.py                        →  lint, types, test, coverage
*.gd                        →  gdlint, gdformat, suite
*.cs                        →  format, build, test, coverage
pyproject.toml, uv.lock     →  test, coverage
wiki/**                     →  brain reindex
```

**Unbekannt heißt: alles laufen lassen.** Eine Endung, die nicht vorkommt, zieht
die volle Kette. Andersherum wäre der Fehler still, und ein Gate, das Prüfungen
unbemerkt überspringt, ist schlimmer als keines.

**Die Abbildung entscheidet zweimal an verschiedenen Mengen**: beim PostToolUse
über die eine geschriebene Datei, beim Stop über die Änderungsmenge seit dem
Basis-Commit. Deshalb liegt sie an einer Stelle, nicht in jedem Hook.

Alle drei Repos tun das heute schon verschieden — `kinds_for(path)` in
ultraloom, `Classification` im Stop-Gate von iam_backend,
`python_changes_pending` in space. Das zu vereinheitlichen ist der Kern.

## Antworten und Interview

**`answers.toml` hält Entscheidungen, alles andere ist Ausgabe.** Wer etwas
anders will, ändert die Antwort und lässt neu erzeugen. Das ist die Naht, an der
`sync` später ohne Zusammenführungslogik funktionieren kann.

```toml
# .ultraloom/answers.toml — written by `ultraloom init`, safe to edit by hand.
[project]
stacks          = ["godot", "gdscript", "csharp"]
docs_language   = "de"
commit_language = "en"

[gates]
coverage_threshold = 100          # reported only; enforced by the tool's own fail_under
tests_in_stop      = true
types_in_stop      = true

[gates.wiki]
mode   = "brain"                  # brain | neighbour_repo | none
bundle = "wiki/"

[policy]
protected_paths    = ["migrations/[0-9][0-9][0-9][0-9]_*.py"]
forbidden_commands = ["git push", "pip install"]

[relevance]
"*.md" = []
"*.cs" = ["format", "build", "test", "coverage"]
```

Bezeichner englisch, Prosa deutsch — wie überall in diesen Repos.

**Drei Sorten Fragen**: Bestätigung des Erkannten (Enter nimmt die Vorbelegung),
nicht Ablesbares (Commit-Sprache, Coverage-Schwelle, Wiki-Modus), und Fragen,
die nur bei Mehrdeutigkeit auftauchen (Django erkannt → Migrationen schützen?).

**Flags übersteuern jede Frage.** `--commit-language en --coverage 100 --yes`
lässt `init` unbeaufsichtigt laufen; das Interview ist nur der bequeme Weg zur
selben Datei.

**Der zweite Lauf** fragt nur, was noch nicht in `answers.toml` steht.

**Keine Fragen zu Werkzeugversionen.** Welche ruff-Version prüft, steht im
Projekt (`uv.lock`, `.csproj`) oder ist die des Rechners. Ein zweiter Pin in der
Antwortdatei wäre genau der doppelte Pflegeort, der bei ultraloom am 2026-08-28
gegen den Shim entschieden hat.

## Rendering

Jede Antwort hat genau einen Leser zur Laufzeit. Das verhindert Drift
*innerhalb* eines Projekts.

| Antwort | Datei | Wer liest sie |
|---|---|---|
| `[policy]` | `.ultraloom/policy.toml` | der Guard (P5), bis dahin `ultraloom policy hook` |
| `[gates]`, `[relevance]` | `.ultraloom/config.toml` | Python: post-edit und stop |
| `[project].commit_language` | `.ultraloom/config.toml` | der commit-msg-Hook |
| `[gates].coverage_threshold` | **nirgends** | siehe unten |
| Hook-Einträge | `.claude/settings.json` | Claude Code |
| brain-Anbindung | `.mcp.json` | Claude Code |

Jede erzeugte Datei trägt eine Kopfzeile
`generated from .ultraloom/answers.toml — edit that and re-run init`, und
`installed.toml` hält einen Hash der Antworten. Damit ist erkennbar, dass jemand
die Ausgabe statt der Quelle geändert hat.

### Die Coverage-Schwelle wird geprüft, nicht gesetzt

ultralooms README benennt den einen wirklich schädlichen Fehler dieses Systems:
eine grüne Zeile für eine Schwelle, die niemand erzwingt. Erzwungen wird sie
allein von `fail_under` in der Konfiguration des Coverage-Werkzeugs — und die
steht in `pyproject.toml`, die es überall schon gibt und die unter "vorhandene
Dateien in Ruhe lassen" fällt.

**`init` schreibt dort nicht hinein. Es prüft und meldet.** Fehlt `fail_under`,
wird der Coverage-Check gar nicht erst eingerichtet, mit Begründung. Lieber eine
fehlende Prüfung als eine behauptete.

### Der Merge in settings.json

Gleichheit wird an **(Ereignis, Matcher, Eigentümer)** gemessen, nicht an der
Befehlszeile. Eigentümer heißt: der Eintrag weist sich als von ultraloom erzeugt
aus.

| Fall | Verhalten |
|---|---|
| kein Eintrag für dieses Ereignis | anlegen |
| eigener Eintrag vorhanden | ersetzen — das ist die Aktualisierung |
| **fremder** Eintrag für dasselbe Ereignis | **nicht anlegen**, melden |

Der dritte Fall ist der wichtige. space hat einen eigenen PostToolUse-Hook;
einen zweiten danebenzustellen hieße, beide feuern zu lassen. Am 2026-08-27
hingen zwei parallele `quality.py`-Läufe über Nacht, weil sie sich um dieselben
Coverage-Dateien stritten.

## Verteilung

Zwei Rollen, zwei Verfahren.

| | Rolle | Verfahren |
|---|---|---|
| **ultraloom** | zustandsloses Werkzeug, je Projekt konfiguriert | **vendoren und pinnen** |
| **brain** | Dienst mit einem Index über alle Projekte | **finden, nicht pinnen** |

### ultraloom wird gevendort

Das Binary ist der Installer, der Klon ist die Laufzeit: die Hooks rufen
weiterhin Python-ultraloom auf, und der soll je Projekt eine festgelegte Version
sein. `init` klont ihn nach `.ultraloom/vendor/ultraloom` auf einen festen Ref
und notiert ihn in `installed.toml`; die Hook-Einträge in `settings.json` zeigen
relativ dorthin. `.mcp.json` ist davon nicht betroffen — dort steht brain, und
brain wird gefunden statt gevendort (siehe unten).

```
projekt/
  .ultraloom/
    vendor/ultraloom/     Klon auf festem Ref
    answers.toml          Entscheidungen
    installed.toml        ref, commit, Antworten-Hash, erzeugte Dateien
  .mcp.json               relative Pfade, versionierbar
```

Das löst drei Dinge auf einmal: kein absoluter Maschinenpfad in versionierten
Dateien, jedes Projekt auf seiner eigenen Version, und ein Upgrade ist ein
bewusster Schritt (`init --upgrade`) statt eines stillen Mitwanderns. Drift wird
sichtbar, weil der Ref dasteht.

### brain wird gefunden

brain hält *einen* Index über alle Projekte — `project/space` neben
`project/ecoflow`, dazu `knowledge` und `engineering/*`; am 2026-08-28 waren es
für space allein 189 indizierte Dokumente und 1586 Links. Ein Klon je Projekt
machte daraus getrennte Gehirne und zerstörte genau das, was `search` und
`neighbors` wertvoll macht.

Gesucht wird in dieser Reihenfolge: `uv tool`-Shim, PATH, `ULTRA_BRAIN_DIR`.
Gefunden → `.mcp.json`-Eintrag und Wiki-Hooks. Nicht gefunden → **die Wiki-Hooks
werden nicht eingerichtet**, mit Begründung.

**Das No-Substrate-Versprechen gilt für den Kern.** `wiki.mode = "brain"` bringt
Voraussetzungen mit: Python ≥3.14, uv, brain auf dem Rechner. Das steht hier,
damit es niemand beim ersten Hook-Fehler herausfinden muss.

## Das Wiki über brain

Was heute als Kopie in drei Repos liegt, hat brain schon:

| Heute | Künftig |
|---|---|
| `wiki_gate.py` in drei iam-Repos, zwei davon veraltet | `brain lint` im Stop-Hook |
| `generate_index.py` in space | `brain reindex` |
| `lint.py` (OKF) in space | `brain lint` |
| `wiki_guard.py` in ultra-brain | Policy-Regel: Schreibgefängnis auf die Wiki-Wurzeln |

**Diese Tabelle ist eine Absicht, keine geprüfte Äquivalenz.** Der Regelvergleich
vom 2026-08-28: brain kennt `broken-frontmatter`, `missing-type`, `dead-link`,
`absolute-link`, `outside-area`, `unlisted-area`, `orphan`, `no-sources`,
`stale`, `untouched`, `planned`/`implemented`/`implemented-without-commit`,
`long-planned`, `wrong-direction`, `conflict-count`. space' `lint.py` deckt
weniger OKF-Regeln ab, bringt aber **Gate-Mechanik** mit, die keine Regel ist:
einen Sitzungs-Fingerabdruck gegen Doppelblockieren und eine Prüfung gelöschter
Konzepte gegen Git. Regeln gehen an brain, die Gate-Mechanik bleibt im Hook —
**welche Fassung im Detail gewinnt, entscheidet P2.**

`neighbour_repo` bleibt als eigener Modus für die iam-Repos: ein Commit im
Nachbar-Repo seit Sessionstart ist eine andere Zusicherung als OKF-Struktur und
wird von `brain lint` nicht beantwortet.

**brain-Aufrufe sind kalt.** `brain lint` hängt am Stop (1–3× je Sitzung),
`brain reindex` an der Relevanzabbildung — nur `wiki/**` löst es aus. Ein
`uv run`-Aufruf kostet 300+ ms und darf nicht bei jedem Edit feuern.

**Ein minimaler OKF-Richtlinienblock gehört in P1s Vorlagensatz.** `brain lint`
als Gate ohne Richtlinientext hieße, Agenten mit Befunden zu blocken, ohne
irgendwo zu sagen, wie eine richtige Seite aussieht. Der ausgebaute Text ist P3.

**Vorhandene Integration**: `ultra-brain/hooks/git/ultraloom.sh` ruft ultraloom
bereits aus dem Nachbar-Checkout auf und dokumentiert dabei, dass `uvx --from`
an einem `#` im Pfad scheitert — nachgemessen am 2026-08-28: für
`uv run --directory` gilt das nicht, beide Formen liefern dasselbe. `init` fasst
vorhandene Git-Hooks in P1 nicht an, es meldet sie. Ob die Vendor-Fassung diese
Datei ablöst, gehört zu P2.

## Verhalten gegenüber vorhandenen Dateien

| Fall | Verhalten |
|---|---|
| Datei fehlt | anlegen |
| Datei ist da | in Ruhe lassen, am Ende benennen |
| `.claude/settings.json` | **einzige Ausnahme**: zusammenführen, siehe oben |

`init` läuft in leeren wie in gewachsenen Projekten. Die sechs bestehenden Repos
sind das eigentliche Ziel; ein Werkzeug, das nur leere Verzeichnisse bedient,
löst das Vereinheitlichungsproblem nicht.

## Fehlerverhalten

**Kein TTY heißt nicht warten.** `init` wird auch von einem Agenten aufgerufen,
und dort ist stdin leer. Ohne TTY und ohne vollständige Antworten bricht es ab,
nennt die fehlenden Antworten samt der Flags, und ändert nichts. Dieselbe
Lektion wie der unsichtbare `uv`-Fehler in space' `run.sh`, der eine ganze
Sitzung lang als "nichts zu melden" gelesen wurde.

| Exit | Bedeutung |
|---|---|
| 0 | fertig, oder nichts zu tun |
| 1 | eigener Fehler — kaputte Vorlage, unlesbares Verzeichnis. Nichts geschrieben |
| 2 | das Projekt sagt nein — fremder Hook im Weg, `fail_under` fehlt, Antwort fehlt ohne TTY |

**Ganz oder gar nicht.** Erst rendern, alles im Speicher, dann prüfen, dann
schreiben. Ein Fehler in der dritten Datei darf kein Projekt hinterlassen, in dem
zwei Dateien neu und die Hook-Einträge alt sind. `--dry-run` ist derselbe Lauf
ohne den letzten Schritt.

**Drei Randfälle:**

- **kein Git-Repo** → weitermachen, aber die Hooks weglassen, die Git brauchen
  (Stop-Gate mit Basis-Commit, commit-msg, Wiki-Pflicht), und das sagen
- **`settings.json` ist kaputtes JSON** → Exit 2. Nicht reparieren, nicht
  überschreiben
- **kein Stack erkannt** → nur Policy und `settings.json`. Ein leeres Projekt ist
  ein gültiger Startpunkt

## Tests

- **Erkennung**: reine Funktion, gegen Verzeichnis-Fixtures, ohne Unterprozesse
- **Rendering**: Golden Files — Antworten plus Fakten hinein, erzeugte Dateien
  byteweise vergleichen
- **settings.json-Merge**: Tabellentests über die drei Fälle, darunter einer, der
  beweist, dass ein fremder Eintrag *nicht* zu zwei feuernden Hooks führt
- **Go-Spur im Gate**: ultralooms Stop-Gate kennt heute nur Python-Prüfarten. Mit
  `cmd/init/` kommen `gofmt -l`, `go vet` und `go test` dazu, und die
  Coverage-Schwelle gilt über beide Sprachen. Das gehört in P1, nicht in P5

**Drei Windows-Fallen als Prüfliste an jede Vorlage**, alle am 2026-08-27/28
gemessen:

- mypys kompilierte `_mypyc`-DLL lädt unter langen Pfaden nicht ("Der Dateiname
  oder die Erweiterung ist zu lang") — erzeugte Werkzeugpfade dürfen nicht tief
  liegen
- ein dmypy-Daemon aus einer ephemeren `uvx`-Umgebung stirbt, wenn deren Cache
  weggeräumt wird (`AssertionError` in `mypy/modulefinder.py`) — Daemons nur
  hinter fester Installation
- ein global installiertes mypy sieht die Projektabhängigkeiten nicht ohne
  `--python-executable`

## Was P1 ausdrücklich nicht ist

- **kein `sync`** — keine Prüfsummen-Auswertung, keine Zusammenführung, keine
  Drift-Meldung. `init` schreibt nur `installed.toml`, damit `sync` später eine
  Vergleichsbasis hat
- **kein Guard-Binary** — der Policy-Hook bleibt vorerst Python. Das ist P5
- **kein konsolidierter Hook-Bestand** — P1 liefert einen minimalen Satz, damit
  das Ergebnis läuft. Welche Fassung je Hook gewinnt, ist P2
- **kein ausgebautes AGENTS.md** — das ist P3
