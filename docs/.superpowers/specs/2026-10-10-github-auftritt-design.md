# GitHub-Auftritt: README, Leitfäden, Repo-Metadaten

Datum: 2026-10-10 · Label: `release:none` · Implementierer: Modell `sonnet`

## Ziel

Wer loomux auf GitHub zum ersten Mal sieht, versteht in einer Minute, was es
ist und was es kann, und kommt in drei Schritten zu einer laufenden
Installation. Die Leitfäden für Installation, Ersteinrichtung und
`.loomux/config.toml` bilden einen klaren Pfad in beiden Sprachen.

**Gesagt (Auftrag):**

1. `README.md` und `README.de.md`: kurzer Pitch, Quickstart, genau ein
   Diagramm; der Abschnitt `Roadmap` bleibt.
2. Leitfäden für Installation, Ersteinrichtung und `.loomux/config.toml` unter
   `docs/en` und `docs/de` neu ordnen: klare Navigation, gleiche Dateinamen,
   Parität en/de je Datei geprüft.
3. Repo-Metadaten: neue Beschreibung (ohne den heutigen Klammerzusatz, der
   die beiden Vorgängerwerkzeuge nennt), Topics, Homepage, Social Preview. Der Plan druckt nur die
   Befehle; kein Agent ändert sie.
4. Was die Migration oder die Vorgängerwerkzeuge betrifft, fliegt aus README,
   Roadmap und den neu geordneten Leitfäden. Das README zählt die Funktionen
   auf und sagt, was sie ausmacht.
5. Der Feature-Branch committet `.claude/team-gate` mit der einzigen Zeile
   `sh ci/gate.sh`.

**Entscheidungen im Gespräch:**

- Homepage: `https://github.com/xidus90/loomux/tree/master/docs/en`.
- README schlank: Was nicht Pitch, Quickstart, Diagramm, Funktionsliste,
  Doku-Tabelle, Roadmap, Credits oder Lizenz ist, wandert nach `docs/` oder
  entfällt.
- Bereinigung gründlich: Die Roadmap verliert die Spalte `Stage`,
  Abhängigkeiten heißen nach Funktionen statt nach Stufenkürzeln, die Zeile
  `space on loomux alone` und alle Vorgänger- und Archivhinweise entfallen.
  Die neu geordneten Leitfäden verlieren ihre Stufenhistorie.
- Gepinnte Wiki-Quellen: Alle Änderungen an gepinnten Dateien liegen in
  einem Task, der als letzter läuft und mit einer Repin-Pause des Menschen
  endet (§3).
- Social Preview: Eine SVG-Quelle liegt im Repo, und der Mensch exportiert und
  lädt sie hoch.

## Randbedingungen aus dem Code

Vom Verifier geprüft (Befehle und Ausgaben in seinem Bericht vom 2026-10-10).

- **Wiki-Pins.** `lint/wiki-sources` (`internal/hooks/wikigate.go:85-138`,
  `internal/brain/wiki/sources.go:140-196`) vergleicht den Hash jeder
  zitierten Datei mit ihrer Zeile in `_identities.tsv`.
  - Zitiert werden `docs/de/getting-started.md` (von
    `docs/wiki/entities/qmd.md`) und `docs/de/configuration.md` (von
    `topics/datenschutz-und-kanaele.md`, `topics/schreibschranke.md` und
    `topics/datenmodell-und-bereiche.md`). Ebenso zitiert sind
    `docs/de/cli-reference.md`, `hooks.md`, `architecture.md` und
    `benchmarks.md`.
  - Gemessen: Eine angehängte Zeile in `docs/de/getting-started.md` ergibt
    `[wiki-sources:register-stale] … cited by entities/qmd.md`, und
    `check precommit` endet mit exit 1. Es gibt kein `.loomux/armed.toml`,
    also ist jede Lane scharf (`internal/verify/armed.go:45-51`). Dieselbe
    Änderung in `docs/en/*`, `README*.md`, `docs/de/README.md` oder
    `docs/wiki/log.md` lässt die Lane grün.
  - Grün wird die Lane nur durch `loomux wiki repin`, und das verweigert der
    Guard Agenten, auch als `go run ./cmd/loomux wiki repin`
    (`internal/hooks/guard.go:652-657`). Das installierte Binary 1.3.0 kennt
    `wiki repin` noch nicht; der Mensch ruft es deshalb als
    `go run ./cmd/loomux wiki repin --root <worktree>` auf
    (`internal/cli/wikicmd.go:41-55`: liest keine Registry, geht auch in
    einem verknüpften Worktree).
  - Eine umbenannte zitierte Datei ergibt `source-missing`, und das heilt
    repin nicht.

  Daraus folgen diese Regeln:
  - Gepinnte Dateien werden **nicht umbenannt** und nicht verschoben.
  - `docs/{en,de}/cli-reference.md`, `hooks.md`, `architecture.md`,
    `flows.md` und `benchmarks.md` bleiben **unverändert**.
  - In `configuration.md` behalten alle bestehenden Überschriften beider
    Sprachen Wortlaut und Nummer, denn die eingefrorenen Dateien verlinken
    sie (z. B. `#verify-prüfketten--quality-gates`,
    `#guard-wie-genau-der-wächter-liest`,
    `#5-die-schreibschranke-und-das-memory-der-agenten`). Die einzige
    Ausnahme ist die `[graph]`-Überschrift (§2); auf sie verlinkt nichts.
  - `## Roadmap` bleibt in beiden READMEs wörtlich so stehen, denn
    `architecture.md`, `flows.md` und `hooks.md` verlinken
    `README*.md#roadmap`.
  - In `getting-started.md` darf neu nummeriert werden, denn kein Dokument
    verlinkt einen Anker dort.
- **Schon heute kaputt:** `docs/en/configuration.md:112` verlinkt
  `cli-reference.md#loomux-check-request---root-path---show--v`, die
  Überschrift endet aber inzwischen auf `[--arm]`. Das deutsche Gegenstück
  steht in Zeile 114. Der Pin-Task korrigiert beide Anker.
- **`go install …@latest` geht nicht.** Gemessen: „The go.mod file for the
  module providing named packages contains one or more replace directives"
  (`go.mod:31`). Die heutige „Recommended"-Option in `getting-started.md` §1
  und §2 ist also falsch.
- **Nur unter Windows:**
  - das maschinenweite Binary und damit der Teil `binary` von `loomux init`
    (`internal/selfupdate/install.go:23`),
  - die Selbstaktualisierung durch `serve` und `loomux upgrade`
    (`internal/selfupdate/update.go:88`),
  - die Hook-Einträge, die `init` schreibt
    (`"${LOCALAPPDATA}/loomux/bin/loomux.exe"`,
    `internal/setup/hostfile/table.go:23`),
  - die Worktree-Spiegel (Junctions).

  Der Satz „Linux / macOS: Fully supported (symlinks, …)" in
  `getting-started.md:13` ist falsch.
- **Zahlen:** 35 ms ist das Budget des Hook-Pfads, kein gemessener
  Kaltstart (`docs/en/benchmarks.md`; `docs/en/README.md` sagt richtig
  „Sub-35ms pre-tool guards"). Die 100 % Abdeckung gelten je Funktion; eine
  Ausnahme braucht `//coverage:exempt <reason>` (`AGENTS.md:48-49`).
- **Namenscheck.** `TestTheRepositoryNamesNoPredecessor` prüft jede
  getrackte Datei, auch ein SVG und diese Spec samt Plan. Die Namen der
  Muster in `internal/namecheck/references.go:17-19` sind nur an den
  Ausnahmen in `references.go:107-128` erlaubt. Im README erlaubt ist
  allein der **Herkunftssatz** des Code-Graphen, also der Satz, auf den die
  Regex `ideaLine` (`references.go:101-103`) passt. Heute steht er in
  `README.md:345` in einem Abschnitt, der entfällt. Spec, Plan und neue
  Dateien schreiben keinen dieser Namen aus; sie verweisen auf die Regex.
- **Verweise auf README-Abschnitte:** Auf „Opening a pull request" zeigen
  nur `AGENTS.md:60-61`, `CONTRIBUTING.md:48` und
  `.github/pull_request_template.md:20`. Auf andere README-Abschnitte, die
  entfallen, zeigt nichts; der einzige weitere eingehende Anker ist
  `#roadmap`.
- **Social Preview** lässt sich weder über `gh` noch über REST oder GraphQL
  setzen, nur über Settings → General → Social preview → Edit. Erlaubt sind
  PNG, JPG oder GIF unter 1 MB; empfohlen sind 1280×640
  (<https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/customizing-your-repositorys-social-media-preview>).
- **`gh repo edit`** (gh 2.102.0) kennt `--description`, `--homepage`,
  `--add-topic` und `--remove-topic` mit Kommaliste. Erlaubt sind höchstens
  20 Topics aus Kleinbuchstaben, Ziffern und Bindestrichen mit höchstens 50
  Zeichen.
- **`release:none`** passt: `docs`- und `chore`-Commits haben die Stufe
  none (`internal/conventional/conventional.go:84-105`), `fix` dagegen
  patch. Jeder Commit dieses Plans ist deshalb `docs:` oder `chore:`, auch
  die Reparatur des kaputten Ankers. Einen CHANGELOG-Eintrag gibt es nicht.

## Entwurf

### 1. README (en und de, gleich aufgebaut)

Reihenfolge der Abschnitte:

1. Titel, Sprach- und Doku-Links (wie heute, Zeile 3).
2. **Pitch:** ein Satz als Untertitel und zwei bis drei Sätze, was loomux für
   wen tut, ohne Superlative und ohne Vorgänger. Dazu Kernaussagen als
   Liste:
   - ein Binary ohne Python- oder Node-Laufzeit,
   - Hook-Entscheidungen innerhalb eines Budgets von 35 ms,
   - 100 % Testabdeckung je Funktion (eine Ausnahme braucht eine
     Begründung im Code),
   - keine Telemetrie; ins Netz geht loomux nur für Updates über `gh` und
     auf ausdrückliche Befehle,
   - Windows zuerst: Binaries gibt es auch für Linux und macOS, getestet
     wird unter Windows. Hook-Einrichtung und Selbstaktualisierung gibt es
     nur unter Windows.

   Der Namensabsatz bleibt.
3. **Quickstart (Windows)** in drei Schritten, jeder als ein Codeblock.
   Voraussetzung (ein Satz davor): GitHub CLI installiert und
   `gh auth login` erledigt; `gh` übernimmt den Download und später die
   Selbstaktualisierung durch `serve`.
   1. Das Release-Binary mit `gh release download` direkt nach
      `$bin = "$env:LOCALAPPDATA\loomux\bin"` laden, mit `Get-FileHash`
      gegen `SHA256SUMS` prüfen (Vergleich mit `-eq`, denn `Get-FileHash`
      liefert Großbuchstaben) und in `loomux.exe` umbenennen. Dann lädt
      `init` nicht ein zweites Mal (`internal/setup/plan.go:97-98`: liegt
      das Binary dort, plant `init` keine Aktion des Teils `binary`).
   2. In einer beliebigen Sitzung im eigenen Projekt (`cd <projekt>`)
      `& "$env:LOCALAPPDATA\loomux\bin\loomux.exe" init` aufrufen. `$bin`
      gilt nur im ersten Block.
   3. Mit `& "$env:LOCALAPPDATA\loomux\bin\loomux.exe" status` prüfen.

   Die vollen Pfade sind nötig, denn nichts setzt den Ordner auf `PATH`.
   `install.md` zeigt, wie man ihn auf Wunsch dauerhaft in den
   Benutzer-`PATH` aufnimmt.

   Darunter steht ein Satz: Linux und macOS, der Bau aus dem Quellcode und
   alle Einzelheiten stehen in `install.md`, der nächste Schritt in
   `getting-started.md`.
4. **Ein Diagramm:** das heutige „5 Pillars"-Flowchart, gestrafft. Es nennt
   keine Stufen und keine Vorgänger. Gestrichelte Kanten bleiben für
   Spezifiziertes, das noch nicht gebaut ist, mit dem Satz darunter. Im
   README steht sonst kein Mermaid-Block.
5. **Funktionen:** eine Tabelle, je Funktion eine Zeile: Name, was sie
   ausmacht, Link in die Doku. Aufgeführt wird nur Gebautes:
   - Guard und Schreibschranke
   - Prüfkette und Gate
   - Code-Graph
   - Wiki und Second Brain
   - MCP-Dienst
   - Flows
   - Worktree-Spiegel (nur Windows)
   - Releases und Selbstaktualisierung (nur Windows)

   Grundlage ist die heutige CLI-Übersicht, README.md:189-329.
6. **Dokumentation:** die Tabelle der Leitfäden, beginnend mit dem Pfad
   Installation → Ersteinrichtung → Konfiguration. Die Beschreibung der
   Benchmarks nennt keine Vorgänger mehr („against predecessor binaries",
   `README.md:370`; ebenso in der Tabelle von `docs/en/README.md:17` und
   `docs/de/README.md:17`, „gegenüber den Vorläufer-Programmen").
7. **Roadmap:** Die Überschrift `## Roadmap` bleibt wörtlich. Die Zeilen
   bleiben bis auf die folgenden Änderungen:
   - Die Spalte `Stage` bzw. `Stufe` entfällt. Die Zeilen, deren Stufe heute
     „… follow-up" heißt, behalten ihre Beschreibung.
   - Die Zeile `space on loomux alone` entfällt. Von ihr hängt keine Zeile
     ab.
   - In `Brain web app` entfällt der Klammerhinweis auf die Web-App des
     Vorgängers und das Archiv-Release.
   - Abhängigkeiten nennen Funktionen. Eine gebaute Funktion trägt ✅:

     | heute | neu (en) |
     |---|---|
     | `Flow A ✅` | `Flow core without models ✅` |
     | `Flow B, C`, bloßes `Flow` | `Flow runtime` |
     | `1b-2 ✅` | `MCP service ✅` |
     | `W1` | `Web OS shell` |
     | `G4a ✅` | `Graph navigation ✅` |
     | `G4b ✅` | `Blast radius of a diff ✅` |
     | `G5a ✅` | `Python in the code graph ✅` |
     | `G5c ✅` | `GDScript in the code graph ✅` |

     Deutsch entsprechend: „Flow-Kern ohne Modelle ✅", „Flow-Laufzeit",
     „MCP-Dienst ✅", „Web-OS-Shell", „Graph-Navigation ✅",
     „Blast-Radius eines Diffs ✅", „Python im Code-Graph ✅", „GDScript im
     Code-Graph ✅".
   - Stufen im Freitext werden umformuliert, nicht stumpf ersetzt. Die
     Grep-Prüfung des Verifiers hat genau diese vier gefunden:
     - „which stage B decides" (Zeile Flow runtime) → „which is still to be
       decided" bzw. „was noch zu entscheiden ist";
     - „the building blocks of stage C" (Flow runtime) → „the building
       blocks it still needs" bzw. sinngemäß;
     - „Flow B also takes a diagnosis run without a model
       (`run --no-model`)" (Flow runtime) → Zukunft: „it will also take …"
       bzw. „sie wird auch … nehmen";
       `--no-model` gibt es noch nicht;
     - „reads Python since G5a" (TypeScript/TSX) → „that already reads
       Python" bzw. „der schon Python liest".

   Der Schlusssatz, der auf den deutschen Zwilling verweist, bleibt.
   Bewusst hingenommen: `architecture.md` („stages G5b and G5d … on the
   roadmap") und `hooks.md` („Planned (stage W1)") nennen weiter Kürzel, die
   in der Roadmap nicht mehr stehen. Diese Dateien bleiben eingefroren.
8. **Credits:** eine Zeile mit dem Herkunftssatz in heutigem Wortlaut
   (`README.md:345` bzw. `README.de.md:352`, unverändert kopiert, damit
   `ideaLine` weiter passt).
9. **Lizenz** wie heute.

Was raus muss und wohin es geht:

| Heute im README | Neues Ziel |
|---|---|
| Core Workflows (3 Diagramme) | entfällt; der Inhalt steht in `hooks.md`, `architecture.md` und `cli-reference.md`, verlinkt aus der Funktionsliste |
| CLI Reference (Befehlsübersicht) | entfällt; `cli-reference.md` deckt jede Zeile ab |
| Architectural Decisions | entfällt (Fusionsgeschichte); „keine Telemetrie" wandert in den Pitch, der Herkunftssatz in die Credits |
| Releases: „Its first run by a human is still pending" (`README.md:432`, `README.de.md:440`) | entfällt; der Quickstart beschreibt genau diesen Lauf |
| Architecture Principles | entfällt; Kernaussagen im Pitch, der Rest steht in `architecture.md` |
| Specifications & Internal Working Papers | entfällt (Archiv) |
| Releases: Download, Prüfsumme, `NOTICE.md`, maschinenweites Binary, Kanäle und Upgrade | `docs/{en,de}/install.md` |
| Releases: Labeltabelle, Versionierung, Beta-Releases | `docs/{en,de}/releases.md` |
| Opening a pull request | `docs/{en,de}/releases.md`, Abschnitt „Opening a pull request" bzw. „Einen Pull Request öffnen" |
| Setting up releases (maintainers) | `docs/{en,de}/releases.md` |

Regel: Aus den Abschnitten, die verschoben werden, geht keine Tatsache
verloren. Aus den Abschnitten, die entfallen, darf nur fehlen, was die
Migration, die Vorgänger, die Stufen oder Doppelungen betrifft. Jede andere
Tatsache steht danach im Zieldokument oder belegbar schon in `docs/`. Das
gilt für beide Sprachen.

Bekannte Fälle:
- „The endpoint must stay on the loopback …; no proxy is taken from the
  environment and no redirect is followed" (`README.md:283-284`) steht
  nicht in der Doku. Der Pin-Task ergänzt ihn in `configuration.md` unter
  „Machine-wide settings" bzw. dem deutschen Gegenstück, wenn er dort noch
  fehlt.
- „Requires Node.js … node-gyp" ist eine Abgrenzung zum Vorgänger und
  entfällt.

Die Verweise ziehen nach:
- `AGENTS.md:60-61` → `docs/en/releases.md`, „Opening a pull request".
- `.github/pull_request_template.md:20` ebenso.
- `CONTRIBUTING.md:48` verlinkt `docs/en/releases.md#opening-a-pull-request`
  und für die Labels `docs/en/releases.md` statt des README.
- Die Roadmap-Regel in `AGENTS.md:104-107`: Aus „each row with its stage,
  dependencies and priority where it has them" wird „each row with its
  dependencies and priority where it has them".

### 2. Leitfäden `docs/en` und `docs/de`

Dateien je Sprache, gleiche Namen:

| Datei | Status | Inhalt |
|---|---|---|
| `README.md` | umgebaut | Navigationsseite. Oben steht **Loslegen** bzw. **Get started** als nummerierter Pfad ① `install.md` → ② `getting-started.md` → ③ `configuration.md`. Darunter die Gruppen **Referenz** (`cli-reference.md`, `configuration.md`, `hooks.md`), **Konzepte** (`architecture.md`, `flows.md`) und **Projekt** (`releases.md`, `benchmarks.md`, `benchmarks/matrix.md`, `open-source-matrix.md`). „Core Philosophy" bzw. sein Zwilling bleibt. |
| `install.md` | neu | Voraussetzungen (auch `gh` mit `gh auth login`); was nur unter Windows geht (maschinenweites Binary, Teil `binary` von `init`, Selbstaktualisierung, Worktree-Spiegel) und dass `init` unter Linux/macOS nicht vorgesehen ist: Der Teil `binary` scheitert, und `.mcp.json`, die Hook-Einträge und die Git-Hooks werden nicht geschrieben, sondern als fehlgeschlagen gemeldet, weil sie `${LOCALAPPDATA}/loomux/bin/loomux.exe` aufriefen (`internal/setup/facts.go:115-123`, `apply.go:85-118`). Wer es dort trotzdem nutzt, setzt das Modul `hooks` auf `none` und wählt im Modul `base` mit `each` die Teile `binary` und `mcp-json` ab (nur interaktiv; für `base` gibt es kein Flag, also kein `--yes`); dass Linux/macOS gebaut, aber nur unter Windows getestet werden; den Ordner auf Wunsch in den Benutzer-`PATH` aufnehmen; Release-Binary mit Assetnamen je OS und `SHA256SUMS` (Windows: `Get-FileHash`, sonst `sha256sum`); `NOTICE.md`; maschinenweites Binary unter `%LOCALAPPDATA%\loomux\bin` mit MCP-Eintrag; Selbstaktualisierung durch `serve`, `loomux upgrade` und die Kanäle beta und stable; Bau aus dem Quellcode; `loomux --version`. Kein `go install …@latest`, und der Satz zu Symlinks fällt weg. |
| `getting-started.md` | umgebaut, **gepinnt (de)** | Nur Ersteinrichtung, neu nummeriert: 1. `loomux init`, 2. die Konfiguration prüfen, 3. die Einrichtung prüfen, 4. Agent-Harness-Integration, 5. einen Bereich von Hand registrieren, 6. nächste Schritte. Voraussetzungen und Installation wandern nach `install.md`; an ihrer Stelle steht ein Satz mit Link. |
| `configuration.md` | ergänzt, **gepinnt (de)** | Siehe unten. |
| `releases.md` | neu | Labeltabelle; Versionierung; stabile Releases und Pre-Releases; `CHANGELOG.md`; Beta-Releases über den Workflow; Einen Pull Request öffnen; Maintainer-Setup (Labels, GitHub App, Rulesets, Runner, verlorenes Release). |

Änderungen an `configuration.md`:
- Oben steht ein thematisch gruppiertes Inhaltsverzeichnis:
  - Schutz: `[policy.*]`, `[guard]`
  - Prüfungen: `[verify]`, `armed.toml`
  - Projekt: `[project]`, `[modules]`, `[commit]`, `[worktree]`
  - Bereiche und Wiki
  - Flows: `[agent]`, `[flow]`
  - Ablage und Schreibschranke
- Dazu kommt ein unnummerierter Abschnitt „Your first day" bzw. „Der erste
  Tag": was `loomux init` schreibt, was man zuerst anpasst und wie eine
  Änderung über `loomux config set … --propose` → `loomux config proposals`
  → `loomux config apply` läuft.
- Die bestehenden Abschnitte bleiben an ihrem Platz und behalten Wortlaut
  und Nummer ihrer Überschriften. Ausnahmen:
  - Die Überschrift „`[graph]` — there is none, and stage G2a settled why"
    heißt neu „`[graph]` — there is none", der deutsche Zwilling
    entsprechend. Ihr Absatz verliert die Stufenkürzel.
  - In „reference implementation's rule" (en ca. Z. 725) und dem deutschen
    Gegenstück fällt der Bezug auf die Referenzimplementierung weg; die
    Regel selbst bleibt.
  - `version = "0.1.0-fusion"` im annotierten Beispiel wird
    `version = "1.0.0"`.
  - Der kaputte Anker `…---show--v` wird zu `…---show--v---arm`, in beiden
    Sprachen.
  - Der Satz zu Loopback und Proxy kommt hinzu (siehe §1).
- „when its predecessor is not in the run" (Z. 223/228) meint die
  Reihenfolge der Lanes und bleibt stehen.

Navigation: Jede Seite des Pfads Loslegen (`install.md`, `getting-started.md`,
`configuration.md`) bekommt unter dem Titel und am Ende je eine Zeile, z. B.
`← [Installation](install.md) · [Übersicht](README.md) · [Konfiguration](configuration.md) →`.
Die erste und die letzte Seite haben nur eine Richtung.

Nicht angefasst werden `architecture.md`, `cli-reference.md`, `hooks.md`,
`flows.md`, `benchmarks*`, `open-source-matrix.md` und `docs/wiki/` außerhalb
des Pin-Tasks.

**Parität je Datei.** Für jedes Paar `docs/en/X` und `docs/de/X`, das dieser
Plan anlegt oder ändert, und für `README.md`/`README.de.md` gilt dasselbe
Gerüst:
- gleiche Folge von Überschriftsebenen;
- gleiche Zahl von Codeblöcken, je Block dieselbe Sprache und dieselbe
  Zeilenzahl (`mermaid`-Blöcke nur gleiche Zahl);
- Inhalt der Blöcke: Befehle, Schlüssel und Flags sind gleich. Abweichen
  dürfen Kommentare (ganze Zeilen und angehängte), Zeichenkettenwerte und
  lokalisierte Beispielnamen und -pfade (z. B. `my-project`/`mein-projekt`,
  `<you>`/`<du>`), wie es die bestehenden Dateien heute schon tun. Das ist
  mechanisch nicht sauber zu trennen; der Verifier prüft es per
  Sichtvergleich von `diff` der Blockinhalte;
- gleiche Zahl von Tabellenzeilen;
- dieselben Linkziele, wobei Pfade auf die Sprache der Datei zeigen und
  Anker übersetzt sein dürfen.

Geprüft wird mit Shell-Befehlen (`awk`, `diff`), die der Plan vorgibt; das
Überschriften-Gerüst überspringt dabei `#`-Zeilen innerhalb von Codeblöcken.
Heute erfüllen alle betroffenen Paare diese Regel (Verifier, Runde 3). Ein
neuer Go-Check entsteht nicht. Jeder relative Link und jeder Anker in den
geänderten Dateien muss auflösen. Der Plan gibt dafür einen Prüfbefehl vor,
der GitHubs Slug-Regel nachbildet.

**Reihenfolge:** `install.md` und `releases.md` entstehen vor den Tasks, die
auf sie verlinken (README, `docs/*/README.md`, `CONTRIBUTING.md`,
`AGENTS.md`).

### 3. Gepinnte Dateien: ein Task mit Repin-Pause

Ein Task, abhängig von allen anderen Doku-Tasks, ändert
`docs/en/getting-started.md`, `docs/de/getting-started.md`,
`docs/en/configuration.md` und `docs/de/configuration.md`. Danach:

1. Er liest die zitierenden Wiki-Seiten (`entities/qmd.md`,
   `topics/datenschutz-und-kanaele.md`, `topics/schreibschranke.md`,
   `topics/datenmodell-und-bereiche.md`) gegen den neuen Text. Was dort aus
   der alten Fassung zitiert oder belegt war und jetzt woanders steht oder
   anders lautet, passt er im Prosatext der Seite an. Die `sources:`-Pins
   ändert er nicht. Keine dieser vier Seiten wird selbst gepinnt.
  (`entities/qmd.md` nennt seinen Pin auf `getting-started.md` nur im
  Frontmatter; dort ist voraussichtlich nichts anzupassen.)
2. Er schreibt einen Eintrag in `docs/wiki/log.md` (`docs/wiki/_schema.md:9`).
3. Er committet noch nicht. Mit der Ausgabe von
   `go run ./cmd/loomux check lint` belegt er, dass nur `lint/wiki-sources`
   rot ist, mit `register-stale` für genau die beiden gepinnten Dateien.
4. Er setzt seinen Task auf `WAITING:` und bittet den Menschen, im Worktree
   `go run ./cmd/loomux wiki repin --root <worktree>` auszuführen. Kein
   `--no-verify`, kein Commit an der Lane vorbei.
5. Danach führt er das Gate aus, es muss grün sein. Erst dann committet er
   die Doku, die Wiki-Prosa, den Log-Eintrag und die von repin geänderten
   Dateien (`_identities.tsv` und die Pins der zitierenden Seiten). Der
   Plan legt die Aufteilung in Commits fest.

Alle anderen Tasks berühren keine gepinnte Datei; ihr Gate bleibt ohne Pause
grün.

### 4. Repo-Metadaten (nur Befehle, ausgeführt vom Menschen)

Der Plan enthält am Ende einen Abschnitt „Für den Menschen" mit:

```sh
gh repo edit xidus90/loomux \
  --description "One Go binary that guards AI coding agents: hooks and a write barrier, a check chain, a code graph and a project wiki. Local, no Python or Node runtime." \
  --homepage "https://github.com/xidus90/loomux/tree/master/docs/en" \
  --add-topic mcp,developer-tools,cli,coding-agents,guardrails
```

Die bestehenden Topics (`ai-agents`, `claude-code`, `code-graph`,
`git-hooks`, `go`, `knowledge-base`, `llm`) bleiben, zusammen sind es 12.

Social Preview: Ein Task legt `docs/assets/social-preview.svg` an, 1280×640.
Es enthält den Namen, den Pitch-Satz und die Kernaussagen, nur
Systemschriften, keine externen Ressourcen, kein eingebettetes base64 und
keinen Vorgängernamen. Der Abschnitt „Für den Menschen" sagt dazu:

1. Das SVG im Browser bei Zoom 100 % und Windows-Anzeigeskalierung 100 %
   öffnen, oder in den DevTools mit devicePixelRatio 1; sonst wird der
   Screenshot z. B. 2560×1280. Dann als PNG mit 1280×640 und unter 1 MB
   exportieren, per Screenshot des Elements oder mit einem SVG-Konverter.
2. Settings → General → Social preview → Edit → „Upload an image…".

### 5. `.claude/team-gate`

Die Datei enthält genau die Zeile `sh ci/gate.sh`. Der Planer committet sie
zusammen mit Spec und Plan auf dem Feature-Branch, bevor der Lauf startet;
`claude-team.ps1` startet ohne sie nicht.

## Nicht im Umfang

- Kein neuer Go-Code, kein neuer Check im Gate, kein Linkchecker im Repo.
  Einzige Go-Datei, die sich ändert: der Kommentar in
  `internal/hosts/hostio_internal_test.go:10`, der sich auf den
  Install-Block des README beruft („names Windows, Linux and macOS"). Er
  verweist danach auf `docs/en/install.md`.
- Keine Änderung an `architecture.md`, `cli-reference.md`, `hooks.md`,
  `flows.md`, `benchmarks*` und `open-source-matrix.md`.
- `AGENTS.md` ändert sich nur an den beiden genannten Stellen. Seine
  Erwähnungen des Archiv-Release bleiben, denn es sind Agentenanweisungen,
  kein Auftritt.
- `CONTRIBUTING.md` bleibt englisch und ändert nur den Verweis.
- Keine GitHub Pages.
- Kein Agent führt `gh repo edit` aus, keiner pusht.

## Erfolgskriterien

- Beide READMEs haben genau einen Mermaid-Block, einen Quickstart mit drei
  Schritten und eine Funktionsliste. Die Überschrift `## Roadmap` steht
  wörtlich, die Roadmap hat keine Spalte `Stage`/`Stufe`, und keines der
  beiden nennt einen Vorgänger (außer im Herkunftssatz der Credits), ein
  Archiv-Release oder ein Stufenkürzel (`G5a`, `4e`, `Flow B`, `W1`,
  `1b-2` …).
- `docs/en` und `docs/de` enthalten dieselben Dateinamen, auch `install.md`
  und `releases.md`, und die Paritätsbefehle laufen für jedes geänderte Paar
  ohne Unterschied.
- Jeder relative Link und jeder Anker in den geänderten Dateien löst auf.
- `go test ./internal/namecheck/` ist grün.
- `sh ci/gate.sh` ist auf dem Feature-Branch grün, auch
  `lint/wiki-sources` nach dem Repin.
- `.claude/team-gate` enthält genau die Zeile `sh ci/gate.sh`.
- Der Plan enthält die `gh repo edit`-Befehle und die Schritte für die
  Social Preview; ausgeführt hat sie niemand.
