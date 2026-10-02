# loomux Stufe 4e: die Umstellung, vom LLM vorbereitet — Design

**Stand:** Entwurf vom 2026-09-29, gegen `origin/master` 709266ee (v5.3.0), vom Nutzer
freigegeben am 2026-09-30; seit dem Entwurf umgesetzt bis zu den Piloten: Die Werkzeuge sind gebaut, `ecoflow` (2026-09-29) und `space`
(2026-09-30) sind umgestellt und gemessen; die Welle lief nach der Lane-Probation am
2026-10-01/02 über fünf Ziele (siehe „Entschieden und berichtigt in der Welle“).
**Bezug:** löst Stück B und ändert Stück C der Spec `2026-09-28-loomux-stufe-4e-design.md`
(dort A `area check`, B Checkliste, C Aufräum-PR); Fusions-Spec Nachtrag #19, #24,
#25 und der neue #26.

## Anlass

Der Nutzer hat am 2026-09-29 den Zuschnitt geändert:

- Alle Projekte haben Hooks, Brain und Graph eingerichtet, alle Guards sind aktiv.
- Das Wiki liegt immer im Projekt. Liegt es woanders, wird es zuerst dorthin migriert.
- Die Einrichtung übernimmt das LLM. Was von Hand bleibt, bereitet es so vor, dass der
  Nutzer einen Befehl ausführt oder eine Datei umbenennt.
- Benchmarks, kalt und warm (Mittel aus fünf Warmläufen), für alles, was schon in den
  Projekten läuft, danach für die neue Umsetzung, ein Vergleich je Projekt und ein
  anonymisierter Bericht in der Doku von loomux.
- „Alle Projekte“ heißt: die zehn Bereiche der Registry, `iam_backend`, `iam_frontend`,
  `iam_workers`, die es dort nicht gibt, **und** der Vault `brain-knowledge` mit eigener Form
  (Hooks und Guards, Übersetzung seiner vier Manifeste, kein Wiki-Umzug).

## Gemessen am 2026-09-29 (Ausgangslage)

Quellen: `%LOCALAPPDATA%\brain\registry.toml`, die Sonden des Wächters
(`loomux hook pre-tool-use` mit einer Write-Nutzlast je Ziel, nichts geschrieben),
`git ls-files`, `test -f`.

| Projekt | Bereich in der Registry | Agent darf schreiben | Wiki heute | alte Einrichtung im Projekt |
|---|---|---|---|---|
| `space` | `project/space`, readonly, workspace | ja | **die Seiten liegen schon im Projekt** (`docs/wiki`, 217 Dateien, 204 Markdown, versioniert); im Vault `91 Projekte/space` nur ein Gerüst (5 Dateien: `_identities.tsv`, `_schema.md`, `audit.md`, `index.md`, `log.md`), im Zustandsverzeichnis ein Bestand von 72 Dateien (`.agents`, `.ultraloom`, `docs`, `wiki`, `handovers`, `graph.json`, `layout.json`, `_identities.tsv`) | `.ultraloom/config.toml`, `.claude/settings.json`, `.brain.toml` |
| `iam_wiki` | `project/iam-wiki`, readonly, ohne workspace | **nein** | Vault `91 Projekte/iam-wiki` | `.claude/settings.json`, `.brain.toml`; kein `.ultraloom` |
| `#Obsidian/AI` | `project/obsidian-ai`, readonly, ohne workspace | **nein** | Vault `91 Projekte/obsidian-ai` | **kein Git-Repo**; wird **nicht umgestellt** (Entscheidung 2026-09-29) |
| `ecoflow` | `project/ecoflow`, workspace | ja | im Vault `91 Projekte/ecoflow` nur ein Gerüst (5 Dateien wie bei `space`); das Projekt hat `docs/`, ein Ordner `wiki` fehlt; Inhalt und Überschneidung nicht gemessen | `.brain.toml`; weder `.ultraloom` noch `.claude/settings.json` |
| `ultra-brain` | `project/ultra-brain`, workspace | ja | im Projekt `docs/wiki` | nicht gemessen |
| `ultraloom` | `project/ultraloom`, workspace | ja | im Projekt `docs/wiki` | `.ultra-brain/config.toml` |
| `iam_backend`, `iam_frontend`, `iam_workers` | **nicht registriert** | **nein** | **keines** (weder `wiki` noch `docs/wiki`) | `.ultraloom/config.toml`, `.claude/settings.json` mit `ulguard`, `ulguard post-edit`, `brain wiki-gate` und einem im Projekt vendorten Python (`.ultraloom/vendor/ultraloom`, `uv run … ultraloom hook …`); kein `.brain.toml` |
| `brain-knowledge` (Vault) | `knowledge`, `engineering/python`, `engineering/craft`, `hub` | ja | `90 Wiki` und die Unterbäume, bleiben | `.brain.toml` mit fünf Schlüsseln, die kein Leser liest (Messung von `area check`) |

Weiter gemessen:

- **Berichtigung (2026-09-29, nach der ersten Fassung dieser Spec):** Der Pfad `wiki` der
  Registry zeigt bei `space`, `iam-wiki`, `obsidian-ai` und `ecoflow` in den Vault, aber das
  sagt nicht, wo die Seiten liegen. Bei `space` liegen sie im Projekt, der Vault-Ordner ist
  ein Gerüst. Was bei den übrigen Zielen wo liegt, misst Task 7 des Plans (Dateizahlen je Ort
  und die Überschneidung der Dateinamen); bis dahin gilt für keines der Vault-Wikis, dass
  „das Wiki zieht um“.

- Der Wächter verweigert dem Agenten `iam_backend/docs/wiki/…` (Exit 2, „outside every
  writable tree“) und `%LOCALAPPDATA%\loomux\registry.toml.new`; er lässt `ecoflow/docs/wiki/…`
  und ein `apply.sh` im Scratchpad durch, auch mit `loomux init` als Text darin.
- Die alten Werkzeuge liegen noch: `~/go/bin/ulguard.exe`, `brain.exe`, `ulflow.exe`.
  `loomux dev bench hooks` ruft beliebige Programme (`Argv[0]` eines Falls), die Baseline
  ist also messbar.
- `init` registriert ein Projekt nur, wo an der Wurzel noch kein Bereich steht
  (`internal/setup/parts.go:41`, `plan.go:157`: „skipped; the registry has an area at this
  root already“). Für `iam_backend`, `iam_frontend`, `iam_workers` genügt also `init`; für
  die schon registrierten (`space`, `iam_wiki`, `ecoflow`, `ultra-brain`, `ultraloom`) muss
  die Registry geändert werden, soweit sich `readonly` oder der Wiki-Pfad ändert.
- Der Bereich `hub` (`91 Projekte/.brain.toml`) indiziert `**/*.md`: nach dem Umzug melden
  `reindex` und `reconcile` auf ihm Löschungen, das ist erwartet.
- Der Vault-Ordner `91 Projekte` enthält je Projekt einen Ordner (`ecoflow`: `index.md`,
  `_schema.md`, `audit.md`, `log.md`, `_identities.tsv`) und im Wurzelpfad `index.md`,
  `graph.json`, `layout.json`, `_identities.tsv`; keine einzelnen Zeigerseiten. Ein
  verwaister Ordner `ultraloom/` liegt dort.

## Zielzustand je Projekt

- `.loomux/config.toml` mit `[area]`, den Modulen Hooks, Brain und Graph an und allen
  Guards aktiv (`[guard]` und die Richtlinie des Projekts bleiben in Kraft).
- Bereich in der Registry, **schreibbar** und mit `workspace = true` **(Annahme, dem
  Nutzer zu nennen:** wo ein Agent programmiert, ist der ganze Baum beschreibbar; die
  Guards beschränken trotzdem Befehle und geschützte Pfade, `workspace` öffnet nur die
  Schranke für Pfade).
- Wiki im Projekt: `docs/wiki`, oder der vorhandene Ort (`wiki`), wo das Projekt schon
  einen hat (`wiki.Root` findet beide). Es zieht mit: `_identities.tsv` (sonst neue
  `doc_id` für jede Seite), `index.md`, `log.md`, `audit.md`, `_schema.md`, `graph.json`.
  **Nach der Berichtigung oben zieht nur, was neben den Seiten liegt und im Projekt fehlt**
  (Register, Schema, Audit, Log, Katalog, Graph); die Seiten selbst liegen bei `space` schon im
  Projekt. Das Skript **fügt nur hinzu**: eine Datei, die im Ziel fehlt, wird kopiert; eine, die
  dort schon liegt und gleich ist, bleibt; eine, die dort liegt und abweicht, bricht das Skript
  ab, und der Nutzer entscheidet je Datei (behalten oder ersetzen), bevor es neu läuft.
  **Für Register, Kataloge und `graph.json` (nicht für die Seiten, siehe die Berichtigung oben) hängt es vom Bereich ab, welche Kopie die lebende ist:** loomux schreibt in einen
  read-only-Baum nie; für `space` und `iam_wiki` liegen Register, Kataloge und `graph.json`
  im Zustandsverzeichnis (`areas/project-<name>`, für `space` 69.144 Byte, 26.09.), für
  `ecoflow` (workspace) ist die Kopie im Vault die lebende. Wer die falsche kopiert, vergibt
  neue `doc_id` oder legt Dubletten an. `iam_backend`,
  `iam_frontend` und `iam_workers` bekommen kein Bündel (`--brain none`); ihr Wiki ist
  `iam_wiki` (Entscheidung 2026-09-29, unten).
- In der Übersetzung des Manifests steht `[maintenance] on_merge = true`, sonst nimmt
  `merge-hook install` den Bereich nicht mit.
- Claude- und Antigravity-Hooks, Git-Hooks (`core.hooksPath`, commit-msg, pre-commit) und
  der post-merge-Hook eingerichtet; die alten Einträge (`ultraLoomOwned`,
  `ultraloom-wiki-guard`, `brain guard`, `ulguard`, `.ultraloom/`, `.brain.toml`,
  `.ultra-brain/`) entfernt.
- Ausnahme `#Obsidian/AI`: **wird nicht umgestellt** (Entscheidung 2026-09-29). Es bleibt
  ein read-only-Bereich mit dem Wiki im Vault, ohne Git-Repo, und fällt aus der Welle und
  aus den Benchmarks. Der Pfad trägt zweimal `#`; wer ihn später anfasst, setzt jede Zeile
  mit ihm in Anführungszeichen.
- Die read-only-Bereiche verschwinden für die übrigen Projekte: `space` und `iam-wiki`
  werden schreibbar. Damit entfällt die Überschneidung ihrer Wiki-Pfade mit `hub`, die die
  Registry als Fehler der Spec 3.2 duldet, bis auf die von `obsidian-ai`, das bleibt.

**Der Vault (`brain-knowledge`) hat eine eigene Form.** Seine Bereiche `knowledge`,
`engineering/python`, `engineering/craft` und `hub` sind keine Projekte und ziehen nicht um;
aber er ist einer der vier Wirte der bisherigen Specs und braucht Hooks und Guards, sonst
bliebe er ohne sie und „alle Guards aktiv“ wäre nicht erfüllt. Für ihn: `init`, die Übersetzung
seiner vier Manifeste (Wurzel, zwei unter `92 Engineering`, `91 Projekte`), das Entfernen der
alten Einträge, **kein** Wiki-Umzug. Er ist das neunte Ziel.
**Folge, dem Nutzer schon jetzt genannt:** Obsidian sieht die verschobenen Wikis nicht mehr,
sie liegen außerhalb des Vaults.

## Ablauf: alles Schreibende steht im `apply.sh`

Der Agent darf für vier der acht Projekte nichts schreiben (`iam_backend`, `iam_frontend`,
`iam_workers`, `iam_wiki`) und nie in die Registry. Deshalb gibt es keine Sonderfälle: Der
Agent liest, bereitet vor und prüft danach nur lesend; jede Schreibaktion steht im
`apply.sh`, das der Nutzer mit einer Zeile aufruft (`! sh "<pfad>/apply.sh"`). **Ablageort
der Vorbereitung:** `.superpowers/switchover/<projekt>/` unter loomux (git-ignoriert,
beschreibbar), nicht der Scratchpad der Sitzung: der stirbt mit ihr, das Skript muss den
nächsten Tag überleben. Mit „Vorschlag“ ist hier eine **ganze Datei** gemeint, die das
`apply.sh` an ihren Platz kopiert; `config set --propose` setzt einen einzelnen Schlüssel
und schriebe in das Projekt, das dem Agenten bei vier Projekten verboten ist. Das Skript
des Menschen schreibt sie, der Agent nie (AGENTS.md).

1. **Baseline (Phase 0) für alle Ziele, bevor das erste `apply.sh` läuft.** Sie ist nur
   lesend. Sie muss vor dem ersten Skript liegen, weil `merge-hook install` sonst in einem
   anderen Projekt loomux-Hooks anlegt und dessen Baseline verfälscht.
2. **Diagnose, nur lesend:** `area check` je Manifest, `init --dry-run`,
   `init --detect-only`, Inhalt und Überschneidung der Wiki-Orte, Zustand von Git.
3. **Vorbereitung** (Dateien im Ablageort, nichts davon liegt schon am Platz):
   - die Übersetzung des alten Manifests in eine vollständige `.loomux/config.toml`
     (das Umlegen von Hand aus Stück B entfällt; `area check` zeigt, was verloren ginge);
   - die geänderte Registry **nur für bereits registrierte Projekte**, aus der lebenden
     Datei erzeugt, mit deren Prüfsumme; das Skript bricht ab, wenn die lebende Registry
     inzwischen eine andere Prüfsumme hat (nie acht Fassungen auf Vorrat);
   - eine Liste der abgelösten Einträge je Projekt (aus `init --detect-only` und der
     Ablösetabelle in `internal/hooks/status.go`) und ein kleines Entfernungsskript für
     die **lebende** `settings.json`;
   - das `apply.sh` aus einer Vorlage, in Git-Bash-Syntax, ohne Rückfragen.
4. **`apply.sh`, mit `--check` (schreibt nichts, zeigt, was es täte); jeder Schritt hat
   einen Existenzwächter, damit ein zweiter Lauf nichts ändert und ein `git rm` unter
   `set -e` nicht scheitert:**
   1. bricht ab, wenn das Projekt oder der Vault ungesichert ist (`git status`), oder wenn
      `brain-knowledge` keinen Commit oder keinen Remote hat;
   2. Registry ersetzen (Sicherung als `registry.toml.bak`), nur für bereits registrierte
      Projekte;
   3. Wiki ins Projekt kopieren, aus der **lebenden** Kopie (Zustandsverzeichnis oder
      Vault), Dateizahl und Prüfsummen vergleichen;
   4. `.loomux/config.toml` aus der Datei des Ablageorts anlegen;
   5. `init --yes` (dabei registriert es unregistrierte Projekte selbst);
   6. **danach** die abgelösten Einträge aus der lebenden `settings.json` entfernen
      (`init` hängt eigene an und lässt die alten stehen; eine vorher erzeugte Zieldatei
      würde `init`s Einträge überschreiben) und die alten Dateien (`.ultraloom/` samt dem
      vendorten Python, `.brain.toml`, `.ultra-brain/`) entfernen; danach meldet der Status
      nichts Abgelöstes mehr;
   7. **erst nach bestandener Prüfung von 3:** im Vault den alten Wiki-Ordner mit `git rm`
      und einem Commit entfernen; der Ordner `areas/project-<name>` im Zustandsverzeichnis
      wird umbenannt (`.alt`), nicht gelöscht.
   `merge-hook install` steht **nicht** im Skript je Projekt: es wirkt auf die ganze Registry
   und gehört in ein Abschlussskript, das nach dem letzten Ziel einmal läuft.
5. **Prüfung durch den Agenten, nur lesend:** `status`, `doctor`, `area check` (Exit 0), ein
   erlaubter und ein verweigerter Edit (Nutzlast durch den Wächter), Suche über MCP; nach dem
   Umzug eines Vault-Wikis ein `reindex` des Bereichs `hub` (Löschungen erwartet).
6. **Phase 2: Benchmarks der neuen Umsetzung.**

## Zuschnitt: zwei Piloten, dann eine Welle

`ecoflow` ist der **saubere** Pilot: registriert, `workspace`, 49 Dateien, im Vault nur ein Gerüst,
ohne alte Hook-Einrichtung im Projekt (kein `.ultraloom`, keine `settings.json`). Er übt
Registry, Wiki-Umzug, `config.toml` und `init` auf leerem Boden. **Er beweist nicht:** das
Entfernen alter Hooks, die Bereinigung der `settings.json` und einen Alt/Neu-Vergleich der
Hooks (seine Baseline ist praktisch leer, nur `brain.exe`).
`space` ist der zweite Pilot und der erste **volle** Durchgang: alle drei Altdateien, ein
Seiten schon im Projekt (`docs/wiki`) und ein Gerüst im Vault, dazu ein Bestand im Zustandsverzeichnis, `workspace`, für den Agenten schreibbar. Er übt genau, was
`ecoflow` nicht kann. Danach die Welle mit dem, was beide gelehrt haben: `ultra-brain`,
`ultraloom`, `iam_backend`, `iam_frontend`, `iam_workers` (ohne Vault-Wiki), dann `iam_wiki`
(mit Vault-Wiki, selbst ein Wiki: 94 von 105 Dateien Markdown; ob dort `docs/wiki` oder die
Wurzel gilt, entscheidet die Welle), zuletzt der Vault. Danach das Abschlussskript
(`merge-hook install`).

## Benchmarks

**Wann:** je Ziel zweimal, Phase 0 für alle **vor dem ersten** `apply.sh`, Phase 2 danach.
Die Such-Messungen hängen am Zustand des qmd-Dienstes; alt und neu laufen gegen denselben.
**Wie:** `loomux dev bench hooks` (ruft beliebige Programme): ein Kaltlauf (der erste Lauf
eines Falls), danach **fünf** Warmläufe. Berichtet werden Mittelwert **und** Median; Min und
Max stehen im Bericht von `dev bench hooks` und im JSON des Vergleichs, die Markdown-Tabelle
von `dev bench compare` zeigt sie nicht (die vorhandenen Messungen in `docs/en/benchmarks.md` nutzen den Median über 20 Läufe;
die Wahl der fünf ist die des Nutzers und wird beim Bericht genannt). Payload je Fall wie in
den bisherigen Messungen (ein `Edit` auf eine Markdown-Datei des Projekts).
**Was:**

- **alt und neu vergleichbar:** Guard (`ulguard` gegen `loomux hook pre-tool-use`),
  Post-Edit, Stop, Session-Start, Subagent, commit-msg, pre-commit-Gate, Suche
  (`brain search` gegen `loomux search`), `reindex`, `reconcile`, Wiki-Prüfung, Graph-Abfragen
  soweit das Alte sie hatte;
- **nur neu:** Graph und Blast, `area check`, `convert`, `fetch`, `config`, `init --dry-run`
  und was sonst hinzugekommen ist;
- **weggefallen:** was das Alte hatte und das Neue nicht mehr braucht (`hooks.tsv`,
  `[relevance]`, der Python-Einstiegspunkt).
Das Inventar entsteht nicht von Hand: `init --detect-only` und die Ablösetabelle in
`internal/hooks/status.go` liefern die alten Einträge, die Hook-Konfiguration des Projekts
die neuen.
**Berichte:** je Projekt `docs/.superpowers/parity/bench-4e-<projekt>.md` (Arbeitspapier,
deutsch, mit echtem Namen): je Fall alt, neu, Faktor, und drei Listen „schneller/langsamer“,
„neu“, „weggefallen“. In `docs/en/benchmarks.md` und `docs/de/benchmarks.md` steht **ein**
anonymisierter Bericht mit allen neun Zielen (acht Projekte und der Vault) als „Beispielprojekt 1“ bis „9“ mit den
Messwerten und einer Zusammenfassung („was wie viel schneller ist, was hinzukam, was
wegfiel“). Die Zuordnung Name zu Nummer steht nur in den Arbeitspapieren. Dateizahlen sind
Quasi-Kennungen; der Nutzer nimmt das in Kauf.

## Was von Stück B und C bleibt

- **Stück B (Checkliste):** Block 4 (Deklarationen der read-only-Bereiche) **entfällt**,
  ebenso Block 5 (`init`, alte Einträge entfernen: steht im `apply.sh`). Bleiben als
  Menschenschritt vor der Welle: Block 1 (`brain-knowledge` hat einen Remote und ist
  committet), Block 2 (`[index]` für `project/loomux`, die Entscheidungen #1, #3, #8 aus
  `parity/artefakte-nach-lebensdauer.md`) und Block 3 (Maschinenzustand: `last-run.txt`,
  `merge-events`, `qmd-collections.json`). Block 6 (Rauchtest) ist der Schritt 5 oben.
- **Stück C (Aufräum-PR):** Die Aside-Auflösung in `ResolvedAreaDir` (Entscheidung 1′ aus #25)
  **bleibt**: `#Obsidian/AI` ist nach der Entscheidung vom 2026-09-29 der einzige read-only-Bereich,
  dessen Bestand im Zustandsverzeichnis getauscht wird, und ohne 1′ gäbe ein erschlagener Tausch
  für ihn den ganzen Tresor auf. Die Tasks 6 und 7 des Plans bleiben, mit dem einen Bereich
  als Testfall. Der Rest (Rückfälle entfernen, Ratschläge umstellen) wartet, bis die acht
  Projekte umgestellt sind. `lock.Recover` im Schreibpfad von `approve` bleibt, bis
  `moveStock` mit seinem Rückfall entfällt.
- **Stück A (`area check`)** bleibt unverändert; es dient der Diagnose in Schritt 2.

## Entschieden und berichtigt in der Welle (2026-10-01/02)

Die Welle ist gelaufen; Ablauf und Messung stehen in `parity/stufe-4e.md`,
Messung 10. Was dabei von dieser Spec abwich:

- **`ultra-brain` und `ultraloom` werden nicht umgestellt** (Entscheidung des
  Nutzers vom 2026-10-01): Beide Repos werden nach der Migration gelöscht. Die
  Welle umfasste `iam_backend`, `iam_frontend`, `iam_workers`, `iam_wiki` und
  den Vault; „acht Projekte und der Vault“ oben und „alle neun Ziele“ unter
  „Benchmarks“ gelten mit dieser Einschränkung (sieben Ziele samt Piloten).
- **`--brain=none` registriert keinen Bereich und installiert keinen
  Merge-Hook.** Der Satz „`init --yes --brain none`, Hooks, Guards und Graph
  wie bei allen anderen“ stimmt für Hooks, Guards und Graph; Bereich und
  Merge-Hook gehören zum Brain-Modul und entfallen. Folge: Die
  Schreibschranke verweigert in den drei Code-Projekten jeden Edit (gemessen
  am 2026-10-02). Folgepunkt unten.
- **Die `iam_*` trugen vendorte ultraloom-Formen** in ihren eigenen Git-Hooks
  (`uv run ultraloom check all`, `ulinit`). `iam_backend` und `iam_workers`:
  Variante A, das Skript entfernt die alten Git-Hooks, ein zweites `init`
  legt die von loomux an. `iam_frontend`: husky weicht `.githooks` (H2).
- **`iam_wiki` ist das gemeinsame Wiki der drei Code-Projekte, kein eigenes
  Projekt:** nur ein Wiki-Bereich, mit eigenem Skript (die Vorlage kann `init`
  nicht auslassen), Registry-Eintrag `wiki` auf die Wurzel, `readonly`
  entfällt, kein `init`, keine Hooks, keine Lanes; die `doc_id` sind
  übernommen. `[layout] wiki` lehnt die Wurzel ab, die Wurzel ist nur über die
  Registry das Wiki: **kein Wiki-Lint** dort (Folgepunkt).
- **Der Vault bekommt ein pre-commit-Profil nur mit Lint:** Eine Art, für die
  es nichts zu prüfen gibt, lässt das Tor mit Exit 1 fallen, und ohne Code
  hat der Vault nur Lint. `verify.profiles` ist über `config set` nicht
  setzbar; der Mensch hat es von Hand eingetragen. Das Stop-Profil hat
  dieselbe Lücke (gemessen: Exit 1).

**Folgepunkte für loomux** (ohne Stufe in der Fusions-Spec; Liste in der
Akte): eine Art ohne Prüfgegenstand darf ein Profil nicht fallen lassen;
`verify.profiles` über `config set`; ein Bereich (oder ein Schreibrecht) für
Projekte unter `--brain=none`; ein Wiki im Wurzelverzeichnis eines Repos;
`dev bench cases` ohne Hook.

## Entschieden am 2026-09-30 (Nutzer)

- **Die Welle startet erst nach der Schonfrist je Lane.** Die Piloten haben gezeigt, dass ein
  frisch eingerichtetes Tor an altem Code sofort rot ist (`ecoflow`: 83 alte ruff-Befunde am
  Umstellungs-Commit; `space`: der Stop-Hook blockierte jede Sitzung, `wiki lint` meldete 790
  Befunde). Reihenfolge: dieser Zweig wird gemergt und veröffentlicht, dann das Feature aus
  `2026-09-30-loomux-lane-probation-design.md` (eine Lane warnt nur, bis sie einmal grün war),
  **dann** die Welle mit der installierten Version, die beides hat. Das `apply.sh` legt dann in
  seinem Konfigurationsschritt die leere `.loomux/armed.toml` mit an. Der Satz „Danach die
  Welle“ unter „Zuschnitt“ gilt mit dieser Einschränkung.
- **Die Piloten bekommen die Schonfrist nicht von selbst.** `ecoflow` und `space` haben ihre
  `.loomux/config.toml` schon; `init` legt die Zustandsdatei der Schonfrist nur an, wo vor dem
  Lauf keine Konfiguration stand, und das `apply.sh` schreibt dort keine Konfiguration mehr.
  Beide bleiben scharf, bis ein Mensch dort `loomux gate disarm --all` ausführt; das ist ein
  Schritt der Welle, sobald die Schonfrist installiert ist.
- **Für `space` keine Überschreibung der Test-Lane.** Das GDScript-Test-Preset ist gestrichen;
  das vollständige Tor von `space` bleibt sein eigener pre-commit-Hook.

## Entschieden am 2026-09-29 (Nutzer)

- **Guard-Modus:** `default` (nicht `strict`). Der Plan prüft, dass `init --yes` ihn setzt
  und die Wächterregeln des Projekts nicht lockert.
- **`workspace = true`** überall, wo ein Agent programmiert; die Guards beschränken
  trotzdem Befehle und geschützte Pfade.
- **`#Obsidian/AI`: nicht umstellen.** Es bleibt read-only mit dem Wiki im Vault und ohne
  Repo und fällt aus Welle und Benchmarks; es sind acht Projekte, nicht neun. Folge: 1′ aus
  #25 bleibt für diesen einen Bereich.
- **`iam_wiki` ist das Wiki aller `iam_*`-Projekte.** Die Seiten liegen heute im
  Wurzelverzeichnis von `iam_wiki` (`architecture/`, `projects/backend|frontend|workers/` …,
  gemessen: 106 Markdown-Dateien außerhalb von `docs/`, davon 80 versioniert) und bleiben dort:
  In der Registry-Datei (Schritt 2 des Skripts) bekommt `project/iam-wiki` `path` und `wiki`
  auf das Wurzelverzeichnis von `iam_wiki`, `workspace = true` und kein `readonly` mehr.
  `WikiSrcs` ist leer; das Gerüst in `docs/wiki` und der Vault-Ordner `91 Projekte/iam-wiki`
  (ein Gerüst aus fünf Dateien) tragen keine Seite. `iam_backend`, `iam_frontend` und
  `iam_workers` bekommen **keinen eigenen Bereich und kein Wiki**: `init --yes --brain none`,
  Hooks, Guards und Graph wie bei allen anderen. Wer dort eine Wikiseite schreibt, schreibt in
  `iam_wiki`.
- **Die Vorgaben von loomux sind bindend**, weil die alten Werkzeuge wegfallen. Gemessen
  gegen das Gerüst, das `wiki.InitBundle` schreibt (`internal/brain/wiki/scaffold.go`):
  In den Vault-Ordnern `91 Projekte/{ecoflow,space,iam-wiki,ultraloom}` ist jede Datei
  byte-gleich mit der Vorgabe. Nur `index.md` weicht ab, und das ist ein Katalog, den die
  alten Werkzeuge erzeugt haben. Bei `ultraloom` weichen die Projektdateien nur in der
  Arbeitskopie ab, durch CRLF; im Index stehen sie mit LF und gleich. Folge: `WikiSrcs` ist
  für jedes Ziel leer, und aus dem Vault wird nichts übernommen. Die vier Vault-Ordner
  entfernt nicht ein `apply.sh`: Als `vault_old` verlangt `Render`, dass der Ordner eine
  Quelle ist, und dann bräche der Abgleich an `index.md` ab; als `old_files` des Vaults
  lehnt `Render` den Pfad ab, weil `91 Projekte` ein Leerzeichen enthält. Der Mensch
  entfernt sie nach dem letzten Projekt der Welle mit einem `git rm -r` und einem Commit im
  Vault. Den Befehl nennt der Agent.
  Echter Inhalt im Projekt bleibt stehen, etwa `space/docs/wiki/log.md` und
  `space/docs/wiki/index.md`. Ein erzeugter alter Katalog im Projekt (`index.md` von
  `ecoflow`) bleibt stehen, bis loomux ihn neu schreibt; `InitBundle` schreibt nie über eine
  vorhandene Datei. Das Gerüst `iam_wiki/docs/wiki` ist nach der Entscheidung oben nicht das
  Wiki und fällt über `old_files` weg. Ob das Wurzelverzeichnis von `iam_wiki` ein
  `_schema.md` und `audit.md` braucht, zeigt der erste `lint` nach der Umstellung.

## Offene Punkte, vor dem Plan zu klären

- ~~Wo `iam_backend`, `iam_frontend`, `iam_workers` heute ein Wiki haben~~ — entschieden
  2026-09-29: keines, `iam_wiki` ist ihr Wiki (siehe oben). Offen bleibt, ob die
  Schreibschranke einer Sitzung in `iam_backend` Schreiben nach `iam_wiki` zulässt, sobald
  `project/iam-wiki` ein workspace ist; zu proben im Pilot, bevor die `iam_*` umgestellt werden.
- Welche Module `init --yes` ohne Antwortflags einschaltet (Hooks, Brain und Graph an?) und
  welchen `[guard] mode` es setzt; das Ziel ist `default`.
- Ob `init --dry-run` die ganze Ziel-`settings.json` zeigt (dann genügt sie als Vergleich);
  bis dahin gilt das Entfernen aus der lebenden Datei (Schritt 6).
- Ob `area add` einen vorhandenen Bereich ändern kann (`readonly` weg, `wiki` neu). Wenn ja,
  entfällt der Registry-Schritt; der erste Pilot klärt es, bis dahin gilt die Registry-Datei.
- Ob die Session-Hooks der `iam_*` (`uv run … ultraloom hook …`) außer dem vendorten Python
  im Projekt noch etwas aus `~/go/bin` oder dem Suchpfad brauchen (für die Baseline).
- Ob die Sicherung vor dem Löschen im Vault genügt (Commit) oder ein Tag nötig ist.

## Tests und Abnahme

- Das `apply.sh` ist Shell-Text; es trägt `set -e`, prüft seine Voraussetzungen und kennt
  `--check`. Berichtigt am 2026-09-30: Die erste Fassung sagte hier, das Skript habe keinen
  Go-Test. Gebaut ist es anders: `internal/switchover/render_test.go` prüft die Vorlage mit
  `sh -n` auf Syntax, und `internal/switchover/apply_test.go` fährt das gerenderte Skript in
  39 Tests durch Wegwerfwelten (`git init`, ein Vault, ein Projekt, ein Zustandsverzeichnis),
  im `--check`-Modus, der nichts ändert, und im echten Lauf samt zweitem Lauf und Abbruch.
- Der Pilot ist bestanden, wenn ein zweiter Lauf des Skripts nichts ändert, `area check`
  Exit 0 gibt, die Prüfung des Agenten grün ist und beide Benchmarkphasen einen Bericht
  ergeben.
- 100 % Coverage je Funktion gilt für jeden neuen Go-Code (die Vorlagenprüfung, falls sie
  im Go-Baum liegt); die Mutationsrunde der Stufe gilt für ihn.

## Nachtrag 2026-10-02: „kein eigener Bereich“ heißt „kein Brain-Bereich“

Der Abschnitt „Entschieden am 2026-09-29“ sagt für `iam_backend`, `iam_frontend` und
`iam_workers` „keinen eigenen Bereich und kein Wiki“ und zugleich „Hooks, Guards und Graph
wie bei allen anderen“. Beides zugleich ging mit `init` bis 6.1.0 nicht: Ohne
Registry-Eintrag verweigert der Schreibwächter jeden Write im Projekt (gemessen am
2026-10-02). Gemeint ist **kein Brain-Bereich**: kein Wiki, keine Erklärung `[area]`, kein
Merge-Hook, aber ein Registry-Eintrag mit `path` und `workspace = true` ohne `wiki`.

Die drei Projekte haben diesen Eintrag seit dem 2026-10-02, von Hand eingetragen. Künftig
schreibt ihn `init --yes --brain=none` selbst (Teil `workspace`, Spec
`2026-10-02-loomux-brain-none-workspace-design.md`); an einer Wurzel mit Eintrag lässt `init`
ihn unberührt und nennt den Grund als Notiz. Der Reindex überspringt die drei Bereiche,
weil sie kein `[area]` erklären, und meldet bei jedem Lauf `skipping project/<name>: …`.
